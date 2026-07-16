package router

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/security"
)

const defaultHTTPTimeout = 10 * time.Minute

// HTTPClient is a pure-Go OpenAI-compatible chat completions client.
type HTTPClient struct {
	cfg        ModelConfig
	httpClient *http.Client
}

// NewHTTPClient builds a client for cfg. httpClient may be nil for defaults.
func NewHTTPClient(cfg ModelConfig, httpClient *http.Client) *HTTPClient {
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = defaultHTTPTimeout
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	return &HTTPClient{cfg: cfg, httpClient: httpClient}
}

// Name implements LLMClient.
func (c *HTTPClient) Name() string { return c.cfg.Name }

// ChatCompletion implements LLMClient (non-streaming).
func (c *HTTPClient) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	req.Model = c.cfg.ModelID
	req.Stream = false
	req.StreamOptions = nil

	body, err := json.Marshal(req)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, body)
	if err != nil {
		return ChatResponse{}, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	// Cap error/success body reads to avoid memory blowups from misbehaving endpoints.
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("read body: %w", err)
	}
	if err := checkStatus(resp.StatusCode, respBody); err != nil {
		return ChatResponse{}, err
	}

	var result ChatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return ChatResponse{}, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

// ChatCompletionStream implements LLMClient with SSE parsing.
func (c *HTTPClient) ChatCompletionStream(ctx context.Context, req ChatRequest, onDelta func(StreamDelta) error) (ChatResponse, error) {
	req.Model = c.cfg.ModelID
	req.Stream = true
	if req.StreamOptions == nil {
		req.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, body)
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return ChatResponse{}, checkStatus(resp.StatusCode, respBody)
	}

	return consumeSSE(resp.Body, onDelta)
}

func (c *HTTPClient) newRequest(ctx context.Context, body []byte) (*http.Request, error) {
	base := c.cfg.ResolvedBaseURL()
	if base == "" {
		return nil, fmt.Errorf("empty base_url")
	}
	// Reject missing GCP project (unexpanded ${…} or expanded-empty path segment).
	if strings.Contains(base, "${GOOGLE_CLOUD_PROJECT}") || strings.Contains(base, "$GOOGLE_CLOUD_PROJECT") ||
		strings.Contains(base, "projects//locations") {
		return nil, fmt.Errorf("base_url missing GOOGLE_CLOUD_PROJECT — set GOOGLE_CLOUD_PROJECT for Vertex models")
	}
	if err := security.ValidateHTTPURL(base, true); err != nil {
		return nil, fmt.Errorf("base_url: %w", err)
	}
	url := strings.TrimRight(base, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	key := c.cfg.ResolvedAPIKey()
	if key != "" {
		httpReq.Header.Set("Authorization", "Bearer "+key)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range c.cfg.ExtraHeaders {
		httpReq.Header.Set(k, expandEnvPlaceholders(v))
	}
	return httpReq, nil
}

func checkStatus(code int, body []byte) error {
	if code == http.StatusOK {
		return nil
	}
	retryable := code == http.StatusTooManyRequests || code == http.StatusRequestTimeout ||
		code == http.StatusBadGateway || code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout || code >= 500
	// Redact any credentials a provider might echo in error bodies.
	return &APIError{
		StatusCode: code,
		Body:       security.Redact(string(body)),
		Retryable:  retryable,
		RateLimit:  code == http.StatusTooManyRequests,
	}
}

// streamChunk is the OpenAI SSE JSON object.
type streamChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role             string     `json:"role"`
			Content          string     `json:"content"`
			ReasoningContent string     `json:"reasoning_content"`
			ToolCalls        []ToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *Usage `json:"usage"`
}

func consumeSSE(r io.Reader, onDelta func(StreamDelta) error) (ChatResponse, error) {
	scanner := bufio.NewScanner(r)
	// Large tool-call argument streams can exceed the default 64K buffer.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 8*1024*1024)

	var (
		out          ChatResponse
		content      strings.Builder
		finishReason string
		toolCalls    = map[int]*ToolCall{}
	)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // ignore malformed keep-alives
		}
		if chunk.ID != "" {
			out.ID = chunk.ID
		}
		if chunk.Model != "" {
			out.Model = chunk.Model
		}
		if chunk.Usage != nil {
			out.Usage = *chunk.Usage
		}

		for _, ch := range chunk.Choices {
			delta := StreamDelta{}
			if ch.Delta.Content != "" {
				content.WriteString(ch.Delta.Content)
				delta.Content = ch.Delta.Content
			}
			if ch.Delta.ReasoningContent != "" {
				delta.ReasoningContent = ch.Delta.ReasoningContent
			}
			if ch.FinishReason != nil {
				finishReason = *ch.FinishReason
				delta.FinishReason = finishReason
			}
			for i, tc := range ch.Delta.ToolCalls {
				idx := i
				// Providers often send index on tool_calls; fall back to order.
				// OpenAI streams partial tool call fragments by index field.
				_ = idx
				mergeStreamToolCall(toolCalls, tc)
				delta.ToolCalls = append(delta.ToolCalls, tc)
			}
			if chunk.Usage != nil {
				u := *chunk.Usage
				delta.Usage = &u
			}
			if onDelta != nil {
				if err := onDelta(delta); err != nil {
					return ChatResponse{}, err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ChatResponse{}, fmt.Errorf("sse scan: %w", err)
	}

	msg := Message{Role: "assistant", Content: content.String()}
	if len(toolCalls) > 0 {
		// Stable order by ascending index.
		maxIdx := -1
		for i := range toolCalls {
			if i > maxIdx {
				maxIdx = i
			}
		}
		for i := 0; i <= maxIdx; i++ {
			if tc, ok := toolCalls[i]; ok {
				msg.ToolCalls = append(msg.ToolCalls, *tc)
			}
		}
	}
	out.Choices = []Choice{{
		Message:      msg,
		FinishReason: finishReason,
	}}
	return out, nil
}

func mergeStreamToolCall(m map[int]*ToolCall, tc ToolCall) {
	// Prefer explicit Index if we extend ToolCall later; for now key by len / id.
	// OpenAI partials often only fill Function.Arguments incrementally with Index in JSON.
	// We re-parse index from a side channel: if ID is set, find existing; else append slot.
	idx := len(m)
	if tc.ID != "" {
		for i, existing := range m {
			if existing != nil && existing.ID == tc.ID {
				idx = i
				break
			}
		}
		// If not found, use next free index.
		if m[idx] != nil && m[idx].ID != tc.ID {
			for i := 0; ; i++ {
				if m[i] == nil {
					idx = i
					break
				}
			}
		}
	} else if len(m) > 0 {
		// Continuation fragment without ID — append to last index.
		last := 0
		for i := range m {
			if i > last {
				last = i
			}
		}
		idx = last
	}

	if m[idx] == nil {
		cp := tc
		m[idx] = &cp
		return
	}
	if tc.ID != "" {
		m[idx].ID = tc.ID
	}
	if tc.Type != "" {
		m[idx].Type = tc.Type
	}
	if tc.Function.Name != "" {
		m[idx].Function.Name = tc.Function.Name
	}
	if tc.Function.Arguments != "" {
		m[idx].Function.Arguments += tc.Function.Arguments
	}
}

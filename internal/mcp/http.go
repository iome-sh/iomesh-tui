package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/security"
)

// DialHTTP connects to a streamable HTTP MCP endpoint, then initialize + tools/list.
// Accepts application/json and text/event-stream responses (SSE).
func DialHTTP(ctx context.Context, cfg ServerConfig, logger *slog.Logger) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("mcp: server %q: empty url", cfg.Name)
	}
	if logger == nil {
		logger = slog.Default()
	}
	allowLoop := true
	if cfg.AllowLoopback != nil {
		allowLoop = *cfg.AllowLoopback
	}
	if err := security.ValidateHTTPURL(cfg.URL, allowLoop); err != nil {
		return nil, fmt.Errorf("mcp: server %q url: %w", cfg.Name, err)
	}
	startup := time.Duration(cfg.StartupTimeoutSec) * time.Second
	if startup <= 0 {
		startup = 30 * time.Second
	}

	c := &Client{
		cfg:        cfg,
		logger:     logger,
		httpClient: &http.Client{Timeout: 0}, // per-call timeouts via context
		baseURL:    strings.TrimRight(cfg.URL, "/"),
		pending:    map[int64]chan rpcResponse{},
		maxOut:     DefaultMaxOutputBytes,
	}

	initCtx, cancel := context.WithTimeout(ctx, startup)
	defer cancel()
	if err := c.initialize(initCtx); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("mcp: initialize %s: %w", cfg.Name, err)
	}
	if err := c.refreshTools(initCtx); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("mcp: tools/list %s: %w", cfg.Name, err)
	}
	_ = c.refreshResources(initCtx)
	_ = c.refreshPrompts(initCtx)
	return c, nil
}

func (c *Client) callHTTP(ctx context.Context, method string, params any, result any) error {
	id := c.nextID.Add(1)
	rawParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  rawParams,
	})
	if err != nil {
		return err
	}

	resp, err := c.httpPost(ctx, reqBody, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		// Async accept without body — treat as success with empty result.
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("mcp http %s: status %d: %s", method, resp.StatusCode, security.Redact(string(b)))
	}

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	var rpcResp rpcResponse
	if strings.Contains(ct, "text/event-stream") {
		rpcResp, err = readSSEResponse(resp.Body, id)
		if err != nil {
			return err
		}
	} else {
		// application/json (or default)
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&rpcResp); err != nil {
			return fmt.Errorf("mcp http decode: %w", err)
		}
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("mcp rpc: %s", rpcResp.Error.Message)
	}
	if result != nil && len(rpcResp.Result) > 0 {
		if err := json.Unmarshal(rpcResp.Result, result); err != nil {
			return fmt.Errorf("mcp decode: %w", err)
		}
	}
	return nil
}

func (c *Client) notifyHTTP(ctx context.Context, method string, params any) error {
	rawParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
	})
	if err != nil {
		return err
	}
	resp, err := c.httpPost(ctx, reqBody, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 202 Accepted or 2xx with empty body are fine for notifications.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("mcp http notify %s: status %d: %s", method, resp.StatusCode, security.Redact(string(b)))
}

func (c *Client) httpPost(ctx context.Context, body []byte, expectJSON bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if expectJSON {
		req.Header.Set("Accept", "application/json, text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json, text/event-stream")
	}
	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}
	c.ApplyAuthHeaders(ctx, req.Header)
	c.mu.Lock()
	sid := c.sessionID
	c.mu.Unlock()
	if sid != "" {
		req.Header.Set("Mcp-Session-Id", sid)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mcp http post: %w", err)
	}
	if s := resp.Header.Get("Mcp-Session-Id"); s != "" {
		c.mu.Lock()
		c.sessionID = s
		c.mu.Unlock()
	}
	return resp, nil
}

// readSSEResponse scans an SSE body for a JSON-RPC response matching wantID.
func readSSEResponse(r io.Reader, wantID int64) (rpcResponse, error) {
	sc := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 4*1024*1024)

	var dataLines []string
	flush := func() (rpcResponse, bool, error) {
		if len(dataLines) == 0 {
			return rpcResponse{}, false, nil
		}
		payload := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		payload = strings.TrimSpace(payload)
		if payload == "" || payload == "[DONE]" {
			return rpcResponse{}, false, nil
		}
		var resp rpcResponse
		if err := json.Unmarshal([]byte(payload), &resp); err != nil {
			return rpcResponse{}, false, fmt.Errorf("mcp sse json: %w", err)
		}
		// Skip pure notifications (no id).
		if resp.ID == nil {
			return rpcResponse{}, false, nil
		}
		if coerceID(resp.ID) != wantID {
			return rpcResponse{}, false, nil
		}
		return resp, true, nil
	}

	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			// event boundary
			if resp, ok, err := flush(); err != nil {
				return rpcResponse{}, err
			} else if ok {
				return resp, nil
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue // comment
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			continue
		}
		// Some servers send bare JSON lines without SSE framing.
		if strings.HasPrefix(strings.TrimSpace(line), "{") {
			var resp rpcResponse
			if err := json.Unmarshal([]byte(line), &resp); err == nil && resp.ID != nil && coerceID(resp.ID) == wantID {
				return resp, nil
			}
		}
	}
	if err := sc.Err(); err != nil {
		return rpcResponse{}, err
	}
	// Final flush if stream ended without blank line.
	if resp, ok, err := flush(); err != nil {
		return rpcResponse{}, err
	} else if ok {
		return resp, nil
	}
	return rpcResponse{}, fmt.Errorf("mcp sse: no response for id %d", wantID)
}

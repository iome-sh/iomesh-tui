// Package router provides OpenAI-compatible LLM selection, execution, and
// cascading fallbacks for the I/O Mesh TUI agent harness.
//
// Design goals (aligned with the Grok Build Go rewrite):
//   - Pure Go core (net/http, encoding/json, context) — no vendor SDKs
//   - Default to cost-efficient coding models (DeepSeek V4 Flash / Pro)
//   - Escalate to premium models on complexity, failure, or explicit override
//   - Observable: cost estimates, usage logging, metrics hooks
package router

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Complexity classifies task stakes for heuristic routing.
type Complexity int

const (
	// ComplexityRoutine is exploration, simple edits, subagent fan-out.
	ComplexityRoutine Complexity = 0
	// ComplexityPlan is multi-step planning, multi-file reasoning.
	ComplexityPlan Complexity = 1
	// ComplexityHighStakes is risky edits, production ops, final reviews.
	ComplexityHighStakes Complexity = 2
)

// String returns a stable label for logs and metrics.
func (c Complexity) String() string {
	switch c {
	case ComplexityRoutine:
		return "routine"
	case ComplexityPlan:
		return "plan"
	case ComplexityHighStakes:
		return "high_stakes"
	default:
		return fmt.Sprintf("complexity(%d)", int(c))
	}
}

// TaskType is a free-form task category used by SelectModel heuristics.
// Common values: "routine", "plan", "edit", "subagent", "review".
type TaskType string

const (
	TaskRoutine  TaskType = "routine"
	TaskPlan     TaskType = "plan"
	TaskEdit     TaskType = "edit"
	TaskSubagent TaskType = "subagent"
	TaskReview   TaskType = "review"
)

// ModelConfig defines a single OpenAI-compatible LLM endpoint.
type ModelConfig struct {
	// Name is the logical name used in config, /model, and logs.
	Name string `toml:"name" json:"name"`
	// BaseURL is the API root, e.g. https://api.deepseek.com/v1
	BaseURL string `toml:"base_url" json:"base_url"`
	// APIKey is optional inline credential; prefer EnvKey for secrets.
	APIKey string `toml:"api_key" json:"api_key,omitempty"`
	// EnvKey is the environment variable holding the API key.
	EnvKey string `toml:"env_key" json:"env_key,omitempty"`
	// ModelID is the provider model identifier sent in the request body.
	ModelID string `toml:"model_id" json:"model_id"`
	// CostTier is a relative cost multiplier (lower = cheaper). Used when
	// per-token prices are unset.
	CostTier float64 `toml:"cost_tier" json:"cost_tier"`
	// InputCostPerM is USD per 1M input tokens (cache miss).
	InputCostPerM float64 `toml:"input_cost_per_m" json:"input_cost_per_m"`
	// OutputCostPerM is USD per 1M output tokens.
	OutputCostPerM float64 `toml:"output_cost_per_m" json:"output_cost_per_m"`
	// CacheHitCostPerM is USD per 1M cached input tokens (optional).
	CacheHitCostPerM float64 `toml:"cache_hit_cost_per_m" json:"cache_hit_cost_per_m"`
	// MaxContext is the context window in tokens.
	MaxContext int `toml:"max_context" json:"max_context"`
	// Capabilities tags models for heuristics (fast, coding, tool-calling, premium).
	Capabilities []string `toml:"capabilities" json:"capabilities"`
	// Priority orders fallback chains (lower = preferred / try first).
	Priority int `toml:"priority" json:"priority"`
	// Timeout is optional per-model HTTP timeout; zero uses client default.
	Timeout time.Duration `toml:"timeout" json:"timeout,omitempty"`
	// ExtraHeaders are merged into every request (e.g. attribution tags).
	ExtraHeaders map[string]string `toml:"extra_headers" json:"extra_headers,omitempty"`
}

// ResolvedAPIKey returns APIKey or the value of EnvKey.
// For Vertex models (capability "vertex"), also tries GOOGLE_OAUTH_ACCESS_TOKEN
// when VERTEX_API_KEY is unset (both hold a short-lived gcloud access token).
func (m ModelConfig) ResolvedAPIKey() string {
	if m.APIKey != "" {
		return expandEnvPlaceholders(m.APIKey)
	}
	if m.EnvKey != "" {
		if v := getenv(m.EnvKey); v != "" {
			return v
		}
	}
	// Vertex OpenAI-compat uses OAuth access tokens, not a long-lived API key.
	if hasCapability(m.Capabilities, "vertex") {
		if v := getenv("VERTEX_API_KEY"); v != "" {
			return v
		}
		if v := getenv("GOOGLE_OAUTH_ACCESS_TOKEN"); v != "" {
			return v
		}
	}
	// Gemini AI Studio common alias.
	if hasCapability(m.Capabilities, "gemini") && !hasCapability(m.Capabilities, "vertex") {
		if v := getenv("GEMINI_API_KEY"); v != "" {
			return v
		}
		if v := getenv("GOOGLE_API_KEY"); v != "" {
			return v
		}
	}
	return ""
}

// ResolvedBaseURL expands ${ENV} placeholders in BaseURL (e.g. GOOGLE_CLOUD_PROJECT).
func (m ModelConfig) ResolvedBaseURL() string {
	return expandEnvPlaceholders(m.BaseURL)
}

func hasCapability(caps []string, want string) bool {
	for _, c := range caps {
		if c == want {
			return true
		}
	}
	return false
}

// LLMClient abstracts an OpenAI-compatible chat completions endpoint.
type LLMClient interface {
	// Name returns the logical model name this client serves.
	Name() string
	// ChatCompletion performs a non-streaming chat completion.
	ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error)
	// ChatCompletionStream streams deltas via onDelta until complete.
	// onDelta may be called many times; the final aggregated response is returned.
	ChatCompletionStream(ctx context.Context, req ChatRequest, onDelta func(StreamDelta) error) (ChatResponse, error)
}

// ChatRequest mirrors a minimal OpenAI chat completion payload.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  any       `json:"tool_choice,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	// StreamOptions enables usage in streaming responses when supported.
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
}

// StreamOptions controls streaming response metadata.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// Message is a chat message (content may be string or multimodal later).
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Tool describes a function tool for the model.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction is an OpenAI-style function definition.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// ToolCall is a model-requested tool invocation.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall holds the function name and JSON arguments string.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatResponse is the minimal non-stream response shape.
type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice is one completion alternative.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	// PromptTokensDetails may include cache hits on supporting providers.
	PromptTokensDetails *PromptTokensDetails `json:"prompt_tokens_details,omitempty"`
}

// PromptTokensDetails carries optional cache accounting.
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

// StreamDelta is one streaming chunk from the provider.
type StreamDelta struct {
	Content          string
	ReasoningContent string
	ToolCalls        []ToolCall
	FinishReason     string
	Usage            *Usage
}

// SelectParams controls model selection for a single call.
type SelectParams struct {
	TaskType        TaskType
	EstimatedTokens int
	Complexity      Complexity
	// PreferCapabilities requires at least one matching capability when set.
	PreferCapabilities []string
	// Override forces a specific logical model name (e.g. from /model or CLI).
	Override string
}

// CallMeta is returned alongside a successful completion for observability.
type CallMeta struct {
	ModelName    string
	ModelID      string
	Attempts     int
	Duration     time.Duration
	EstimatedUSD float64
	FallbackUsed bool
}

// CostEstimate is a breakdown of projected or actual USD cost.
type CostEstimate struct {
	ModelName         string
	InputTokens       int
	OutputTokens      int
	CachedInputTokens int
	USD               float64
}

// APIError classifies provider HTTP failures for fallback policy.
type APIError struct {
	StatusCode int
	Body       string
	Retryable  bool
	RateLimit  bool
}

func (e *APIError) Error() string {
	return fmt.Sprintf("llm api http %d: %s", e.StatusCode, truncate(e.Body, 512))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

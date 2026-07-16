package router

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testModels(baseFlash, basePro, basePremium string) []ModelConfig {
	return []ModelConfig{
		{
			Name: "deepseek-v4-flash", BaseURL: baseFlash, ModelID: "deepseek-v4-flash",
			APIKey: "test", CostTier: 1, InputCostPerM: 0.14, OutputCostPerM: 0.28,
			MaxContext: 1_000_000, Capabilities: []string{"fast", "coding", "tool-calling"}, Priority: 10,
		},
		{
			Name: "deepseek-v4-pro", BaseURL: basePro, ModelID: "deepseek-v4-pro",
			APIKey: "test", CostTier: 3, InputCostPerM: 0.435, OutputCostPerM: 0.87,
			MaxContext: 1_000_000, Capabilities: []string{"coding", "tool-calling", "reasoning"}, Priority: 20,
		},
		{
			Name: "grok-4.5", BaseURL: basePremium, ModelID: "grok-4.5",
			APIKey: "test", CostTier: 20, InputCostPerM: 2, OutputCostPerM: 6,
			MaxContext: 500_000, Capabilities: []string{"coding", "tool-calling", "premium"}, Priority: 30,
		},
	}
}

func TestSelectModel_RoutinePrefersFlash(t *testing.T) {
	r, err := New(testModels("http://a", "http://b", "http://c"), DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	got := r.SelectModel(SelectParams{TaskType: TaskRoutine, Complexity: ComplexityRoutine, EstimatedTokens: 10_000})
	if got != "deepseek-v4-flash" {
		t.Fatalf("got %q, want deepseek-v4-flash", got)
	}
}

func TestSelectModel_PlanPrefersProStepUp(t *testing.T) {
	r, err := New(testModels("http://a", "http://b", "http://c"), DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	got := r.SelectModel(SelectParams{TaskType: TaskPlan, Complexity: ComplexityPlan, EstimatedTokens: 50_000})
	if got != "deepseek-v4-pro" {
		t.Fatalf("got %q, want deepseek-v4-pro", got)
	}
}

func TestSelectModel_HighStakesPrefersPremium(t *testing.T) {
	r, err := New(testModels("http://a", "http://b", "http://c"), DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	got := r.SelectModel(SelectParams{TaskType: TaskReview, Complexity: ComplexityHighStakes})
	if got != "grok-4.5" {
		t.Fatalf("got %q, want grok-4.5", got)
	}
}

func TestSelectModel_OverrideWins(t *testing.T) {
	r, err := New(testModels("http://a", "http://b", "http://c"), DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.SetOverride("grok-4.5"); err != nil {
		t.Fatal(err)
	}
	got := r.SelectModel(SelectParams{TaskType: TaskRoutine, Complexity: ComplexityRoutine})
	if got != "grok-4.5" {
		t.Fatalf("got %q, want grok-4.5", got)
	}
}

func TestSelectModel_ContextWindowFilter(t *testing.T) {
	models := []ModelConfig{
		{Name: "small", BaseURL: "http://a", ModelID: "s", APIKey: "k", CostTier: 1, MaxContext: 1000, Capabilities: []string{"fast"}, Priority: 10},
		{Name: "large", BaseURL: "http://b", ModelID: "l", APIKey: "k", CostTier: 5, MaxContext: 1_000_000, Capabilities: []string{"fast", "coding"}, Priority: 20},
	}
	r, err := New(models, "small")
	if err != nil {
		t.Fatal(err)
	}
	got := r.SelectModel(SelectParams{EstimatedTokens: 50_000, Complexity: ComplexityRoutine})
	if got != "large" {
		t.Fatalf("got %q, want large", got)
	}
}

func TestEstimateCost_FlashWithCache(t *testing.T) {
	r, err := New(DefaultModels(), DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	// 900k cached + 100k miss input, 10k output
	est := r.EstimateCostTokens(DefaultModelName, 1_000_000, 10_000, 900_000)
	// 100k * 0.14/1M + 900k * 0.0028/1M + 10k * 0.28/1M
	// = 0.014 + 0.00252 + 0.0028 = 0.01932
	want := 0.014 + 0.00252 + 0.0028
	if diff := est.USD - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("USD=%v want %v", est.USD, want)
	}
}

func TestExecuteWithFallback_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(ChatResponse{
			ID:    "1",
			Model: "deepseek-v4-flash",
			Choices: []Choice{{
				Message:      Message{Role: "assistant", Content: "hello"},
				FinishReason: "stop",
			}},
			Usage: Usage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12},
		})
	}))
	defer srv.Close()

	models := testModels(srv.URL, srv.URL, srv.URL)
	r, err := New(models, DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	resp, meta, err := r.ExecuteWithFallback(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	}, SelectParams{TaskType: TaskRoutine, Complexity: ComplexityRoutine})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Choices[0].Message.Content != "hello" {
		t.Fatalf("content=%q", resp.Choices[0].Message.Content)
	}
	if meta.ModelName != "deepseek-v4-flash" {
		t.Fatalf("model=%q", meta.ModelName)
	}
	if meta.Attempts != 1 {
		t.Fatalf("attempts=%d", meta.Attempts)
	}
}

func TestExecuteWithFallback_RateLimitThenNext(t *testing.T) {
	var hits atomic.Int32
	flash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer flash.Close()

	pro := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ChatResponse{
			Choices: []Choice{{Message: Message{Role: "assistant", Content: "from-pro"}}},
			Usage:   Usage{TotalTokens: 5},
		})
	}))
	defer pro.Close()

	models := testModels(flash.URL, pro.URL, pro.URL)
	r, err := New(models, DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	resp, meta, err := r.ExecuteWithFallback(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	}, SelectParams{TaskType: TaskRoutine, Complexity: ComplexityRoutine})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Choices[0].Message.Content != "from-pro" {
		t.Fatalf("content=%q", resp.Choices[0].Message.Content)
	}
	if !meta.FallbackUsed {
		t.Fatal("expected fallback")
	}
	if hits.Load() < 1 {
		t.Fatal("expected flash hit")
	}
}

func TestExecuteStreamWithFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		chunks := []string{
			`data: {"id":"s1","choices":[{"delta":{"content":"hel"}}]}`,
			`data: {"choices":[{"delta":{"content":"lo"},"finish_reason":"stop"}]}`,
			`data: {"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			_, _ = io.WriteString(w, c+"\n\n")
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer srv.Close()

	models := testModels(srv.URL, srv.URL, srv.URL)
	r, err := New(models, DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	var got strings.Builder
	resp, _, err := r.ExecuteStreamWithFallback(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	}, SelectParams{Complexity: ComplexityRoutine}, func(d StreamDelta) error {
		got.WriteString(d.Content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "hello" {
		t.Fatalf("streamed=%q", got.String())
	}
	if resp.Choices[0].Message.Content != "hello" {
		t.Fatalf("aggregated=%q", resp.Choices[0].Message.Content)
	}
}

func TestHTTPClient_AuthHeaderAndModelID(t *testing.T) {
	var sawAuth, sawModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		var body ChatRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		sawModel = body.Model
		_ = json.NewEncoder(w).Encode(ChatResponse{Choices: []Choice{{Message: Message{Content: "ok"}}}})
	}))
	defer srv.Close()

	c := NewHTTPClient(ModelConfig{
		Name: "m", BaseURL: srv.URL, ModelID: "deepseek-v4-flash", APIKey: "secret",
	}, nil)
	_, err := c.ChatCompletion(context.Background(), ChatRequest{
		Model:    "ignored",
		Messages: []Message{{Role: "user", Content: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if sawAuth != "Bearer secret" {
		t.Fatalf("auth=%q", sawAuth)
	}
	if sawModel != "deepseek-v4-flash" {
		t.Fatalf("model=%q", sawModel)
	}
}

func TestNew_RejectsEmptyCatalog(t *testing.T) {
	_, err := New(nil, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsRetryable(t *testing.T) {
	if !isRetryable(&APIError{StatusCode: 429, RateLimit: true, Retryable: true}) {
		t.Fatal("429 should be retryable")
	}
	if isRetryable(&APIError{StatusCode: 401, Retryable: false}) {
		t.Fatal("401 should not be retryable via flag")
	}
	if !isRetryable(errors.New("connection reset")) {
		t.Fatal("network errors retryable")
	}
}

func TestExpandEnvPlaceholders(t *testing.T) {
	t.Setenv("IOMESH_TEST_KEY", "abc")
	if got := expandEnvPlaceholders("${IOMESH_TEST_KEY}"); got != "abc" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultModels_CascadeOrder(t *testing.T) {
	r, err := New(DefaultModels(), DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	names := r.fallbackChain("deepseek-v4-flash")
	if len(names) != 3 {
		t.Fatalf("chain=%v", names)
	}
	if names[0] != "deepseek-v4-flash" || names[1] != "deepseek-v4-pro" || names[2] != "grok-4.5" {
		t.Fatalf("chain=%v", names)
	}
}

func TestExecute_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()
	r, err := New(testModels(srv.URL, srv.URL, srv.URL), DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, _, err = r.ExecuteWithFallback(ctx, ChatRequest{
		Messages: []Message{{Role: "user", Content: "x"}},
	}, SelectParams{})
	if err == nil {
		t.Fatal("expected cancel/timeout error")
	}
}

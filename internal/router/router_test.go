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
	// DeepSeek cascade first; Gemini/Vertex/Ollama are opt-in (later priority) but present in catalog.
	if len(names) < 3 {
		t.Fatalf("chain=%v", names)
	}
	if names[0] != "deepseek-v4-flash" || names[1] != "deepseek-v4-pro" || names[2] != "grok-4.5" {
		t.Fatalf("chain=%v", names)
	}
	// Default model remains DeepSeek Flash (cascade unchanged by Ollama pin entry).
	if r.DefaultModel() != DefaultModelName || DefaultModelName != "deepseek-v4-flash" {
		t.Fatalf("default model=%q want deepseek-v4-flash", r.DefaultModel())
	}
	// Gemini / Vertex / Ollama built-ins must load without GOOGLE_CLOUD_PROJECT set.
	foundGemini, foundVertex, foundOllama := false, false, false
	for _, n := range names {
		if n == GeminiFlashModelName {
			foundGemini = true
		}
		if n == VertexGeminiFlashModelName {
			foundVertex = true
		}
		if n == OllamaLlama32ModelName {
			foundOllama = true
		}
	}
	if !foundGemini || !foundVertex || !foundOllama {
		t.Fatalf("expected gemini+vertex+ollama in catalog, chain=%v", names)
	}
}

func TestDefaultModels_OllamaCatalog(t *testing.T) {
	var ollama *ModelConfig
	for i := range DefaultModels() {
		m := DefaultModels()[i]
		if m.Name == OllamaLlama32ModelName {
			ollama = &m
			break
		}
	}
	if ollama == nil {
		t.Fatal("missing ollama-llama3.2 in DefaultModels")
	}
	if ollama.BaseURL != OllamaOpenAIBaseURL {
		t.Fatalf("BaseURL=%q want %q", ollama.BaseURL, OllamaOpenAIBaseURL)
	}
	if ollama.ModelID != "llama3.2" {
		t.Fatalf("ModelID=%q", ollama.ModelID)
	}
	if ollama.CostTier != 0 || ollama.InputCostPerM != 0 || ollama.OutputCostPerM != 0 {
		t.Fatalf("want free local costs, got tier=%v in=%v out=%v", ollama.CostTier, ollama.InputCostPerM, ollama.OutputCostPerM)
	}
	if !hasCapability(ollama.Capabilities, "ollama") || !hasCapability(ollama.Capabilities, "local") {
		t.Fatalf("capabilities=%v", ollama.Capabilities)
	}
	if ollama.Priority != 60 {
		t.Fatalf("priority=%d want 60", ollama.Priority)
	}
	// Empty API key is OK for Ollama (no EnvKey required).
	if ollama.EnvKey != "" {
		t.Fatalf("EnvKey=%q want empty", ollama.EnvKey)
	}
	if got := ollama.ResolvedAPIKey(); got != "" {
		t.Fatalf("ResolvedAPIKey=%q want empty", got)
	}
}

// TestDefaultModels_OllamaLocalEdgeCompletenessPin is the s765 completeness pin
// after s761 first-class Ollama product. It locks local-edge catalog inventory
// needles without re-implementing product or inventing new models:
// name ollama-llama3.2, model id llama3.2, cost 0, caps local+ollama,
// priority after DeepSeek cascade / Premium (priority > 30). Cascade default
// remains deepseek-v4-flash. Completeness pin only — does not re-claim s761.
func TestDefaultModels_OllamaLocalEdgeCompletenessPin(t *testing.T) {
	// Cascade default unchanged (catalog pin ≠ cascade default).
	if DefaultModelName != "deepseek-v4-flash" {
		t.Fatalf("DefaultModelName=%q want deepseek-v4-flash (cascade unchanged)", DefaultModelName)
	}

	var ollama *ModelConfig
	var premium *ModelConfig
	for i := range DefaultModels() {
		m := DefaultModels()[i]
		switch m.Name {
		case OllamaLlama32ModelName:
			cp := m
			ollama = &cp
		case PremiumModelName:
			cp := m
			premium = &cp
		}
	}
	if ollama == nil {
		t.Fatal("s765 completeness inventory missing ollama-llama3.2 in DefaultModels")
	}

	// Inventory needles: name, model id, $0 costs, local+ollama caps.
	if ollama.Name != "ollama-llama3.2" {
		t.Fatalf("Name=%q want ollama-llama3.2", ollama.Name)
	}
	if ollama.ModelID != "llama3.2" {
		t.Fatalf("ModelID=%q want llama3.2", ollama.ModelID)
	}
	if ollama.CostTier != 0 || ollama.InputCostPerM != 0 || ollama.OutputCostPerM != 0 {
		t.Fatalf("want cost 0 local tier, got tier=%v in=%v out=%v",
			ollama.CostTier, ollama.InputCostPerM, ollama.OutputCostPerM)
	}
	if !hasCapability(ollama.Capabilities, "local") || !hasCapability(ollama.Capabilities, "ollama") {
		t.Fatalf("capabilities=%v want local+ollama", ollama.Capabilities)
	}

	// Priority after DeepSeek cascade / Premium (Premium priority is 30).
	if ollama.Priority <= 30 {
		t.Fatalf("priority=%d want > 30 (after Premium / DeepSeek cascade)", ollama.Priority)
	}
	if premium != nil && ollama.Priority <= premium.Priority {
		t.Fatalf("ollama priority=%d want > Premium priority=%d", ollama.Priority, premium.Priority)
	}

	// Router still defaults to DeepSeek Flash — Ollama remains pin-only.
	r, err := New(DefaultModels(), DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	if r.DefaultModel() != DefaultModelName {
		t.Fatalf("router default=%q want %q", r.DefaultModel(), DefaultModelName)
	}
}

func TestResolvedBaseURL_OllamaEnvOverrides(t *testing.T) {
	m := ModelConfig{
		Name:         OllamaLlama32ModelName,
		BaseURL:      OllamaOpenAIBaseURL,
		ModelID:      "llama3.2",
		Capabilities: []string{"local", "ollama"},
	}

	t.Run("default", func(t *testing.T) {
		t.Setenv("OLLAMA_URL", "")
		t.Setenv("OLLAMA_HOST", "")
		if got := m.ResolvedBaseURL(); got != OllamaOpenAIBaseURL {
			t.Fatalf("got %q want %q", got, OllamaOpenAIBaseURL)
		}
	})

	t.Run("OLLAMA_URL host root", func(t *testing.T) {
		t.Setenv("OLLAMA_URL", "http://127.0.0.1:11434")
		t.Setenv("OLLAMA_HOST", "ignored")
		if got := m.ResolvedBaseURL(); got != "http://127.0.0.1:11434/v1" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("OLLAMA_URL already /v1", func(t *testing.T) {
		t.Setenv("OLLAMA_URL", "http://10.0.0.2:11434/v1/")
		if got := m.ResolvedBaseURL(); got != "http://10.0.0.2:11434/v1" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("OLLAMA_HOST host:port", func(t *testing.T) {
		t.Setenv("OLLAMA_URL", "")
		t.Setenv("OLLAMA_HOST", "192.168.1.10:11434")
		if got := m.ResolvedBaseURL(); got != "http://192.168.1.10:11434/v1" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("non-ollama ignores env", func(t *testing.T) {
		other := ModelConfig{BaseURL: "https://api.deepseek.com/v1", Capabilities: []string{"fast"}}
		t.Setenv("OLLAMA_URL", "http://127.0.0.1:11434")
		if got := other.ResolvedBaseURL(); got != "https://api.deepseek.com/v1" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestNew_AcceptsOllamaLoopback(t *testing.T) {
	r, err := New(DefaultModels(), DefaultModelName)
	if err != nil {
		t.Fatal(err)
	}
	cfg, ok := r.Model(OllamaLlama32ModelName)
	if !ok {
		t.Fatal("ollama model missing from router")
	}
	if cfg.BaseURL != OllamaOpenAIBaseURL {
		t.Fatalf("BaseURL=%q", cfg.BaseURL)
	}
}

func TestResolvedBaseURL_ExpandsProject(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "iomesh-stage-001")
	m := ModelConfig{BaseURL: VertexOpenAIBaseURLTemplate}
	got := m.ResolvedBaseURL()
	if !strings.Contains(got, "iomesh-stage-001") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "${") {
		t.Fatalf("unexpanded placeholder in %q", got)
	}
}

func TestResolvedAPIKey_VertexFallback(t *testing.T) {
	t.Setenv("VERTEX_API_KEY", "")
	t.Setenv("GOOGLE_OAUTH_ACCESS_TOKEN", "ya29.test-token")
	m := ModelConfig{Capabilities: []string{"vertex", "gemini"}, EnvKey: "VERTEX_API_KEY"}
	if got := m.ResolvedAPIKey(); got != "ya29.test-token" {
		t.Fatalf("got %q", got)
	}
}

func TestNewRequest_RejectsUnexpandedVertexProject(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	c := NewHTTPClient(ModelConfig{
		Name: "v", BaseURL: VertexOpenAIBaseURLTemplate, ModelID: "google/gemini-2.5-flash",
		APIKey: "tok", Capabilities: []string{"vertex"},
	}, nil)
	_, err := c.newRequest(context.Background(), []byte(`{}`))
	if err == nil || !strings.Contains(err.Error(), "GOOGLE_CLOUD_PROJECT") {
		t.Fatalf("want missing GOOGLE_CLOUD_PROJECT error, got %v", err)
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

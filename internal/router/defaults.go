package router

// Default catalog and cascade for the I/O Mesh TUI agent harness.
// Pricing reflects DeepSeek public rates (July 2026): Flash $0.14/$0.28,
// Pro $0.435/$0.87 per 1M tokens; cache hits substantially lower.
// Model IDs use the current DeepSeek identifiers (deepseek-chat is deprecated).
//
// Gemini (Google AI Studio) and Vertex AI Gemini are first-class OpenAI-compatible
// options (opt-in via -m / /model / IOMESH_DEFAULT_MODEL). They do NOT replace the
// DeepSeek-first default cascade — pin them when you want a Google runtime.
//
// Ollama local models are pin-only (`-m ollama-llama3.2` / `/model` / IOMESH_DEFAULT_MODEL).
// They are not part of the DeepSeek default cascade. OpenAI-compatible /v1 only.

const (
	// DefaultModelName is the primary value leader for routine agent work.
	DefaultModelName = "deepseek-v4-flash"
	// StepUpModelName is the mid-tier escalation for plan / multi-file work.
	StepUpModelName = "deepseek-v4-pro"
	// PremiumModelName is the high-stakes / ecosystem fallback.
	PremiumModelName = "grok-4.5"

	// GeminiFlashModelName is Google AI Studio Gemini Flash (OpenAI-compat).
	GeminiFlashModelName = "gemini-2.5-flash"
	// GeminiProModelName is Google AI Studio Gemini Pro (OpenAI-compat).
	GeminiProModelName = "gemini-2.5-pro"
	// VertexGeminiFlashModelName is Vertex AI OpenAI-compat Gemini Flash.
	VertexGeminiFlashModelName = "vertex-gemini-2.5-flash"
	// VertexGeminiProModelName is Vertex AI OpenAI-compat Gemini Pro.
	VertexGeminiProModelName = "vertex-gemini-2.5-pro"

	// OllamaLlama32ModelName is local Ollama llama3.2 via OpenAI-compat /v1 (pin-only).
	OllamaLlama32ModelName = "ollama-llama3.2"

	// GeminiOpenAIBaseURL is the Gemini API OpenAI-compatible root (no trailing slash).
	// Docs: https://ai.google.dev/gemini-api/docs/openai
	GeminiOpenAIBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"

	// VertexOpenAIBaseURLTemplate uses ${GOOGLE_CLOUD_PROJECT}; region defaults to us-central1.
	// Expand with expandEnvPlaceholders at request time. Override via [model.*] base_url.
	// Docs: Vertex OpenAI-compatible chat completions.
	VertexOpenAIBaseURLTemplate = "https://us-central1-aiplatform.googleapis.com/v1/projects/${GOOGLE_CLOUD_PROJECT}/locations/us-central1/endpoints/openapi"

	// OllamaOpenAIBaseURL is the default Ollama OpenAI-compatible root (no trailing slash).
	// Override at runtime via OLLAMA_URL or OLLAMA_HOST (see ResolvedBaseURL).
	OllamaOpenAIBaseURL = "http://127.0.0.1:11434/v1"
)

// DefaultModels returns the built-in catalog:
// DeepSeek V4 Flash → DeepSeek V4 Pro → Grok 4.5 (default cascade by priority),
// plus optional Gemini / Vertex Gemini / Ollama models (higher priority = later fallback only).
// Ollama entries are pin-only and must not replace the DeepSeek cascade default.
func DefaultModels() []ModelConfig {
	return []ModelConfig{
		{
			Name:             DefaultModelName,
			BaseURL:          "https://api.deepseek.com/v1",
			EnvKey:           "DEEPSEEK_API_KEY",
			ModelID:          "deepseek-v4-flash",
			CostTier:         1,
			InputCostPerM:    0.14,
			OutputCostPerM:   0.28,
			CacheHitCostPerM: 0.0028,
			MaxContext:       1_000_000,
			Capabilities:     []string{"fast", "coding", "tool-calling"},
			Priority:         10,
		},
		{
			Name:             StepUpModelName,
			BaseURL:          "https://api.deepseek.com/v1",
			EnvKey:           "DEEPSEEK_API_KEY",
			ModelID:          "deepseek-v4-pro",
			CostTier:         3,
			InputCostPerM:    0.435,
			OutputCostPerM:   0.87,
			CacheHitCostPerM: 0.003625,
			MaxContext:       1_000_000,
			Capabilities:     []string{"coding", "tool-calling", "reasoning"},
			Priority:         20,
		},
		{
			Name:           PremiumModelName,
			BaseURL:        "https://api.x.ai/v1",
			EnvKey:         "XAI_API_KEY",
			ModelID:        "grok-4.5",
			CostTier:       20,
			InputCostPerM:  2.00,
			OutputCostPerM: 6.00,
			MaxContext:     500_000,
			Capabilities:   []string{"coding", "tool-calling", "premium", "mcp"},
			Priority:       30,
		},
		// --- Google Gemini (AI Studio API key) — pin with -m gemini-2.5-flash ---
		{
			Name:           GeminiFlashModelName,
			BaseURL:        GeminiOpenAIBaseURL,
			EnvKey:         "GEMINI_API_KEY",
			ModelID:        "gemini-2.5-flash",
			CostTier:       4,
			InputCostPerM:  0.30,
			OutputCostPerM: 2.50,
			MaxContext:     1_000_000,
			Capabilities:   []string{"fast", "coding", "tool-calling", "gemini", "google"},
			Priority:       40,
		},
		{
			Name:           GeminiProModelName,
			BaseURL:        GeminiOpenAIBaseURL,
			EnvKey:         "GEMINI_API_KEY",
			ModelID:        "gemini-2.5-pro",
			CostTier:       12,
			InputCostPerM:  1.25,
			OutputCostPerM: 10.00,
			MaxContext:     1_000_000,
			Capabilities:   []string{"coding", "tool-calling", "reasoning", "gemini", "google", "premium"},
			Priority:       45,
		},
		// --- Vertex AI Gemini (GCP project + OAuth access token as Bearer) ---
		// export GOOGLE_CLOUD_PROJECT=… VERTEX_API_KEY=$(gcloud auth print-access-token)
		{
			Name:           VertexGeminiFlashModelName,
			BaseURL:        VertexOpenAIBaseURLTemplate,
			EnvKey:         "VERTEX_API_KEY",
			ModelID:        "google/gemini-2.5-flash",
			CostTier:       4,
			InputCostPerM:  0.30,
			OutputCostPerM: 2.50,
			MaxContext:     1_000_000,
			Capabilities:   []string{"fast", "coding", "tool-calling", "gemini", "vertex", "google"},
			Priority:       50,
		},
		{
			Name:           VertexGeminiProModelName,
			BaseURL:        VertexOpenAIBaseURLTemplate,
			EnvKey:         "VERTEX_API_KEY",
			ModelID:        "google/gemini-2.5-pro",
			CostTier:       12,
			InputCostPerM:  1.25,
			OutputCostPerM: 10.00,
			MaxContext:     1_000_000,
			Capabilities:   []string{"coding", "tool-calling", "reasoning", "gemini", "vertex", "google", "premium"},
			Priority:       55,
		},
		// --- Ollama (local OpenAI-compat /v1) — pin-only; not cascade default ---
		// ollama serve && ollama pull llama3.2
		// iomesh -m ollama-llama3.2  ·  /model ollama-llama3.2  ·  IOMESH_DEFAULT_MODEL=ollama-llama3.2
		// Optional: OLLAMA_URL / OLLAMA_HOST override BaseURL; no API key required.
		{
			Name:           OllamaLlama32ModelName,
			BaseURL:        OllamaOpenAIBaseURL,
			EnvKey:         "", // Ollama does not require a key; empty Authorization already works
			ModelID:        "llama3.2",
			CostTier:       0,
			InputCostPerM:  0,
			OutputCostPerM: 0,
			MaxContext:     128_000,
			Capabilities:   []string{"local", "ollama", "fast", "coding", "tool-calling"},
			Priority:       60,
		},
	}
}

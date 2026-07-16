package router

// Default catalog and cascade for the I/O Mesh TUI agent harness.
// Pricing reflects DeepSeek public rates (July 2026): Flash $0.14/$0.28,
// Pro $0.435/$0.87 per 1M tokens; cache hits substantially lower.
// Model IDs use the current DeepSeek identifiers (deepseek-chat is deprecated).

const (
	// DefaultModelName is the primary value leader for routine agent work.
	DefaultModelName = "deepseek-v4-flash"
	// StepUpModelName is the mid-tier escalation for plan / multi-file work.
	StepUpModelName = "deepseek-v4-pro"
	// PremiumModelName is the high-stakes / ecosystem fallback.
	PremiumModelName = "grok-4.5"
)

// DefaultModels returns the built-in cascade:
// DeepSeek V4 Flash → DeepSeek V4 Pro → Grok 4.5.
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
	}
}

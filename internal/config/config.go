// Package config loads and merges I/O Mesh TUI configuration from TOML,
// environment variables, and CLI flags (highest precedence last at the call site).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/pelletier/go-toml/v2"
)

// FileName is the default config file basename.
const FileName = "config.toml"

// Config is the root configuration document.
type Config struct {
	Models    ModelsSection        `toml:"models"`
	Model     map[string]ModelTOML `toml:"model"`
	Router    RouterSection        `toml:"router"`
	Agent     AgentSection         `toml:"agent"`
	IOMesh    IOMeshSection        `toml:"iomesh"`
	UI        UISection            `toml:"ui"`
	Features  FeaturesSection      `toml:"features"`
	Subagents SubagentsSection     `toml:"subagents"`
	// Catalog is the resolved runtime model list (not from TOML directly).
	Catalog []router.ModelConfig `toml:"-"`
}

// ModelsSection holds global model defaults.
type ModelsSection struct {
	Default             string            `toml:"default"`
	Temperature         *float64          `toml:"temperature"`
	TopP                *float64          `toml:"top_p"`
	MaxCompletionTokens int               `toml:"max_completion_tokens"`
	MaxRetries          int               `toml:"max_retries"`
	ExtraHeaders        map[string]string `toml:"extra_headers"`
	StreamToolCalls     *bool             `toml:"stream_tool_calls"`
}

// ModelTOML is a single [model.<name>] table.
type ModelTOML struct {
	Model            string            `toml:"model"`
	BaseURL          string            `toml:"base_url"`
	Name             string            `toml:"name"`
	Description      string            `toml:"description"`
	APIKey           string            `toml:"api_key"`
	EnvKey           string            `toml:"env_key"`
	APIBackend       string            `toml:"api_backend"`
	Temperature      *float64          `toml:"temperature"`
	MaxCompletionTok int               `toml:"max_completion_tokens"`
	ContextWindow    int               `toml:"context_window"`
	CostTier         float64           `toml:"cost_tier"`
	InputCostPerM    float64           `toml:"input_cost_per_m"`
	OutputCostPerM   float64           `toml:"output_cost_per_m"`
	CacheHitCostPerM float64           `toml:"cache_hit_cost_per_m"`
	Capabilities     []string          `toml:"capabilities"`
	Priority         int               `toml:"priority"`
	ExtraHeaders     map[string]string `toml:"extra_headers"`
}

// RouterSection tunes the fallback router.
type RouterSection struct {
	MaxAttempts int `toml:"max_attempts"`
}

// AgentSection controls agent runtime defaults.
type AgentSection struct {
	SystemPromptLabel string `toml:"system_prompt_label"`
	Yolo              bool   `toml:"yolo"`
	Workspace         string `toml:"workspace"`
}

// IOMeshSection configures I/O Mesh platform integration.
type IOMeshSection struct {
	Enabled   bool   `toml:"enabled"`
	Endpoint  string `toml:"endpoint"`
	Tenant    string `toml:"tenant"`
	APIKeyEnv string `toml:"api_key_env"`
	// EmitDeptStreams publishes dept.* operational events when true.
	EmitDeptStreams bool `toml:"emit_dept_streams"`
	// ContextPlane injects governed operational context into prompts.
	ContextPlane bool `toml:"context_plane"`
}

// UISection is reserved for TUI preferences.
type UISection struct {
	SimpleMode   bool   `toml:"simple_mode"`
	ScreenMode   string `toml:"screen_mode"`
	ShowThinking bool   `toml:"show_thinking_blocks"`
}

// FeaturesSection holds feature flags.
type FeaturesSection struct {
	Telemetry     bool `toml:"telemetry"`
	Subagents     bool `toml:"subagents"`
	CodebaseIndex bool `toml:"codebase_indexing"`
}

// SubagentsSection tunes child-session orchestration.
type SubagentsSection struct {
	Enabled       bool `toml:"enabled"`
	MaxConcurrent int  `toml:"max_concurrent"`
	MaxDepth      int  `toml:"max_depth"`
	MaxBatch      int  `toml:"max_batch"`
	// WorktreeBase is relative to the workspace (default .iomesh/worktrees).
	WorktreeBase string `toml:"worktree_base"`
	// WorktreeAutoRemove deletes successful worktrees (default false = keep for inspection).
	WorktreeAutoRemove bool `toml:"worktree_auto_remove"`
}

// Default returns built-in configuration (DeepSeek Flash primary cascade).
func Default() *Config {
	cfg := &Config{
		Models: ModelsSection{
			Default:             router.DefaultModelName,
			MaxCompletionTokens: 8192,
			MaxRetries:          8,
		},
		Router: RouterSection{MaxAttempts: 3},
		Agent:  AgentSection{SystemPromptLabel: "iomesh-tui"},
		IOMesh: IOMeshSection{
			Enabled:         false,
			APIKeyEnv:       "IOMESH_API_KEY",
			EmitDeptStreams: true,
			ContextPlane:    true,
		},
		UI: UISection{
			SimpleMode:   true,
			ScreenMode:   "fullscreen",
			ShowThinking: true,
		},
		Features: FeaturesSection{
			Subagents:     true,
			CodebaseIndex: true,
		},
		Subagents: SubagentsSection{
			Enabled:       true,
			MaxConcurrent: 16, // max parallel running children
			MaxDepth:      2,
			MaxBatch:      32, // max tasks per spawn_subagents call
		},
		Catalog: router.DefaultModels(),
	}
	return cfg
}

// Load reads path if it exists and merges onto defaults.
// Missing file is not an error. Environment overrides always apply.
func Load(path string) (*Config, error) {
	cfg := Default()
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("read config: %w", err)
			}
			// missing file → defaults + env
		} else {
			if err := toml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config %s: %w", path, err)
			}
			if err := cfg.resolveCatalog(); err != nil {
				return nil, err
			}
		}
	}
	cfg.applyEnvOverrides()
	return cfg, nil
}

// LoadUser loads ~/.iomesh/config.toml (or IOMESH_CONFIG / XDG path).
func LoadUser() (*Config, error) {
	path, err := UserConfigPath()
	if err != nil {
		return Default(), nil
	}
	return Load(path)
}

// UserConfigPath returns the default user config file path.
func UserConfigPath() (string, error) {
	if p := os.Getenv("IOMESH_CONFIG"); p != "" {
		return p, nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "iomesh", FileName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".iomesh", FileName), nil
}

// resolveCatalog builds Catalog from defaults + [model.*] overrides/additions.
func (c *Config) resolveCatalog() error {
	byName := map[string]router.ModelConfig{}
	for _, m := range router.DefaultModels() {
		byName[m.Name] = m
	}

	for key, mt := range c.Model {
		name := key
		if mt.Name != "" {
			name = mt.Name
		}
		base, ok := byName[name]
		if !ok {
			base = router.ModelConfig{Name: name, Priority: 50}
		}
		if mt.Model != "" {
			base.ModelID = mt.Model
		}
		if mt.BaseURL != "" {
			base.BaseURL = mt.BaseURL
		}
		if mt.APIKey != "" {
			base.APIKey = mt.APIKey
		}
		if mt.EnvKey != "" {
			base.EnvKey = mt.EnvKey
		}
		if mt.ContextWindow > 0 {
			base.MaxContext = mt.ContextWindow
		}
		if mt.CostTier > 0 {
			base.CostTier = mt.CostTier
		}
		if mt.InputCostPerM > 0 {
			base.InputCostPerM = mt.InputCostPerM
		}
		if mt.OutputCostPerM > 0 {
			base.OutputCostPerM = mt.OutputCostPerM
		}
		if mt.CacheHitCostPerM > 0 {
			base.CacheHitCostPerM = mt.CacheHitCostPerM
		}
		if len(mt.Capabilities) > 0 {
			base.Capabilities = append([]string(nil), mt.Capabilities...)
		}
		if mt.Priority > 0 {
			base.Priority = mt.Priority
		}
		if len(mt.ExtraHeaders) > 0 {
			if base.ExtraHeaders == nil {
				base.ExtraHeaders = map[string]string{}
			}
			for k, v := range mt.ExtraHeaders {
				base.ExtraHeaders[k] = v
			}
		}
		// Merge global extra headers (model wins on key).
		if len(c.Models.ExtraHeaders) > 0 {
			if base.ExtraHeaders == nil {
				base.ExtraHeaders = map[string]string{}
			}
			for k, v := range c.Models.ExtraHeaders {
				if _, exists := base.ExtraHeaders[k]; !exists {
					base.ExtraHeaders[k] = v
				}
			}
		}
		if base.ModelID == "" {
			return fmt.Errorf("model %q: missing model id", name)
		}
		if base.BaseURL == "" {
			return fmt.Errorf("model %q: missing base_url", name)
		}
		byName[name] = base
	}

	// Also apply global headers to defaults not listed under [model.*].
	if len(c.Models.ExtraHeaders) > 0 {
		for name, base := range byName {
			if _, listed := c.Model[name]; listed {
				continue
			}
			if base.ExtraHeaders == nil {
				base.ExtraHeaders = map[string]string{}
			}
			for k, v := range c.Models.ExtraHeaders {
				if _, exists := base.ExtraHeaders[k]; !exists {
					base.ExtraHeaders[k] = v
				}
			}
			byName[name] = base
		}
	}

	c.Catalog = c.Catalog[:0]
	for _, m := range byName {
		c.Catalog = append(c.Catalog, m)
	}
	if c.Models.Default == "" {
		c.Models.Default = router.DefaultModelName
	}
	return nil
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("IOMESH_DEFAULT_MODEL"); v != "" {
		c.Models.Default = v
	}
	if v := os.Getenv("IOMESH_ENDPOINT"); v != "" {
		c.IOMesh.Endpoint = v
		c.IOMesh.Enabled = true
	}
	if v := os.Getenv("IOMESH_TENANT"); v != "" {
		c.IOMesh.Tenant = v
	}
	if strings.EqualFold(os.Getenv("IOMESH_YOLO"), "1") || strings.EqualFold(os.Getenv("IOMESH_YOLO"), "true") {
		c.Agent.Yolo = true
	}
	if v := os.Getenv("IOMESH_SUBAGENTS"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.Subagents.Enabled = false
			c.Features.Subagents = false
		case "1", "true", "on", "yes":
			c.Subagents.Enabled = true
			c.Features.Subagents = true
		}
	}
}

// NewRouter constructs a router.Router from this config.
func (c *Config) NewRouter(opts ...router.Option) (*router.Router, error) {
	if c.Router.MaxAttempts > 0 {
		opts = append([]router.Option{router.WithMaxAttempts(c.Router.MaxAttempts)}, opts...)
	}
	return router.New(c.Catalog, c.Models.Default, opts...)
}

// Package config loads and merges I/O Mesh TUI configuration from TOML,
// environment variables, and CLI flags (highest precedence last at the call site).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	Skills    SkillsSection        `toml:"skills"`
	MCP       MCPSection           `toml:"mcp"`
	Memory    MemorySection        `toml:"memory"`
	// Plugins is opt-in Agent Plugins package discovery (s1331). Default disabled.
	// Discover/load success ≠ Connected / install APPLY green · dual_write OFF.
	Plugins PluginsSection `toml:"plugins"`
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
	// Org is optional org id for PlanGate / MEMORY_INGEST entitlements (X-IOMesh-Org).
	// Prefer the console /me public id (org_ + cuid2). Name slugs remain aliases.
	Org string `toml:"org"`
	// Workspace is optional workspace id for memory entitlements (X-IOMesh-Workspace).
	// Distinct from [agent].workspace (filesystem path).
	Workspace string `toml:"workspace"`
	// EmitDeptStreams publishes dept.* operational events when true.
	EmitDeptStreams bool `toml:"emit_dept_streams"`
	// ContextPlane injects governed operational context into prompts.
	ContextPlane bool `toml:"context_plane"`
	// IncludeLineage requests lineage refs on context plane queries.
	IncludeLineage bool `toml:"include_lineage"`
	// PolicyMode: off | advisory | enforce (remote Rego/OPA evaluate; fail-open on transport).
	PolicyMode string `toml:"policy_mode"`
	// CatalogPlane enables catalog data-product discovery (list_mesh_catalog, mesh catalog CLI).
	CatalogPlane bool `toml:"catalog_plane"`
	// InjectCatalog injects a short catalog snippet each agent turn (fail-open).
	InjectCatalog bool `toml:"inject_catalog"`
}

// UISection is reserved for TUI preferences.
type UISection struct {
	SimpleMode   bool   `toml:"simple_mode"`
	ScreenMode   string `toml:"screen_mode"`
	ShowThinking bool   `toml:"show_thinking_blocks"`
	// Theme: default | mono | high-contrast | dim
	Theme string `toml:"theme"`
}

// FeaturesSection holds feature flags.
type FeaturesSection struct {
	Telemetry     bool `toml:"telemetry"`
	Subagents     bool `toml:"subagents"`
	CodebaseIndex bool `toml:"codebase_indexing"`
	Skills        bool `toml:"skills"`
	MCP           bool `toml:"mcp"`
}

// SkillsSection configures SKILL.md discovery.
type SkillsSection struct {
	Enabled bool     `toml:"enabled"`
	Dirs    []string `toml:"dirs"` // extra dirs; workspace/user defaults always scanned when enabled
}

// PluginsSection configures Agent Plugins package discovery (s1331 runtime wire).
// Opt-in: Enabled defaults false. Dirs are absolute or ~/ expanded plugin package
// roots (or parent dirs whose immediate children are package roots).
// DataDir is the root for per-plugin PLUGIN_DATA; empty → workspace .iomesh/plugin-data/<name>.
// Residual honesty: package wire ≠ Agent Plugins GA · dual_write OFF · secrets never
// from portable plugin fields · TOML [[mcp.servers]] remains primary (plugins append).
type PluginsSection struct {
	Enabled bool     `toml:"enabled"`
	Dirs    []string `toml:"dirs"`
	DataDir string   `toml:"data_dir"`
}

// MCPServerTOML is one MCP server (stdio command and/or HTTP url).
type MCPServerTOML struct {
	Name              string            `toml:"name"`
	Command           string            `toml:"command"`
	Args              []string          `toml:"args"`
	Env               map[string]string `toml:"env"`
	URL               string            `toml:"url"`
	Headers           map[string]string `toml:"headers"`
	AllowLoopback     *bool             `toml:"allow_loopback"`
	Enabled           *bool             `toml:"enabled"`
	Mutating          *bool             `toml:"mutating"`
	StartupTimeoutSec int               `toml:"startup_timeout_sec"`
	ToolTimeoutSec    int               `toml:"tool_timeout_sec"`
	// OAuthTokenEnv injects Authorization: Bearer from env (simplest).
	OAuthTokenEnv string `toml:"oauth_token_env"`
	// Nested oauth table for client_credentials.
	OAuth *MCPOAuthTOML `toml:"oauth"`
	// InjectIOMeshContext overrides [mcp].inject_iomesh_context for this server.
	// nil = inherit global (default false). s1267 residual-honest opt-in only.
	// Applies to HTTP URL servers (stdio has no request headers).
	InjectIOMeshContext *bool `toml:"inject_iomesh_context"`
}

// MCPOAuthTOML is optional OAuth2 client_credentials for HTTP MCP.
type MCPOAuthTOML struct {
	TokenURL        string   `toml:"token_url"`
	ClientID        string   `toml:"client_id"`
	ClientSecretEnv string   `toml:"client_secret_env"`
	Scopes          []string `toml:"scopes"`
	AccessTokenEnv  string   `toml:"access_token_env"`
	AllowLoopback   *bool    `toml:"allow_loopback"`
}

// MCPSection configures Model Context Protocol clients.
type MCPSection struct {
	Enabled        bool            `toml:"enabled"`
	MaxOutputBytes int             `toml:"max_output_bytes"`
	Servers        []MCPServerTOML `toml:"servers"`
	// InjectIOMeshContext (s1267) opt-in: when true, merge X-IOMesh-Tenant/Org/Workspace
	// into each server's HTTP Headers at ServerConfig build time (non-empty only;
	// never overwrite explicit headers). Default false — not install green / not dual-auth.
	// Per-server inject_iomesh_context overrides this. Stdio servers ignore headers.
	InjectIOMeshContext bool `toml:"inject_iomesh_context"`
}

// WantsInjectIOMeshContext reports whether multi-tenant context headers should be
// merged for this server (s1267). Per-server pointer wins; else global default false.
func (m MCPSection) WantsInjectIOMeshContext(s MCPServerTOML) bool {
	if s.InjectIOMeshContext != nil {
		return *s.InjectIOMeshContext
	}
	return m.InjectIOMeshContext
}

// IOMeshMCPContext returns tenant/org/workspace for MCP header inject (s1267).
// Tenant: [iomesh].tenant, else [memory].tenant. Org/workspace from [iomesh] only.
// Empty strings mean "do not send" — never invent values.
func (c *Config) IOMeshMCPContext() (tenant, org, workspace string) {
	if c == nil {
		return "", "", ""
	}
	tenant = strings.TrimSpace(c.IOMesh.Tenant)
	if tenant == "" {
		tenant = strings.TrimSpace(c.Memory.Tenant)
	}
	org = strings.TrimSpace(c.IOMesh.Org)
	workspace = strings.TrimSpace(c.IOMesh.Workspace)
	return tenant, org, workspace
}

// MemorySection configures Memory Palace hooks (auto-recall / auto-ingest).
// Recall prefers mesh sync HTTP RetrieveMemory when [iomesh] is enabled or Endpoint
// (memory sidecar) is set; else MCP (connected [[mcp.servers]] — stdio or HTTP).
// DualWrite: optional async publish to mesh MEMORY_INGEST (no SDK dep).
type MemorySection struct {
	Enabled bool   `toml:"enabled"`
	Server  string `toml:"server"` // MCP server name; default "memory"
	Tenant  string `toml:"tenant"`
	// Endpoint is optional memory sidecar base for sync POST /v1/memory/retrieve.
	// When set, overrides [iomesh] endpoint for retrieve only (stage warm plane).
	// Env: IOMESH_MEMORY_ENDPOINT / MEMORY_SIDECAR_URL
	Endpoint   string `toml:"endpoint"`
	AutoRecall bool   `toml:"auto_recall"`
	AutoIngest bool   `toml:"auto_ingest"`
	// DualWrite also emits memory_ingest envelopes to MEMORY_INGEST when mesh is enabled.
	// Cost-max: dual_write is optional mesh **audit** (default OFF) — not primary cloud palace.
	DualWrite       bool `toml:"dual_write"`
	Limit           int  `toml:"limit"`
	MaxSnippetBytes int  `toml:"max_snippet_bytes"`
	// RecallSince / RecallUntil optional RFC3339 bounds for auto-recall + default /memory recall
	// (s1068 temporal retrieve → platform since/until). Empty = no time filter.
	// Env: IOMESH_MEMORY_RECALL_SINCE / IOMESH_MEMORY_RECALL_UNTIL
	RecallSince string `toml:"recall_since"`
	RecallUntil string `toml:"recall_until"`
	// RecallSessionSeq optional session_seq filter for temporal recall; 0 omits. s1068.
	// Env: IOMESH_MEMORY_RECALL_SESSION_SEQ
	RecallSessionSeq int `toml:"recall_session_seq"`
	// RecallCacheTTLMS short-TTL client-side sync RetrieveMemory reuse (s1069).
	// Default 3000; 0 disables. Env: IOMESH_MEMORY_RECALL_CACHE_TTL_MS
	RecallCacheTTLMS int `toml:"recall_cache_ttl_ms"`
	// Pull* configure `iomesh memory pull` (mesh durable consumer → local MCP palace). s652 M1.
	// Primary product path under cost-max local-memory charter (dual_write remains optional audit).
	PullStream    string `toml:"pull_stream"`      // e.g. EVENTS or MEMORY_INGEST
	PullConsumer  string `toml:"pull_consumer"`    // durable consumer name
	PullFilter    string `toml:"pull_filter"`      // optional filter_subject
	PullBatch     int    `toml:"pull_batch"`       // default 8
	PullMaxWaitMS int    `toml:"pull_max_wait_ms"` // default 2000
	// PullContinuous opt-in in-session continuous memory pull on agent Runtime (s1530 P5).
	// Default OFF. Requires pull_consumer. Env: IOMESH_MEMORY_PULL_CONTINUOUS.
	// pull running ≠ invent install green / Ops Pack GA · dual_write OFF · not Memory GA.
	PullContinuous bool `toml:"pull_continuous"`
	// PullRole optional X-IOMesh-Role on mesh auth (operator|admin|agent|auditor|viewer|memory|custom).
	// Fail-open empty → omit header. Beta federated ACL (s675/s687); not full IdP RBAC.
	// role=memory → default filter tenant.memory.> (peer aion s686).
	PullRole string `toml:"pull_role"`
	// PullAllowSuffix optional X-IOMesh-Pull-Allow-Suffix (comma-separated tokens for role=custom).
	// Fail-open empty → omit. s675 / aion s671 peer.
	PullAllowSuffix string `toml:"pull_allow_suffix"`
	// AnalyzeContinuous opt-in in-session analyze ticks on agent Runtime (s1534 P6).
	// Default OFF. Env: IOMESH_MEMORY_ANALYZE_CONTINUOUS.
	// analyze tick ≠ invent Connected / Memory GA · dual_write OFF · not Memory GA.
	AnalyzeContinuous bool `toml:"analyze_continuous"`
	// AnalyzeIntervalSec tick period seconds (default 0 → treat as 300; Runtime floors at 30).
	// Env: IOMESH_MEMORY_ANALYZE_INTERVAL_SEC.
	AnalyzeIntervalSec int `toml:"analyze_interval_sec"`
	// AnalyzeMode "status" | "digest" (default "status" when empty).
	// Env: IOMESH_MEMORY_ANALYZE_MODE.
	AnalyzeMode string `toml:"analyze_mode"`
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
			IncludeLineage:  true,
			PolicyMode:      "off",
			CatalogPlane:    true,
			InjectCatalog:   false, // opt-in: use list_mesh_catalog or set true
		},
		UI: UISection{
			SimpleMode:   true,
			ScreenMode:   "fullscreen",
			ShowThinking: true,
			Theme:        "default",
		},
		Features: FeaturesSection{
			Subagents:     true,
			CodebaseIndex: true,
			Skills:        true,
			MCP:           true,
		},
		Subagents: SubagentsSection{
			Enabled:       true,
			MaxConcurrent: 32, // max parallel running children
			MaxDepth:      2,
			MaxBatch:      64, // max tasks per spawn_subagents call
		},
		Skills: SkillsSection{
			Enabled: true,
		},
		MCP: MCPSection{
			Enabled: false, // opt-in: no servers until configured
		},
		Plugins: PluginsSection{
			Enabled: false, // s1331: opt-in Agent Plugins package wire
		},
		Memory: MemorySection{
			Enabled:          false,
			Server:           "memory",
			AutoRecall:       true,  // when enabled
			AutoIngest:       false, // opt-in write path
			DualWrite:        false, // s768: dual_write default OFF (local-primary honesty)
			Limit:            8,
			MaxSnippetBytes:  6000,
			RecallCacheTTLMS: 3000, // s1069
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
	if v := os.Getenv("IOMESH_ORG"); v != "" {
		c.IOMesh.Org = v
	} else if v := os.Getenv("MEMORY_ORG"); v != "" && c.IOMesh.Org == "" {
		c.IOMesh.Org = v
	}
	if v := os.Getenv("IOMESH_WORKSPACE"); v != "" {
		c.IOMesh.Workspace = v
	}
	if v := os.Getenv("IOMESH_POLICY_MODE"); v != "" {
		c.IOMesh.PolicyMode = strings.ToLower(strings.TrimSpace(v))
	}
	if v := os.Getenv("IOMESH_INCLUDE_LINEAGE"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.IOMesh.IncludeLineage = false
		case "1", "true", "on", "yes":
			c.IOMesh.IncludeLineage = true
		}
	}
	if v := os.Getenv("IOMESH_CATALOG_PLANE"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.IOMesh.CatalogPlane = false
		case "1", "true", "on", "yes":
			c.IOMesh.CatalogPlane = true
		}
	}
	if v := os.Getenv("IOMESH_INJECT_CATALOG"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.IOMesh.InjectCatalog = false
		case "1", "true", "on", "yes":
			c.IOMesh.InjectCatalog = true
		}
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
	if v := os.Getenv("IOMESH_SKILLS"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.Skills.Enabled = false
			c.Features.Skills = false
		case "1", "true", "on", "yes":
			c.Skills.Enabled = true
			c.Features.Skills = true
		}
	}
	if v := os.Getenv("IOMESH_MCP"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.MCP.Enabled = false
			c.Features.MCP = false
		case "1", "true", "on", "yes":
			c.MCP.Enabled = true
			c.Features.MCP = true
		}
	}
	if v := os.Getenv("IOMESH_MEMORY"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.Memory.Enabled = false
		case "1", "true", "on", "yes":
			c.Memory.Enabled = true
		}
	}
	if v := os.Getenv("IOMESH_MEMORY_TENANT"); v != "" {
		c.Memory.Tenant = v
	} else if v := os.Getenv("MEMORY_TENANT"); v != "" && c.Memory.Tenant == "" {
		c.Memory.Tenant = v
	}
	if v := os.Getenv("IOMESH_MEMORY_AUTO_RECALL"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.Memory.AutoRecall = false
		case "1", "true", "on", "yes":
			c.Memory.AutoRecall = true
		}
	}
	if v := os.Getenv("IOMESH_MEMORY_AUTO_INGEST"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.Memory.AutoIngest = false
		case "1", "true", "on", "yes":
			c.Memory.AutoIngest = true
		}
	}
	if v := os.Getenv("IOMESH_MEMORY_DUAL_WRITE"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.Memory.DualWrite = false
		case "1", "true", "on", "yes":
			c.Memory.DualWrite = true
		}
	}
	// Memory pull (mesh → local palace) s652 / continuous s1530 P5.
	if v := os.Getenv("IOMESH_MEMORY_PULL_STREAM"); v != "" {
		c.Memory.PullStream = v
	}
	if v := os.Getenv("IOMESH_MEMORY_PULL_CONSUMER"); v != "" {
		c.Memory.PullConsumer = v
	}
	if v := os.Getenv("IOMESH_MEMORY_PULL_FILTER"); v != "" {
		c.Memory.PullFilter = v
	}
	if v := os.Getenv("IOMESH_MEMORY_PULL_CONTINUOUS"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.Memory.PullContinuous = false
		case "1", "true", "on", "yes":
			c.Memory.PullContinuous = true
		}
	}
	if v := os.Getenv("IOMESH_MEMORY_PULL_ROLE"); v != "" {
		c.Memory.PullRole = v
	}
	if v := os.Getenv("IOMESH_MEMORY_PULL_ALLOW_SUFFIX"); v != "" {
		c.Memory.PullAllowSuffix = v
	}
	// Analyze ticks (s1534 P6 · opt-in; default OFF).
	if v := os.Getenv("IOMESH_MEMORY_ANALYZE_CONTINUOUS"); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "off", "no":
			c.Memory.AnalyzeContinuous = false
		case "1", "true", "on", "yes":
			c.Memory.AnalyzeContinuous = true
		}
	}
	if v := os.Getenv("IOMESH_MEMORY_ANALYZE_INTERVAL_SEC"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			c.Memory.AnalyzeIntervalSec = n
		}
	}
	if v := os.Getenv("IOMESH_MEMORY_ANALYZE_MODE"); v != "" {
		c.Memory.AnalyzeMode = v
	}
	// Memory sidecar base for sync retrieve (stage warm plane). Prefer explicit IOMESH_*.
	if v := os.Getenv("IOMESH_MEMORY_ENDPOINT"); v != "" {
		c.Memory.Endpoint = v
	} else if v := os.Getenv("MEMORY_SIDECAR_URL"); v != "" && c.Memory.Endpoint == "" {
		c.Memory.Endpoint = v
	}
	// s1068 temporal auto-recall window (RFC3339).
	if v := os.Getenv("IOMESH_MEMORY_RECALL_SINCE"); v != "" {
		c.Memory.RecallSince = v
	}
	if v := os.Getenv("IOMESH_MEMORY_RECALL_UNTIL"); v != "" {
		c.Memory.RecallUntil = v
	}
	if v := os.Getenv("IOMESH_MEMORY_RECALL_SESSION_SEQ"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			c.Memory.RecallSessionSeq = n
		}
	}
	if c.Memory.Server == "" {
		c.Memory.Server = "memory"
	}
}

// NewRouter constructs a router.Router from this config.
func (c *Config) NewRouter(opts ...router.Option) (*router.Router, error) {
	if c.Router.MaxAttempts > 0 {
		opts = append([]router.Option{router.WithMaxAttempts(c.Router.MaxAttempts)}, opts...)
	}
	return router.New(c.Catalog, c.Models.Default, opts...)
}

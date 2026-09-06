// Package runtimewire maps config → agent runtime skill dirs and MCP server configs.
// Shared by cmd/iomesh agent bootstrap, mcp --connect, skills list, ACP session build,
// and post-setup hot reload (s1526 P4 ReplaceMCP · s1670 ReplaceSkills / skills re-scan ·
// s2055 ReplaceMesh when [iomesh] or inferred hooks change).
//
// Wire returns SkillDirs for LoadWithBuiltin on bootstrap and /setup reload.
// ConnectMCP builds managers from Wire MCPServers. NewMesh applies inferred broker
// then builds the iomesh client. Callers hot-swap via Runtime.ReplaceMCP,
// Runtime.ReplaceSkills, and Runtime.ReplaceMesh.
//
// Residual honesty:
//   - package wire ≠ Connected / install APPLY green / Agent Plugins GA / Memory GA
//   - dual_write OFF (not flipped here)
//   - Discover / map success ≠ process Connected
//   - skills re-scan ≠ invent Connected
//   - infer ≠ Connected · catalog MCP ≠ hooks streams
//   - TOML [[mcp.servers]] remains primary; plugins append after TOML
package runtimewire

import (
	"context"
	"log/slog"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/agentplugins"
	"github.com/iome-sh/iomesh-tui/internal/config"
	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/skills"
)

// Result holds config→runtime wiring for skills discovery and MCP attach.
// package wire ≠ Connected · dual_write OFF.
type Result struct {
	// SkillDirs is the full load path: DefaultDirs(workspace) + [skills].dirs + plugin skill dirs.
	SkillDirs []string
	// PluginSkillDirs is the plugin-only subset (may be empty).
	PluginSkillDirs []string
	// MCPServers is TOML [[mcp.servers]] then plugin-mapped servers (ready for mcp.NewManager).
	MCPServers []mcp.ServerConfig
	// TOMLServerCount is len of servers built from TOML only.
	TOMLServerCount int
	// PluginServerCount is len of servers mapped from plugins.
	PluginServerCount int
	// PluginCount is number of discovered plugins (Discover ≠ Connected).
	PluginCount int
	// Warnings are fail-open discover/map messages (also logged when Logger set).
	Warnings []string
}

// Wire derives skill dirs and MCP server configs from cfg + workspace.
// Mirrors cmd/iomesh agent bootstrap:
//   - plugins DiscoverAll when Plugins.Enabled && Dirs non-empty
//   - SkillDirs from plugins after [skills].dirs
//   - MCPServersFromPlugins after TOML servers (BuildMCPServerConfig / s1267 inject)
//
// Does not connect servers or load skills. dual_write not flipped.
func Wire(cfg *config.Config, workspace string, logger *slog.Logger) Result {
	var res Result
	if cfg == nil {
		cfg = &config.Config{}
	}
	if logger == nil {
		logger = slog.Default()
	}

	// Skills path always computed (CLI list + agent gate separately).
	dirs := skills.DefaultDirs(workspace)
	dirs = append(dirs, cfg.Skills.Dirs...)

	var discovered []*agentplugins.Plugin
	if cfg.Plugins.Enabled && len(cfg.Plugins.Dirs) > 0 {
		var pw []string
		discovered, pw = agentplugins.DiscoverAll(cfg.Plugins.Dirs)
		res.PluginCount = len(discovered)
		for _, w := range pw {
			res.Warnings = append(res.Warnings, w)
			logger.Warn("plugins discover", "warn", w)
		}
		logger.Info("plugins discovered", "count", len(discovered), "dirs", len(cfg.Plugins.Dirs))
		if skillDirs := agentplugins.SkillDirs(discovered); len(skillDirs) > 0 {
			res.PluginSkillDirs = skillDirs
			dirs = append(dirs, skillDirs...)
			logger.Info("plugins skill dirs", "count", len(skillDirs))
		}
	}
	res.SkillDirs = dirs

	// TOML primary; plugin MCP append (s1331). Built even when MCP feature off so
	// list CLIs can show config; callers gate NewManager on Enabled + Features.MCP.
	var servers []mcp.ServerConfig
	for _, s := range cfg.MCP.Servers {
		servers = append(servers, cfg.BuildMCPServerConfig(s))
	}
	res.TOMLServerCount = len(servers)

	if len(discovered) > 0 {
		dataRoot := agentplugins.DefaultPluginDataRoot(workspace, cfg.Plugins.DataDir)
		pluginServers, mw := agentplugins.MCPServersFromPlugins(discovered, dataRoot)
		for _, w := range mw {
			res.Warnings = append(res.Warnings, w)
			logger.Warn("plugins mcp map", "warn", w)
		}
		if len(pluginServers) > 0 {
			servers = append(servers, pluginServers...)
			res.PluginServerCount = len(pluginServers)
			logger.Info("plugins mcp servers mapped", "count", len(pluginServers), "data_root", dataRoot)
		}
	}
	res.MCPServers = servers
	return res
}

// MCPFeatureOn reports whether agent/MCP attach paths should connect servers.
func MCPFeatureOn(cfg *config.Config) bool {
	return cfg != nil && cfg.MCP.Enabled && cfg.Features.MCP
}

// SkillsFeatureOn reports whether agent should load skill catalogs.
func SkillsFeatureOn(cfg *config.Config) bool {
	return cfg != nil && cfg.Skills.Enabled && cfg.Features.Skills
}

// ConnectMCP builds an mcp.Manager from Wire when MCP feature is on and servers exist.
// Returns nil when disabled, no servers, or cfg nil. Fail-open per server inside Manager.
// Connect success ≠ Connected product green · dual_write OFF · package wire ≠ GA.
func ConnectMCP(ctx context.Context, cfg *config.Config, workspace string, logger *slog.Logger) *mcp.Manager {
	if !MCPFeatureOn(cfg) {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	w := Wire(cfg, workspace, logger)
	if len(w.MCPServers) == 0 {
		return nil
	}
	return mcp.NewManager(ctx, w.MCPServers, logger)
}

// NewMesh builds the session mesh client from cfg, applying inferred hooks
// when [iomesh] is unset. Infer ≠ Connected. Empty infer does not invent Enabled.
// The second return is zero unless infer actually filled [iomesh].
func NewMesh(cfg *config.Config, logger *slog.Logger) (*iomesh.Client, config.InferredBroker) {
	if cfg == nil {
		return iomesh.New(iomesh.Config{}, logger), config.InferredBroker{}
	}
	inf := config.ApplyInferredBroker(cfg)
	if logger == nil {
		logger = slog.Default()
	}
	if inf.Endpoint != "" {
		logger.Info("inferred broker from portal MCP (catalog ≠ streams · infer ≠ Connected)",
			"endpoint", inf.Endpoint)
	}
	return iomesh.New(iomesh.Config{
		Enabled:         cfg.IOMesh.Enabled,
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          cfg.IOMesh.Tenant,
		APIKeyEnv:       cfg.IOMesh.APIKeyEnv,
		OrgID:           cfg.IOMesh.Org,
		WorkspaceID:     cfg.IOMesh.Workspace,
		Department:      cfg.IOMesh.Department,
		DualWrite:       cfg.Memory.DualWrite,
		MemoryEndpoint:  cfg.Memory.Endpoint,
		EmitDeptStreams: cfg.IOMesh.EmitDeptStreams,
		ContextPlane:    cfg.IOMesh.ContextPlane,
		IncludeLineage:  cfg.IOMesh.IncludeLineage,
		PolicyMode:      iomesh.PolicyMode(cfg.IOMesh.PolicyMode),
		CatalogPlane:    cfg.IOMesh.CatalogPlane,
		InjectCatalog:   cfg.IOMesh.InjectCatalog,
		Role:            strings.TrimSpace(cfg.Memory.PullRole),
		PullAllowSuffix: strings.TrimSpace(cfg.Memory.PullAllowSuffix),
	}, logger), inf
}

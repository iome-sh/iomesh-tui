// Command iomesh is the I/O Mesh TUI coding agent (Go rewrite of Grok Build),
// with DeepSeek-first model routing and optional I/O Mesh platform integration.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/acp"
	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/agentplugins"
	"github.com/iome-sh/iomesh-tui/internal/config"
	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/runtimewire"
	"github.com/iome-sh/iomesh-tui/internal/session"
	"github.com/iome-sh/iomesh-tui/internal/setup"
	"github.com/iome-sh/iomesh-tui/internal/skills"
	"github.com/iome-sh/iomesh-tui/internal/tui"
	"github.com/iome-sh/iomesh-tui/internal/workspace"
)

// Overridden at link time by make build: -X main.version=$(VERSION)
// (must be a var, not const, for -ldflags -X).
var version = "1.1.0"

func main() {
	// Identify mesh HTTP traffic for operator support (parity with iomesh-client-sdk-go User-Agent).
	iomesh.SetUserAgent("iomesh-tui/" + version)
	// Product version for StatusLine version= and dogfood report default.
	iomesh.SetProductVersion(version)
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "version", "--version", "-V":
			fmt.Printf("iomesh %s\n", version)
			return 0
		case "models":
			return cmdModels(args[1:])
		case "sessions":
			return cmdSessions(args[1:])
		case "skills":
			return cmdSkills(args[1:])
		case "mcp":
			return cmdMCP(args[1:])
		case "plugins":
			return cmdPlugins(args[1:])
		case "mesh":
			return cmdMesh(args[1:])
		case "memory":
			return cmdMemory(args[1:])
		case "setup":
			return cmdSetup(args[1:])
		case "agent":
			return cmdAgent(args[1:])
		case "help", "-h", "--help":
			printUsage()
			return 0
		}
	}

	fs := flag.NewFlagSet("iomesh", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		prompt       = fs.String("p", "", "headless single-prompt mode (print response and exit)")
		model        = fs.String("m", "", "logical model name override (e.g. deepseek-v4-flash)")
		configPath   = fs.String("config", "", "path to config.toml (default: ~/.iomesh/config.toml)")
		workspace    = fs.String("C", "", "workspace directory (default: cwd)")
		yolo         = fs.Bool("yolo", false, "auto-approve mutating tools")
		verbose      = fs.Bool("v", false, "verbose logging")
		continueS    = fs.Bool("c", false, "continue latest session in workspace")
		continueLong = fs.Bool("continue", false, "alias for -c")
		sessionID    = fs.String("session", "", "load session id from .iomesh/sessions")
		noSave       = fs.Bool("no-save", false, "disable session autosave")
		repl         = fs.Bool("repl", false, "force classic line REPL instead of full-screen TUI")
	)
	// Also accept --prompt long form.
	promptLong := fs.String("prompt", "", "alias for -p")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *prompt == "" && *promptLong != "" {
		*prompt = *promptLong
	}
	if *continueLong {
		*continueS = true
	}

	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *workspace != "" {
		cfg.Agent.Workspace = *workspace
	}
	if *yolo {
		cfg.Agent.Yolo = true
	}

	var metrics router.MetricsSink = router.NopMetrics{}
	// s675/s1530: wire [memory].pull_role / pull_allow_suffix onto Client so in-session
	// continuous pull (and consumer create/fetch) send federated ACL headers.
	// Fail-open empty → omit. dual_write remains report-only default OFF.
	// s2055: infer hooks from portal MCP when [iomesh] unset (infer ≠ Connected).
	mesh, inf := runtimewire.NewMesh(cfg, logger)
	if inf.Endpoint != "" {
		fmt.Fprintf(os.Stderr, "note: inferred broker %s from portal MCP (catalog ≠ streams)\n", inf.Endpoint)
	}
	if mesh.Enabled() {
		metrics = mesh
	}

	rtr, err := cfg.NewRouter(router.WithLogger(logger), router.WithMetrics(metrics))
	if err != nil {
		fmt.Fprintf(os.Stderr, "router: %v\n", err)
		return 1
	}
	if *model != "" {
		if err := rtr.SetOverride(*model); err != nil {
			fmt.Fprintf(os.Stderr, "model: %v\n", err)
			return 1
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mode := agent.ModeTUI
	if *prompt != "" {
		mode = agent.ModeHeadless
	}

	// [subagents].enabled is SSOT; features.subagents=false forces off.
	subEnabled := cfg.Subagents.Enabled && cfg.Features.Subagents

	processCfgPath, err := config.ResolvePath(*configPath)
	if err != nil {
		processCfgPath = strings.TrimSpace(*configPath)
	}

	rt, err := agent.New(agent.Config{
		Mode:                  mode,
		Workspace:             cfg.Agent.Workspace,
		Yolo:                  cfg.Agent.Yolo,
		SubagentsEnabled:      subEnabled,
		MaxSubagentDepth:      cfg.Subagents.MaxDepth,
		MaxSubagentConcurrent: cfg.Subagents.MaxConcurrent,
		MaxSubagentBatch:      cfg.Subagents.MaxBatch,
		WorktreeBase:          cfg.Subagents.WorktreeBase,
		WorktreeAutoRemove:    cfg.Subagents.WorktreeAutoRemove,
		ConfigPath:            processCfgPath,
	}, rtr, mesh, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent: %v\n", err)
		return 1
	}
	defer func() { _ = rt.Close() }()

	// Mesh catalog tools (list_mesh_catalog / mesh_status) when catalog plane on.
	rt.AttachMeshTools()

	// Config→runtime package wire (s1331 + s1526 P4): plugins DiscoverAll → skill dirs +
	// MCP server configs (TOML primary, plugins append). package wire ≠ Connected / GA.
	// dual_write OFF · Discover success ≠ install APPLY green.
	wired := runtimewire.Wire(cfg, rt.Workspace().Root(), logger)

	// Skills: builtin (s1251 connector-integrations-setup) + workspace + user dirs.
	// Builtin always present when skills enabled so residual-honest connector setup
	// guidance appears even if user/workspace skill dirs are empty.
	// Plugin skills (s1331) append after [skills].dirs when plugins enabled.
	if runtimewire.SkillsFeatureOn(cfg) {
		if cat, err := skills.LoadWithBuiltin(wired.SkillDirs...); err != nil {
			logger.Warn("skills load", "err", err)
		} else if cat.Len() > 0 {
			rt.AttachSkills(cat)
			logger.Info("skills loaded", "count", cat.Len())
		}
	}

	// MCP stdio/HTTP servers (opt-in). Fail-open per server inside manager.
	// TOML [[mcp.servers]] remains primary; plugin MCP servers append after TOML.
	// Plugins-only MCP is allowed when [mcp] enabled and TOML servers empty.
	if runtimewire.MCPFeatureOn(cfg) && len(wired.MCPServers) > 0 {
		mgr := mcp.NewManager(ctx, wired.MCPServers, logger)
		rt.AttachMCP(mgr)
	}

	// Memory Palace hooks (MCP server and/or dual-write MEMORY_INGEST).
	if cfg.Memory.Enabled {
		rt.AttachMemory(agent.MemoryConfig{
			Enabled:          true,
			Server:           cfg.Memory.Server,
			Tenant:           cfg.Memory.Tenant,
			AutoRecall:       cfg.Memory.AutoRecall,
			AutoIngest:       cfg.Memory.AutoIngest,
			DualWrite:        cfg.Memory.DualWrite,
			Limit:            cfg.Memory.Limit,
			MaxSnippetBytes:  cfg.Memory.MaxSnippetBytes,
			RecallSince:      cfg.Memory.RecallSince,
			RecallUntil:      cfg.Memory.RecallUntil,
			RecallSessionSeq: cfg.Memory.RecallSessionSeq,
			RecallCacheTTLMS: cfg.Memory.RecallCacheTTLMS,
		})
	}

	// s1530 P5: opt-in in-session continuous memory pull (default OFF).
	// Only when pull_continuous && pull_consumer set. Fail-open log warn on start error
	// (do not fail process start). pull running ≠ invent install green / Ops Pack GA.
	if cfg.Memory.PullContinuous {
		consumer := strings.TrimSpace(cfg.Memory.PullConsumer)
		if consumer != "" {
			stream := strings.TrimSpace(cfg.Memory.PullStream)
			server := strings.TrimSpace(cfg.Memory.Server)
			tenant := strings.TrimSpace(cfg.Memory.Tenant)
			if tenant == "" {
				tenant = strings.TrimSpace(cfg.IOMesh.Tenant)
			}
			if err := rt.StartContinuousMemoryPull(agent.ContinuousPullConfig{
				Enabled:   true,
				Stream:    stream,
				Consumer:  consumer,
				Filter:    strings.TrimSpace(cfg.Memory.PullFilter),
				Batch:     cfg.Memory.PullBatch,
				MaxWaitMS: cfg.Memory.PullMaxWaitMS,
				Server:    server,
				Tenant:    tenant,
			}); err != nil {
				logger.Warn("continuous memory pull auto-start failed", "err", err)
			} else {
				logger.Info("continuous memory pull started",
					"stream", stream, "consumer", consumer,
					"role", strings.TrimSpace(cfg.Memory.PullRole))
			}
		} else {
			logger.Warn("pull_continuous set but pull_consumer empty; continuous pull not started")
		}
	}

	// s1534 P6: opt-in in-session analyze ticks (default OFF).
	// Fail-open log warn on start error — do not fail process start.
	// analyze tick ≠ invent Connected / Memory GA · dual_write OFF.
	if cfg.Memory.AnalyzeContinuous {
		mode := strings.TrimSpace(cfg.Memory.AnalyzeMode)
		if err := rt.StartAnalyzeTick(agent.AnalyzeTickConfig{
			Enabled:     true,
			IntervalSec: cfg.Memory.AnalyzeIntervalSec,
			Mode:        mode,
		}); err != nil {
			logger.Warn("analyze tick auto-start failed", "err", err)
		} else {
			logger.Info("analyze tick started",
				"mode", mode, "interval_sec", cfg.Memory.AnalyzeIntervalSec)
		}
	}

	store, err := session.Open(rt.Workspace().Root())
	if err != nil {
		fmt.Fprintf(os.Stderr, "session store: %v\n", err)
		return 1
	}
	if !*noSave {
		rt.EnableAutoSave(true)
	}

	// Resume session (subagent registry + transcript).
	if *sessionID != "" || *continueS {
		var snap *session.Snapshot
		if *sessionID != "" {
			snap, err = store.Load(*sessionID)
		} else {
			snap, err = store.Latest()
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "session load: %v\n", err)
			return 1
		}
		if snap == nil {
			fmt.Fprintln(os.Stderr, "session: no saved sessions in workspace")
			return 1
		}
		if err := rt.LoadSession(snap); err != nil {
			fmt.Fprintf(os.Stderr, "session restore: %v\n", err)
			return 1
		}
		logger.Info("session resumed", "id", snap.ID, "messages", len(snap.Messages), "subagents", len(snap.Subagents))
		fmt.Fprintf(os.Stderr, "resumed session %s (%d messages, %d subagents)\n", snap.ID, len(snap.Messages), len(snap.Subagents))
	}

	// Headless single prompt.
	if *prompt != "" {
		if err := rt.RunHeadless(ctx, *prompt, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		if rt.SessionID() == "" || !*noSave {
			if _, err := rt.SaveSession(store, 0); err != nil {
				logger.Warn("session save", "err", err)
			} else {
				fmt.Fprintf(os.Stderr, "session saved: %s\n", rt.SessionID())
			}
		}
		// Local process metering rollup (stderr so stdout stays clean for scripts).
		if snap := mesh.Usage(); snap.Calls > 0 {
			fmt.Fprint(os.Stderr, iomesh.FormatUsage(snap))
		}
		return 0
	}

	// Interactive TUI: full-screen Bubble Tea by default; --repl for classic line mode.
	var tuiErr error
	if *repl {
		tuiErr = tui.RunREPL(ctx, rt, store, logger)
	} else {
		tuiErr = tui.RunWithStoreOpts(ctx, rt, store, logger, tui.UIOptions{
			Theme: cfg.UI.Theme,
		})
	}
	if tuiErr != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", tuiErr)
		return 1
	}
	return 0
}

func cmdSessions(args []string) int {
	fs := flag.NewFlagSet("sessions", flag.ContinueOnError)
	workspace := fs.String("C", "", "workspace directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ws := *workspace
	if ws == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		ws = wd
	}
	st, err := session.Open(ws)
	if err != nil {
		fmt.Fprintf(os.Stderr, "session: %v\n", err)
		return 1
	}
	list, err := st.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "session list: %v\n", err)
		return 1
	}
	if len(list) == 0 {
		fmt.Println("no sessions")
		return 0
	}
	fmt.Printf("%-28s %5s %5s  %-20s  %s\n", "ID", "MSGS", "SUBS", "UPDATED", "TITLE")
	for _, s := range list {
		fmt.Printf("%-28s %5d %5d  %-20s  %s\n",
			s.ID, s.Messages, s.Subagents, s.UpdatedAt.Format(time.RFC3339), s.Title)
	}
	return 0
}

func cmdSkills(args []string) int {
	fs := flag.NewFlagSet("skills", flag.ContinueOnError)
	workspace := fs.String("C", "", "workspace directory")
	configPath := fs.String("config", "", "path to config.toml")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	ws := *workspace
	if ws == "" {
		ws = cfg.Agent.Workspace
	}
	if ws == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		ws = wd
	}
	// s1331 + s1526 P4: shared wire (DefaultDirs + [skills].dirs + plugin skill dirs).
	wired := runtimewire.Wire(cfg, ws, slog.Default())
	for _, w := range wired.Warnings {
		fmt.Fprintf(os.Stderr, "plugins: %s\n", w)
	}
	// Include builtin skills (s1251 connector-integrations-setup) so CLI mirrors agent.
	cat, err := skills.LoadWithBuiltin(wired.SkillDirs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skills: %v\n", err)
		return 1
	}
	if cat.Len() == 0 {
		fmt.Println("no skills found")
		fmt.Fprintf(os.Stderr, "searched: builtin + %s\n", strings.Join(wired.SkillDirs, ", "))
		return 0
	}
	fmt.Printf("%-24s  %s\n", "NAME", "DESCRIPTION")
	for _, sk := range cat.List() {
		desc := strings.ReplaceAll(sk.Description, "\n", " ")
		if len(desc) > 80 {
			desc = desc[:77] + "…"
		}
		fmt.Printf("%-24s  %s\n", sk.Name, desc)
	}
	return 0
}

func cmdMCP(args []string) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to config.toml")
	connect := fs.Bool("connect", false, "actually spawn servers and list tools (slow)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	// TOML primary; plugin MCP appended (s1331 / s1526 P4 shared wire).
	ws := cfg.Agent.Workspace
	if ws == "" {
		if wd, err := os.Getwd(); err == nil {
			ws = wd
		}
	}
	wired := runtimewire.Wire(cfg, ws, slog.Default())
	for _, w := range wired.Warnings {
		fmt.Fprintf(os.Stderr, "plugins: %s\n", w)
	}
	if wired.TOMLServerCount == 0 && wired.PluginServerCount == 0 {
		fmt.Println("no MCP servers configured ([mcp] / [[mcp.servers]] / [plugins] in config.toml)")
		return 0
	}
	fmt.Printf("mcp.enabled=%v features.mcp=%v plugins.enabled=%v\n", cfg.MCP.Enabled, cfg.Features.MCP, cfg.Plugins.Enabled)
	fmt.Printf("%-24s %-8s %-8s %-6s %s\n", "NAME", "ENABLED", "MUTATING", "MODE", "ENDPOINT")
	for _, s := range cfg.MCP.Servers {
		en := true
		if s.Enabled != nil {
			en = *s.Enabled
		}
		mut := true
		if s.Mutating != nil {
			mut = *s.Mutating
		}
		mode, ep := "stdio", s.Command+" "+strings.Join(s.Args, " ")
		if s.URL != "" {
			mode, ep = "http", s.URL
		}
		fmt.Printf("%-24s %-8v %-8v %-6s %s\n", s.Name, en, mut, mode, strings.TrimSpace(ep))
	}
	// Plugin-mapped servers (after TOML slice in wired.MCPServers).
	for _, s := range wired.MCPServers[wired.TOMLServerCount:] {
		mode, ep := "stdio", s.Command+" "+strings.Join(s.Args, " ")
		if s.URL != "" {
			mode, ep = "http", s.URL
		}
		// Plugin-mapped servers: mutating default true (nil).
		fmt.Printf("%-24s %-8v %-8v %-6s %s\n", s.Name, true, true, mode, strings.TrimSpace(ep))
	}
	if !*connect || !cfg.MCP.Enabled {
		if !*connect {
			fmt.Fprintln(os.Stderr, "(pass --connect to probe tools/list)")
		}
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	mgr := mcp.NewManager(ctx, wired.MCPServers, slog.Default())
	defer mgr.Close()
	fmt.Printf("\nconnected=%d\n", mgr.Len())
	for _, b := range mgr.Bindings() {
		fmt.Printf("  %s  (server=%s tool=%s mutating=%v)\n", b.Qualified, b.Server, b.Tool, b.Mutating)
	}
	for _, c := range mgr.Clients() {
		fmt.Printf("  server %s resources=%d prompts=%d\n", c.Name(), len(c.Resources()), len(c.Prompts()))
	}
	return 0
}

// mcpServerFromTOML maps config TOML to mcp.ServerConfig (s1267 inject via BuildMCPServerConfig).
// Kept for memory-pull path; agent/mcp/skills bootstrap use runtimewire.Wire.
func mcpServerFromTOML(s config.MCPServerTOML, cfg *config.Config) mcp.ServerConfig {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return cfg.BuildMCPServerConfig(s)
}

// cmdSetup is setup lifecycle CLI (s1525 P1–P2): init managed config + residual-honest preflight.
// dual_write OFF · not Memory GA · catalog ≠ Connected · portal HITL · PASS ≠ invent install green.
func cmdSetup(args []string) int {
	if len(args) == 0 {
		printSetupUsage()
		return 2
	}
	switch args[0] {
	case "init":
		return cmdSetupInit(args[1:])
	case "preflight", "check", "status":
		return cmdSetupPreflight(args[1:])
	case "help", "-h", "--help":
		printSetupUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown setup subcommand %q\n", args[0])
		printSetupUsage()
		return 2
	}
}

func printSetupUsage() {
	fmt.Fprint(os.Stderr, `iomesh setup — agent-native setup lifecycle (s1525 · residual-honest · s1686)

  iomesh setup init [profiles]   write managed config fragment (default: local-memory)
  iomesh setup preflight         probe config + local memory healthz (not invent Connected)
  iomesh setup help

  (no iomesh setup reload — in-session /setup reload only · hot-swap MCP + skills · package wire ≠ Connected)

Profiles: local-memory | plugins | mesh | platform-mcp | all
  (positional and/or --profiles csv)

Flags (init):
  --config path           config.toml (default: user config path)
  --profiles list         csv profiles (alternative to positionals)
  --stdio                 local-memory via stdio command iomesh-memory-mcp
  --memory-url URL        default http://127.0.0.1:8080/mcp
  --plugins-dir path      repeatable [plugins].dirs entry
  --mesh-endpoint URL     optional mesh base
  --mesh-tenant id        optional tenant
  --mesh-org id           optional [iomesh].org / IOMESH_ORG (X-IOMesh-Org; empty fail-opens)
  --platform-mcp-url URL  portal Agent/MCP streamable HTTP URL
  --print-only            print fragment only (do not write)

Flags (preflight):
  --config path
  --json                  always-emit PreflightReport JSON

After init: memory host (if local-memory) · secret env vars ·
  TUI already running → /setup preflight · /setup reload · else cold start → restart iomesh · iomesh setup preflight.

Honesty: dual_write OFF · not Memory GA · secrets via env refs only ·
  portal HITL for OAuth/install · setup PASS ≠ invent Connected / INSTALL_STORE green ·
  package wire ≠ Connected · free eng s1686.
  Continuous pull: iomesh memory pull (in-session /setup pull). Analyze: /memory digest.
`)
}

func cmdSetupInit(args []string) int {
	fs := flag.NewFlagSet("setup init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "", "config.toml path (default user path)")
	profilesFlag := fs.String("profiles", "", "comma-separated profiles (local-memory|plugins|mesh|platform-mcp|all)")
	stdio := fs.Bool("stdio", false, "use stdio iomesh-memory-mcp instead of HTTP URL")
	memoryURL := fs.String("memory-url", "", "memory MCP HTTP URL")
	meshEP := fs.String("mesh-endpoint", "", "mesh endpoint URL")
	meshTenant := fs.String("mesh-tenant", "", "mesh tenant")
	meshOrg := fs.String("mesh-org", "", "mesh org id ([iomesh].org / IOMESH_ORG; X-IOMesh-Org)")
	platformURL := fs.String("platform-mcp-url", "", "platform MCP URL from portal")
	printOnly := fs.Bool("print-only", false, "print managed fragment only")
	var pluginDirs multiFlag
	fs.Var(&pluginDirs, "plugins-dir", "plugins package dir (repeatable)")
	// Allow flags after profile tokens: scan flags first by moving all -flags to front.
	args = hoistFlags(args)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	var profiles []setup.Profile
	if p := strings.TrimSpace(*profilesFlag); p != "" {
		profiles = setup.ParseProfiles(p)
	} else if len(fs.Args()) > 0 {
		profiles = setup.ParseProfiles(strings.Join(fs.Args(), ","))
	} else {
		profiles = []setup.Profile{setup.ProfileLocalMemory}
	}

	opt := setup.DefaultInitOptions()
	opt.UseStdioMemory = *stdio
	if strings.TrimSpace(*memoryURL) != "" {
		opt.MemoryHTTPURL = strings.TrimSpace(*memoryURL)
	}
	opt.MeshEndpoint = strings.TrimSpace(*meshEP)
	opt.MeshTenant = strings.TrimSpace(*meshTenant)
	opt.MeshOrg = strings.TrimSpace(*meshOrg)
	opt.PlatformMCPURL = strings.TrimSpace(*platformURL)
	opt.PluginsDirs = append([]string{}, pluginDirs...)

	frag, err := setup.BuildManagedFragment(profiles, opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup init: %v\n", err)
		return 1
	}
	if *printOnly {
		fmt.Print(frag)
		if !strings.HasSuffix(frag, "\n") {
			fmt.Println()
		}
		return 0
	}
	path := strings.TrimSpace(*configPath)
	if path == "" {
		path, err = config.UserConfigPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "setup init: config path: %v\n", err)
			return 1
		}
	}
	if err := config.WriteSetupManagedFragment(path, frag); err != nil {
		fmt.Fprintf(os.Stderr, "setup init: write: %v\n", err)
		return 1
	}
	fmt.Printf("setup init: wrote managed fragment → %s\n", path)
	fmt.Println("profiles:", profiles)
	for _, line := range setup.SetupInitNextStepLines() {
		fmt.Println(line)
	}
	if setup.ProfilesWantMesh(profiles) {
		for _, line := range setup.SetupInitMeshNextStepLines() {
			fmt.Println(line)
		}
	}
	return 0
}

func cmdSetupPreflight(args []string) int {
	fs := flag.NewFlagSet("setup preflight", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "", "config.toml path")
	jsonOut := fs.Bool("json", false, "JSON PreflightReport")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	rep, err := setup.Preflight(ctx, strings.TrimSpace(*configPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup preflight: %v\n", err)
		return 1
	}
	if *jsonOut {
		fmt.Print(setup.FormatPreflightJSON(rep))
	} else {
		fmt.Print(setup.FormatPreflightText(rep))
	}
	if !rep.OK {
		return 1
	}
	return 0
}

// multiFlag collects repeatable string flags.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

// hoistFlags moves -flag and -flag=val / -flag val pairs before positionals so
// `setup init local-memory --config path` works with the stdlib flag package.
func hoistFlags(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			// boolean flags and --flag=value need no extra arg
			if strings.Contains(a, "=") {
				continue
			}
			// known flags that take a value
			name := strings.TrimLeft(a, "-")
			switch name {
			case "config", "profiles", "memory-url", "mesh-endpoint", "mesh-tenant",
				"mesh-org", "platform-mcp-url", "plugins-dir":
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					flags = append(flags, args[i])
				}
			}
			continue
		}
		pos = append(pos, a)
	}
	return append(flags, pos...)
}

// cmdPlugins is operator DX for Agent Plugins packages (s1336 list/validate · s1357 dogfood).
// Residual honesty: list/validate/dogfood ≠ invent Agent Plugins GA · dual_write OFF ·
// Discover ≠ Connected · not Memory GA · PATH residual for binary · book-demo OFF.
// Fail-open discover; validate exits non-zero on fatal package errors or zero plugins when dirs set.
// Dogfood validates both in-repo samples offline — no MCP dial / connect.
func cmdPlugins(args []string) int {
	sub := "list"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		rest = args[1:]
	}
	switch sub {
	case "list":
		return cmdPluginsList(rest)
	case "validate":
		return cmdPluginsValidate(rest)
	case "smoke", "dogfood":
		// Public name: smoke. dogfood = legacy alias (compat · s1521).
		return cmdPluginsDogfood(rest)
	case "help", "-h", "--help":
		printPluginsUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown plugins subcommand %q\n", sub)
		printPluginsUsage()
		return 2
	}
}

func printPluginsUsage() {
	fmt.Fprint(os.Stderr, `iomesh plugins — Agent Plugins package operator DX (s1336 · s1357 · s1521 public smoke)

  iomesh plugins [list]           discover packages; table NAME VERSION SKILLS MCP WARN ROOT
  iomesh plugins validate         OK/FAIL per package root; exit 1 on fatal or zero found
  iomesh plugins smoke            offline residual-honest validate of both in-repo samples (public name)
  iomesh plugins dogfood          legacy alias for smoke (compat)
  iomesh plugins help

Flags (list|validate):
  --config path         config.toml (default: ~/.iomesh/config.toml)
  -dir path             package root or parent of roots (repeatable; comma-separated OK)
                        supplements [plugins].dirs for one-shot list/validate without enable

Flags (smoke):
  -module-root path     module root containing examples/agent-plugins/* (default: walk up from cwd for go.mod)

Honesty: list/validate/smoke ≠ invent Agent Plugins GA · dual_write OFF · Discover ≠ Connected ·
  not Memory GA · PATH residual for binary · book-demo OFF. [plugins] is opt-in (default enabled=false).
  Fail-open discover (list); validate surfaces fatals and exits non-zero. Smoke = discover/validate
  only (no MCP dial / connect). Runtime wire is separate (s1331). dogfood remains a legacy alias.
`)
}

// cmdPluginsDogfood runs offline residual-honest dogfood of both in-repo product sample packages
// (hello-iome + iomesh-memory-mcp). Discover/validate only — does not Dial MCP or require
// iomesh-memory-mcp on PATH (PATH residual; connect skip). s1357+s1478.
func cmdPluginsDogfood(args []string) int {
	fs := flag.NewFlagSet("plugins smoke", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	moduleRoot := fs.String("module-root", "", "module root with examples/agent-plugins (default: find go.mod from cwd)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	root := strings.TrimSpace(*moduleRoot)
	if root == "" {
		found, err := agentplugins.FindModuleRoot("")
		if err != nil {
			// Fallback: treat cwd as module root (operator may have samples without go.mod walk).
			cwd, cwdErr := os.Getwd()
			if cwdErr != nil {
				fmt.Fprintf(os.Stderr, "smoke: find module root: %v\n", err)
				fmt.Fprintln(os.Stderr, agentplugins.ResidualDogfoodHonesty)
				return 1
			}
			root = cwd
			fmt.Fprintf(os.Stderr, "smoke: go.mod not found above cwd; using cwd as module root (%s)\n", cwd)
		} else {
			root = found
		}
	}
	outcomes, warns, err := agentplugins.DogfoodSamples(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "smoke: %v\n", err)
		fmt.Fprintln(os.Stderr, agentplugins.ResidualDogfoodHonesty)
		return 1
	}
	for _, w := range warns {
		fmt.Fprintf(os.Stderr, "plugins: %s\n", w)
	}
	for _, o := range outcomes {
		if o.OK {
			fmt.Println(agentplugins.FormatValidateOK(o))
			for _, w := range o.Warnings {
				fmt.Fprintf(os.Stderr, "plugins %s: %s\n", o.Name, w)
			}
		} else {
			fmt.Println(agentplugins.FormatValidateFail(o.Path, o.Error))
		}
	}
	fmt.Println(agentplugins.FormatDogfoodSummary(outcomes))
	fmt.Fprintln(os.Stderr, agentplugins.ResidualDogfoodHonesty)
	// Exit 1 if any fatal, missing samples, or not both expected samples OK.
	if !agentplugins.DogfoodPass(outcomes) {
		return 1
	}
	return 0
}

func cmdPluginsList(args []string) int {
	fs := flag.NewFlagSet("plugins list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "", "path to config.toml")
	var dirsFlag agentplugins.DirFlag
	fs.Var(&dirsFlag, "dir", "plugin package root or parent (repeatable; comma-separated)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	dirs := agentplugins.MergePluginDirs(cfg.Plugins.Dirs, []string(dirsFlag))
	dirsSpecified := len(dirs) > 0
	if !dirsSpecified {
		fmt.Println(agentplugins.FormatListEmptyFooter(cfg.Plugins.Enabled, false))
		fmt.Fprintln(os.Stderr, agentplugins.ResidualCLIHonesty)
		return 0
	}
	plugins, warns := agentplugins.DiscoverAll(dirs)
	for _, w := range warns {
		fmt.Fprintf(os.Stderr, "plugins: %s\n", w)
	}
	if len(plugins) == 0 {
		fmt.Println(agentplugins.FormatListEmptyFooter(cfg.Plugins.Enabled, true))
		fmt.Fprintln(os.Stderr, agentplugins.ResidualCLIHonesty)
		return 0
	}
	fmt.Println(agentplugins.FormatListHeader())
	for _, p := range plugins {
		fmt.Println(agentplugins.FormatListRow(agentplugins.PluginToListRow(p)))
		for _, w := range p.Warnings {
			fmt.Fprintf(os.Stderr, "plugins %s: %s\n", p.Manifest.Name, w)
		}
	}
	fmt.Fprintln(os.Stderr, agentplugins.ResidualCLIHonesty)
	return 0
}

func cmdPluginsValidate(args []string) int {
	fs := flag.NewFlagSet("plugins validate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "", "path to config.toml")
	var dirsFlag agentplugins.DirFlag
	fs.Var(&dirsFlag, "dir", "plugin package root or parent (repeatable; comma-separated)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	dirs := agentplugins.MergePluginDirs(cfg.Plugins.Dirs, []string(dirsFlag))
	if len(dirs) == 0 {
		fmt.Println(agentplugins.FormatListEmptyFooter(cfg.Plugins.Enabled, false))
		fmt.Fprintln(os.Stderr, agentplugins.ResidualCLIHonesty)
		// No dirs specified → residual-honest guidance; not a fatal package error.
		// When dirs were intended, operators pass -dir or set [plugins].dirs.
		if !cfg.Plugins.Enabled {
			return 0
		}
		// enabled but empty dirs: treat as zero plugins when "dirs specified" is false —
		// residual message only (exit 0). Exit 1 only when dirs were provided.
		return 0
	}
	outcomes, scanWarns := agentplugins.ValidateDirs(dirs)
	for _, w := range scanWarns {
		fmt.Fprintf(os.Stderr, "plugins: %s\n", w)
	}
	okCount := agentplugins.ValidateOKCount(outcomes)
	hasFatal := agentplugins.ValidateHasFatal(outcomes)
	for _, o := range outcomes {
		if o.OK {
			fmt.Println(agentplugins.FormatValidateOK(o))
			for _, w := range o.Warnings {
				fmt.Fprintf(os.Stderr, "plugins %s: %s\n", o.Name, w)
			}
		} else {
			fmt.Println(agentplugins.FormatValidateFail(o.Path, o.Error))
		}
	}
	if okCount == 0 {
		fmt.Fprintln(os.Stderr, "validate: zero plugins OK (dirs were specified)")
	}
	fmt.Fprintln(os.Stderr, agentplugins.ResidualCLIHonesty)
	// Exit 1 if any fatal OR zero plugins found when dirs specified.
	if hasFatal || okCount == 0 {
		return 1
	}
	return 0
}

func cmdModels(args []string) int {
	fs := flag.NewFlagSet("models", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to config.toml")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	rtr, err := cfg.NewRouter()
	if err != nil {
		fmt.Fprintf(os.Stderr, "router: %v\n", err)
		return 1
	}
	fmt.Printf("default: %s\n\n", rtr.DefaultModel())
	fmt.Printf("%-22s %-28s %8s %10s %s\n", "NAME", "MODEL_ID", "PRIORITY", "COST_TIER", "CAPABILITIES")
	for _, m := range rtr.Models() {
		mark := " "
		if m.Name == rtr.DefaultModel() {
			mark = "*"
		}
		fmt.Printf("%s %-20s %-28s %8d %10.2f %s\n",
			mark, m.Name, m.ModelID, m.Priority, m.CostTier, strings.Join(m.Capabilities, ","))
	}
	return 0
}

func cmdMesh(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh smoke|probe|dogfood|usage|catalog|streams|kv|pub|consumer|wait|status [flags]")
		return 2
	}
	switch args[0] {
	case "smoke", "probe", "dogfood":
		// Public name: smoke. probe/dogfood = legacy aliases (compat · s1521).
		return cmdMeshDogfood(args[1:])
	case "usage":
		return cmdMeshUsage(args[1:])
	case "catalog":
		return cmdMeshCatalog(args[1:])
	case "streams":
		return cmdMeshStreams(args[1:])
	case "kv":
		return cmdMeshKV(args[1:])
	case "pub":
		return cmdMeshPub(args[1:])
	case "consumer":
		return cmdMeshConsumer(args[1:])
	case "wait":
		return cmdMeshWait(args[1:])
	case "status":
		return cmdMeshStatus(args[1:])
	case "help", "-h", "--help":
		fmt.Fprintln(os.Stderr, `iomesh mesh — I/O Mesh platform probes

  iomesh mesh smoke     stage smoke (health → ready → context → emit → [pub] → policy → catalog → streams → [consumer] → [kv] → memory_*)
  iomesh mesh probe     alias for smoke
  iomesh mesh dogfood   legacy alias for smoke (compat)
  iomesh mesh usage     local LLM metering rollup for this process (UsagePrint always-emit --json)
  iomesh mesh catalog   list/detail governed data products (--id detail; CatalogPrint / CatalogProductPrint --json)
  iomesh mesh streams   list/get/delete/messages/create broker streams (GET|POST|DELETE /v1/streams; explicit errors)
  iomesh mesh kv        KV list/get/put/delete/create-bucket (GET|PUT|DELETE|POST /v1/kv; mutate ops require --yes)
  iomesh mesh pub       ephemeral fire-and-forget publish (POST /v1/pub; requires --yes; PubPrint always-emit)
  iomesh mesh consumer  durable pull consumer create/fetch/ack/nack/delete (.../consumers; requires --yes)
  iomesh mesh wait      poll Ready until OK or timeout (operator preflight)
  iomesh mesh status    operator snapshot (StatusLine + optional Health/Ready)

Flags (status):
  --config path           config.toml
  --endpoint url          override IOMESH_ENDPOINT
  --json                  print status as JSON
  --strict                exit 1 when aggregate result is err (skipped/partial still 0)
  -v                      verbose
  Identity always-emits pull_role / pull_allow_suffix from [memory].pull_role / pull_allow_suffix
  (empty when unset; Beta federated ACL; fail-open; not full mesh RBAC GA; dual_write default OFF).

Flags (smoke · legacy dogfood/probe):
  --config path           config.toml
  --strict                require context + emit + ready (+ policy/catalog/memory/streams/kv/pub/consumer when on)
  --skip-context          skip context plane
  --skip-emit             skip dept stream emit
  --skip-memory           skip memory_ingest / memory_recall / memory_retrieve
  --skip-streams          skip streams list probe (GET /v1/streams)
  --kv-bucket NAME        soft KV list-keys probe on bucket (omit = skip kv step)
  --kv-ensure             with --kv-bucket: best-effort create bucket before list (soft fail-open)
  --pub-subject SUBJECT   soft ephemeral Pub probe (omit = skip pub step)
  --consumer-stream S     soft consumer create probe stream (requires --consumer-name)
  --consumer-name C       soft consumer create probe name (requires --consumer-stream)
  --consumer-filter F     optional filter_subject for consumer create probe
  --consumer-fetch        after create: soft fetch batch=1 max_wait=500ms (empty OK; no ack)
  --consumer-delete       after create: best-effort DeleteConsumer cleanup (soft fail-open)
  --wait-ready dur        soft WaitReady preflight budget (0=off; timeout SKIP unless --strict)
  --wait-interval dur     WaitReady poll interval (default 500ms when --wait-ready set)
  --wait-require-health   WaitReady requires Health OK each attempt
  --endpoint url          override IOMESH_ENDPOINT
  --memory-endpoint url   memory sidecar base (sync retrieve / warm plane)
  --json                  JSON report for stage CI evidence
  -C dir                  workspace for context query
  -v                      verbose

Flags (pub):
  --subject S             subject (required)
  --payload STR           payload string (raw wire; not base64)
  --payload-file PATH     read payload bytes from file
  --yes                   required gate (mutating)
  --json                  PubPrint always-emit {ok,subject,bytes} (s732; empty/0 honest; no payload echo)
  --endpoint / --tenant / --config / -v

Flags (consumer):
  create --stream S --name C [--filter F] --yes [--json]
  fetch  --stream S --name C [--batch N] --yes [--json]
  ack    --stream S --name C --seq N [--seq N...] --yes
  nack   --stream S --name C --seq N [--seq N...] --yes
  --stream / --name required; --yes required (mutating)
  --filter F              create: optional filter_subject
  --batch N               fetch: max messages (default 1)
  --seq N                 ack/nack: message sequence (repeatable)
  --endpoint / --tenant / --config / -v

Flags (catalog):
  --query q         optional search filter (list path)
  --id ID           product id detail path (GetCatalogProduct; omit for list)
  --json            list: CatalogPrint (s735); detail: CatalogProductPrint {source,detail,id,found,product} (s744; empty/0/[]/false honest)
  --endpoint url    override mesh endpoint
  --tenant id       override tenant

Flags (streams):
  --name NAME       get/delete/messages one stream (omit to list); create defaults OPERATIONAL_EVENTS
  --json            JSON array (list/messages) or object (get/create); delete: StreamDeletePrint
  --messages        list messages for --name (requires --name; incompatible with --delete / --create)
  --from-seq N      messages: lower seq bound (query from_seq)
  --to-seq N        messages: upper seq bound (query to_seq)
  --limit N         messages: max rows (default 20 for CLI comfort)
  --create          create stream (requires --yes; default name OPERATIONAL_EVENTS; 409 = already exists)
  --subject S       create: subject override (default from tenant: dept.{tenant}.events.github)
  --delete          delete stream named by --name (requires --name and --yes; DESTRUCTIVE)
  --yes             confirm destructive delete or mutating create
  --endpoint url    override mesh endpoint
  --config path     config.toml
  --tenant id       override tenant
  -v                verbose

Flags (kv):
  --bucket NAME     KV bucket (required)
  --list            list keys in bucket (optional --prefix)
  --get KEY         get one key value
  --prefix P        list: key prefix filter
  --json            JSON output (keys array or entry object; put: KVPutPrint; delete: KVDeletePrint)
  --endpoint url    override mesh endpoint
  --config path     config.toml
  --tenant id       override tenant
  -v                verbose

Flags (pub):
  --subject S       pub subject (required)
  --payload STR     payload string (raw wire, not base64)
  --payload-file F  read payload from file (not both with --payload)
  --yes             confirm mutating ephemeral pub (required)
  --json            PubPrint always-emit {ok,subject,bytes} (s732; empty/0 honest; no payload echo)
  --endpoint url    override mesh endpoint
  --config path     config.toml
  --tenant id       override tenant
  -v                verbose

Flags (wait):
  --timeout dur       max wait (default 30s)
  --interval dur      poll interval (default 500ms)
  --require-health    require Health OK each attempt before Ready
  --json              print wait evidence as JSON (always-emits pull_role / pull_allow_suffix)
  --endpoint url      override IOMESH_ENDPOINT
  --config path       config.toml
  -v                  verbose
  Identity always-emits pull_role / pull_allow_suffix from [memory].pull_role / pull_allow_suffix
  (empty when unset; Beta federated ACL; fail-open; not full mesh RBAC GA; dual_write default OFF).

Flags (status):
  --endpoint url      override IOMESH_ENDPOINT
  --config path       config.toml
  --json              structured status object
  -v                  verbose`)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown mesh subcommand %q\n", args[0])
		return 2
	}
}

// applyInferredBroker fills [iomesh] from portal MCP when unset.
// Infer ≠ Connected. --endpoint still wins. Empty infer does not invent Enabled.
func applyInferredBroker(cfg *config.Config) {
	inf := config.ApplyInferredBroker(cfg)
	if inf.Endpoint == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "note: inferred broker %s from portal MCP (catalog ≠ streams)\n", inf.Endpoint)
}

// applyOrgFlag overlays --org onto [iomesh].org / IOMESH_ORG when the flag is set.
// Empty flag keeps config/env. Empty org still fail-opens (aion #2721).
func applyOrgFlag(cfg *config.Config, org string) {
	if cfg == nil {
		return
	}
	if s := strings.TrimSpace(org); s != "" {
		cfg.IOMesh.Org = s
	}
}

func cmdMeshWait(args []string) int {
	fs := flag.NewFlagSet("mesh wait", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath    = fs.String("config", "", "config.toml path")
		endpoint      = fs.String("endpoint", "", "override IOMESH_ENDPOINT")
		timeout       = fs.Duration("timeout", 30*time.Second, "max wait duration")
		interval      = fs.Duration("interval", 500*time.Millisecond, "poll interval")
		requireHealth = fs.Bool("require-health", false, "require Health OK each attempt before Ready")
		jsonOut       = fs.Bool("json", false, "print {ok,elapsed_ms,require_health,timeout_ms,interval_ms,attempts,result,exit_code,version,user_agent,endpoint,tenant,org,workspace,pull_role,pull_allow_suffix[,error]} as JSON")
		verbose       = fs.Bool("v", false, "verbose logs")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	applyInferredBroker(cfg)
	// s693: wire [memory].pull_role / pull_allow_suffix onto Client so wait
	// always-emits pull identity from Config (empty when unset; peers status s690).
	pullRole := strings.TrimSpace(cfg.Memory.PullRole)
	pullAllowSuffix := strings.TrimSpace(cfg.Memory.PullAllowSuffix)
	mesh := iomesh.New(iomesh.Config{
		Enabled:         cfg.IOMesh.Enabled,
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          cfg.IOMesh.Tenant,
		APIKeyEnv:       cfg.IOMesh.APIKeyEnv,
		OrgID:           cfg.IOMesh.Org,
		WorkspaceID:     cfg.IOMesh.Workspace,
		Role:            pullRole,
		PullAllowSuffix: pullAllowSuffix,
	}, logger)

	parent, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(parent, *timeout)
	defer cancel()

	// Wall-clock WaitReady for operator/CI evidence (always emitted as elapsed_ms).
	// timeout_ms / interval_ms are configured preflight budget evidence (always emit).
	// attempts is the number of WaitReady probe cycles (always emit).
	// result is ok|err derived from OK / waitErr (always emit; peer mesh status result).
	// exit_code is the process exit (0 when OK, 1 when not) always emit for scrapers.
	// version is package ProductVersion (set from main; empty when unset) always emit.
	// user_agent is package UserAgent (set from main; default "iomesh-tui") always emit.
	// endpoint/tenant/org/workspace/pull_role/pull_allow_suffix are configured identity
	// (empty when unset) always emit; peer mesh status s690 identity continuum —
	// does not invent readiness from identity.
	start := time.Now()
	attempts, waitErr := mesh.WaitReadyAttempts(ctx, iomesh.WaitReadyOptions{
		Interval:      *interval,
		RequireHealth: *requireHealth,
	})
	elapsedMS := iomesh.ElapsedMS(time.Since(start))
	ev := iomesh.MeshWaitEvidence{
		OK:              waitErr == nil,
		ElapsedMS:       elapsedMS,
		RequireHealth:   *requireHealth,
		TimeoutMS:       int(timeout.Milliseconds()),
		IntervalMS:      int(interval.Milliseconds()),
		Attempts:        attempts,
		Version:         iomesh.ProductVersion(),
		UserAgent:       iomesh.UserAgent(),
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          cfg.IOMesh.Tenant,
		Org:             strings.TrimSpace(cfg.IOMesh.Org),
		Workspace:       strings.TrimSpace(cfg.IOMesh.Workspace),
		PullRole:        pullRole,
		PullAllowSuffix: pullAllowSuffix,
	}
	if waitErr != nil {
		ev.Error = waitErr.Error()
	}
	// Result + ExitCode are re-derived from OK in normalize; set here for process-exit parity.
	ev.Result = iomesh.MeshWaitResult(ev)
	ev.ExitCode = iomesh.MeshWaitExitCode(ev)
	var out string
	if *jsonOut {
		out = iomesh.FormatMeshWaitResultJSON(ev)
	} else {
		out = iomesh.FormatMeshWaitResult(ev)
	}
	if !mesh.Enabled() {
		fmt.Fprintln(os.Stderr, iomesh.MeshDisabledHooksHint())
	}
	if ev.OK {
		fmt.Print(out)
		return iomesh.MeshWaitExitCode(ev)
	}
	// FAIL + elapsed_ms on stderr (text) or stdout (json) for CI greps.
	if *jsonOut {
		fmt.Print(out)
	} else {
		fmt.Fprint(os.Stderr, out)
	}
	return iomesh.MeshWaitExitCode(ev)
}

// cmdMeshStatus prints an operator snapshot: StatusLine fields + one-shot Health/Ready.
// Default exit is fail-open (0 on probe err). With --strict, aggregate result "err" exits 1.
func cmdMeshStatus(args []string) int {
	fs := flag.NewFlagSet("mesh status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath = fs.String("config", "", "config.toml path")
		endpoint   = fs.String("endpoint", "", "override IOMESH_ENDPOINT")
		jsonOut    = fs.Bool("json", false, "print status as JSON")
		strict     = fs.Bool("strict", false, "exit 1 when aggregate result is err (probe failure); skipped/partial still 0")
		verbose    = fs.Bool("v", false, "verbose logs")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	applyInferredBroker(cfg)
	// s690: wire [memory].pull_role / pull_allow_suffix onto Client so status
	// always-emits pull identity from Config (empty when unset; peers dogfood s687).
	pullRole := strings.TrimSpace(cfg.Memory.PullRole)
	pullAllowSuffix := strings.TrimSpace(cfg.Memory.PullAllowSuffix)
	mesh := iomesh.New(iomesh.Config{
		Enabled:         cfg.IOMesh.Enabled,
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          cfg.IOMesh.Tenant,
		APIKeyEnv:       cfg.IOMesh.APIKeyEnv,
		OrgID:           cfg.IOMesh.Org,
		WorkspaceID:     cfg.IOMesh.Workspace,
		EmitDeptStreams: cfg.IOMesh.EmitDeptStreams,
		ContextPlane:    cfg.IOMesh.ContextPlane,
		IncludeLineage:  cfg.IOMesh.IncludeLineage,
		PolicyMode:      iomesh.PolicyMode(cfg.IOMesh.PolicyMode),
		CatalogPlane:    cfg.IOMesh.CatalogPlane,
		Role:            pullRole,
		PullAllowSuffix: pullAllowSuffix,
	}, logger)

	policyMode := strings.ToLower(strings.TrimSpace(cfg.IOMesh.PolicyMode))
	if policyMode == "" {
		policyMode = "off"
	}
	out := iomesh.MeshStatusSnapshot{
		Enabled:         mesh.Enabled(),
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          cfg.IOMesh.Tenant,
		Org:             strings.TrimSpace(cfg.IOMesh.Org),
		Workspace:       strings.TrimSpace(cfg.IOMesh.Workspace),
		PullRole:        pullRole,
		PullAllowSuffix: pullAllowSuffix,
		Version:         version,
		PolicyMode:      policyMode,
		ContextPlane:    cfg.IOMesh.ContextPlane,
		CatalogPlane:    cfg.IOMesh.CatalogPlane,
		IncludeLineage:  cfg.IOMesh.IncludeLineage,
		EmitDept:        cfg.IOMesh.EmitDeptStreams,
		UserAgent:       iomesh.UserAgent(),
		StatusLine:      mesh.StatusLine(),
		Health:          "skipped",
		Ready:           "skipped",
		// health_ms / ready_ms / duration_ms always 0 when mesh disabled / probes skipped
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// One-shot Health/Ready with latencies — fail-open display by default.
	// With --strict, exit 1 only when aggregate result is "err" (see MeshStatusExitCode).
	// duration_ms is wall-clock for the whole probe path (always emitted; ~0 when skipped).
	probeStart := time.Now()
	if mesh.Enabled() {
		t0 := time.Now()
		out.Health, out.HealthErr = iomesh.ProbeStatus(mesh.Health(ctx))
		out.HealthMS = iomesh.ElapsedMS(time.Since(t0))

		t1 := time.Now()
		out.Ready, out.ReadyErr = iomesh.ProbeStatus(mesh.Ready(ctx))
		out.ReadyMS = iomesh.ElapsedMS(time.Since(t1))
	}
	out.DurationMS = iomesh.ElapsedMS(time.Since(probeStart))
	out.Result = iomesh.AggregateProbeResult(out.Health, out.Ready)
	out.Strict = *strict
	out.ExitCode = iomesh.MeshStatusExitCode(out.Strict, out.Result)

	if *jsonOut {
		fmt.Print(iomesh.FormatMeshStatusJSON(out))
	} else {
		fmt.Print(iomesh.FormatMeshStatus(out))
	}
	if !mesh.Enabled() {
		fmt.Fprintln(os.Stderr, iomesh.MeshDisabledHooksHint())
	}
	return out.ExitCode
}

func cmdMeshCatalog(args []string) int {
	fs := flag.NewFlagSet("mesh catalog", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath = fs.String("config", "", "config.toml path")
		query      = fs.String("query", "", "optional catalog search filter (list path)")
		id         = fs.String("id", "", "product id for detail path (omit for list)")
		endpoint   = fs.String("endpoint", "", "override IOMESH_ENDPOINT")
		tenant     = fs.String("tenant", "", "override tenant")
		jsonOut    = fs.Bool("json", false, "list: CatalogPrint always-emit; detail (--id): CatalogProductPrint always-emit (empty/0/[]/false honest)")
		verbose    = fs.Bool("v", false, "verbose logs")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	if *tenant != "" {
		cfg.IOMesh.Tenant = *tenant
	}
	applyInferredBroker(cfg)
	cfg.IOMesh.CatalogPlane = true
	mesh := iomesh.New(iomesh.Config{
		Enabled:      cfg.IOMesh.Enabled,
		Endpoint:     cfg.IOMesh.Endpoint,
		Tenant:       cfg.IOMesh.Tenant,
		APIKeyEnv:    cfg.IOMesh.APIKeyEnv,
		OrgID:        cfg.IOMesh.Org,
		WorkspaceID:  cfg.IOMesh.Workspace,
		CatalogPlane: true,
	}, logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Detail path: GetCatalogProduct + CatalogProductPrint (s744).
	// List path (id empty) unchanged: CatalogPrint (s735).
	productID := strings.TrimSpace(*id)
	if productID != "" {
		p, meta := mesh.GetCatalogProduct(ctx, productID)
		// found=false when empty product / fail-open not found / off — no invent.
		found := p.ID != "" || p.Name != ""
		if *jsonOut {
			fmt.Print(iomesh.FormatCatalogProductJSON(iomesh.NewCatalogProductPrint(productID, p, meta, found)))
		} else {
			fmt.Print(iomesh.FormatProductDetail(p, meta))
		}
		// Exit 1 only when source off (mesh/catalog disabled). fail-open not-found
		// keeps exit 0 (match list fail-open honesty — operator sees found=false).
		if meta.Source == "off" {
			return 1
		}
		return 0
	}

	res := mesh.ListCatalog(ctx, *query)
	if *jsonOut {
		// CatalogPrint always-emit (s735); Source=="off" still exit 1 (honesty).
		fmt.Print(iomesh.FormatCatalogJSON(iomesh.NewCatalogPrint(res, *query)))
	} else {
		fmt.Print(iomesh.FormatCatalog(res))
	}
	if res.Source == "off" {
		return 1
	}
	return 0
}

// cmdMeshStreams lists, gets, deletes, inspects messages, or creates broker streams
// via lean /v1/streams (explicit errors; no SDK dep). --delete is destructive and
// requires --name and --yes. --create requires --yes (409 = already exists).
// --create / --delete / --messages are mutually exclusive.
func cmdMeshStreams(args []string) int {
	fs := flag.NewFlagSet("mesh streams", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath = fs.String("config", "", "config.toml path")
		name       = fs.String("name", "", "get/delete/messages one stream by name (omit to list; create defaults OPERATIONAL_EVENTS)")
		endpoint   = fs.String("endpoint", "", "override IOMESH_ENDPOINT")
		tenant     = fs.String("tenant", "", "override tenant")
		jsonOut    = fs.Bool("json", false, "print streams/messages as JSON (messages: StreamMessagesPrint; delete: StreamDeletePrint; create: StreamInfoPrint)")
		doMessages = fs.Bool("messages", false, "list messages for --name (requires --name; GET /v1/streams/{name}/messages)")
		fromSeq    = fs.Uint64("from-seq", 0, "messages: from_seq lower bound (0=omit; broker default)")
		toSeq      = fs.Uint64("to-seq", 0, "messages: to_seq upper bound (0=omit; broker default)")
		limit      = fs.Int("limit", 20, "messages: max rows (default 20 for CLI comfort)")
		doCreate   = fs.Bool("create", false, "create stream (requires --yes; default name OPERATIONAL_EVENTS; 409 = already exists)")
		subject    = fs.String("subject", "", "create: subject override (default from tenant: dept.{tenant}.events.github)")
		doDelete   = fs.Bool("delete", false, "delete stream named by --name (requires --name and --yes; DESTRUCTIVE)")
		yes        = fs.Bool("yes", false, "confirm destructive delete or mutating create")
		verbose    = fs.Bool("v", false, "verbose logs")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	streamName := strings.TrimSpace(*name)
	nMutate := 0
	if *doCreate {
		nMutate++
	}
	if *doDelete {
		nMutate++
	}
	if *doMessages {
		nMutate++
	}
	if nMutate > 1 {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh streams --create|--messages|--delete (not both)")
		fmt.Fprintln(os.Stderr, "  --create, --messages, and --delete are mutually exclusive")
		return 2
	}
	if *doCreate && !*yes {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh streams --create --yes [--name NAME] [--subject S] [--json]")
		fmt.Fprintln(os.Stderr, "  --create requires --yes (mutating; 409 = already exists)")
		fmt.Fprintln(os.Stderr, "  default name OPERATIONAL_EVENTS; subject from tenant (console defaults)")
		return 2
	}
	if *doCreate && streamName == "" {
		streamName = "OPERATIONAL_EVENTS"
	}
	if *doDelete {
		if streamName == "" || !*yes {
			fmt.Fprintln(os.Stderr, "usage: iomesh mesh streams --delete --name NAME --yes [--json]")
			fmt.Fprintln(os.Stderr, "  --delete is destructive; requires --name and --yes")
			fmt.Fprintln(os.Stderr, "  text/JSON always-emit StreamDeletePrint {ok,name} (s726; empty name honest)")
			return 2
		}
	}
	if *doMessages {
		if streamName == "" {
			fmt.Fprintln(os.Stderr, "usage: iomesh mesh streams --messages --name NAME [--from-seq N] [--to-seq N] [--limit N] [--json]")
			fmt.Fprintln(os.Stderr, "  --messages requires --name")
			return 2
		}
	}
	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	if *tenant != "" {
		cfg.IOMesh.Tenant = *tenant
	}
	applyInferredBroker(cfg)
	mesh := iomesh.New(iomesh.Config{
		Enabled:     cfg.IOMesh.Enabled,
		Endpoint:    cfg.IOMesh.Endpoint,
		Tenant:      cfg.IOMesh.Tenant,
		APIKeyEnv:   cfg.IOMesh.APIKeyEnv,
		OrgID:       cfg.IOMesh.Org,
		WorkspaceID: cfg.IOMesh.Workspace,
	}, logger)
	if !mesh.Enabled() {
		fmt.Fprintln(os.Stderr, "FAIL mesh streams: mesh disabled")
		fmt.Fprintln(os.Stderr, iomesh.MeshDisabledHooksHint())
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if *doCreate {
		createCfg := iomesh.DefaultOperationalEventsCreate(cfg.IOMesh.Tenant)
		createCfg.Name = streamName
		if sub := strings.TrimSpace(*subject); sub != "" {
			createCfg.Subjects = []string{sub}
		}
		info, err := mesh.CreateStream(ctx, createCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL mesh streams create: %v\n", err)
			return 1
		}
		if info == nil {
			fmt.Fprintln(os.Stderr, "FAIL mesh streams create: empty response")
			return 1
		}
		if *jsonOut {
			fmt.Print(iomesh.FormatStreamInfoJSON(iomesh.NewStreamInfoPrint(*info)))
			return 0
		}
		// 409 already-exists is success (idempotent). Create ≠ PULSE.
		fmt.Print("PASS mesh streams create\n")
		fmt.Print(iomesh.FormatStreamDetail(*info))
		for _, line := range iomesh.StreamsInboxNextStepLines() {
			fmt.Println(line)
		}
		return 0
	}

	if *doDelete {
		if err := mesh.DeleteStream(ctx, streamName); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL mesh streams delete: %v\n", err)
			return 1
		}
		// s726: always-emit StreamDeletePrint on delete success (mold ConsumerDeletePrint s708).
		printDTO := iomesh.NewStreamDeletePrint(streamName)
		if *jsonOut {
			fmt.Print(iomesh.FormatStreamDeleteJSON(printDTO))
			return 0
		}
		fmt.Print(iomesh.FormatStreamDelete(printDTO))
		return 0
	}

	if *doMessages {
		msgs, err := mesh.ListStreamMessages(ctx, streamName, iomesh.ListStreamMessagesOptions{
			FromSeq: *fromSeq,
			ToSeq:   *toSeq,
			Limit:   *limit,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL mesh streams messages: %v\n", err)
			return 1
		}
		// s720: print DTO always-emits stream + knobs + count + messages (not bare array).
		// s741: FormatStreamMessagesJSON (helper completeness; no ad-hoc MarshalIndent).
		printDTO := iomesh.NewStreamMessagesPrint(streamName, *fromSeq, *toSeq, *limit, msgs)
		if *jsonOut {
			fmt.Print(iomesh.FormatStreamMessagesJSON(printDTO))
			return 0
		}
		fmt.Print(iomesh.FormatStreamMessagesPrint(printDTO))
		return 0
	}

	if streamName != "" {
		info, err := mesh.GetStream(ctx, streamName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL mesh streams: %v\n", err)
			return 1
		}
		if *jsonOut {
			// s699: print DTO always-emits retention knobs (0 / "" when unset).
			// s741: FormatStreamInfoJSON (helper completeness; no ad-hoc MarshalIndent).
			fmt.Print(iomesh.FormatStreamInfoJSON(iomesh.NewStreamInfoPrint(*info)))
			return 0
		}
		fmt.Print(iomesh.FormatStreamDetail(*info))
		return 0
	}

	streams, err := mesh.ListStreams(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL mesh streams: %v\n", err)
		return 1
	}
	if *jsonOut {
		// s702: list --json always-emits retention knobs + retention_tier via print DTO
		// (closes s699 half-gap that marshaled lean wire []StreamInfo).
		// s741: FormatStreamInfoListJSON (nil → []; no ad-hoc MarshalIndent).
		printList := make([]iomesh.StreamInfoPrint, 0, len(streams))
		for _, s := range streams {
			printList = append(printList, iomesh.NewStreamInfoPrint(s))
		}
		fmt.Print(iomesh.FormatStreamInfoListJSON(printList))
		return 0
	}
	fmt.Print(iomesh.FormatStreams(streams))
	return 0
}

// cmdMeshConsumer creates, fetches, acks, nacks, or deletes a durable pull consumer
// (lean /v1/streams/{s}/consumers; requires --yes).
func cmdMeshConsumer(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh consumer create|fetch|ack|nack|delete [flags]")
		return 2
	}
	switch args[0] {
	case "create":
		return cmdMeshConsumerCreate(args[1:])
	case "fetch":
		return cmdMeshConsumerFetch(args[1:])
	case "ack":
		return cmdMeshConsumerAckNack(args[1:], false)
	case "nack":
		return cmdMeshConsumerAckNack(args[1:], true)
	case "delete":
		return cmdMeshConsumerDelete(args[1:])
	case "help", "-h", "--help":
		fmt.Fprintln(os.Stderr, `iomesh mesh consumer — durable pull consumers

  iomesh mesh consumer create --stream S --name C [--filter F] [--role R] [--pull-allow-suffix S] --yes [--json]
  iomesh mesh consumer fetch  --stream S --name C [--batch N] [--role R] [--pull-allow-suffix S] --yes [--json]
  iomesh mesh consumer ack    --stream S --name C --seq N [--seq N...] [--role R] [--pull-allow-suffix S] --yes [--json]
  iomesh mesh consumer nack   --stream S --name C --seq N [--seq N...] [--role R] [--pull-allow-suffix S] --yes [--json]
  iomesh mesh consumer delete --stream S --name C [--role R] [--pull-allow-suffix S] --yes [--json]

Create: POST /v1/streams/{stream}/consumers (201 full info; 409 idempotent name-only).
  --role / [memory].pull_role → X-IOMesh-Role; --pull-allow-suffix / pull_allow_suffix → allow-suffix.
  Roles: operator|admin|agent|auditor|viewer|memory|custom (s687 memory → tenant.memory.>).
  Empty --filter → role-aware default (s681/s687; same as memory pull s678).
  Text/JSON always-emit pull_role / pull_allow_suffix next to filter_subject (s696; empty when unset).
  Beta; fail-open; dual_write default OFF; not full mesh RBAC GA.
Fetch:  POST /v1/streams/{stream}/consumers/{name}/fetch (default batch=1, max_wait 2s).
  Same --role / --pull-allow-suffix headers (s684; aion validates role on fetch). Fail-open empty.
  Text/JSON always-emit pull identity + knobs + count + messages (s708 ConsumerFetchPrint; empty role honest).
Ack:    POST .../consumers/{name}/ack body {"seqs":[...]} (optional ack_floor on response).
  Text/JSON always-emit pull identity + op/seqs/ack_floor/count (s711 ConsumerAckPrint; empty role honest).
Nack:   POST .../consumers/{name}/nack body {"seqs":[...]} (same print DTO shape as ack; op=nack).
Delete: DELETE .../consumers/{name} (204/2xx success).
  Text/JSON always-emit {ok,stream,name,pull_role,pull_allow_suffix} (s708; empty role honest).
  Ack/nack/delete also accept --role / --pull-allow-suffix for defense-in-depth auth headers.
All require --yes (mutating).`)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown mesh consumer subcommand %q\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh consumer create|fetch|ack|nack|delete [flags]")
		return 2
	}
}

// uint64List is a flag.Value for repeatable --seq N flags.
type uint64List []uint64

func (u *uint64List) String() string {
	if u == nil || len(*u) == 0 {
		return ""
	}
	parts := make([]string, len(*u))
	for i, v := range *u {
		parts[i] = strconv.FormatUint(v, 10)
	}
	return strings.Join(parts, ",")
}

func (u *uint64List) Set(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("empty seq")
	}
	// Allow a single value or CSV fragment so --seq 1,2 works too.
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		v, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid seq %q: %w", part, err)
		}
		*u = append(*u, v)
	}
	if len(*u) == 0 {
		return fmt.Errorf("empty seq")
	}
	return nil
}

func cmdMeshConsumerAckNack(args []string, nack bool) int {
	op := "ack"
	if nack {
		op = "nack"
	}
	fs := flag.NewFlagSet("mesh consumer "+op, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var seqs uint64List
	var (
		configPath      = fs.String("config", "", "config.toml path")
		stream          = fs.String("stream", "", "stream name (required)")
		name            = fs.String("name", "", "consumer name (required)")
		role            = fs.String("role", "", "optional X-IOMesh-Role (operator|admin|agent|auditor|viewer|memory|custom); [memory].pull_role")
		pullAllowSuffix = fs.String("pull-allow-suffix", "", "optional X-IOMesh-Pull-Allow-Suffix (comma tokens; role=custom); [memory].pull_allow_suffix")
		yes             = fs.Bool("yes", false, "confirm mutating "+op+" (required)")
		jsonOut         = fs.Bool("json", false, "print ConsumerAckPrint JSON (always-emits pull_role / pull_allow_suffix)")
		verbose         = fs.Bool("v", false, "verbose logs")
		endpoint        = fs.String("endpoint", "", "override IOMESH_ENDPOINT / config")
		tenant          = fs.String("tenant", "", "override tenant")
		org             = fs.String("org", "", "override [iomesh].org / IOMESH_ORG (X-IOMesh-Org; empty fail-opens)")
	)
	fs.Var(&seqs, "seq", "message sequence to "+op+" (repeatable; CSV ok)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	streamName := strings.TrimSpace(*stream)
	consumerName := strings.TrimSpace(*name)
	if streamName == "" || consumerName == "" || len(seqs) == 0 || !*yes {
		fmt.Fprintf(os.Stderr, "usage: iomesh mesh consumer %s --stream S --name C --seq N [--seq N...] [--role R] [--pull-allow-suffix S] --yes [--json]\n", op)
		fmt.Fprintln(os.Stderr, "  --stream, --name, and at least one --seq required; --yes required (mutating)")
		fmt.Fprintln(os.Stderr, "  text/JSON always-emit pull_role / pull_allow_suffix (s711; empty when unset)")
		return 2
	}

	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	if *tenant != "" {
		cfg.IOMesh.Tenant = *tenant
	}
	applyOrgFlag(cfg, *org)
	applyInferredBroker(cfg)
	// s684: same role/suffix auth headers as create/fetch (defense-in-depth; fail-open empty).
	pullRole, allowSuffix := iomesh.ResolveMeshPullAuth(
		*role, *pullAllowSuffix, cfg.Memory.PullRole, cfg.Memory.PullAllowSuffix,
	)
	mesh := iomesh.New(iomesh.Config{
		Enabled:         cfg.IOMesh.Enabled,
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          cfg.IOMesh.Tenant,
		APIKeyEnv:       cfg.IOMesh.APIKeyEnv,
		OrgID:           cfg.IOMesh.Org,
		WorkspaceID:     cfg.IOMesh.Workspace,
		Role:            pullRole,
		PullAllowSuffix: allowSuffix,
	}, logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var floor uint64
	if nack {
		floor, err = mesh.ConsumerNack(ctx, streamName, consumerName, seqs...)
	} else {
		floor, err = mesh.ConsumerAck(ctx, streamName, consumerName, seqs...)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL mesh consumer %s: %v\n", op, err)
		return 1
	}
	// s711: always-emit pull identity on ack/nack success (peer create s696 / fetch/delete s708).
	printDTO := iomesh.NewConsumerAckPrint(op, streamName, consumerName, pullRole, allowSuffix, []uint64(seqs), floor)
	if *jsonOut {
		fmt.Print(iomesh.FormatConsumerAckJSON(printDTO))
		return 0
	}
	fmt.Print(iomesh.FormatConsumerAck(printDTO))
	return 0
}

func cmdMeshConsumerDelete(args []string) int {
	fs := flag.NewFlagSet("mesh consumer delete", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath      = fs.String("config", "", "config.toml path")
		stream          = fs.String("stream", "", "stream name (required)")
		name            = fs.String("name", "", "consumer name (required)")
		role            = fs.String("role", "", "optional X-IOMesh-Role (operator|admin|agent|auditor|viewer|memory|custom); [memory].pull_role")
		pullAllowSuffix = fs.String("pull-allow-suffix", "", "optional X-IOMesh-Pull-Allow-Suffix (comma tokens; role=custom); [memory].pull_allow_suffix")
		yes             = fs.Bool("yes", false, "confirm mutating delete (required)")
		jsonOut         = fs.Bool("json", false, "print ConsumerDeletePrint JSON (always-emits pull_role / pull_allow_suffix)")
		verbose         = fs.Bool("v", false, "verbose logs")
		endpoint        = fs.String("endpoint", "", "override IOMESH_ENDPOINT / config")
		tenant          = fs.String("tenant", "", "override tenant")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	streamName := strings.TrimSpace(*stream)
	consumerName := strings.TrimSpace(*name)
	if streamName == "" || consumerName == "" || !*yes {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh consumer delete --stream S --name C [--role R] [--pull-allow-suffix S] --yes [--json]")
		fmt.Fprintln(os.Stderr, "  --stream and --name required; --yes required (mutating)")
		fmt.Fprintln(os.Stderr, "  text/JSON always-emit pull_role / pull_allow_suffix (s708; empty when unset)")
		return 2
	}

	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	if *tenant != "" {
		cfg.IOMesh.Tenant = *tenant
	}
	applyInferredBroker(cfg)
	// s684: same role/suffix auth headers as create/fetch (defense-in-depth; fail-open empty).
	pullRole, allowSuffix := iomesh.ResolveMeshPullAuth(
		*role, *pullAllowSuffix, cfg.Memory.PullRole, cfg.Memory.PullAllowSuffix,
	)
	mesh := iomesh.New(iomesh.Config{
		Enabled:         cfg.IOMesh.Enabled,
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          cfg.IOMesh.Tenant,
		APIKeyEnv:       cfg.IOMesh.APIKeyEnv,
		OrgID:           cfg.IOMesh.Org,
		WorkspaceID:     cfg.IOMesh.Workspace,
		Role:            pullRole,
		PullAllowSuffix: allowSuffix,
	}, logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := mesh.DeleteConsumer(ctx, streamName, consumerName); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL mesh consumer delete: %v\n", err)
		return 1
	}
	// s708: always-emit pull identity on delete success (peer create s696 / fetch print DTO).
	printDTO := iomesh.NewConsumerDeletePrint(streamName, consumerName, pullRole, allowSuffix)
	if *jsonOut {
		fmt.Print(iomesh.FormatConsumerDeleteJSON(printDTO))
		return 0
	}
	fmt.Print(iomesh.FormatConsumerDelete(printDTO))
	return 0
}

func cmdMeshConsumerCreate(args []string) int {
	fs := flag.NewFlagSet("mesh consumer create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath      = fs.String("config", "", "config.toml path")
		stream          = fs.String("stream", "", "stream name (required)")
		name            = fs.String("name", "", "consumer name (required)")
		filter          = fs.String("filter", "", "optional filter_subject (role-aware default when empty; s681)")
		role            = fs.String("role", "", "optional X-IOMesh-Role (operator|admin|agent|auditor|viewer|memory|custom); [memory].pull_role")
		pullAllowSuffix = fs.String("pull-allow-suffix", "", "optional X-IOMesh-Pull-Allow-Suffix (comma tokens; role=custom); [memory].pull_allow_suffix")
		yes             = fs.Bool("yes", false, "confirm mutating create (required)")
		jsonOut         = fs.Bool("json", false, "print ConsumerInfoPrint JSON (always-emits pull_role / pull_allow_suffix)")
		verbose         = fs.Bool("v", false, "verbose logs")
		endpoint        = fs.String("endpoint", "", "override IOMESH_ENDPOINT / config")
		tenant          = fs.String("tenant", "", "override tenant")
		org             = fs.String("org", "", "override [iomesh].org / IOMESH_ORG (X-IOMesh-Org; empty fail-opens)")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	streamName := strings.TrimSpace(*stream)
	consumerName := strings.TrimSpace(*name)
	if streamName == "" || consumerName == "" || !*yes {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh consumer create --stream S --name C [--filter F] [--role R] [--pull-allow-suffix S] --yes [--json]")
		fmt.Fprintln(os.Stderr, "  --stream and --name required; --yes required (mutating)")
		fmt.Fprintln(os.Stderr, "  text/JSON always-emit pull_role / pull_allow_suffix next to filter_subject (s696)")
		return 2
	}

	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	if *tenant != "" {
		cfg.IOMesh.Tenant = *tenant
	}
	applyOrgFlag(cfg, *org)
	applyInferredBroker(cfg)
	// s681: federated role + allow-suffix (flags override [memory] config) + role-aware default filter.
	// Tenant is IOMesh tenant (mesh command pattern). Fail-open empty role/suffix → omit headers.
	meshTenant := strings.TrimSpace(cfg.IOMesh.Tenant)
	filterSub, pullRole, allowSuffix := iomesh.ResolveConsumerCreateAuthAndFilter(
		*filter, meshTenant, *role, *pullAllowSuffix, cfg.Memory.PullRole, cfg.Memory.PullAllowSuffix,
	)
	mesh := iomesh.New(iomesh.Config{
		Enabled:         cfg.IOMesh.Enabled,
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          meshTenant,
		APIKeyEnv:       cfg.IOMesh.APIKeyEnv,
		OrgID:           cfg.IOMesh.Org,
		WorkspaceID:     cfg.IOMesh.Workspace,
		Role:            pullRole,
		PullAllowSuffix: allowSuffix,
	}, logger)
	// Log effective filter/role/suffix once (same honesty as memory pull s675/s678).
	fmt.Fprintf(os.Stderr, "mesh consumer create filter_subject=%q tenant=%q role=%q pull_allow_suffix=%q\n",
		filterSub, meshTenant, pullRole, allowSuffix)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	info, err := mesh.CreateConsumer(ctx, streamName, consumerName, filterSub)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL mesh consumer create: %v\n", err)
		return 1
	}
	if info == nil {
		fmt.Fprintln(os.Stderr, "FAIL mesh consumer create: empty response")
		return 1
	}
	// s696: always-emit pull_role / pull_allow_suffix next to filter_subject for CI scrapers.
	// CLI print DTO keeps wire ConsumerInfo free of auth identity fields.
	// s741: FormatConsumerInfoJSON (helper completeness; no ad-hoc MarshalIndent).
	if *jsonOut {
		fmt.Print(iomesh.FormatConsumerInfoJSON(iomesh.NewConsumerInfoPrint(*info, pullRole, allowSuffix)))
		return 0
	}
	fmt.Print(iomesh.FormatConsumerInfoWithAuth(*info, pullRole, allowSuffix))
	return 0
}

func cmdMeshConsumerFetch(args []string) int {
	fs := flag.NewFlagSet("mesh consumer fetch", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath      = fs.String("config", "", "config.toml path")
		stream          = fs.String("stream", "", "stream name (required)")
		name            = fs.String("name", "", "consumer name (required)")
		batch           = fs.Int("batch", 1, "max messages to fetch (default 1)")
		role            = fs.String("role", "", "optional X-IOMesh-Role (operator|admin|agent|auditor|viewer|memory|custom); [memory].pull_role")
		pullAllowSuffix = fs.String("pull-allow-suffix", "", "optional X-IOMesh-Pull-Allow-Suffix (comma tokens; role=custom); [memory].pull_allow_suffix")
		yes             = fs.Bool("yes", false, "confirm mutating fetch (required)")
		jsonOut         = fs.Bool("json", false, "print ConsumerFetchPrint JSON (always-emits pull_role / pull_allow_suffix)")
		verbose         = fs.Bool("v", false, "verbose logs")
		endpoint        = fs.String("endpoint", "", "override IOMESH_ENDPOINT / config")
		tenant          = fs.String("tenant", "", "override tenant")
		org             = fs.String("org", "", "override [iomesh].org / IOMESH_ORG (X-IOMesh-Org; empty fail-opens)")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	streamName := strings.TrimSpace(*stream)
	consumerName := strings.TrimSpace(*name)
	if streamName == "" || consumerName == "" || !*yes {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh consumer fetch --stream S --name C [--batch N] [--role R] [--pull-allow-suffix S] [--org id] --yes [--json]")
		fmt.Fprintln(os.Stderr, "  --stream and --name required; --yes required (long-poll mutate)")
		fmt.Fprintln(os.Stderr, "  text/JSON always-emit pull_role / pull_allow_suffix + knobs + count (s708; empty when unset)")
		return 2
	}
	if *batch <= 0 {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh consumer fetch --batch N (N > 0)")
		return 2
	}

	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	if *tenant != "" {
		cfg.IOMesh.Tenant = *tenant
	}
	applyOrgFlag(cfg, *org)
	applyInferredBroker(cfg)
	// s684: federated role + allow-suffix on fetch (flags override [memory] config).
	// Fail-open empty role/suffix → Client.auth omits headers. Peer aion s683 continuum.
	pullRole, allowSuffix := iomesh.ResolveMeshPullAuth(
		*role, *pullAllowSuffix, cfg.Memory.PullRole, cfg.Memory.PullAllowSuffix,
	)
	mesh := iomesh.New(iomesh.Config{
		Enabled:         cfg.IOMesh.Enabled,
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          cfg.IOMesh.Tenant,
		APIKeyEnv:       cfg.IOMesh.APIKeyEnv,
		OrgID:           cfg.IOMesh.Org,
		WorkspaceID:     cfg.IOMesh.Workspace,
		Role:            pullRole,
		PullAllowSuffix: allowSuffix,
	}, logger)
	// Log effective role/suffix once when set (stderr; same honesty as create s681).
	if pullRole != "" || allowSuffix != "" {
		fmt.Fprintf(os.Stderr, "mesh consumer fetch role=%q pull_allow_suffix=%q\n", pullRole, allowSuffix)
	}
	// Default long-poll budget (matches ConsumerFetch maxWait default path).
	const maxWait = 2 * time.Second
	maxWaitMS := int(maxWait / time.Millisecond)
	// Allow slightly more than the 2s long-poll budget.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	msgs, err := mesh.ConsumerFetch(ctx, streamName, consumerName, *batch, maxWait)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL mesh consumer fetch: %v\n", err)
		return 1
	}
	// s708: print DTO always-emits pull identity + knobs + count (not raw []StreamMessage).
	printDTO := iomesh.NewConsumerFetchPrint(streamName, consumerName, pullRole, allowSuffix, *batch, maxWaitMS, msgs)
	if *jsonOut {
		fmt.Print(iomesh.FormatConsumerFetchJSON(printDTO))
		return 0
	}
	fmt.Print(iomesh.FormatConsumerFetch(printDTO))
	return 0
}

// cmdMeshPub publishes an ephemeral fire-and-forget message via lean POST /v1/pub (SDK wire; requires --yes).
func cmdMeshPub(args []string) int {
	fs := flag.NewFlagSet("mesh pub", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath  = fs.String("config", "", "config.toml path")
		subject     = fs.String("subject", "", "pub subject (required)")
		payloadStr  = fs.String("payload", "", "payload string (raw wire, not base64)")
		payloadFile = fs.String("payload-file", "", "read payload from file")
		yes         = fs.Bool("yes", false, "confirm mutating pub (required)")
		jsonOut     = fs.Bool("json", false, "print success as JSON (PubPrint always-emit {ok,subject,bytes})")
		verbose     = fs.Bool("v", false, "verbose logs")
		endpoint    = fs.String("endpoint", "", "override IOMESH_ENDPOINT / config")
		tenant      = fs.String("tenant", "", "override tenant")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	subj := strings.TrimSpace(*subject)
	hasPayload := strings.TrimSpace(*payloadStr) != "" || strings.TrimSpace(*payloadFile) != ""
	if subj == "" || !hasPayload || !*yes {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh pub --subject S --payload STR|--payload-file F --yes [--json]")
		fmt.Fprintln(os.Stderr, "  --subject and --payload or --payload-file required; --yes required (ephemeral mutate)")
		fmt.Fprintln(os.Stderr, "  text/JSON always-emit PubPrint {ok,subject,bytes} (s732; empty/0 honest; no payload echo)")
		return 2
	}
	if strings.TrimSpace(*payloadStr) != "" && strings.TrimSpace(*payloadFile) != "" {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh pub --payload|--payload-file (not both)")
		return 2
	}

	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	if *tenant != "" {
		cfg.IOMesh.Tenant = *tenant
	}
	applyInferredBroker(cfg)
	mesh := iomesh.New(iomesh.Config{
		Enabled:     cfg.IOMesh.Enabled,
		Endpoint:    cfg.IOMesh.Endpoint,
		Tenant:      cfg.IOMesh.Tenant,
		APIKeyEnv:   cfg.IOMesh.APIKeyEnv,
		OrgID:       cfg.IOMesh.Org,
		WorkspaceID: cfg.IOMesh.Workspace,
	}, logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var payload []byte
	if vf := strings.TrimSpace(*payloadFile); vf != "" {
		raw, err := os.ReadFile(vf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL mesh pub: read payload-file: %v\n", err)
			return 1
		}
		payload = raw
	} else {
		payload = []byte(*payloadStr)
	}
	if !mesh.Enabled() {
		fmt.Fprintln(os.Stderr, "FAIL mesh pub: mesh disabled")
		fmt.Fprintln(os.Stderr, iomesh.MeshDisabledHooksHint())
		return 1
	}
	if err := mesh.Pub(ctx, subj, payload, nil); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL mesh pub: %v\n", err)
		if strings.Contains(err.Error(), "mesh disabled") {
			fmt.Fprintln(os.Stderr, iomesh.MeshDisabledHooksHint())
		}
		return 1
	}
	// s732: always-emit PubPrint on pub success (mold StreamDeletePrint s726 + KVPutPrint s729).
	printDTO := iomesh.NewPubPrint(subj, len(payload))
	if *jsonOut {
		fmt.Print(iomesh.FormatPubJSON(printDTO))
		return 0
	}
	fmt.Print(iomesh.FormatPub(printDTO))
	return 0
}

// cmdMeshKV lists, gets, puts, deletes, or creates broker KV entries/buckets (lean /v1/kv; explicit errors; no SDK dep).
// Mutating ops (--put / --delete / --create-bucket) require --yes. Ops are mutually exclusive.
func cmdMeshKV(args []string) int {
	fs := flag.NewFlagSet("mesh kv", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath   = fs.String("config", "", "config.toml path")
		bucket       = fs.String("bucket", "", "KV bucket name (required)")
		list         = fs.Bool("list", false, "list keys in bucket (optional --prefix)")
		getKey       = fs.String("get", "", "get one key value")
		putKey       = fs.String("put", "", "put key (requires --value or --value-file and --yes)")
		valueStr     = fs.String("value", "", "put: raw string value")
		valueFile    = fs.String("value-file", "", "put: read value bytes from file path")
		deleteKey    = fs.String("delete", "", "delete key (requires --yes; DESTRUCTIVE)")
		createBucket = fs.Bool("create-bucket", false, "create --bucket (requires --yes; 409 = already exists)")
		yes          = fs.Bool("yes", false, "confirm mutating put/delete/create-bucket")
		prefix       = fs.String("prefix", "", "list: key prefix filter")
		endpoint     = fs.String("endpoint", "", "override IOMESH_ENDPOINT")
		tenant       = fs.String("tenant", "", "override tenant")
		jsonOut      = fs.Bool("json", false, "print keys/entry/bucket/put/delete as JSON (put: KVPutPrint; delete: KVDeletePrint)")
		verbose      = fs.Bool("v", false, "verbose logs")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	bucketName := strings.TrimSpace(*bucket)
	getName := strings.TrimSpace(*getKey)
	putName := strings.TrimSpace(*putKey)
	delName := strings.TrimSpace(*deleteKey)

	printKVUsage := func() {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh kv --bucket NAME --list [--prefix P] [--json]")
		fmt.Fprintln(os.Stderr, "       iomesh mesh kv --bucket NAME --get KEY [--json]")
		fmt.Fprintln(os.Stderr, "       iomesh mesh kv --bucket NAME --put KEY --value STR|--value-file PATH --yes [--json]")
		fmt.Fprintln(os.Stderr, "       iomesh mesh kv --bucket NAME --delete KEY --yes [--json]")
		fmt.Fprintln(os.Stderr, "       iomesh mesh kv --bucket NAME --create-bucket --yes [--json]")
		fmt.Fprintln(os.Stderr, "  --bucket required; exactly one of --list|--get|--put|--delete|--create-bucket")
	}

	if bucketName == "" {
		printKVUsage()
		fmt.Fprintln(os.Stderr, "  --bucket required")
		return 2
	}

	// Count mutually exclusive ops.
	nOps := 0
	if *list {
		nOps++
	}
	if getName != "" {
		nOps++
	}
	if putName != "" {
		nOps++
	}
	if delName != "" {
		nOps++
	}
	if *createBucket {
		nOps++
	}
	if nOps == 0 {
		printKVUsage()
		fmt.Fprintln(os.Stderr, "  --list or --get or --put or --delete or --create-bucket required")
		return 2
	}
	if nOps > 1 {
		printKVUsage()
		fmt.Fprintln(os.Stderr, "  --list|--get|--put|--delete|--create-bucket (not both)")
		return 2
	}

	if putName != "" {
		hasVal := strings.TrimSpace(*valueStr) != "" || strings.TrimSpace(*valueFile) != ""
		if !hasVal || !*yes {
			fmt.Fprintln(os.Stderr, "usage: iomesh mesh kv --bucket NAME --put KEY --value STR|--value-file PATH --yes [--json]")
			fmt.Fprintln(os.Stderr, "  --put requires --value or --value-file and --yes")
			return 2
		}
		if strings.TrimSpace(*valueStr) != "" && strings.TrimSpace(*valueFile) != "" {
			fmt.Fprintln(os.Stderr, "usage: iomesh mesh kv --put --value|--value-file (not both)")
			return 2
		}
	}
	if delName != "" && !*yes {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh kv --bucket NAME --delete KEY --yes [--json]")
		fmt.Fprintln(os.Stderr, "  --delete is destructive; requires --yes")
		return 2
	}
	if *createBucket && !*yes {
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh kv --bucket NAME --create-bucket --yes")
		fmt.Fprintln(os.Stderr, "  --create-bucket requires --yes (idempotent; 409 = already exists)")
		return 2
	}

	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	if *tenant != "" {
		cfg.IOMesh.Tenant = *tenant
	}
	applyInferredBroker(cfg)
	mesh := iomesh.New(iomesh.Config{
		Enabled:     cfg.IOMesh.Enabled,
		Endpoint:    cfg.IOMesh.Endpoint,
		Tenant:      cfg.IOMesh.Tenant,
		APIKeyEnv:   cfg.IOMesh.APIKeyEnv,
		OrgID:       cfg.IOMesh.Org,
		WorkspaceID: cfg.IOMesh.Workspace,
	}, logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if *createBucket {
		info, err := mesh.KVCreateBucket(ctx, bucketName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL mesh kv create-bucket: %v\n", err)
			return 1
		}
		if *jsonOut {
			// s714: print DTO always-emits name/history/max_bytes/ttl_seconds (0 when nil).
			// s741: FormatKVBucketInfoJSON (helper completeness; no ad-hoc MarshalIndent).
			fmt.Print(iomesh.FormatKVBucketInfoJSON(iomesh.NewKVBucketInfoPrint(*info)))
			return 0
		}
		fmt.Print(iomesh.FormatKVBucketInfo(*info))
		return 0
	}

	if putName != "" {
		var val []byte
		if vf := strings.TrimSpace(*valueFile); vf != "" {
			raw, err := os.ReadFile(vf)
			if err != nil {
				fmt.Fprintf(os.Stderr, "FAIL mesh kv put: read value-file: %v\n", err)
				return 1
			}
			val = raw
		} else {
			val = []byte(*valueStr)
		}
		rev, err := mesh.KVPut(ctx, bucketName, putName, val)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL mesh kv put: %v\n", err)
			return 1
		}
		// s729: always-emit KVPutPrint on put success (mold StreamDeletePrint s726 + s714).
		printDTO := iomesh.NewKVPutPrint(bucketName, putName, rev)
		if *jsonOut {
			fmt.Print(iomesh.FormatKVPutJSON(printDTO))
			return 0
		}
		fmt.Print(iomesh.FormatKVPut(printDTO))
		return 0
	}

	if delName != "" {
		if err := mesh.KVDelete(ctx, bucketName, delName); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL mesh kv delete: %v\n", err)
			return 1
		}
		// s729: always-emit KVDeletePrint on delete success (mold StreamDeletePrint s726 + s714).
		printDTO := iomesh.NewKVDeletePrint(bucketName, delName)
		if *jsonOut {
			fmt.Print(iomesh.FormatKVDeleteJSON(printDTO))
			return 0
		}
		fmt.Print(iomesh.FormatKVDelete(printDTO))
		return 0
	}

	if getName != "" {
		entry, err := mesh.KVGet(ctx, bucketName, getName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL mesh kv get: %v\n", err)
			return 1
		}
		if *jsonOut {
			// s714: print DTO always-emits bucket/key/value/revision/created_at
			// ("" when zero; value base64, never omitempty-hide created_at).
			// s741: FormatKVEntryJSON (helper completeness; no ad-hoc MarshalIndent).
			fmt.Print(iomesh.FormatKVEntryJSON(iomesh.NewKVEntryPrint(*entry)))
			return 0
		}
		fmt.Print(iomesh.FormatKVEntry(*entry))
		return 0
	}

	keys, err := mesh.KVListKeys(ctx, bucketName, *prefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL mesh kv list: %v\n", err)
		return 1
	}
	if *jsonOut {
		// s714: list envelope always-emits bucket/prefix/count/keys (not bare array).
		// s741: FormatKVKeysJSON (helper completeness; no ad-hoc MarshalIndent).
		fmt.Print(iomesh.FormatKVKeysJSON(iomesh.NewKVKeysPrint(bucketName, *prefix, keys)))
		return 0
	}
	fmt.Print(iomesh.FormatKVKeys(bucketName, keys))
	return 0
}

func cmdMeshUsage(args []string) int {
	// Local process meter is empty in a fresh CLI process; still print schema + guidance.
	// When wired as MetricsSink during agent runs, snapshots are in-process only.
	// Remote multi-tenant dashboards consume dept.agent.llm_call on the platform (not this CLI).
	// s738: --json marshals UsagePrint always-emit (via FormatUsageJSON → NewUsagePrint).
	fs := flag.NewFlagSet("mesh usage", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "print UsagePrint always-emit JSON (started/as_of \"\" when zero; by_model []; s738)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	mesh := iomesh.New(iomesh.Config{}, nil)
	snap := mesh.Usage()
	if *jsonOut {
		// FormatUsageJSON → UsagePrint always-emit (empty-time honesty; by_model [] not null).
		fmt.Print(iomesh.FormatUsageJSON(snap))
	} else {
		fmt.Print(iomesh.FormatUsage(snap))
		fmt.Fprintln(os.Stderr, "note: metering accumulates during agent runs in-process (MetricsSink); CLI `mesh usage` shows the current process only. Use --json for UsagePrint scrapers (s738; empty-time honest). Platform remote dashboards use dept.agent.llm_call when mesh is enabled.")
	}
	return 0
}

func cmdMeshDogfood(args []string) int {
	fs := flag.NewFlagSet("mesh smoke", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath        = fs.String("config", "", "config.toml path")
		workspace         = fs.String("C", "", "workspace for context query")
		strict            = fs.Bool("strict", false, "fail if context/emit/ready/memory soft-fail")
		skipContext       = fs.Bool("skip-context", false, "skip context plane probe")
		skipEmit          = fs.Bool("skip-emit", false, "skip dept emit probe")
		skipMemory        = fs.Bool("skip-memory", false, "skip memory_ingest / memory_recall / memory_retrieve probes")
		skipStreams       = fs.Bool("skip-streams", false, "skip streams list probe (GET /v1/streams)")
		kvBucket          = fs.String("kv-bucket", "", "soft KV list-keys probe on bucket (omit = skip kv step)")
		kvEnsure          = fs.Bool("kv-ensure", false, "with --kv-bucket: best-effort KVCreateBucket before list (soft fail-open)")
		pubSubject        = fs.String("pub-subject", "", "soft ephemeral Pub probe subject (omit = skip pub step)")
		consumerStream    = fs.String("consumer-stream", "", "soft consumer create probe stream (requires --consumer-name)")
		consumerName      = fs.String("consumer-name", "", "soft consumer create probe name (requires --consumer-stream)")
		consumerFilter    = fs.String("consumer-filter", "", "optional filter_subject for consumer create probe")
		consumerFetch     = fs.Bool("consumer-fetch", false, "after create: soft fetch batch=1 max_wait=500ms (empty OK; no ack)")
		consumerDelete    = fs.Bool("consumer-delete", false, "after create: best-effort DeleteConsumer cleanup (soft fail-open)")
		waitReady         = fs.Duration("wait-ready", 0, "soft WaitReady preflight budget before ready (0=off)")
		waitInterval      = fs.Duration("wait-interval", 0, "WaitReady poll interval (default 500ms when --wait-ready set)")
		waitRequireHealth = fs.Bool("wait-require-health", false, "WaitReady requires Health OK each attempt")
		jsonOut           = fs.Bool("json", false, "print smoke report as JSON (stage CI evidence)")
		verbose           = fs.Bool("v", false, "verbose logs")
		endpoint          = fs.String("endpoint", "", "override IOMESH_ENDPOINT / config")
		memoryEndpoint    = fs.String("memory-endpoint", "", "memory sidecar base (IOMESH_MEMORY_ENDPOINT / MEMORY_SIDECAR_URL)")
		tenant            = fs.String("tenant", "", "override tenant")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	if *memoryEndpoint != "" {
		cfg.Memory.Endpoint = *memoryEndpoint
	}
	if *tenant != "" {
		cfg.IOMesh.Tenant = *tenant
	}
	applyInferredBroker(cfg)
	// Env already applied by config.Load; allow empty endpoint → SKIP report.
	// s687: wire [memory].pull_role / pull_allow_suffix onto Client so consumer probe
	// sends federated ACL headers + dogfood report always-emits pull_role/pull_allow_suffix.
	mesh := iomesh.New(iomesh.Config{
		Enabled:         cfg.IOMesh.Enabled,
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          cfg.IOMesh.Tenant,
		APIKeyEnv:       cfg.IOMesh.APIKeyEnv,
		OrgID:           cfg.IOMesh.Org,
		WorkspaceID:     cfg.IOMesh.Workspace,
		DualWrite:       cfg.Memory.DualWrite, // report-only; does not gate memory_ingest probe
		MemoryEndpoint:  cfg.Memory.Endpoint,  // stage warm sidecar for memory_retrieve
		EmitDeptStreams: cfg.IOMesh.EmitDeptStreams,
		ContextPlane:    cfg.IOMesh.ContextPlane,
		IncludeLineage:  cfg.IOMesh.IncludeLineage,
		PolicyMode:      iomesh.PolicyMode(cfg.IOMesh.PolicyMode),
		CatalogPlane:    cfg.IOMesh.CatalogPlane,
		InjectCatalog:   cfg.IOMesh.InjectCatalog,
		Role:            strings.TrimSpace(cfg.Memory.PullRole),
		PullAllowSuffix: strings.TrimSpace(cfg.Memory.PullAllowSuffix),
	}, logger)

	ws := *workspace
	if ws == "" {
		ws = cfg.Agent.Workspace
	}
	if ws == "" {
		if wd, err := os.Getwd(); err == nil {
			ws = wd
		} else {
			ws = "."
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	rep := mesh.Dogfood(ctx, iomesh.DogfoodOptions{
		Workspace:         ws,
		Strict:            *strict,
		SkipContext:       *skipContext,
		SkipEmit:          *skipEmit,
		SkipMemory:        *skipMemory,
		SkipStreams:       *skipStreams,
		KVBucket:          strings.TrimSpace(*kvBucket),
		KVEnsure:          *kvEnsure,
		PubSubject:        strings.TrimSpace(*pubSubject),
		ConsumerStream:    strings.TrimSpace(*consumerStream),
		ConsumerName:      strings.TrimSpace(*consumerName),
		ConsumerFilter:    strings.TrimSpace(*consumerFilter),
		ConsumerFetch:     *consumerFetch,
		ConsumerDelete:    *consumerDelete,
		WaitReady:         *waitReady,
		WaitReadyInterval: *waitInterval,
		WaitRequireHealth: *waitRequireHealth,
		Version:           version,
	})
	if *jsonOut {
		fmt.Print(iomesh.FormatReportJSON(rep))
	} else {
		fmt.Print(iomesh.FormatReport(rep))
	}
	return rep.ExitCode
}

func cmdAgent(args []string) int {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		model       = fs.String("m", "", "model override")
		configPath  = fs.String("config", "", "config.toml path")
		workspace   = fs.String("C", "", "workspace directory")
		yolo        = fs.Bool("yolo", false, "auto-approve mutating tools")
		yoloAlias   = fs.Bool("always-approve", false, "alias for --yolo")
		verbose     = fs.Bool("v", false, "verbose logging to stderr")
		listen      = fs.String("listen", acp.DefaultListen, "WebSocket listen addr (serve mode; default loopback)")
		listenShort = fs.String("l", "", "alias for --listen")
		wsPath      = fs.String("path", acp.DefaultWSPath, "WebSocket path (serve mode)")
		token       = fs.String("token", "", "require Bearer/?token= for WebSocket (serve mode)")
	)
	// Parse flags; remaining is mode (stdio|serve).
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "usage: iomesh agent [flags] stdio|serve")
		return 2
	}
	if *yoloAlias {
		*yolo = true
	}
	if *listenShort != "" {
		*listen = *listenShort
	}
	logger := newLogger(*verbose)

	opts := acp.Options{
		ConfigPath: *configPath,
		Workspace:  *workspace,
		Model:      *model,
		Yolo:       *yolo,
		Version:    version,
		Logger:     logger,
	}

	switch rest[0] {
	case "stdio":
		srv := acp.New(opts)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		// Protocol on stdout; logs on stderr only.
		if err := srv.Run(ctx, os.Stdin, os.Stdout); err != nil && err != context.Canceled && err != io.EOF {
			fmt.Fprintf(os.Stderr, "acp: %v\n", err)
			return 1
		}
		return 0
	case "serve":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		// Prefer loopback; warn if binding non-loopback without token.
		loopback := strings.HasPrefix(*listen, "127.") ||
			strings.HasPrefix(*listen, "localhost") ||
			strings.HasPrefix(*listen, "[::1]") ||
			strings.HasPrefix(*listen, ":") // :port binds all interfaces — not loopback
		// ":7400" binds 0.0.0.0 — treat as non-loopback.
		if strings.HasPrefix(*listen, ":") {
			loopback = false
		}
		if !loopback && *token == "" {
			fmt.Fprintln(os.Stderr, "warning: binding non-loopback without --token is unsafe")
		}
		// Allow any browser Origin only on loopback (local IDE DX). Remote binds reject cross-origin by default.
		err := acp.ListenAndServe(ctx, acp.ServeOptions{
			Listen:         *listen,
			Path:           *wsPath,
			Token:          *token,
			AllowAnyOrigin: loopback,
			Options:        opts,
		})
		if err != nil && err != context.Canceled {
			fmt.Fprintf(os.Stderr, "acp serve: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown agent mode %q\n", rest[0])
		return 2
	}
}

// cmdMemory is the Memory Palace operator surface (local-first cost-max path).
func cmdMemory(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: iomesh memory [pull|ingest|ingest-dir] [flags]")
		return 2
	}
	switch args[0] {
	case "pull":
		return cmdMemoryPull(args[1:])
	case "ingest":
		return cmdMemoryIngest(args[1:])
	case "ingest-dir", "ingestdir", "ingest_dir":
		return cmdMemoryIngestDir(args[1:])
	case "help", "-h", "--help":
		fmt.Fprint(os.Stderr, `iomesh memory — local-first Memory Palace operators (cost-max M1)

  iomesh memory pull         durable mesh pull → local MCP memory_ingest_turn
  iomesh memory ingest       ingest text via MCP memory_ingest_turn (session_id minted)
  iomesh memory ingest-dir   folder ingest into private overlay (session_id minted)

Flags (pull):
  --config path         config.toml
  --stream S            durable stream (default: [memory].pull_stream or EVENTS)
  --name C              durable consumer name (required unless config pull_consumer)
  --filter F            optional filter_subject (role-aware default when empty; s660/s678/s687)
  --batch N             fetch batch (default 8)
  --max-wait dur        long-poll wait (default 2s)
  --once                single fetch cycle then exit
  --dry-run             map messages only (no MCP ingest); still acks when --ack
  --no-ack              do not ack after ingest (default: ack)
  --yes                 confirm mutating pull loop (required unless --dry-run)
  --json                print MemoryPullStatsPrint JSON (always-emits identity + knobs + counters + process evidence; complete s747)
  --endpoint url        override IOMESH_ENDPOINT
  --org id              override [iomesh].org / IOMESH_ORG (X-IOMesh-Org; empty fail-opens)
  --mcp-server name     MCP server name for memory tools (default memory)
  --role R              optional X-IOMesh-Role (operator|admin|agent|auditor|viewer|memory|custom); [memory].pull_role
  --pull-allow-suffix S optional X-IOMesh-Pull-Allow-Suffix (comma tokens; role=custom); [memory].pull_allow_suffix
  -v                    verbose

Flags (ingest):
  --config path         config.toml
  --yes                 confirm mutating ingest (required)
  --session-id id       override (default: minted local-overlay when the walk has none)
  --mcp-server name     MCP server name for memory tools (default memory)
  --tenant T            palace tenant (default [memory].tenant)

Flags (ingest-dir):
  --config path         config.toml
  --yes                 confirm mutating folder ingest (required unless --dry-run)
  --dry-run             list files only (no MCP)
  --limit N             max files (default 32)
  --session-id id       override (default: minted local-overlay when the walk has none)
  --mcp-server name     MCP server name for memory tools (default memory)
  --tenant T            palace tenant (default [memory].tenant)
  -C dir                workspace root for path jail (default cwd)

Honesty: dual_write remains optional audit (default OFF). Hosted Palace sunset until scale.
  /memory ingest and iomesh memory ingest mint session_id=local-overlay when the operator
  has none so iomesh-memory-mcp v0.1.0 memory_ingest_turn can complete. Retrieve without
  a session_id stays unfiltered and finds the private overlay. Catalog list ≠ consume.
  Role/suffix headers are Beta federated ACL (s675); role-aware default filter is s678/s687 Beta —
  memory → tenant.memory.> (peer aion s686); fail-open when empty — not full IdP RBAC GA.
  s705: PASS/summary and --json always emit stream/consumer/filter_subject/pull_role/pull_allow_suffix/tenant
  + knobs (dry_run/dual_write/batch/max_wait_ms/once) + counters; empty identity honest; peer aion s704.
  s717: always emit process evidence endpoint/org/workspace (empty honest) + result(ok|err)/exit_code(0|1)
  + duration_ms/ack; peer aion s716 — process evidence ≠ invent pull success.
  s747: process evidence complete (identity + knobs/counters + process evidence); completeness pin only —
  no new fields; peer aion s746 — process evidence ≠ invent pull success · dual_write OFF · not full mesh RBAC GA.
`)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown memory subcommand %q\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: iomesh memory [pull|ingest|ingest-dir] [flags]")
		return 2
	}
}

func cmdMemoryIngest(args []string) int {
	fs := flag.NewFlagSet("memory ingest", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath = fs.String("config", "", "config.toml path")
		yes        = fs.Bool("yes", false, "confirm mutating ingest")
		sessionID  = fs.String("session-id", "", "override session_id (default minted local-overlay)")
		mcpServer  = fs.String("mcp-server", "", "MCP memory server name")
		tenantFlag = fs.String("tenant", "", "palace tenant")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	content := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if content == "" {
		fmt.Fprintln(os.Stderr, "usage: iomesh memory ingest --yes <text>")
		fmt.Fprintln(os.Stderr, "  session_id minted as local-overlay when the operator has none")
		return 2
	}
	if !*yes {
		fmt.Fprintln(os.Stderr, "memory ingest: mutating — pass --yes (dual_write stays off)")
		return 2
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	serverName := strings.TrimSpace(*mcpServer)
	if serverName == "" {
		serverName = strings.TrimSpace(cfg.Memory.Server)
	}
	if serverName == "" {
		serverName = "memory"
	}
	tenant := strings.TrimSpace(*tenantFlag)
	if tenant == "" {
		tenant = strings.TrimSpace(cfg.Memory.Tenant)
	}
	sid := agent.ResolveMemoryIngestSessionID(*sessionID, "")
	cl, closer, err := connectMemoryMCP(cfg, serverName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "memory ingest: %v\n", err)
		return 1
	}
	defer closer()
	argsIn := map[string]any{
		"role":       "user",
		"content":    content,
		"session_id": sid,
	}
	if tenant != "" {
		argsIn["tenant"] = tenant
	}
	out, err := cl.CallTool(context.Background(), "memory_ingest_turn", argsIn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "memory ingest: %v\n", err)
		return 1
	}
	minted := strings.TrimSpace(*sessionID) == ""
	fmt.Printf("memory ingest: session_id=%s", sid)
	if minted {
		fmt.Print(" (minted · operator had none)")
	}
	fmt.Printf(" dual_write=%v · not Memory GA · catalog list ≠ consume\n", cfg.Memory.DualWrite)
	if s := strings.TrimSpace(out); s != "" {
		fmt.Println(s)
	}
	return 0
}

func cmdMemoryIngestDir(args []string) int {
	fs := flag.NewFlagSet("memory ingest-dir", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath = fs.String("config", "", "config.toml path")
		yes        = fs.Bool("yes", false, "confirm mutating folder ingest")
		dryRun     = fs.Bool("dry-run", false, "list files only (no MCP)")
		limit      = fs.Int("limit", 0, "max files (default 32)")
		sessionID  = fs.String("session-id", "", "override session_id (default minted local-overlay)")
		mcpServer  = fs.String("mcp-server", "", "MCP memory server name")
		tenantFlag = fs.String("tenant", "", "palace tenant")
		workDir    = fs.String("C", "", "workspace root for path jail")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	dir := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if dir == "" {
		fmt.Fprintln(os.Stderr, "usage: iomesh memory ingest-dir [--dry-run|--yes] <path>")
		fmt.Fprintln(os.Stderr, "  folder ingest; session_id minted as local-overlay when the walk has none")
		return 2
	}
	if !*dryRun && !*yes {
		fmt.Fprintln(os.Stderr, "memory ingest-dir: mutating — pass --yes or --dry-run (dual_write stays off)")
		return 2
	}
	root := strings.TrimSpace(*workDir)
	ws, err := workspace.Open(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "memory ingest-dir: workspace: %v\n", err)
		return 1
	}
	plan, err := agent.ListIngestDirFiles(ws, dir, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "memory ingest-dir: %v\n", err)
		return 1
	}
	sid := agent.ResolveMemoryIngestSessionID(*sessionID, "")
	minted := strings.TrimSpace(*sessionID) == ""
	if *dryRun {
		fmt.Println(agent.FormatIngestDirPlan(plan, sid, minted, true))
		return 0
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	serverName := strings.TrimSpace(*mcpServer)
	if serverName == "" {
		serverName = strings.TrimSpace(cfg.Memory.Server)
	}
	if serverName == "" {
		serverName = "memory"
	}
	tenant := strings.TrimSpace(*tenantFlag)
	if tenant == "" {
		tenant = strings.TrimSpace(cfg.Memory.Tenant)
	}
	cl, closer, err := connectMemoryMCP(cfg, serverName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "memory ingest-dir: %v\n", err)
		return 1
	}
	defer closer()
	ingested := 0
	failed := 0
	var lines []string
	for _, f := range plan.Files {
		content := "file: " + f.Rel + "\n\n" + f.Text
		callArgs := map[string]any{
			"role":       "user",
			"content":    content,
			"session_id": sid,
		}
		if tenant != "" {
			callArgs["tenant"] = tenant
		}
		out, ierr := cl.CallTool(context.Background(), "memory_ingest_turn", callArgs)
		if ierr != nil {
			failed++
			lines = append(lines, f.Rel+": "+ierr.Error())
			continue
		}
		ingested++
		if s := strings.TrimSpace(out); s != "" {
			lines = append(lines, f.Rel+": "+s)
		} else {
			lines = append(lines, f.Rel+": ok")
		}
	}
	fmt.Printf("ingest-dir: dir=%s ingested=%d failed=%d skipped=%d session_id=%s",
		plan.Dir, ingested, failed, len(plan.Skipped), sid)
	if minted {
		fmt.Print(" (minted · operator had none)")
	}
	fmt.Printf(" dual_write=%v · not Memory GA · catalog list ≠ consume · private overlay\n", cfg.Memory.DualWrite)
	for _, line := range lines {
		fmt.Printf("  %s\n", line)
	}
	for _, s := range plan.Skipped {
		fmt.Printf("  skip %s\n", s)
	}
	if ingested == 0 && failed > 0 {
		return 1
	}
	return 0
}

func connectMemoryMCP(cfg *config.Config, serverName string) (*mcp.Client, func(), error) {
	if cfg == nil {
		return nil, func() {}, fmt.Errorf("config required")
	}
	if !cfg.MCP.Enabled && !cfg.Features.MCP {
		return nil, func() {}, fmt.Errorf("MCP disabled — enable [mcp] or use --dry-run")
	}
	var servers []mcp.ServerConfig
	for _, s := range cfg.MCP.Servers {
		servers = append(servers, mcpServerFromTOML(s, cfg))
	}
	if len(servers) == 0 {
		return nil, func() {}, fmt.Errorf("no MCP servers configured")
	}
	ctx := context.Background()
	mgr := mcp.NewManager(ctx, servers, slog.Default())
	cl := mgr.ClientByName(serverName)
	if cl == nil {
		_ = mgr.Close()
		return nil, func() {}, fmt.Errorf("MCP server %q not connected", serverName)
	}
	return cl, func() { _ = mgr.Close() }, nil
}

func cmdMemoryPull(args []string) int {
	fs := flag.NewFlagSet("memory pull", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath      = fs.String("config", "", "config.toml path")
		stream          = fs.String("stream", "", "durable stream name")
		name            = fs.String("name", "", "durable consumer name")
		filter          = fs.String("filter", "", "optional filter_subject")
		batch           = fs.Int("batch", 0, "fetch batch size")
		maxWait         = fs.Duration("max-wait", 0, "long-poll max wait")
		once            = fs.Bool("once", false, "single fetch cycle")
		dryRun          = fs.Bool("dry-run", false, "map only; no MCP local ingest")
		noAck           = fs.Bool("no-ack", false, "do not ack after success")
		yes             = fs.Bool("yes", false, "confirm mutating pull (required unless --dry-run)")
		jsonOut         = fs.Bool("json", false, "print MemoryPullStatsPrint JSON (always-emits identity + knobs + counters + process evidence; complete s747)")
		endpoint        = fs.String("endpoint", "", "override mesh endpoint")
		org             = fs.String("org", "", "override [iomesh].org / IOMESH_ORG (X-IOMesh-Org; empty fail-opens)")
		mcpServer       = fs.String("mcp-server", "", "MCP memory server name")
		role            = fs.String("role", "", "optional X-IOMesh-Role (operator|admin|agent|auditor|viewer|memory|custom)")
		pullAllowSuffix = fs.String("pull-allow-suffix", "", "optional X-IOMesh-Pull-Allow-Suffix (comma tokens; role=custom)")
		verbose         = fs.Bool("v", false, "verbose logs")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	logger := newLogger(*verbose)
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	if *endpoint != "" {
		cfg.IOMesh.Endpoint = *endpoint
		cfg.IOMesh.Enabled = true
	}
	applyOrgFlag(cfg, *org)
	applyInferredBroker(cfg)

	streamName := strings.TrimSpace(*stream)
	if streamName == "" {
		streamName = strings.TrimSpace(cfg.Memory.PullStream)
	}
	if streamName == "" {
		streamName = "EVENTS"
	}
	consumerName := strings.TrimSpace(*name)
	if consumerName == "" {
		consumerName = strings.TrimSpace(cfg.Memory.PullConsumer)
	}
	filterSub := strings.TrimSpace(*filter)
	if filterSub == "" {
		filterSub = strings.TrimSpace(cfg.Memory.PullFilter)
	}
	// s660/s678: empty --filter / pull_filter → role-aware default from memory or mesh tenant.
	pullTenant := strings.TrimSpace(cfg.Memory.Tenant)
	if pullTenant == "" {
		pullTenant = strings.TrimSpace(cfg.IOMesh.Tenant)
	}
	// s675: federated pull role + custom allow-suffix (flags override [memory] config). Fail-open empty → omit headers.
	// Resolved before default filter so s678 can use role + single custom suffix.
	pullRole := strings.TrimSpace(*role)
	if pullRole == "" {
		pullRole = strings.TrimSpace(cfg.Memory.PullRole)
	}
	allowSuffix := strings.TrimSpace(*pullAllowSuffix)
	if allowSuffix == "" {
		allowSuffix = strings.TrimSpace(cfg.Memory.PullAllowSuffix)
	}
	filterSub = iomesh.DefaultMemoryPullFilterForRole(filterSub, pullTenant, pullRole, allowSuffix)
	batchN := *batch
	if batchN <= 0 {
		batchN = cfg.Memory.PullBatch
	}
	if batchN <= 0 {
		batchN = 8
	}
	wait := *maxWait
	if wait <= 0 && cfg.Memory.PullMaxWaitMS > 0 {
		wait = time.Duration(cfg.Memory.PullMaxWaitMS) * time.Millisecond
	}
	if wait <= 0 {
		wait = 2 * time.Second
	}
	serverName := strings.TrimSpace(*mcpServer)
	if serverName == "" {
		serverName = strings.TrimSpace(cfg.Memory.Server)
	}
	if serverName == "" {
		serverName = "memory"
	}

	if consumerName == "" {
		fmt.Fprintln(os.Stderr, "usage: iomesh memory pull --name C [--stream S] --yes")
		fmt.Fprintln(os.Stderr, "  consumer name required (flag or [memory].pull_consumer)")
		return 2
	}
	if !*dryRun && !*yes {
		fmt.Fprintln(os.Stderr, "memory pull: --yes required for mutating local ingest (or use --dry-run)")
		return 2
	}

	// Prefer [iomesh].tenant for X-IOMesh-Tenant; fall back to memory tenant (s660).
	meshTenant := strings.TrimSpace(cfg.IOMesh.Tenant)
	if meshTenant == "" {
		meshTenant = strings.TrimSpace(cfg.Memory.Tenant)
	}
	// s717: process mesh identity from [iomesh] (empty string honest when unset).
	meshEndpoint := strings.TrimSpace(cfg.IOMesh.Endpoint)
	meshOrg := strings.TrimSpace(cfg.IOMesh.Org)
	meshWorkspace := strings.TrimSpace(cfg.IOMesh.Workspace)
	ackKnob := !*noAck

	// s705+s717: print meta for always-emit identity + knobs + process evidence.
	// dual_write is report-only from [memory].dual_write (default OFF); does not gate pull.
	// Result/exit_code/duration_ms filled on each exit path.
	printMeta := iomesh.MemoryPullPrintMeta{
		Tenant:          pullTenant,
		PullRole:        pullRole,
		PullAllowSuffix: allowSuffix,
		Endpoint:        meshEndpoint,
		Org:             meshOrg,
		Workspace:       meshWorkspace,
		DryRun:          *dryRun,
		DualWrite:       cfg.Memory.DualWrite,
		Batch:           batchN,
		MaxWaitMS:       int(wait / time.Millisecond),
		Once:            *once,
		Ack:             ackKnob,
	}
	// Seed partial stats for early fail emit (stream/consumer/filter known pre-run).
	seedStats := func() iomesh.MemoryPullStats {
		return iomesh.MemoryPullStats{
			Stream:   streamName,
			Consumer: consumerName,
			Filter:   filterSub,
		}
	}
	emitPullPrint := func(st iomesh.MemoryPullStats, meta iomesh.MemoryPullPrintMeta, ok bool, errMsg string) {
		dto := iomesh.NewMemoryPullStatsPrint(st, meta)
		if *jsonOut {
			fmt.Fprint(os.Stdout, iomesh.FormatMemoryPullStatsJSON(dto))
			if !ok && strings.TrimSpace(errMsg) != "" {
				fmt.Fprintf(os.Stderr, "FAIL memory pull: %s\n", errMsg)
			}
		} else {
			fmt.Fprint(os.Stdout, iomesh.FormatMemoryPullStats(dto, ok, errMsg))
		}
	}

	mesh := iomesh.New(iomesh.Config{
		Enabled:         cfg.IOMesh.Enabled,
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          meshTenant,
		APIKeyEnv:       cfg.IOMesh.APIKeyEnv,
		OrgID:           cfg.IOMesh.Org,
		WorkspaceID:     cfg.IOMesh.Workspace,
		Role:            pullRole,
		PullAllowSuffix: allowSuffix,
	}, logger)
	if !mesh.Enabled() {
		printMeta.Result = "err"
		printMeta.ExitCode = 1
		emitPullPrint(seedStats(), printMeta, false, "mesh disabled (set IOMESH_ENDPOINT / [iomesh])")
		return 1
	}

	// Always log effective filter once at start (s660/s678); role/suffix once (s675). Empty role/suffix = fail-open omit headers.
	fmt.Fprintf(os.Stderr, "memory pull filter_subject=%q tenant=%q role=%q pull_allow_suffix=%q\n",
		filterSub, pullTenant, pullRole, allowSuffix)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opt := iomesh.MemoryPullOptions{
		Stream:  streamName,
		Name:    consumerName,
		Filter:  filterSub,
		Batch:   batchN,
		MaxWait: wait,
		Ack:     ackKnob,
		DryRun:  *dryRun,
		OnMessage: func(msg iomesh.StreamMessage, env iomesh.MemoryEnvelope, skipped bool, err error) {
			if !*verbose && err == nil && !skipped {
				return
			}
			status := "ingest"
			if skipped {
				status = "skip"
			}
			if err != nil {
				status = "err"
			}
			fmt.Fprintf(os.Stderr, "memory pull %s seq=%d subject=%s content_len=%d err=%v\n",
				status, msg.Seq, msg.Subject, len(env.Content), err)
		},
	}
	if *once {
		opt.MaxLoops = 1
	}

	if !*dryRun {
		// Local palace via MCP memory_ingest_turn (fail-closed if no server).
		if !cfg.MCP.Enabled && !cfg.Features.MCP {
			printMeta.Result = "err"
			printMeta.ExitCode = 1
			emitPullPrint(seedStats(), printMeta, false, "MCP disabled — enable [mcp] or use --dry-run")
			return 1
		}
		var servers []mcp.ServerConfig
		for _, s := range cfg.MCP.Servers {
			servers = append(servers, mcpServerFromTOML(s, cfg))
		}
		mgr := mcp.NewManager(ctx, servers, logger)
		defer func() { _ = mgr.Close() }()
		cl := mgr.ClientByName(serverName)
		if cl == nil {
			printMeta.Result = "err"
			printMeta.ExitCode = 1
			emitPullPrint(seedStats(), printMeta, false, fmt.Sprintf("MCP server %q not connected", serverName))
			return 1
		}
		tenant := strings.TrimSpace(cfg.Memory.Tenant)
		if tenant == "" {
			tenant = strings.TrimSpace(cfg.IOMesh.Tenant)
		}
		opt.LocalIngest = func(cctx context.Context, env iomesh.MemoryEnvelope) error {
			args := map[string]any{
				"role":    env.Role,
				"content": env.Content,
			}
			if env.EventTime != "" {
				args["event_time"] = env.EventTime
			}
			if env.SessionID != "" {
				args["session_id"] = env.SessionID
			}
			if tenant != "" {
				args["tenant"] = tenant
			}
			_, err := cl.CallTool(cctx, "memory_ingest_turn", args)
			return err
		}
	}

	started := time.Now()
	st, err := mesh.RunMemoryPull(ctx, opt)
	printMeta.DurationMS = int(time.Since(started).Milliseconds())
	// Seed identity on stats when create failed early (st may be partial).
	if st.Stream == "" {
		st.Stream = streamName
	}
	if st.Consumer == "" {
		st.Consumer = consumerName
	}
	if st.Filter == "" {
		st.Filter = filterSub
	}

	// Hard err (non-cancel): result=err, exit_code=1.
	if err != nil && ctx.Err() == nil {
		printMeta.Result = "err"
		printMeta.ExitCode = 1
		emitPullPrint(st, printMeta, false, err.Error())
		return 1
	}
	// Soft-fail (errors>0 && ingested==0 && !dryRun): result=err, exit_code=1.
	// Success (incl. cancel with partial progress): result=ok, exit_code=0.
	// Process evidence ≠ invent pull success from identity alone (s717 / peer aion s716).
	softFail := st.Errors > 0 && st.Ingested == 0 && !*dryRun
	if softFail {
		printMeta.Result = "err"
		printMeta.ExitCode = 1
		emitPullPrint(st, printMeta, false, "")
		return 1
	}
	printMeta.Result = "ok"
	printMeta.ExitCode = 0
	// Summary always-emits identity (s705) + process evidence (s717).
	emitPullPrint(st, printMeta, true, "")
	return 0
}

func loadConfig(path string) (*config.Config, error) {
	if path != "" {
		return config.Load(path)
	}
	return config.LoadUser()
}

func newLogger(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `iomesh — I/O Mesh coding agent TUI (Go rewrite of Grok Build)

Usage:
  iomesh [flags]                 interactive full-screen TUI
  iomesh --repl                  classic line REPL
  iomesh -p "prompt"             headless single prompt
  iomesh -c                      continue latest session
  iomesh --session <id>          resume session by id
  iomesh sessions                list sessions in workspace
  iomesh skills                  list SKILL.md catalogs
  iomesh mcp [--connect]         list configured MCP servers
  iomesh plugins [list|validate|smoke] Agent Plugins package discover/validate/smoke (opt-in; ≠ GA)
  iomesh setup init|preflight    setup lifecycle (write managed config · residual-honest preflight)
  iomesh mesh smoke              I/O Mesh smoke (health/context/emit/pub/memory; needs IOMESH_ENDPOINT)
  iomesh mesh pub                ephemeral POST /v1/pub (--subject --payload|--payload-file --yes; PubPrint always-emit)
  iomesh mesh consumer create    durable pull consumer create (--stream --name --yes)
  iomesh mesh consumer delete    durable pull consumer delete (--stream --name --yes)
  iomesh mesh consumer ack|nack  ack/nack sequences (--stream --name --seq --yes)
  iomesh mesh wait               poll mesh Ready until OK (operator preflight)
  iomesh mesh status             operator snapshot (StatusLine + Health/Ready; --strict gates result=err)
  iomesh memory pull             mesh durable pull → local MCP palace (cost-max M1; --yes)
  iomesh memory ingest           local overlay text ingest (session_id minted; --yes)
  iomesh memory ingest-dir       folder ingest into private overlay (--dry-run|--yes)
  iomesh models                  list configured models
  iomesh agent stdio             ACP JSON-RPC over stdio (IDE integration)
  iomesh agent serve             ACP JSON-RPC over WebSocket (default 127.0.0.1:7400/acp)
  iomesh agent --yolo stdio      ACP with auto-approve tools
  iomesh version

Flags:
  -p, --prompt string   headless prompt
  -m string             model override (logical name)
  -C string             workspace directory
  -c, --continue        resume latest session
  --session id          resume specific session id
  --no-save             disable session autosave
  --repl                classic line REPL (non-alt-screen)
  --config path         config.toml path
  --yolo                auto-approve mutating tools
  -v                    verbose logs

Agent serve (WebSocket) flags:
  --listen, -l addr     bind address (default 127.0.0.1:7400)
  --path /acp           WebSocket path
  --token secret        require Bearer or ?token=

Default model cascade: deepseek-v4-flash → deepseek-v4-pro → grok-4.5
  Optional Google: gemini-2.5-flash|pro (GEMINI_API_KEY) · vertex-gemini-2.5-* (VERTEX_API_KEY + GOOGLE_CLOUD_PROJECT)
Config: ~/.iomesh/config.toml  (or $IOMESH_CONFIG)

Environment:
  DEEPSEEK_API_KEY    DeepSeek API key (Flash/Pro)
  XAI_API_KEY         xAI / Grok fallback
  IOMESH_API_KEY      I/O Mesh platform
  IOMESH_ENDPOINT     enable mesh integration
  IOMESH_MEMORY_ENDPOINT / MEMORY_SIDECAR_URL  sync memory retrieve base (sidecar)
  IOMESH_DEFAULT_MODEL  override default model name
`)
}

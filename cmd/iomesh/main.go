// Command iomesh is the I/O Mesh TUI coding agent (Go rewrite of Grok Build),
// with DeepSeek-first model routing and optional I/O Mesh platform integration.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/acp"
	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/config"
	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/mcp"
	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/session"
	"github.com/iome-sh/iomesh-tui/internal/skills"
	"github.com/iome-sh/iomesh-tui/internal/tui"
)

// Overridden at link time by make build: -X main.version=$(VERSION)
// (must be a var, not const, for -ldflags -X).
var version = "0.16.0"

func main() {
	// Identify mesh HTTP traffic for operator support (parity with iomesh-client-sdk-go User-Agent).
	iomesh.SetUserAgent("iomesh-tui/" + version)
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
		case "mesh":
			return cmdMesh(args[1:])
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
	mesh := iomesh.New(iomesh.Config{
		Enabled:         cfg.IOMesh.Enabled,
		Endpoint:        cfg.IOMesh.Endpoint,
		Tenant:          cfg.IOMesh.Tenant,
		APIKeyEnv:       cfg.IOMesh.APIKeyEnv,
		OrgID:           cfg.IOMesh.Org,
		WorkspaceID:     cfg.IOMesh.Workspace,
		DualWrite:       cfg.Memory.DualWrite, // report evidence only; does not gate memory_ingest probe
		MemoryEndpoint:  cfg.Memory.Endpoint,  // optional sidecar for sync retrieve / auto-recall
		EmitDeptStreams: cfg.IOMesh.EmitDeptStreams,
		ContextPlane:    cfg.IOMesh.ContextPlane,
		IncludeLineage:  cfg.IOMesh.IncludeLineage,
		PolicyMode:      iomesh.PolicyMode(cfg.IOMesh.PolicyMode),
		CatalogPlane:    cfg.IOMesh.CatalogPlane,
		InjectCatalog:   cfg.IOMesh.InjectCatalog,
	}, logger)
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
	}, rtr, mesh, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent: %v\n", err)
		return 1
	}
	defer func() { _ = rt.Close() }()

	// Mesh catalog tools (list_mesh_catalog / mesh_status) when catalog plane on.
	rt.AttachMeshTools()

	// Skills: workspace + user dirs (+ config extras). Fail-open on load errors.
	if cfg.Skills.Enabled && cfg.Features.Skills {
		dirs := skills.DefaultDirs(rt.Workspace().Root())
		dirs = append(dirs, cfg.Skills.Dirs...)
		if cat, err := skills.LoadDirs(dirs...); err != nil {
			logger.Warn("skills load", "err", err)
		} else if cat.Len() > 0 {
			rt.AttachSkills(cat)
			logger.Info("skills loaded", "count", cat.Len())
		}
	}

	// MCP stdio/HTTP servers (opt-in). Fail-open per server inside manager.
	if cfg.MCP.Enabled && cfg.Features.MCP && len(cfg.MCP.Servers) > 0 {
		var servers []mcp.ServerConfig
		for _, s := range cfg.MCP.Servers {
			servers = append(servers, mcpServerFromTOML(s))
		}
		mgr := mcp.NewManager(ctx, servers, logger)
		rt.AttachMCP(mgr)
	}

	// Memory Palace hooks (MCP server and/or dual-write MEMORY_INGEST).
	if cfg.Memory.Enabled {
		rt.AttachMemory(agent.MemoryConfig{
			Enabled:         true,
			Server:          cfg.Memory.Server,
			Tenant:          cfg.Memory.Tenant,
			AutoRecall:      cfg.Memory.AutoRecall,
			AutoIngest:      cfg.Memory.AutoIngest,
			DualWrite:       cfg.Memory.DualWrite,
			Limit:           cfg.Memory.Limit,
			MaxSnippetBytes: cfg.Memory.MaxSnippetBytes,
		})
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
	dirs := skills.DefaultDirs(ws)
	dirs = append(dirs, cfg.Skills.Dirs...)
	cat, err := skills.LoadDirs(dirs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skills: %v\n", err)
		return 1
	}
	if cat.Len() == 0 {
		fmt.Println("no skills found")
		fmt.Fprintf(os.Stderr, "searched: %s\n", strings.Join(dirs, ", "))
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
	if len(cfg.MCP.Servers) == 0 {
		fmt.Println("no MCP servers configured ([mcp] / [[mcp.servers]] in config.toml)")
		return 0
	}
	fmt.Printf("mcp.enabled=%v features.mcp=%v\n", cfg.MCP.Enabled, cfg.Features.MCP)
	fmt.Printf("%-16s %-8s %-8s %-6s %s\n", "NAME", "ENABLED", "MUTATING", "MODE", "ENDPOINT")
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
		fmt.Printf("%-16s %-8v %-8v %-6s %s\n", s.Name, en, mut, mode, strings.TrimSpace(ep))
	}
	if !*connect || !cfg.MCP.Enabled {
		if !*connect {
			fmt.Fprintln(os.Stderr, "(pass --connect to probe tools/list)")
		}
		return 0
	}
	var servers []mcp.ServerConfig
	for _, s := range cfg.MCP.Servers {
		servers = append(servers, mcpServerFromTOML(s))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	mgr := mcp.NewManager(ctx, servers, slog.Default())
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

func mcpServerFromTOML(s config.MCPServerTOML) mcp.ServerConfig {
	sc := mcp.ServerConfig{
		Name: s.Name, Command: s.Command, Args: s.Args, Env: s.Env,
		URL: s.URL, Headers: s.Headers, AllowLoopback: s.AllowLoopback,
		Enabled: s.Enabled, Mutating: s.Mutating,
		StartupTimeoutSec: s.StartupTimeoutSec, ToolTimeoutSec: s.ToolTimeoutSec,
		AccessTokenEnv: s.OAuthTokenEnv,
	}
	if s.OAuth != nil {
		sc.OAuth = &mcp.OAuthConfig{
			TokenURL:        s.OAuth.TokenURL,
			ClientID:        s.OAuth.ClientID,
			ClientSecretEnv: s.OAuth.ClientSecretEnv,
			Scopes:          s.OAuth.Scopes,
			AccessTokenEnv:  s.OAuth.AccessTokenEnv,
			AllowLoopback:   s.OAuth.AllowLoopback,
		}
	}
	return sc
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
		fmt.Fprintln(os.Stderr, "usage: iomesh mesh dogfood|probe|usage|catalog|streams|wait|status [flags]")
		return 2
	}
	switch args[0] {
	case "dogfood", "probe":
		return cmdMeshDogfood(args[1:])
	case "usage":
		return cmdMeshUsage(args[1:])
	case "catalog":
		return cmdMeshCatalog(args[1:])
	case "streams":
		return cmdMeshStreams(args[1:])
	case "wait":
		return cmdMeshWait(args[1:])
	case "status":
		return cmdMeshStatus(args[1:])
	case "help", "-h", "--help":
		fmt.Fprintln(os.Stderr, `iomesh mesh — I/O Mesh platform probes

  iomesh mesh dogfood   stage smoke (health → ready → context → emit → policy → catalog → streams → memory_*)
  iomesh mesh probe     alias for dogfood
  iomesh mesh usage     local LLM metering rollup for this process (--json for scrapers)
  iomesh mesh catalog   list governed data products (broker + portal federation)
  iomesh mesh streams   list/get/delete broker streams (GET|DELETE /v1/streams; explicit errors)
  iomesh mesh wait      poll Ready until OK or timeout (operator preflight)
  iomesh mesh status    operator snapshot (StatusLine + optional Health/Ready)

Flags (dogfood):
  --config path           config.toml
  --strict                require context + emit + ready (+ policy/catalog/memory/streams when on)
  --skip-context          skip context plane
  --skip-emit             skip dept stream emit
  --skip-memory           skip memory_ingest / memory_recall / memory_retrieve
  --skip-streams          skip streams list probe (GET /v1/streams)
  --wait-ready dur        soft WaitReady preflight budget (0=off; timeout SKIP unless --strict)
  --wait-interval dur     WaitReady poll interval (default 500ms when --wait-ready set)
  --wait-require-health   WaitReady requires Health OK each attempt
  --endpoint url          override IOMESH_ENDPOINT
  --memory-endpoint url   memory sidecar base (sync retrieve / warm plane)
  --json                  JSON report for stage CI evidence
  -C dir                  workspace for context query
  -v                      verbose

Flags (catalog):
  --query q         optional search filter
  --endpoint url    override mesh endpoint
  --tenant id       override tenant

Flags (streams):
  --name NAME       get one stream (omit to list all); required with --delete
  --json            JSON array (list) or object (get)
  --delete          delete stream named by --name (requires --name and --yes; DESTRUCTIVE)
  --yes             confirm destructive delete
  --endpoint url    override mesh endpoint
  --config path     config.toml
  --tenant id       override tenant
  -v                verbose

Flags (wait):
  --timeout dur       max wait (default 30s)
  --interval dur      poll interval (default 500ms)
  --require-health    require Health OK each attempt before Ready
  --endpoint url      override IOMESH_ENDPOINT
  --config path       config.toml
  -v                  verbose

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

func cmdMeshWait(args []string) int {
	fs := flag.NewFlagSet("mesh wait", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath    = fs.String("config", "", "config.toml path")
		endpoint      = fs.String("endpoint", "", "override IOMESH_ENDPOINT")
		timeout       = fs.Duration("timeout", 30*time.Second, "max wait duration")
		interval      = fs.Duration("interval", 500*time.Millisecond, "poll interval")
		requireHealth = fs.Bool("require-health", false, "require Health OK each attempt before Ready")
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
	mesh := iomesh.New(iomesh.Config{
		Enabled:     cfg.IOMesh.Enabled,
		Endpoint:    cfg.IOMesh.Endpoint,
		Tenant:      cfg.IOMesh.Tenant,
		APIKeyEnv:   cfg.IOMesh.APIKeyEnv,
		OrgID:       cfg.IOMesh.Org,
		WorkspaceID: cfg.IOMesh.Workspace,
	}, logger)

	parent, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(parent, *timeout)
	defer cancel()

	if err := mesh.WaitReady(ctx, iomesh.WaitReadyOptions{
		Interval:      *interval,
		RequireHealth: *requireHealth,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL mesh wait: %v\n", err)
		return 1
	}
	fmt.Println("PASS mesh wait: ready")
	return 0
}

// cmdMeshStatus prints an operator snapshot: StatusLine fields + one-shot Health/Ready (fail-open).
func cmdMeshStatus(args []string) int {
	fs := flag.NewFlagSet("mesh status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath = fs.String("config", "", "config.toml path")
		endpoint   = fs.String("endpoint", "", "override IOMESH_ENDPOINT")
		jsonOut    = fs.Bool("json", false, "print status as JSON")
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
	}, logger)

	type statusOut struct {
		Enabled    bool   `json:"enabled"`
		Endpoint   string `json:"endpoint,omitempty"`
		Tenant     string `json:"tenant,omitempty"`
		Org        string `json:"org,omitempty"`
		Workspace  string `json:"workspace,omitempty"`
		UserAgent  string `json:"user_agent"`
		StatusLine string `json:"status_line"`
		Health     string `json:"health"` // ok|err|skipped
		HealthErr  string `json:"health_err,omitempty"`
		Ready      string `json:"ready"` // ok|err|skipped
		ReadyErr   string `json:"ready_err,omitempty"`
	}
	out := statusOut{
		Enabled:    mesh.Enabled(),
		Endpoint:   cfg.IOMesh.Endpoint,
		Tenant:     cfg.IOMesh.Tenant,
		Org:        strings.TrimSpace(cfg.IOMesh.Org),
		Workspace:  strings.TrimSpace(cfg.IOMesh.Workspace),
		UserAgent:  iomesh.UserAgent(),
		StatusLine: mesh.StatusLine(),
		Health:     "skipped",
		Ready:      "skipped",
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// One-shot Health/Ready — fail-open display (never exit non-zero for probe errs).
	if mesh.Enabled() {
		if err := mesh.Health(ctx); err != nil {
			out.Health = "err"
			out.HealthErr = err.Error()
		} else {
			out.Health = "ok"
		}
		if err := mesh.Ready(ctx); err != nil {
			out.Ready = "err"
			out.ReadyErr = err.Error()
		} else {
			out.Ready = "ok"
		}
	}

	if *jsonOut {
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "json: %v\n", err)
			return 1
		}
		fmt.Println(string(b))
		return 0
	}

	fmt.Println("iomesh mesh status")
	fmt.Printf("  status_line: %s\n", out.StatusLine)
	fmt.Printf("  endpoint:    %s\n", out.Endpoint)
	if out.Tenant != "" {
		fmt.Printf("  tenant:      %s\n", out.Tenant)
	}
	if out.Org != "" {
		fmt.Printf("  org:         %s\n", out.Org)
	}
	if out.Workspace != "" {
		fmt.Printf("  workspace:   %s\n", out.Workspace)
	}
	fmt.Printf("  user_agent:  %s\n", out.UserAgent)
	if out.HealthErr != "" {
		fmt.Printf("  health:      %s (%s)\n", out.Health, out.HealthErr)
	} else {
		fmt.Printf("  health:      %s\n", out.Health)
	}
	if out.ReadyErr != "" {
		fmt.Printf("  ready:       %s (%s)\n", out.Ready, out.ReadyErr)
	} else {
		fmt.Printf("  ready:       %s\n", out.Ready)
	}
	return 0
}

func cmdMeshCatalog(args []string) int {
	fs := flag.NewFlagSet("mesh catalog", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath = fs.String("config", "", "config.toml path")
		query      = fs.String("query", "", "optional catalog search filter")
		endpoint   = fs.String("endpoint", "", "override IOMESH_ENDPOINT")
		tenant     = fs.String("tenant", "", "override tenant")
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
	res := mesh.ListCatalog(ctx, *query)
	fmt.Print(iomesh.FormatCatalog(res))
	if res.Source == "off" {
		return 1
	}
	return 0
}

// cmdMeshStreams lists, gets, or deletes broker streams via lean /v1/streams (explicit errors; no SDK dep).
// --delete is destructive and requires --name and --yes (s302).
func cmdMeshStreams(args []string) int {
	fs := flag.NewFlagSet("mesh streams", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath = fs.String("config", "", "config.toml path")
		name       = fs.String("name", "", "get/delete one stream by name (omit to list all)")
		endpoint   = fs.String("endpoint", "", "override IOMESH_ENDPOINT")
		tenant     = fs.String("tenant", "", "override tenant")
		jsonOut    = fs.Bool("json", false, "print streams as JSON")
		doDelete   = fs.Bool("delete", false, "delete stream named by --name (requires --name and --yes; DESTRUCTIVE)")
		yes        = fs.Bool("yes", false, "confirm destructive delete")
		verbose    = fs.Bool("v", false, "verbose logs")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	streamName := strings.TrimSpace(*name)
	if *doDelete {
		if streamName == "" || !*yes {
			fmt.Fprintln(os.Stderr, "usage: iomesh mesh streams --delete --name NAME --yes")
			fmt.Fprintln(os.Stderr, "  --delete is destructive; requires --name and --yes")
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

	if *doDelete {
		if err := mesh.DeleteStream(ctx, streamName); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL mesh streams delete: %v\n", err)
			return 1
		}
		fmt.Printf("PASS mesh streams delete name=%s\n", streamName)
		return 0
	}

	if streamName != "" {
		info, err := mesh.GetStream(ctx, streamName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL mesh streams: %v\n", err)
			return 1
		}
		if *jsonOut {
			b, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "json: %v\n", err)
				return 1
			}
			fmt.Println(string(b))
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
		if streams == nil {
			streams = []iomesh.StreamInfo{}
		}
		b, err := json.MarshalIndent(streams, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "json: %v\n", err)
			return 1
		}
		fmt.Println(string(b))
		return 0
	}
	fmt.Print(iomesh.FormatStreams(streams))
	return 0
}

func cmdMeshUsage(args []string) int {
	// Local process meter is empty in a fresh CLI process; still print schema + guidance.
	// When wired as MetricsSink during agent runs, snapshots are in-process only.
	// Remote multi-tenant dashboards consume dept.agent.llm_call on the platform (not this CLI).
	fs := flag.NewFlagSet("mesh usage", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "print usage snapshot as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	mesh := iomesh.New(iomesh.Config{}, nil)
	snap := mesh.Usage()
	if *jsonOut {
		fmt.Print(iomesh.FormatUsageJSON(snap))
	} else {
		fmt.Print(iomesh.FormatUsage(snap))
		fmt.Fprintln(os.Stderr, "note: metering accumulates during agent runs in-process (MetricsSink); CLI `mesh usage` shows the current process only. Use --json for scrapers. Platform remote dashboards use dept.agent.llm_call when mesh is enabled.")
	}
	return 0
}

func cmdMeshDogfood(args []string) int {
	fs := flag.NewFlagSet("mesh dogfood", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		configPath        = fs.String("config", "", "config.toml path")
		workspace         = fs.String("C", "", "workspace for context query")
		strict            = fs.Bool("strict", false, "fail if context/emit/ready/memory soft-fail")
		skipContext       = fs.Bool("skip-context", false, "skip context plane probe")
		skipEmit          = fs.Bool("skip-emit", false, "skip dept emit probe")
		skipMemory        = fs.Bool("skip-memory", false, "skip memory_ingest / memory_recall / memory_retrieve probes")
		skipStreams       = fs.Bool("skip-streams", false, "skip streams list probe (GET /v1/streams)")
		waitReady         = fs.Duration("wait-ready", 0, "soft WaitReady preflight budget before ready (0=off)")
		waitInterval      = fs.Duration("wait-interval", 0, "WaitReady poll interval (default 500ms when --wait-ready set)")
		waitRequireHealth = fs.Bool("wait-require-health", false, "WaitReady requires Health OK each attempt")
		jsonOut           = fs.Bool("json", false, "print dogfood report as JSON (stage CI evidence)")
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
	// Env already applied by config.Load; allow empty endpoint → SKIP report.
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
		WaitReady:         *waitReady,
		WaitReadyInterval: *waitInterval,
		WaitRequireHealth: *waitRequireHealth,
	})
	if *jsonOut {
		fmt.Print(iomesh.FormatReportJSON(rep))
	} else {
		fmt.Print(iomesh.FormatReport(rep))
	}
	if !rep.OK {
		return 1
	}
	return 0
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
  iomesh mesh dogfood            stage I/O Mesh smoke (health/context/emit/memory)
  iomesh mesh wait               poll mesh Ready until OK (operator preflight)
  iomesh mesh status             operator snapshot (StatusLine + Health/Ready)
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

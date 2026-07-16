// Command iomesh is the I/O Mesh TUI coding agent (Go rewrite of Grok Build),
// with DeepSeek-first model routing and optional I/O Mesh platform integration.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/config"
	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/session"
	"github.com/iome-sh/iomesh-tui/internal/tui"
)

const version = "0.1.0-dev"

func main() {
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
		EmitDeptStreams: cfg.IOMesh.EmitDeptStreams,
		ContextPlane:    cfg.IOMesh.ContextPlane,
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
		return 0
	}

	// Interactive TUI (scaffold) with session store for /save and autosave.
	if err := tui.RunWithStore(ctx, rt, store, logger); err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
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

func cmdAgent(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: iomesh agent stdio | serve")
		return 2
	}
	switch args[0] {
	case "stdio":
		fmt.Fprintln(os.Stderr, "ACP stdio mode: scaffold — not yet implemented (see docs/architecture)")
		return 1
	case "serve":
		fmt.Fprintln(os.Stderr, "ACP serve mode: scaffold — not yet implemented")
		return 1
	default:
		fmt.Fprintf(os.Stderr, "unknown agent mode %q\n", args[0])
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
  iomesh [flags]                 interactive TUI
  iomesh -p "prompt"             headless single prompt
  iomesh -c                      continue latest session
  iomesh --session <id>          resume session by id
  iomesh sessions                list sessions in workspace
  iomesh models                  list configured models
  iomesh agent stdio             ACP server (scaffold)
  iomesh version

Flags:
  -p, --prompt string   headless prompt
  -m string             model override (logical name)
  -C string             workspace directory
  -c, --continue        resume latest session
  --session id          resume specific session id
  --no-save             disable session autosave
  --config path         config.toml path
  --yolo                auto-approve mutating tools
  -v                    verbose logs

Default model cascade: deepseek-v4-flash → deepseek-v4-pro → grok-4.5
Config: ~/.iomesh/config.toml  (or $IOMESH_CONFIG)

Environment:
  DEEPSEEK_API_KEY    DeepSeek API key (Flash/Pro)
  XAI_API_KEY         xAI / Grok fallback
  IOMESH_API_KEY      I/O Mesh platform
  IOMESH_ENDPOINT     enable mesh integration
  IOMESH_DEFAULT_MODEL  override default model name
`)
}

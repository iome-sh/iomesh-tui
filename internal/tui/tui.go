// Package tui is the interactive terminal front-end.
//
// Default interactive mode is a full-screen Bubble Tea UI (scrollback, streaming,
// approvals). Classic line REPL remains available via RunREPL / --repl for
// scripts, tests, and non-TTY environments.
package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/session"
)

// Runner is the subset of agent.Runtime needed by the TUI.
type Runner interface {
	RunTurn(ctx context.Context, userText string, onEvent func(agent.Event)) (string, error)
	Router() *router.Router
	Workspace() workspaceRoot
}

type workspaceRoot interface {
	Root() string
}

// runtimeAdapter lets us accept *agent.Runtime without exporting extra interfaces.
type runtimeAdapter struct {
	rt       *agent.Runtime
	store    *session.Store
	readLine func() (string, error)
	out      io.Writer
}

func (a runtimeAdapter) RunTurn(ctx context.Context, userText string, onEvent func(agent.Event)) (string, error) {
	return a.rt.RunTurn(ctx, userText, onEvent)
}
func (a runtimeAdapter) Router() *router.Router { return a.rt.Router() }
func (a runtimeAdapter) Workspace() workspaceRoot {
	return a.rt.Workspace()
}

// Run starts the interactive UI without a session store (full-screen when possible).
func Run(ctx context.Context, rt *agent.Runtime, logger *slog.Logger) error {
	return RunWithStore(ctx, rt, nil, logger)
}

// RunWithStore starts the full-screen TUI with optional session persistence.
// Falls back to the classic line REPL when stdout is not a terminal.
func RunWithStore(ctx context.Context, rt *agent.Runtime, store *session.Store, logger *slog.Logger) error {
	return RunWithStoreOpts(ctx, rt, store, logger, UIOptions{})
}

// RunWithStoreOpts is RunWithStore plus UI theme options.
func RunWithStoreOpts(ctx context.Context, rt *agent.Runtime, store *session.Store, logger *slog.Logger, opts UIOptions) error {
	if !isTerminal(os.Stdout) {
		return runREPL(ctx, rt, store, os.Stdin, os.Stdout, logger)
	}
	return RunFullscreenOpts(ctx, rt, store, logger, opts)
}

// RunREPL forces the classic line-oriented REPL (tests, pipes, --repl).
func RunREPL(ctx context.Context, rt *agent.Runtime, store *session.Store, logger *slog.Logger) error {
	return runREPL(ctx, rt, store, os.Stdin, os.Stdout, logger)
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func runREPL(ctx context.Context, rt *agent.Runtime, store *session.Store, in io.Reader, out io.Writer, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	sc := bufio.NewScanner(in)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 4*1024*1024)

	readLine := func() (string, error) {
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return "", err
			}
			return "", io.EOF
		}
		return sc.Text(), nil
	}

	// Interactive approval for mutating tools (apply_worktree, shell, write, …).
	rt.SetApprover(func(ctx context.Context, tool, args string) (agent.Approval, error) {
		fmt.Fprintf(out, "\n\033[33m⚠ approve tool %s?\033[0m\n  %s\n[y]es / [n]o / [a]lways this session: ",
			tool, truncate(args, 240))
		line, err := readLine()
		if err != nil {
			return agent.ApprovalDeny, err
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "y", "yes":
			return agent.ApprovalOnce, nil
		case "a", "always":
			return agent.ApprovalAlways, nil
		default:
			return agent.ApprovalDeny, nil
		}
	})

	adapter := runtimeAdapter{rt: rt, store: store, readLine: readLine, out: out}

	fmt.Fprintf(out, "iomesh-tui REPL — workspace %s\n", rt.Workspace().Root())
	if sid := rt.SessionID(); sid != "" {
		fmt.Fprintf(out, "session: %s\n", sid)
	}
	fmt.Fprintf(out, "model: %s  |  /model /models /subagents /save /sessions /permissions /cost /mesh /catalog /memory /quit\n", displayModel(rt.Router()))
	fmt.Fprintln(out, "mutating tools (write/shell/apply_worktree/…) prompt for approval unless --yolo")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		fmt.Fprint(out, "iomesh> ")
		line, err := readLine()
		if err == io.EOF {
			fmt.Fprintln(out)
			return nil
		}
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			if quit, err := handleSlash(out, adapter, line); quit {
				return err
			}
			continue
		}

		_, err = adapter.RunTurn(ctx, line, func(ev agent.Event) {
			switch ev.Type {
			case agent.EventModelSelected:
				fmt.Fprintf(out, "\033[2m[model %s]\033[0m\n", ev.Model)
			case agent.EventTextDelta:
				fmt.Fprint(out, ev.Text)
			case agent.EventThinkingDelta:
				fmt.Fprintf(out, "\033[2m%s\033[0m", ev.Text)
			case agent.EventToolStart:
				fmt.Fprintf(out, "\n\033[33m→ tool %s\033[0m\n", ev.Tool)
			case agent.EventToolEnd:
				fmt.Fprintf(out, "\033[32m✓ %s\033[0m %s\n", ev.Tool, truncate(ev.Text, 120))
			case agent.EventToolError, agent.EventToolDenied:
				fmt.Fprintf(out, "\033[31m✗ %s\033[0m %s\n", ev.Tool, ev.Text)
			case agent.EventLLMDone:
				fmt.Fprintf(out, "\n\033[2m— %s · %d tokens · $%.5f · %s\033[0m\n",
					ev.Model, ev.Tokens, ev.CostUSD, ev.Duration)
			case agent.EventMeshContext:
				fmt.Fprintf(out, "\033[36m[iomesh] %s\033[0m\n", ev.Text)
			}
		})
		if err != nil {
			fmt.Fprintf(out, "\nerror: %v\n", err)
		} else if store != nil {
			rt.AutoSaveAfterTurn(store)
			if id := rt.SessionID(); id != "" {
				fmt.Fprintf(out, "\033[2m[session %s saved]\033[0m\n", id)
			}
		}
		fmt.Fprintln(out)
	}
}

func displayModel(r *router.Router) string {
	if ov := r.Override(); ov != "" {
		return ov + " (pinned)"
	}
	return r.DefaultModel()
}

func handleSlash(out io.Writer, rt runtimeAdapter, line string) (quit bool, err error) {
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "/quit", "/exit", "/q":
		return true, nil
	case "/models":
		printModelPicker(out, rt.Router())
	case "/model", "/m":
		if len(parts) < 2 {
			printModelPicker(out, rt.Router())
			fmt.Fprintln(out, "usage: /model <name|number> | default")
			return false, nil
		}
		name := parts[1]
		if name == "default" || name == "auto" {
			_ = rt.Router().SetOverride("")
			fmt.Fprintln(out, "model override cleared (auto cascade)")
			return false, nil
		}
		// Numeric picker
		if n, err := strconv.Atoi(name); err == nil {
			models := rt.Router().Models()
			if n < 1 || n > len(models) {
				fmt.Fprintf(out, "error: pick 1..%d\n", len(models))
				return false, nil
			}
			name = models[n-1].Name
		}
		if err := rt.Router().SetOverride(name); err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return false, nil
		}
		fmt.Fprintf(out, "model override → %s\n", name)
	case "/cost":
		name := rt.Router().Override()
		if name == "" {
			name = rt.Router().DefaultModel()
		}
		// Session / process actuals (via mesh MetricsSink when enabled).
		snap := rt.rt.MeshUsage()
		if snap.Calls > 0 {
			fmt.Fprint(out, iomesh.FormatUsage(snap))
		} else {
			fmt.Fprintln(out, "session usage: (no LLM calls recorded yet in this process)")
		}
		est := rt.Router().EstimateCostTokens(name, 100_000, 4_000, 0)
		fmt.Fprintf(out, "sample estimate for %s @ 100k in / 4k out: $%.5f\n", name, est.USD)
		estHit := rt.Router().EstimateCostTokens(name, 100_000, 4_000, 80_000)
		fmt.Fprintf(out, "sample with 80%% cache hit: $%.5f\n", estHit.USD)
	case "/mesh":
		m := rt.rt.Mesh()
		if m == nil {
			fmt.Fprintln(out, "mesh: no client")
			return false, nil
		}
		fmt.Fprintln(out, m.StatusLine())
		snap := m.Usage()
		if snap.Calls > 0 {
			fmt.Fprint(out, iomesh.FormatUsage(snap))
		}
	case "/catalog":
		m := rt.rt.Mesh()
		if m == nil || !m.CatalogEnabled() {
			fmt.Fprintln(out, "catalog: mesh catalog plane disabled (enable [iomesh] catalog_plane + endpoint)")
			return false, nil
		}
		q := ""
		if len(parts) > 1 {
			q = strings.Join(parts[1:], " ")
		}
		res := m.ListCatalog(context.Background(), q)
		fmt.Fprint(out, iomesh.FormatCatalog(res))
	case "/memory", "/mem":
		if len(parts) < 2 {
			fmt.Fprintln(out, rt.rt.MemoryStatusLine())
			fmt.Fprintln(out, "usage: /memory [recall [query] | ingest <text>]")
			return false, nil
		}
		sub := strings.ToLower(parts[1])
		switch sub {
		case "status", "st":
			fmt.Fprintln(out, rt.rt.MemoryStatusLine())
		case "recall", "r":
			q := strings.Join(parts[2:], " ")
			if strings.TrimSpace(q) == "" {
				q = "*"
			}
			text, err := rt.rt.MemoryRecall(context.Background(), q)
			if err != nil {
				fmt.Fprintf(out, "memory recall: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(no memories)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "ingest", "i":
			content := strings.Join(parts[2:], " ")
			if strings.TrimSpace(content) == "" {
				fmt.Fprintln(out, "usage: /memory ingest <text>")
				return false, nil
			}
			text, err := rt.rt.MemoryIngestTurn(context.Background(), "user", content)
			if err != nil {
				fmt.Fprintf(out, "memory ingest: %v\n", err)
				return false, nil
			}
			fmt.Fprintln(out, text)
		default:
			// Treat remainder as recall query: /memory what did we decide
			q := strings.Join(parts[1:], " ")
			text, err := rt.rt.MemoryRecall(context.Background(), q)
			if err != nil {
				fmt.Fprintf(out, "memory: %v (try /memory status|recall|ingest)\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(no memories)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		}
	case "/subagents", "/sa":
		mgr := rt.rt.Subagents()
		if mgr == nil || !mgr.Enabled() {
			fmt.Fprintln(out, "subagents disabled")
			return false, nil
		}
		list := mgr.Registry().List()
		if len(list) == 0 {
			fmt.Fprintln(out, "no subagents spawned this session")
			return false, nil
		}
		fmt.Fprintf(out, "%-22s %-12s %-16s %-8s %s\n", "ID", "STATUS", "TYPE", "WT", "DESC")
		for _, rec := range list {
			wt := "-"
			if rec.WorktreePath != "" {
				wt = "yes"
			}
			fmt.Fprintf(out, "%-22s %-12s %-16s %-8s %s\n",
				rec.ID, rec.Status, rec.Spec.SubagentType, wt, rec.Spec.Description)
		}
	case "/permissions", "/perms":
		fmt.Fprintln(out, "session always-allow tools:")
		// No export of map keys without iteration API — probe known mutators.
		for _, name := range []string{
			"write_file", "run_shell", "apply_worktree", "apply_worktrees", "remove_worktree",
		} {
			if rt.rt.ToolAllowedSession(name) {
				fmt.Fprintf(out, "  ✓ %s\n", name)
			}
		}
		fmt.Fprintln(out, "(use approval prompt [a]lways, or --yolo)")
	case "/save":
		if rt.store == nil {
			fmt.Fprintln(out, "session store unavailable")
			return false, nil
		}
		compact := 0
		if len(parts) > 1 && parts[1] == "compact" {
			compact = 8
		}
		snap, err := rt.rt.SaveSession(rt.store, compact)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return false, nil
		}
		fmt.Fprintf(out, "saved session %s (%d messages, %d subagents)\n", snap.ID, len(snap.Messages), len(snap.Subagents))
	case "/sessions":
		if rt.store == nil {
			fmt.Fprintln(out, "session store unavailable")
			return false, nil
		}
		list, err := rt.store.List()
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return false, nil
		}
		if len(list) == 0 {
			fmt.Fprintln(out, "no sessions")
			return false, nil
		}
		cur := rt.rt.SessionID()
		for _, s := range list {
			mark := " "
			if s.ID == cur {
				mark = "*"
			}
			fmt.Fprintf(out, "%s %s  msgs=%d subs=%d  %s  %s\n",
				mark, s.ID, s.Messages, s.Subagents, s.UpdatedAt.Format(time.RFC3339), s.Title)
		}
	case "/load":
		if rt.store == nil || len(parts) < 2 {
			fmt.Fprintln(out, "usage: /load <session-id>")
			return false, nil
		}
		snap, err := rt.store.Load(parts[1])
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return false, nil
		}
		if err := rt.rt.LoadSession(snap); err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return false, nil
		}
		fmt.Fprintf(out, "loaded %s (%d messages, %d subagents)\n", snap.ID, len(snap.Messages), len(snap.Subagents))
	case "/theme", "/themes":
		if len(parts) < 2 {
			fmt.Fprintf(out, "themes: %s\nusage: /theme <name>  (fullscreen only applies live)\n", strings.Join(ThemeNames(), ", "))
			return false, nil
		}
		th, err := ParseTheme(parts[1])
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return false, nil
		}
		fmt.Fprintf(out, "theme %s (apply in fullscreen TUI with /theme)\n", th.Name)
	case "/help", "/?":
		fmt.Fprint(out, `commands:
  /models              list models (numbered)
  /model <name|#>      pin model (or default)
  /theme [name]        list or set UI theme (default|mono|high-contrast|dim)
  /subagents           list subagents (id, status, worktree)
  /permissions         show session always-allow tools
  /save [compact]      save session
  /sessions            list saved sessions
  /load <id>           restore session
  /cost                session usage meter + sample estimate
  /mesh                I/O Mesh status + usage
  /catalog [query]     list mesh data products (catalog plane)
  /memory [recall|ingest|status]  Memory Palace (sync HTTP + MCP; see memory-mcp.md)
  /quit                exit

Fullscreen keys: enter send · ctrl+j newline · pgup/pgdn scroll
On mutating tools (write_file, run_shell, apply_worktree, …) you will be prompted:
  [y]es  [n]o  [a]lways this session
`)
	default:
		fmt.Fprintf(out, "unknown command %s (try /help)\n", cmd)
	}
	return false, nil
}

func printModelPicker(out io.Writer, r *router.Router) {
	models := r.Models()
	cur := r.Override()
	if cur == "" {
		cur = r.DefaultModel()
	}
	fmt.Fprintf(out, "%3s  %-22s %-28s %s\n", "#", "NAME", "MODEL_ID", "NOTES")
	for i, m := range models {
		note := ""
		if m.Name == r.DefaultModel() {
			note = "default"
		}
		if m.Name == cur && r.Override() != "" {
			note = "pinned"
		}
		fmt.Fprintf(out, "%3d  %-22s %-28s %s\n", i+1, m.Name, m.ModelID, note)
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

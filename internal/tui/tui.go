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
	"github.com/iome-sh/iomesh-tui/internal/config"
	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/runtimewire"
	"github.com/iome-sh/iomesh-tui/internal/session"
	"github.com/iome-sh/iomesh-tui/internal/setup"
	"github.com/iome-sh/iomesh-tui/internal/skills"
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

// processConfigPath is the config file the running process loaded (--config /
// IOMESH_CONFIG / user default). Empty when the runtime was not given a path.
func (a runtimeAdapter) processConfigPath() string {
	if a.rt == nil {
		return ""
	}
	return a.rt.ConfigPath()
}

// slashConfigPath returns explicit slash --config (empty if omitted).
func slashConfigPath(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--config" && i+1 < len(args) {
			return strings.TrimSpace(args[i+1])
		}
		if strings.HasPrefix(a, "--config=") {
			return strings.TrimSpace(strings.TrimPrefix(a, "--config="))
		}
	}
	return ""
}

// setupProbeConfigPath prefers slash --config, then the process config path.
// Empty means callers should LoadUser() (IOMESH_CONFIG / user default).
func setupProbeConfigPath(args []string, processPath string) string {
	if p := slashConfigPath(args); p != "" {
		return p
	}
	return strings.TrimSpace(processPath)
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
	fmt.Fprintf(out, "model: %s  |  /model /models /subagents /save /sessions /permissions /cost /mesh /catalog /memory /integrations /quit\n", displayModel(rt.Router()))
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
	case "/dashboard", "/heartbeat", "/mesh-console":
		// s1989: landing-page MeshConsole heartbeat live-feed analysis (eval template).
		handleDashboardSlash(out, rt, parts)
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
			fmt.Fprintln(out, "usage: /memory [recall [--since|--until|--session-seq] [query] | related --seed <entity> [--query ...] [--max-hops N] [--prefer-shorter-hops|--legacy-sort] | digest [--window day|week] [--horizon ops|knowledge|analytical|all] [--limit N] [--require-sources mesh,private] | facts-as-of --as-of <RFC3339> [--entity ...] [--query ...] [--limit N] | timeline [--since|--until|--session-id|--query|--limit] | compact-status | trigger-compact --i-confirm | semantic [query|--query ...] [--limit N] | ingest-event --subject <id> --content <text> [--event-time|--session-id|--session-seq|--severity|--source-stream] | patterns [--limit N] | anomalies [--limit N] | supersede --entity <key> [--as-of RFC3339] --i-confirm | ingest <text> | ingest-dir <path> [--dry-run] [--limit N] | status]")
			// s1831: residual-honest dual-path next-step after bare /memory help.
			for _, line := range agent.MemoryNextStepLines() {
				fmt.Fprintln(out, line)
			}
			return false, nil
		}
		sub := strings.ToLower(parts[1])
		switch sub {
		case "status", "st":
			// s1311: base MemoryStatusLine + residual-honest advanced MCP inventory pulse.
			// s1831 next-step is appended inside MemoryAdvancedStatus.
			fmt.Fprintln(out, rt.rt.MemoryStatusLine())
			adv, aerr := rt.rt.MemoryAdvancedStatus(context.Background())
			if aerr != nil {
				fmt.Fprintf(out, "memory advanced status: %v\n", aerr)
				return false, nil
			}
			if strings.TrimSpace(adv) != "" {
				fmt.Fprintln(out, adv)
			}
		case "recall", "r":
			q, ropts := parseMemoryRecallArgs(parts[2:])
			if strings.TrimSpace(q) == "" {
				q = "*"
			}
			text, err := rt.rt.MemoryRecallWithOpts(context.Background(), q, ropts)
			if err != nil {
				fmt.Fprintf(out, "memory recall: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(no memories)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "related", "rel":
			// s1135: opt-in multi-hop lite related recall (HTTP + MCP fallback).
			// Does not change default auto-recall. Not full graph RAG; not Memory GA.
			// s1281: hop ranking path-aware lite — PreferShorterHops default true (nil = true).
			seed, q, ropts, perr := parseMemoryRelatedArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory related: %s\nusage: /memory related --seed person:alice [--query ...] [--max-hops 2] [--prefer-shorter-hops|--legacy-sort]\n", perr)
				return false, nil
			}
			if seed == "" && q == "" {
				fmt.Fprintln(out, "usage: /memory related --seed person:alice [--query ...] [--max-hops 2] [--prefer-shorter-hops|--legacy-sort]\n  multi-hop lite related recall (opt-in; not auto-recall; not full graph RAG; hop ranking path-aware lite)")
				return false, nil
			}
			text, err := rt.rt.MemoryRelated(context.Background(), seed, q, ropts)
			if err != nil {
				fmt.Fprintf(out, "memory related: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(no related memories)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "digest", "dig":
			// s1200: opt-in ops heartbeat digest export (HTTP + MCP ops_digest_export fallback).
			// ops GA-path · knowledge/analytical Beta · never invent GA · dual_write OFF · not Memory GA.
			// #373: --require-sources mesh,private cites both or explicit miss (catalog/grant ≠ cite-both).
			// s1831: residual-honest dual-path next-step after digest (primary honesty surface).
			dopts, perr := parseMemoryDigestArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory digest: %s\nusage: /memory digest [--window day|week] [--horizon ops|knowledge|analytical|all] [--limit N] [--require-sources mesh,private]\n", perr)
				return false, nil
			}
			text, err := rt.rt.MemoryOpsDigest(context.Background(), dopts)
			if err != nil {
				fmt.Fprintf(out, "memory digest: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(empty digest)")
				for _, line := range agent.MemoryNextStepLines() {
					fmt.Fprintln(out, line)
				}
				return false, nil
			}
			fmt.Fprintln(out, text)
			for _, line := range agent.MemoryNextStepLines() {
				fmt.Fprintln(out, line)
			}
		case "facts-as-of", "facts", "as-of", "asof":
			// s1276: opt-in bi-temporal lite validity listing via MCP memory_facts_as_of
			// (aion Beta K4 lite). MCP-first — no lean HTTP facts_as_of route today.
			// Not auto-recall · not full dual-clock Graphiti · not Memory GA · dual_write OFF.
			fopts, perr := parseMemoryFactsAsOfArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory facts-as-of: %s\nusage: /memory facts-as-of --as-of <RFC3339> [--entity ...] [--query ...] [--limit N]\n", perr)
				return false, nil
			}
			if strings.TrimSpace(fopts.AsOf) == "" {
				fmt.Fprintln(out, "usage: /memory facts-as-of --as-of <RFC3339> [--entity ...] [--query ...] [--limit N]\n  bi-temporal lite validity listing (opt-in; not auto-recall; not full dual-clock Graphiti; not Memory GA)")
				return false, nil
			}
			text, err := rt.rt.MemoryFactsAsOf(context.Background(), fopts)
			if err != nil {
				fmt.Fprintf(out, "memory facts-as-of: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				// Residual-honest empty — formatter normally always emits honesty footer.
				fmt.Fprintln(out, "(no facts at as_of · empty ≠ invent memories)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "timeline", "tl":
			// s1296: opt-in temporal timeline via MCP memory_timeline.
			// MCP-first — no lean HTTP timeline invent. Filters before limit.
			// Not auto-recall · not Memory GA · dual_write OFF.
			// Empty ≠ invent memories; offline fail-open residual-honest.
			// Mutating compact: /memory trigger-compact --i-confirm (s1311 HITL).
			topts, perr := parseMemoryTimelineArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory timeline: %s\nusage: /memory timeline [--since RFC3339] [--until RFC3339] [--session-id id] [--query ...] [--limit N]\n", perr)
				return false, nil
			}
			text, err := rt.rt.MemoryTimeline(context.Background(), topts)
			if err != nil {
				fmt.Fprintf(out, "memory timeline: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				// Residual-honest empty — formatter normally always emits honesty footer.
				fmt.Fprintln(out, "(no timeline entries · empty ≠ invent memories · temporal timeline)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "compact-status", "compact", "compactstatus", "cstatus":
			// s1296: opt-in read-only Palace tier counts via MCP memory_compact_status.
			// MCP-first — no lean HTTP invent. Not auto-compact product · not Memory GA.
			// dual_write OFF. Offline fail-open residual-honest (empty ≠ invent green).
			// Mutating compact: /memory trigger-compact --i-confirm (s1311 HITL).
			text, err := rt.rt.MemoryCompactStatus(context.Background(), agent.MemoryCompactStatusOpts{})
			if err != nil {
				fmt.Fprintf(out, "memory compact-status: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(compact-status empty · empty ≠ invent compaction green)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "trigger-compact", "compact-trigger", "tcompact":
			// s1311: opt-in HITL RecMem compaction advisory via MCP memory_trigger_compact.
			// Mutating: publishes memory.compact.trigger for RecMem worker. MCP-first only.
			// Require --i-confirm (HITL) — MemoryTriggerCompact refuses residual-honestly without it.
			// Not invent compaction green · not Memory GA · dual_write OFF · not auto from compact-status.
			topts, perr := parseMemoryTriggerCompactArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory trigger-compact: %s\nusage: /memory trigger-compact --i-confirm\n", perr)
				return false, nil
			}
			// Missing --i-confirm → Confirm=false; MemoryTriggerCompact refuses residual-honestly (no MCP).
			text, err := rt.rt.MemoryTriggerCompact(context.Background(), topts)
			if err != nil {
				fmt.Fprintf(out, "memory trigger-compact: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(trigger-compact empty · not inventing triggered/cluster_size)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "semantic", "search-semantic", "sem":
			// s1301: opt-in tier-4 semantic facts via MCP memory_search_semantic.
			// MCP-first — no lean HTTP invent. Not Memory GA · dual_write OFF.
			// Empty ≠ invent memories; offline fail-open residual-honest.
			sopts, perr := parseMemorySemanticArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory semantic: %s\nusage: /memory semantic [--query ...] [--limit N] [query words…]\n", perr)
				return false, nil
			}
			if strings.TrimSpace(sopts.Query) == "" {
				fmt.Fprintln(out, "usage: /memory semantic [--query ...] [--limit N] [query words…]\n  tier-4 semantic facts residual (opt-in; not auto-recall; not Memory GA; dual_write OFF)")
				return false, nil
			}
			text, err := rt.rt.MemorySearchSemantic(context.Background(), sopts)
			if err != nil {
				fmt.Fprintf(out, "memory semantic: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(no semantic facts · empty ≠ invent memories · tier-4 residual)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "ingest-event", "event", "ingest_event":
			// s1301: opt-in ops/telemetry event ingest via MCP memory_ingest_event (s138 T1).
			// MCP-first — not a conversation turn (use /memory ingest for turns).
			// Not Memory GA · dual_write OFF. Offline fail-open residual-honest (never invent memory_id).
			eopts, perr := parseMemoryIngestEventArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory ingest-event: %s\nusage: /memory ingest-event --subject <id> --content <text> [--event-time RFC3339] [--session-id id] [--session-seq N] [--severity info|warning|error|critical] [--source-stream name]\n", perr)
				return false, nil
			}
			if strings.TrimSpace(eopts.Subject) == "" || strings.TrimSpace(eopts.Content) == "" {
				fmt.Fprintln(out, "usage: /memory ingest-event --subject <id> --content <text> [--event-time RFC3339] [--session-id id] [--session-seq N] [--severity info|warning|error|critical] [--source-stream name]\n  s138 T1 temporal event telemetry (opt-in; not conversation turn; not Memory GA; dual_write OFF)")
				return false, nil
			}
			text, err := rt.rt.MemoryIngestEvent(context.Background(), eopts)
			if err != nil {
				fmt.Fprintf(out, "memory ingest-event: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(ingest-event empty · never invent memory_id · s138 T1)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "patterns", "pattern", "pat":
			// s1287: opt-in ops-pulse pattern listing via MCP memory_patterns_list
			// (aion s138 T2 · s789 Beta). MCP-first — no lean HTTP patterns invent.
			// Suggestive ops pulse only · not medical diagnosis · not OTel host metrics ·
			// not invent GA window engine · dual_write OFF · not Memory GA · book-demo OFF.
			// Empty ≠ invent patterns; offline fail-open residual-honest.
			popts, perr := parseMemoryPatternsArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory patterns: %s\nusage: /memory patterns [--limit N]\n", perr)
				return false, nil
			}
			text, err := rt.rt.MemoryPatterns(context.Background(), popts)
			if err != nil {
				fmt.Fprintf(out, "memory patterns: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				// Residual-honest empty — formatter normally always emits honesty footer.
				fmt.Fprintln(out, "(no patterns · empty ≠ invent patterns · ops pulse Beta)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "anomalies", "anomaly", "anom":
			// s1287: opt-in ops-pulse anomaly listing via MCP memory_anomalies_list
			// (aion s138 T2 · s789 Beta). MCP-first — no lean HTTP anomalies invent.
			// Suggestive ops pulse only · not medical diagnosis · not OTel host metrics ·
			// not invent GA window engine · dual_write OFF · not Memory GA · book-demo OFF.
			// Empty ≠ invent anomalies; offline fail-open residual-honest.
			aopts, perr := parseMemoryAnomaliesArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory anomalies: %s\nusage: /memory anomalies [--limit N]\n", perr)
				return false, nil
			}
			text, err := rt.rt.MemoryAnomalies(context.Background(), aopts)
			if err != nil {
				fmt.Fprintf(out, "memory anomalies: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				// Residual-honest empty — formatter normally always emits honesty footer.
				fmt.Fprintln(out, "(no anomalies · empty ≠ invent anomalies · ops pulse Beta)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "supersede", "super":
			// s1282: opt-in HITL A3 lite entity supersession via MCP memory_supersede_entity
			// (aion s640). Mutating: closes open valid_until windows. MCP-first only.
			// Require --i-confirm (HITL) — MemorySupersede refuses residual-honestly without it.
			// Not NLP contradiction · not full dual-clock Graphiti · not Memory GA · dual_write OFF.
			sopts, perr := parseMemorySupersedeArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory supersede: %s\nusage: /memory supersede --entity <key> [--as-of RFC3339] --i-confirm\n", perr)
				return false, nil
			}
			if strings.TrimSpace(sopts.Entity) == "" {
				fmt.Fprintln(out, "usage: /memory supersede --entity <key> [--as-of RFC3339] --i-confirm\n  A3 lite supersede (HITL mutating; closes valid_until; not NLP contradiction; not Memory GA)")
				return false, nil
			}
			text, err := rt.rt.MemorySupersede(context.Background(), sopts)
			if err != nil {
				fmt.Fprintf(out, "memory supersede: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				// Residual-honest empty — formatter normally always emits honesty footer.
				fmt.Fprintln(out, "(supersede empty · not inventing superseded_count)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "ingest-dir", "ingestdir", "ingest_dir", "idir":
			// #384: folder ingest into private overlay. session_id minted as
			// local-overlay when the operator has none. dual_write OFF.
			// Catalog list ≠ consume. Not hosted Memory GA.
			dopts, perr := parseMemoryIngestDirArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory ingest-dir: %s\nusage: /memory ingest-dir <path> [--dry-run] [--limit N]\n", perr)
				return false, nil
			}
			if strings.TrimSpace(dopts.Path) == "" {
				fmt.Fprintln(out, "usage: /memory ingest-dir <path> [--dry-run] [--limit N]\n  folder ingest into private overlay (session_id minted as local-overlay when the walk has none; dual_write OFF; catalog list ≠ consume)")
				return false, nil
			}
			text, err := rt.rt.MemoryIngestDir(context.Background(), dopts)
			if err != nil {
				fmt.Fprintf(out, "memory ingest-dir: %v\n", err)
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "ingest", "i":
			content := strings.Join(parts[2:], " ")
			if strings.TrimSpace(content) == "" {
				fmt.Fprintln(out, "usage: /memory ingest <text>\n  session_id is minted as local-overlay when the operator has none (iomesh-memory-mcp requires it). Folder: /memory ingest-dir <path>")
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
			// (also accepts --since/--until when first token is not a known subcommand)
			q, ropts := parseMemoryRecallArgs(parts[1:])
			text, err := rt.rt.MemoryRecallWithOpts(context.Background(), q, ropts)
			if err != nil {
				fmt.Fprintf(out, "memory: %v (try /memory status|recall|related|digest|facts-as-of|timeline|compact-status|trigger-compact|semantic|ingest-event|patterns|anomalies|supersede|ingest|ingest-dir)\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(no memories)")
				return false, nil
			}
			fmt.Fprintln(out, text)
		}
	case "/setup", "/setup-lifecycle":
		// s1526 P3–P4 + s1530 P5 + s1534 P6 + s1538 P7: agent-native setup lifecycle slash
		// (init/preflight/portal/reload/pull/analyze/drift/repair). Residual honesty: dual_write OFF ·
		// not Memory GA · catalog ≠ Connected · portal HITL · setup PASS ≠ invent
		// install green · continuous pull/analyze opt-in · CLI iomesh memory pull still valid ·
		// /memory digest still valid · drift report-only ≠ invent install green ·
		// repair plan/apply safe steps only · repair apply ≠ invent Connected · no auto-repair without apply --yes.
		// P4: /setup reload → runtimewire.ConnectMCP + Runtime.ReplaceMCP (package wire ≠ Connected).
		// s1670: /setup reload also Wire + LoadWithBuiltin → Runtime.ReplaceSkills (skills re-scan).
		// P5: /setup pull → Runtime continuous memory pull (opt-in · pull ≠ invent Connected).
		// P6: /setup analyze → Runtime analyze ticks; /setup drift|maintain → report-only drift.
		// P7: /setup repair → PlanRepair / ApplyRepairPlan (safe steps · explicit --yes).
		if len(parts) < 2 {
			fmt.Fprintln(out, setupHelp())
			return false, nil
		}
		sub := strings.ToLower(parts[1])
		switch sub {
		case "help", "?":
			fmt.Fprintln(out, setupHelp())
		case "init":
			handleSetupInit(out, parts[2:])
		case "preflight", "status", "check", "st":
			handleSetupPreflight(out, rt, parts[2:])
		case "portal", "hitl", "urls":
			fmt.Fprintln(out, setup.SetupLifecyclePortalHandoff())
			for _, line := range setup.SetupPortalNextStepLines() {
				fmt.Fprintln(out, line)
			}
		case "reload", "reattach", "hot-reload":
			handleSetupReload(out, rt, parts[2:])
		case "pull":
			handleSetupPull(out, rt, parts[2:])
		case "analyze", "tick", "ticks":
			handleSetupAnalyze(out, rt, parts[2:])
		case "drift", "maintain", "maintenance":
			handleSetupDrift(out, rt, parts[2:])
		case "repair", "fix":
			handleSetupRepair(out, rt, parts[2:])
		default:
			fmt.Fprintf(out, "setup: unknown subcommand %q\n%s\n", parts[1], setupHelp())
		}
	case "/integrations", "/integration", "/connectors":
		// s1238/s1242/s1243/s1247: agent/TUI path for connector integrations setup via MCP tools
		// list_connector_catalog / plan_connector_setup (aion v178) · get_webhook_signing_headers (v30).
		// Residual honesty: browser HITL OAuth · stub ≠ live · dual_write OFF ·
		// no invent GA · catalog Beta honesty · fail-open when MCP unavailable ·
		// never invent install green · signing = discovery only (no secret mint).
		// status (s1247) = residual-honest operator pulse (MCP path · tools · catalog honesty).
		// Not full install CRUD.
		if len(parts) < 2 {
			fmt.Fprintln(out, integrationsHelp())
			return false, nil
		}
		sub := strings.ToLower(parts[1])
		switch sub {
		case "help", "?":
			fmt.Fprintln(out, integrationsHelp())
		case "status", "st":
			// s1247: residual-honest operator pulse (not pure help text).
			text, err := rt.rt.IntegrationsStatus(context.Background())
			if err != nil {
				fmt.Fprintf(out, "integrations status: %v\n", err)
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "list", "ls", "catalog":
			layer, perr := parseIntegrationsListArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "integrations list: %s\nusage: /integrations list [--layer operational|knowledge|analytical]\n", perr)
				return false, nil
			}
			text, err := rt.rt.IntegrationsCatalog(context.Background(), layer)
			if err != nil {
				fmt.Fprintf(out, "integrations list: %v\n", err)
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "plan":
			id, perr := parseIntegrationsPlanArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "integrations plan: %s\nusage: /integrations plan <connector_id>\n", perr)
				return false, nil
			}
			if id == "" {
				fmt.Fprintln(out, "usage: /integrations plan <connector_id>\n  residual-honest setup plan via MCP plan_connector_setup (portal HITL; not install green)")
				return false, nil
			}
			text, err := rt.rt.IntegrationsPlan(context.Background(), id)
			if err != nil {
				fmt.Fprintf(out, "integrations plan: %v\n", err)
				return false, nil
			}
			fmt.Fprintln(out, text)
		case "signing", "headers", "signing-headers", "signing_headers":
			// s1243: discovery-only webhook signing header parity (MCP get_webhook_signing_headers).
			hint, perr := parseIntegrationsSigningArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "integrations signing: %s\nusage: /integrations signing [operational|knowledge|analytical|<connector_id>]\n", perr)
				return false, nil
			}
			text, err := rt.rt.IntegrationsSigning(context.Background(), hint)
			if err != nil {
				fmt.Fprintf(out, "integrations signing: %v\n", err)
				return false, nil
			}
			fmt.Fprintln(out, text)
		default:
			fmt.Fprintf(out, "integrations: unknown subcommand %q\n%s\n", parts[1], integrationsHelp())
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
	case "/gtm", "/gtm-draft", "/gtm-agent":
		// s1352: residual-honest GTM draft-only guidance (no auto-send agency).
		// s1358: /gtm help|checklist → numbered draft-only checklist (no auto-send).
		// Bare /gtm (and aliases) keep guidance note + residual footer.
		// Drafts only · human publish · skill gtm-draft-only-agent via read_skill ·
		// dual_write OFF · not Memory GA · book-demo OFF · not invent suite ops GA.
		if len(parts) >= 2 {
			sub := strings.ToLower(parts[1])
			switch sub {
			case "help", "checklist", "?":
				fmt.Fprintln(out, agent.GtmDraftChecklist())
				return false, nil
			case "brief", "voc-brief", "voc_brief", "market-telling", "market_telling":
				// #372: palace market_telling / voc_brief (source=agent-brief, tenant gtm/founder).
				handleGtmBriefSlash(out, parts[2:])
				return false, nil
			}
			// Unknown subcommand: still print guidance + usage hint for help/checklist.
			fmt.Fprintln(out, agent.GtmDraftOnlyAgentGuidanceNote())
			fmt.Fprintln(out, "— residual: drafts only · human publish · skill gtm-draft-only-agent via read_skill · dual_write OFF · not Memory GA")
			fmt.Fprintln(out, "usage: /gtm [help|checklist|brief]  (aliases /gtm-draft /gtm-agent)")
			return false, nil
		}
		fmt.Fprintln(out, agent.GtmDraftOnlyAgentGuidanceNote())
		fmt.Fprintln(out, "— residual: drafts only · human publish · skill gtm-draft-only-agent via read_skill · dual_write OFF · not Memory GA")
	case "/onboard", "/aion-onboard", "/agent-onboard":
		// s1363+s1368+s1372+s1377+s1382+s1387+s1402+s1417: residual-honest TUI agent ↔ aion CP/MCP onboarding guidance.
		// Bare /onboard (and aliases) → guidance note + residual footer.
		// help|checklist|? → numbered onboarding checklist.
		// portal|agent-mcp|mcp → portal Agent/MCP handoff (mint/copy/probe + TUI [[mcp.servers]]).
		// status → residual-honest offline static status (no MCP dial).
		// next|after|continue|lanes → post-onboard operator lanes overview (plugins·gtm·memory·mesh·memory-pull·agentic·planes·sales·demo·operator·setup·human-gates).
		// next <lane> (s1377): plugins|plugin|dogfood · gtm|drafts · memory|mcp|palace.
		// next mesh (s1402): mesh|stream|streams|heartbeat|heartbeats|pull (NOT pulse — pulse stays status board).
		// next memory-pull (s1407): memory-pull|ops-pack|pull-path|memorypull|ops_pack (NOT bare pull — pull stays mesh).
		// next agentic (s1417): agentic|agentic-integrations|integrations|list-plan (NOT bare mcp/portal/pull · NOT portal-hitl|hitl — s1562).
		// next portal-hitl (s1562): portal-hitl|hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl (journey stage 5 · soft dogfood residual).
		// next e4 (s1566): e4|e4-dogfood|client-attach|edge-memory-e4|e4_attach (journey stage 6 · E4 client-attach soft dogfood residual).
		// next tool-call (s1578): tool-call|tool-calls|deeper-e4|e4-tools|ingest-retrieve|tool_call (deeper tool-call residual after E4 attach).
		// next e10 (s1586): e10|e10-open|edge-memory-e10|ga-signoff|e10_open (E10 Open reaffirm residual-check after OSS packaging).
		// next marketing-demo (s1590): marketing-demo|marketing|sales-demo|demo-script|gtm-demo (plain-language local agent + memory demo path).
		// next planes (s1432): planes|three-planes|product-planes|product|pillars|three_planes (NOT pulse/board · pull · mcp).
		// next sales (s1437): sales|claims|buyer|claim-matrix|sales-claims|buyer-claims (NOT product/planes · gtm · pulse/board · sales-demo).
		// next operator (s1447): operator|operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix (NOT demo/readiness/lighthouse/landgrab · sales/claims · planes/product · pulse/board · export/receipt).
		// next setup (s1542): setup|setup-lifecycle|lifecycle|setup_lifecycle (setup lifecycle P1–P7 closeout residual · stage 4).
		// next wizard (s1570 Wave C): wizard|first-run-wizard|guided|wave-c|wave_c|wizard-residual (guided first-run residual + soft dogfood).
		// next journey (s1558 Wave B): journey|edge-journey|user-journey|first-run|edge_user_journey (7-stage edge-user-journey first-run map).
		// next status|pulse|board (s1382): residual-honest lane status board.
		// next export|receipt|stamp|evidence (s1387): residual-honest status export receipt.
		// next human-gates|human|gates|apply-gates (s1413): residual-honest human-gates still-required vs offline.
		// dual_write OFF · not Memory GA · never invent install green / Connected ·
		// catalog ≠ Connected · portal HITL · agent MCP cannot write installs · mesh ≠ memory.
		if len(parts) >= 2 {
			sub := strings.ToLower(parts[1])
			switch sub {
			case "help", "checklist", "?":
				fmt.Fprintln(out, agent.AionAgentOnboardingChecklist())
				return false, nil
			case "portal", "agent-mcp", "mcp":
				fmt.Fprintln(out, agent.AionAgentOnboardingPortalHandoff())
				fmt.Fprintln(out, "— residual: portal Agent/MCP handoff · dual_write OFF · not Memory GA · probe only ≠ Memory GA · never invent Connected · portal HITL")
				return false, nil
			case "status":
				fmt.Fprintln(out, agent.AionAgentOnboardingStatus())
				return false, nil
			case "next", "after", "continue", "lanes":
				// s1377: optional lane drill-down; s1382: status|pulse|board lane status board;
				// s1387: export|receipt|stamp|evidence status export receipt (+ optional json);
				// s1402: mesh|stream|streams|heartbeat|heartbeats|pull streaming lane;
				// s1407: memory-pull|ops-pack|pull-path|memorypull|ops_pack Ops Pack pull path;
				// s1417: agentic|agentic-integrations|integrations|list-plan plane-3 agentic integrations;
				// s1562: portal-hitl|hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl journey stage-5 portal HITL;
				// s1566: e4|e4-dogfood|client-attach|edge-memory-e4|e4_attach journey stage-6 E4 client-attach;
				// s1578: tool-call|tool-calls|deeper-e4|e4-tools|ingest-retrieve|tool_call deeper tool-call residual;
				// s1586: e10|e10-open|edge-memory-e10|ga-signoff|e10_open E10 Open reaffirm residual-check;
				// s1590: marketing-demo|marketing|sales-demo|demo-script|gtm-demo marketing demo path (local agent + memory);
				// s1432: planes|three-planes|product-planes|product|pillars|three_planes three product planes board;
				// s1437: sales|claims|buyer|claim-matrix|sales-claims|buyer-claims sales/buyer claims board;
				// s1442: demo|demo-ready|readiness|demo-readiness|lighthouse|landgrab demo readiness board;
				// s1447: operator|operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix operator readiness matrix;
				// s1542: setup|setup-lifecycle|lifecycle|setup_lifecycle setup lifecycle P1–P7 closeout residual;
				// s1570 Wave C: wizard|first-run-wizard|guided|wave-c|wave_c|wizard-residual first-run wizard residual + soft dogfood;
				// s1558 Wave B: journey|edge-journey|user-journey|first-run|edge_user_journey edge-user-journey first-run map;
				// s1413: human-gates|human|gates|apply-gates still-required vs offline residual.
				if len(parts) >= 3 {
					lane := strings.ToLower(parts[2])
					switch lane {
					case "plugins", "plugin", "smoke", "dogfood":
						fmt.Fprintln(out, agent.AionAgentOnboardingNextPluginsLane())
						fmt.Fprintln(out, "— residual: plugins smoke lane · dual_write OFF · not Memory GA · plugins dogfood ≠ Agent Plugins GA · plugins smoke ≠ invent Agent Plugins GA · residual PASS ≠ live dogfood · package load ≠ Memory GA · portal HITL")
						return false, nil
					case "gtm", "drafts":
						fmt.Fprintln(out, agent.AionAgentOnboardingNextGtmLane())
						fmt.Fprintln(out, "— residual: gtm draft-only lane · drafts only · no auto-send · human publish · GTM checklist ≠ invent GTM agent GA · dual_write OFF · not Memory GA")
						return false, nil
					case "memory", "mcp", "palace":
						// s1377+s1453+s1458+s1463+s1469+s1478+s1508: local-primary memory + edge OSS + public product attach + E4 client attach tip.
						fmt.Fprintln(out, agent.AionAgentOnboardingNextMemoryLane())
						fmt.Fprintln(out, "— residual: memory local lane · dual_write OFF · not Memory GA · package load ≠ Memory GA · ≠ freemium palace · mesh ≠ memory · iomesh-memory-mcp · public product attach · go install · no GOPRIVATE · 8080/mcp · stdio · docker compose still valid · flip complete residual ≠ invent Memory GA · public OSS ≠ invent platform GA · PASS ≠ invent full platform sidecar parity · E4 client attach (s1508) · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · aion broker private · aion still private · s1517 product-only memory sample · portal HITL")
						return false, nil
					case "mesh", "stream", "streams", "heartbeat", "heartbeats", "pull":
						// s1402: mesh streaming lane (org heartbeats). NOT pulse — pulse stays status board.
						// bare pull stays mesh (s1407 memory-pull uses memory-pull|ops-pack|pull-path|memorypull|ops_pack).
						fmt.Fprintln(out, agent.AionAgentOnboardingNextMeshLane())
						fmt.Fprintln(out, "— residual: mesh streaming lane · dual_write OFF · not Memory GA · mesh = streaming org heartbeats · mesh ≠ memory · never invent stream green · streams_not_probed · not OTel/APM · pull ≠ freemium hosted palace · rates ~$88/$119 optional")
						return false, nil
					case "memory-pull", "ops-pack", "pull-path", "memorypull", "ops_pack":
						// s1407: Ops Pack pull path. Bare pull stays mesh (s1402).
						fmt.Fprintln(out, agent.AionAgentOnboardingNextMemoryPullLane())
						fmt.Fprintln(out, "— residual: memory-pull Ops Pack lane · dual_write OFF · not Memory GA · pull = mesh → local palace egress · pull ≠ freemium hosted palace · Ops Pack ≠ GPU fleet · pull_not_probed · never invent pull green · package load ≠ Ops Pack entitlement · rates ~$88/$119 optional")
						return false, nil
					case "planes", "three-planes", "product-planes", "product", "pillars", "three_planes":
						// s1432: residual-honest three product planes board (mesh · memory-pull · agentic).
						// Do NOT steal pulse|board (status) · pull (mesh) · mcp (memory).
						fmt.Fprintln(out, agent.AionAgentOnboardingNextThreePlanes())
						fmt.Fprintln(out, "— residual: three product planes board · s1432 · no MCP dial · mesh · memory-pull · agentic · streams_not_probed · pull_not_probed · list_plan_not_connected · dual_auth_candidacy_open · dual_write OFF · not Memory GA · never invent stream green / pull green / Connected · residual PASS ≠ live dogfood · PASS ≠ live APPLY · rates ~$88/$119 optional · open boxes stay open")
						return false, nil
					case "sales", "claims", "buyer", "claim-matrix", "sales-claims", "buyer-claims":
						// s1437: residual-honest sales / buyer claims board (may claim / must not claim).
						// Do NOT steal product|planes (three-planes) · gtm|drafts (GTM) · pulse|board (status).
						fmt.Fprintln(out, agent.AionAgentOnboardingNextSalesClaims())
						fmt.Fprintln(out, "— residual: sales/buyer claims board · s1437 · no MCP dial · may claim / must not claim · three-planes grounded · dual_write OFF · book-demo OFF · not Memory GA · never invent Connected · dual_auth_candidacy_open · tool ship ≠ dual-auth live · residual PASS ≠ live dogfood · PASS ≠ live APPLY · rates ~$88/$119 optional · open boxes stay open")
						return false, nil
					case "demo", "demo-ready", "readiness", "demo-readiness", "lighthouse", "landgrab":
						// s1442: residual-honest demo readiness board (Lighthouse · book-demo OFF · Landgrab NOT READY).
						// Do NOT steal sales|claims (sales claims) · product|planes (three-planes) · pulse|board (status) · gtm|drafts.
						// Do NOT steal marketing-demo|marketing|sales-demo|demo-script|gtm-demo (s1590 marketing demo path).
						// landgrab alias stays honesty NOT READY — not invent ready.
						fmt.Fprintln(out, agent.AionAgentOnboardingNextDemoReadiness())
						fmt.Fprintln(out, "— residual: demo readiness board · s1442 · no MCP dial · Lighthouse beachhead packaging · book-demo OFF · Landgrab NOT READY · three planes · sales claims · human gates still open · dual_write OFF · not Memory GA · never invent Connected · residual PASS ≠ live dogfood · PASS ≠ live APPLY · residual PASS ≠ logos met · open boxes stay open · rates ~$88/$119 optional · founder-led walkthrough only when scheduled")
						return false, nil
					case "marketing-demo", "marketing", "sales-demo", "demo-script", "gtm-demo":
						// s1590: plain-language marketing demo path (local agent + local memory for videos/sales).
						// Do NOT steal bare demo|readiness|lighthouse|landgrab (demo readiness) · sales|claims (sales claims) · gtm|drafts (GTM).
						fmt.Fprintln(out, agent.AionAgentOnboardingNextMarketingDemoLane())
						fmt.Fprintln(out, "— marketing-demo: s1590 · plain-language local agent + local memory script · dual_write OFF · local memory · not Memory GA · mesh optional · never invent Connected · book-demo OFF · free eng s1590 · free-floor peer s1592+ mention only · NOT bare demo (demo readiness) · NOT bare sales · NOT bare gtm")
						return false, nil
					case "operator", "operator-matrix", "ops-matrix", "operator-readiness", "ops-readiness", "matrix":
						// s1447: residual-honest operator readiness matrix (demo · sales · planes · human-gates).
						// Do NOT steal demo|readiness|lighthouse|landgrab (demo) · sales|claims · product|planes · pulse|board · export|receipt.
						fmt.Fprintln(out, agent.AionAgentOnboardingNextOperatorMatrix())
						fmt.Fprintln(out, "— residual: operator readiness matrix · s1447 · no MCP dial · demo · sales · planes · human-gates · dual-auth candidacy · policy locks residual-honest · dual_write OFF · book-demo OFF · Landgrab NOT READY · not Memory GA · never invent Connected · dual_auth_candidacy_open · tool ship ≠ dual-auth live · residual PASS ≠ live dogfood · PASS ≠ live APPLY · residual PASS ≠ logos met · open boxes stay open · rates ~$88/$119 optional · residual_only · path_ready · still_human · policy_off · not_ready · portal_hitl_still")
						return false, nil
					case "setup", "setup-lifecycle", "lifecycle", "setup_lifecycle":
						// s1542+s1558: residual-honest setup lifecycle P1–P7 closeout residual map (stage 4 of edge-user-journey).
						// offline static lane ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA.
						// wizard alias is s1570 Wave C first-run wizard residual (not setup).
						fmt.Fprintln(out, agent.AionAgentOnboardingNextSetupLane())
						fmt.Fprintln(out, "— residual: setup lifecycle lane · s1542+s1558 · stage 4 of edge-user-journey · no MCP dial · P1–P7 closeout residual · dual_write OFF · not Memory GA · package wire ≠ Connected · catalog ≠ Connected · portal HITL · pull ≠ invent Connected · analyze tick ≠ invent green · drift PASS ≠ invent install green · repair apply ≠ invent Connected · dual_write never auto ON · still-human APPLY open · E10 Open · setup_not_probed · offline static lane ≠ live dogfood · setup closeout residual ≠ invent Edge Memory GA · Edge Memory GA candidacy only · free eng s1558 · never invent Connected · full first-run: /onboard next journey · guided residual: /onboard next wizard")
						return false, nil
					case "journey", "edge-journey", "user-journey", "first-run", "edge_user_journey":
						// s1558 Wave B: residual-honest 7-stage edge-user-journey first-run map.
						// Do NOT invent auto memory host · TUI portal SSO · Connected · dual_write ON · Memory GA · agent install APPLY.
						fmt.Fprintln(out, agent.AionAgentOnboardingNextJourneyLane())
						fmt.Fprintln(out, "— residual: edge-user-journey first-run lane · s1558 Wave B · no MCP dial · 7 stages residual-honest · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · portal HITL · agent MCP cannot write installs · catalog ≠ Connected · book-demo OFF · no invent TUI portal SSO · host not auto · free eng s1558 · free-floor peer s1560+ mention only · never invent Connected · stage 5: /onboard next portal-hitl · Wave C guided residual: /onboard next wizard")
						return false, nil
					case "wizard", "first-run-wizard", "guided", "wave-c", "wave_c", "wizard-residual":
						// s1570 Wave C: residual-honest guided first-run wizard residual + soft offline dogfood.
						// Bare /onboard next wizard stays board (not auto dogfood · never start host · never dial MCP).
						// NOT setup-lifecycle|lifecycle|setup_lifecycle (setup lane) · NOT invent full interactive auto wizard.
						if len(parts) >= 4 {
							sub := strings.ToLower(parts[3])
							switch sub {
							case "dogfood", "soft", "samples", "offline", "wizard-soft":
								fmt.Fprintln(out, agent.RunFirstRunWizardSoftDogfood())
								fmt.Fprintln(out, "— residual: first-run wizard soft offline dogfood · s1570 Wave C · no MCP dial · never start host · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent full interactive auto wizard · E10 Open · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · no invent TUI portal SSO · host not auto · free eng s1570 · free-floor peer s1572+ mention only")
								return false, nil
							}
						}
						fmt.Fprintln(out, agent.AionAgentOnboardingNextWizardLane())
						fmt.Fprintln(out, "— residual: first-run wizard residual lane · s1570 Wave C · no MCP dial · guided residual map · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · no invent TUI portal SSO · host not auto · residual PASS ≠ invent full interactive auto wizard · residual PASS ≠ live dogfood · soft offline ≠ invent Connected · free eng s1570 · free-floor peer s1572+ mention only · soft dogfood: /onboard next wizard dogfood · companion: /onboard next journey")
						return false, nil
					case "portal-hitl", "hitl", "portal_hitl", "portal-dogfood", "stage5", "connectors-hitl":
						// s1562: journey stage-5 portal HITL connectors board + soft offline dogfood residual.
						// Bare /onboard next portal-hitl stays board (not auto dogfood).
						// NOT bare portal|agent-mcp under /onboard (portal handoff) · NOT agentic list/plan soft (s1422 independent).
						if len(parts) >= 4 {
							sub := strings.ToLower(parts[3])
							switch sub {
							case "dogfood", "soft", "samples", "offline", "portal-hitl-soft":
								fmt.Fprintln(out, agent.RunPortalHITLSoftDogfood())
								fmt.Fprintln(out, "— residual: portal HITL soft offline dogfood · s1562 · no MCP dial · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · portal HITL still · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · template= ≠ install APPLY · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · free eng s1562 · free-floor peer s1564+ mention only")
								return false, nil
							}
						}
						fmt.Fprintln(out, agent.AionAgentOnboardingNextPortalHITLLane())
						fmt.Fprintln(out, "— residual: portal HITL lane · s1562 · journey stage 5 · no MCP dial · portal HITL when connect · agent MCP cannot write installs · catalog ≠ Connected · template= ≠ install APPLY · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ live dogfood · soft offline ≠ invent Connected · free eng s1562 · free-floor peer s1564+ mention only · soft dogfood: /onboard next portal-hitl dogfood")
						return false, nil
					case "e4", "e4-dogfood", "client-attach", "edge-memory-e4", "e4_attach":
						// s1566: journey stage-6 E4 client-attach board + soft offline dogfood residual.
						// Bare /onboard next e4 stays board (not auto dogfood · never start host · never dial MCP).
						// NOT bare mcp|palace (memory lane) · NOT invent dual_write ON / Edge Memory GA declared / E10 closed.
						// NOT tool-call|deeper-e4 (s1578 deeper tool-call residual).
						if len(parts) >= 4 {
							sub := strings.ToLower(parts[3])
							switch sub {
							case "dogfood", "soft", "samples", "offline", "e4-soft":
								fmt.Fprintln(out, agent.RunE4SoftDogfood())
								fmt.Fprintln(out, "— residual: E4 client-attach soft offline dogfood · s1566 · no MCP dial · never start host · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · free eng s1566 · free-floor peer s1568+ mention only")
								return false, nil
							}
						}
						fmt.Fprintln(out, agent.AionAgentOnboardingNextE4Lane())
						fmt.Fprintln(out, "— residual: E4 client-attach lane · s1566 · journey stage 6 · no MCP dial · never start host · E4 client attach · tools=6 · iomesh mcp --connect residual · iomesh-memory-mcp · local-primary · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · soft offline ≠ invent Connected · free eng s1566 · free-floor peer s1568+ mention only · soft dogfood: /onboard next e4 dogfood · deeper: /onboard next tool-call · E10 Open reaffirm: /onboard next e10")
						return false, nil
					case "tool-call", "tool-calls", "deeper-e4", "e4-tools", "ingest-retrieve", "tool_call":
						// s1578: deeper tool-call residual board + soft offline dogfood after E4 attach.
						// Bare /onboard next tool-call stays board (not auto dogfood · never start host · never dial MCP).
						// NOT bare e4|client-attach (E4 attach lane) · NOT bare mcp|palace (memory lane) · NOT e10 (s1586).
						if len(parts) >= 4 {
							sub := strings.ToLower(parts[3])
							switch sub {
							case "dogfood", "soft", "samples", "offline", "tool-call-soft":
								fmt.Fprintln(out, agent.RunDeeperToolCallSoftDogfood())
								fmt.Fprintln(out, "— residual: deeper tool-call soft offline dogfood · s1578 · no MCP dial · never start host · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · free eng s1578 · free-floor peer s1580+ mention only")
								return false, nil
							}
						}
						fmt.Fprintln(out, agent.AionAgentOnboardingNextToolCallLane())
						fmt.Fprintln(out, "— residual: deeper tool-call lane · s1578 · journey stage 6/7 · no MCP dial · never start host · memory_ingest_turn · memory_retrieve · memory_list · memory_facts_as_of · Partial→client-attach-evidence · companion /onboard next e4 · tools=6 · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared · E10 Open · tip ≠ invent forever-green product dogfood · residual PASS ≠ live dogfood · soft offline ≠ invent Connected · free eng s1578 · free-floor peer s1580+ mention only · soft dogfood: /onboard next tool-call dogfood · E10: /onboard next e10")
						return false, nil
					case "e10", "e10-open", "edge-memory-e10", "ga-signoff", "e10_open":
						// s1586: E10 Open reaffirm residual-check after OSS packaging continuum.
						// Bare /onboard next e10 stays board (not auto residual-check · never start host · never dial MCP).
						// NOT invent E10 closed / Edge Memory GA declared / live APPLY green.
						// NOT bare e4|client-attach · NOT bare human-gates · NOT tool-call.
						if len(parts) >= 4 {
							sub := strings.ToLower(parts[3])
							switch sub {
							case "dogfood", "soft", "samples", "offline", "e10-soft", "residual-check":
								fmt.Fprintln(out, agent.RunE10OpenSoftDogfood())
								fmt.Fprintln(out, "— residual: E10 Open soft offline residual-check · s1586 · no MCP dial · never start host · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared · residual PASS ≠ invent E10 closed · E10 Open · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · PASS ≠ live APPLY · free eng s1586 · free-floor peer s1588+ mention only")
								return false, nil
							}
						}
						fmt.Fprintln(out, agent.AionAgentOnboardingNextE10Lane())
						fmt.Fprintln(out, "— residual: E10 Open reaffirm lane · s1586 · Platform residual honesty · no MCP dial · never start host · E10 Open · residual PASS ≠ invent E10 closed · residual PASS ≠ invent Edge Memory GA declared · founder sign-off only if declaring Edge Memory GA · candidacy allowed without E10 · PASS ≠ live APPLY · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · residual-check · residual PASS ≠ live dogfood · soft offline ≠ invent Connected · free eng s1586 · free-floor peer s1588+ mention only · soft residual-check: /onboard next e10 dogfood · companion: /onboard next e4 · /onboard next human-gates · OSS packaging")
						return false, nil
					case "agentic", "agentic-integrations", "integrations", "list-plan":
						// s1417: product plane 3 agentic integrations (MCP list/plan residual-honest).
						// s1422: optional 4th token dogfood|soft|samples|offline|list-plan-soft → soft offline list/plan dogfood.
						// s1427: optional 4th token dual-auth|candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy → dual-auth candidacy board.
						// Bare /onboard next agentic stays board (not auto dogfood / dual-auth).
						// NOT bare mcp (memory) · NOT bare portal|agent-mcp (portal handoff) · NOT bare pull (mesh).
						// NOT portal-hitl|hitl (s1562 portal HITL lane) · NOT bare dogfood under /onboard next (dogfood stays plugins lane).
						if len(parts) >= 4 {
							sub := strings.ToLower(parts[3])
							switch sub {
							case "dogfood", "soft", "samples", "offline", "list-plan-soft":
								fmt.Fprintln(out, agent.RunAgenticListPlanSoftDogfood())
								fmt.Fprintln(out, "— residual: agentic list/plan soft offline dogfood · s1422 · no MCP dial · soft offline list/plan ≠ live dogfood · ≠ invent Connected · portal HITL still · list_org fail-open ≠ empty-as-none · session soft ≠ live dogfood · dual_write OFF · not Memory GA · template= ≠ install APPLY · agent MCP cannot write installs · companion portal HITL soft: /onboard next portal-hitl dogfood")
								return false, nil
							case "dual-auth", "candidacy", "list-org", "org-installs", "dual_auth", "dual-auth-candidacy":
								// s1427: dual-auth candidacy depth (list_org fail-open · tool ship ≠ dual-auth live).
								// Does NOT steal dogfood|soft|samples|offline|list-plan-soft (s1422 soft dogfood).
								fmt.Fprintln(out, agent.AionAgentOnboardingNextAgenticDualAuthCandidacy())
								fmt.Fprintln(out, "— residual: agentic dual-auth candidacy · s1427 · no MCP dial · dual_auth_candidacy_open · list_org_unavailable · list_org_connector_installs available=false status=unavailable installs=null · never invent empty-as-none · tool ship ≠ dual-auth live · never invent dual-auth live · agent MCP cannot write installs · portal HITL · dual_write OFF · not Memory GA · residual PASS ≠ live dogfood · PASS ≠ live APPLY · open boxes stay open")
								return false, nil
							}
						}
						fmt.Fprintln(out, agent.AionAgentOnboardingNextAgenticLane())
						fmt.Fprintln(out, "— residual: agentic integrations lane · product plane 3 · dual_write OFF · not Memory GA · MCP list/plan residual-honest · plan deep links = browser HITL only · template= ≠ install APPLY · catalog ≠ Connected · list_org fail-open ≠ empty-as-none · list_plan_not_connected · portal_hitl_still · agent MCP cannot write installs · never invent Connected · rates ~$88/$119 optional · soft dogfood: /onboard next agentic dogfood · dual-auth: /onboard next agentic dual-auth · companion portal HITL: /onboard next portal-hitl (s1562)")
						return false, nil
					case "human-gates", "human", "gates", "apply-gates", "still-human", "apply-residual":
						// s1413+s1546+s1550+s1574: residual-honest human-gates edge-first residual pin + still-human APPLY soft dogfood.
						// Bare /onboard next human-gates stays board (not auto dogfood · never dial MCP · never start host).
						// edge-first · knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect.
						// dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA declared.
						// PASS ≠ invent Connected · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open.
						if len(parts) >= 4 {
							sub := strings.ToLower(parts[3])
							switch sub {
							case "dogfood", "soft", "samples", "offline", "still-human-soft", "apply-soft":
								fmt.Fprintln(out, agent.RunStillHumanApplySoftDogfood())
								fmt.Fprintln(out, "— residual: still-human APPLY soft offline dogfood · s1574 Wave C continuum · no MCP dial · never start host · soft offline ≠ invent Connected · residual PASS ≠ live dogfood · session soft ≠ live dogfood · residual PASS ≠ invent Edge Memory GA declared · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · E10 Open · dual_write OFF · book-demo OFF · not Memory GA · Edge Memory GA candidacy only · portal HITL when connect · free eng s1574 · free-floor peer s1576+ mention only")
								return false, nil
							}
						}
						fmt.Fprintln(out, agent.AionAgentHumanGatesHonestyBoard())
						fmt.Fprintln(out, "— residual: human-gates honesty board · s1550 edge-first · s1574 Wave C continuum still-human APPLY residual · dual_write OFF · not Memory GA · Edge Memory GA candidacy only · residual PASS ≠ invent Edge Memory GA · residual PASS ≠ invent Edge Memory GA declared · PASS ≠ invent Connected · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · knowledge multi-tenant punted · Slack HMAC punted · portal HITL when connect · book-demo OFF · leave ON_SIGNAL unset · H1/H2 not launch gate · E10 Open · agent MCP cannot write installs · never invent Connected / INSTALL_STORE green / book-demo as ON · soft dogfood: /onboard next human-gates dogfood · E10 Open reaffirm: /onboard next e10 · free eng s1574 · free-floor peer s1576+ mention only")
						return false, nil
					case "status", "pulse", "board":
						fmt.Fprintln(out, agent.AionAgentOnboardingNextLaneStatus())
						fmt.Fprintln(out, "— residual: next lane status board · dual_write OFF · not Memory GA · session soft ≠ live dogfood · portal_hitl_still · streams_not_probed · pull_not_probed · list_plan_not_connected · never invent Connected/GA/APPLY/stream green/pull green · residual PASS ≠ live dogfood · board/export evidence ≠ invent Connected · mesh ≠ memory")
						return false, nil
					case "export", "receipt", "stamp", "evidence":
						// Optional third token: json → JSON receipt; otherwise markdown.
						if len(parts) >= 4 && strings.ToLower(parts[3]) == "json" {
							fmt.Fprintln(out, agent.AionAgentOnboardingNextLaneStatusExportJSON())
							fmt.Fprintln(out, "— residual: next lane status export json · evidence_kind=onboard_next_lane_status_export · offline_static · not_live_dogfood · s1387 · session soft ≠ live dogfood · streams_not_probed · pull_not_probed · list_plan_not_connected · board/export evidence ≠ invent Connected · dual_write OFF · not Memory GA · mesh ≠ memory · agentic tip /onboard next agentic · human-gates tip /onboard next human-gates")
							return false, nil
						}
						fmt.Fprintln(out, agent.AionAgentOnboardingNextLaneStatusExport())
						fmt.Fprintln(out, "— residual: next lane status export receipt · evidence_kind=onboard_next_lane_status_export · offline_static · not_live_dogfood · s1387 · session soft ≠ live dogfood · streams_not_probed · pull_not_probed · list_plan_not_connected · board/export evidence ≠ invent Connected · dual_write OFF · not Memory GA · mesh ≠ memory · agentic tip /onboard next agentic · human-gates tip /onboard next human-gates")
						return false, nil
					default:
						// Unknown next sub → overview + usage hint listing lanes.
						fmt.Fprintln(out, agent.AionAgentOnboardingNextLanes())
						fmt.Fprintln(out, "— residual: post-onboard next lanes · dual_write OFF · not Memory GA · plugins dogfood ≠ Agent Plugins GA · drafts only · no auto-send · package load ≠ Memory GA · mesh ≠ memory · portal HITL · board/export evidence ≠ invent Connected · pull_not_probed · list_plan_not_connected · setup_not_probed · PASS ≠ invent human-gate green · Edge Memory GA candidacy only · free eng s1558 · free eng s1562 · free eng s1566 · free eng s1570 · free eng s1574 · free eng s1578 · free eng s1582 · free eng s1586 · free eng s1590 · OSS packaging residual · E10 Open reaffirm · marketing demo path")
						fmt.Fprintln(out, "— packaging: "+agent.OSSPackagingHonestyOneLiner)
						fmt.Fprintln(out, "usage: /onboard next [plugins|gtm|memory|mesh|memory-pull|agentic|portal-hitl|e4|tool-call|e10|planes|sales|demo|marketing-demo|operator|setup|journey|wizard|status|export|human-gates]  (Edge OSS path: setup|journey|wizard|memory|e4|portal-hitl|marketing-demo · Platform residual honesty optional residual-check: human-gates|tool-call|e10 · soft residual-check = offline residual honesty · slash dogfood kept for compatibility; lane aliases: plugins→plugin|dogfood · gtm→drafts · memory→mcp|palace · mesh→stream|streams|heartbeat|heartbeats|pull · memory-pull→ops-pack|pull-path|memorypull|ops_pack · agentic→agentic-integrations|integrations|list-plan · portal-hitl→hitl|portal_hitl|portal-dogfood|stage5|connectors-hitl · e4→e4-dogfood|client-attach|edge-memory-e4|e4_attach · tool-call→tool-calls|deeper-e4|e4-tools|ingest-retrieve|tool_call · e10→e10-open|edge-memory-e10|ga-signoff|e10_open · planes→three-planes|product-planes|product|pillars|three_planes · sales→claims|buyer|claim-matrix|sales-claims|buyer-claims · demo→demo-ready|readiness|demo-readiness|lighthouse|landgrab · marketing-demo→marketing|sales-demo|demo-script|gtm-demo · operator→operator-matrix|ops-matrix|operator-readiness|ops-readiness|matrix · setup→setup-lifecycle|lifecycle|setup_lifecycle · journey→edge-journey|user-journey|first-run|edge_user_journey · wizard→first-run-wizard|guided|wave-c|wave_c|wizard-residual · status→pulse|board · export→receipt|stamp|evidence · human-gates→human|gates|apply-gates|still-human|apply-residual; soft residual-check dogfood: tool-call dogfood|soft|samples|offline|tool-call-soft · e10 dogfood|soft|samples|offline|e10-soft|residual-check · human-gates dogfood|soft|samples|offline|still-human-soft|apply-soft; parent aliases after|continue|lanes; export json for JSON receipt; pulse stays status board; bare pull stays mesh; bare mcp stays memory; bare portal stays portal handoff; product/planes stay three-planes; landgrab stays Landgrab NOT READY honesty; readiness/lighthouse stay demo board; bare demo stays demo readiness · marketing-demo is the plain-language demo script)")
						return false, nil
					}
				}
				fmt.Fprintln(out, agent.AionAgentOnboardingNextLanes())
				fmt.Fprintln(out, "— residual: post-onboard next lanes · dual_write OFF · not Memory GA · plugins dogfood ≠ Agent Plugins GA · drafts only · no auto-send · package load ≠ Memory GA · mesh ≠ memory · portal HITL · board/export evidence ≠ invent Connected · pull_not_probed · list_plan_not_connected · setup_not_probed · PASS ≠ invent human-gate green · Edge Memory GA candidacy only · free eng s1558 · free eng s1562 · free eng s1566 · free eng s1570 · free eng s1574 · free eng s1578 · free eng s1582 · free eng s1586 · free eng s1590 · OSS packaging residual · E10 Open reaffirm · marketing demo path")
				fmt.Fprintln(out, "— packaging: "+agent.OSSPackagingHonestyOneLiner)
				return false, nil
			}
			// Unknown subcommand: still print guidance + usage hint.
			fmt.Fprintln(out, agent.AionAgentOnboardingGuidanceNote())
			fmt.Fprintln(out, "— residual: TUI ↔ aion onboarding · dual_write OFF · not Memory GA · never invent Connected · portal HITL · skill aion-agent-onboarding via read_skill")
			fmt.Fprintln(out, "— packaging: "+agent.OSSPackagingHonestyOneLiner)
			fmt.Fprintln(out, "usage: /onboard [help|checklist|portal|status|next]  (aliases /aion-onboard /agent-onboard; portal aliases agent-mcp|mcp; next aliases after|continue|lanes; next lanes Edge OSS path: setup|journey|wizard|memory|e4|portal-hitl|marketing-demo · Platform residual honesty (optional residual-check): human-gates|tool-call|e10 · also plugins|gtm|mesh|memory-pull|agentic|planes|sales|demo|operator|status|export)")
			return false, nil
		}
		fmt.Fprintln(out, agent.AionAgentOnboardingGuidanceNote())
		fmt.Fprintln(out, "— residual: TUI ↔ aion onboarding · dual_write OFF · not Memory GA · never invent Connected · portal HITL · skill aion-agent-onboarding via read_skill")
		fmt.Fprintln(out, "— packaging: "+agent.OSSPackagingHonestyOneLiner)
	case "/plugins", "/plugin":
		// s1392: residual-honest /plugins slash soft offline dogfood.
		// Subcommands: help|? · list · validate · dogfood (aliases soft|samples|offline) · status.
		// Bare /plugins → help. Discover/list ≠ Connected · soft offline dogfood ≠ invent Agent Plugins GA ·
		// residual PASS ≠ live dogfood · package load ≠ Memory GA · dual_write OFF · book-demo OFF ·
		// never invent install green / Connected / INSTALL_STORE APPLY · portal HITL · agent MCP cannot write installs.
		if len(parts) < 2 {
			fmt.Fprintln(out, pluginsHelp())
			return false, nil
		}
		sub := strings.ToLower(parts[1])
		switch sub {
		case "help", "?":
			fmt.Fprintln(out, pluginsHelp())
		case "list", "ls":
			handlePluginsList(out, parts[2:])
		case "validate", "check":
			handlePluginsValidate(out, parts[2:])
		case "smoke", "dogfood", "soft", "samples", "offline":
			// Public name: smoke. dogfood = legacy alias (s1521). check stays validate.
			handlePluginsDogfood(out)
		case "status", "st", "pulse":
			handlePluginsStatus(out)
		default:
			fmt.Fprintf(out, "plugins: unknown subcommand %q\n%s\n", parts[1], pluginsHelp())
		}
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
  /dashboard [help|preview|focus|ack]  empty until consume · preview = eval not your org · ack = brief ritual (aliases /heartbeat /mesh-console)
  /mesh                I/O Mesh status + usage
  /catalog [query]     list mesh data products (catalog plane)
  /memory [recall|related|digest|facts-as-of|timeline|compact-status|trigger-compact|semantic|ingest-event|patterns|anomalies|supersede|ingest|ingest-dir|status]  Memory Palace (sync HTTP + MCP; related multi-hop · digest ops pulse · facts-as-of bi-temporal lite · timeline/compact-status · trigger-compact HITL · semantic tier-4 · ingest-event s138 T1 · patterns/anomalies ops pulse Beta · supersede A3 lite HITL · ingest-dir folder overlay · status advanced inventory)
  /integrations [list|plan|signing|status]  list/plan a source via MCP, then finish in portal HITL (not install CRUD)
  /setup [init|preflight|portal|reload|pull|analyze|drift|repair]  setup lifecycle (managed config · preflight · portal HITL · hot MCP reload · opt-in continuous pull/analyze · drift report · guided repair; alias /setup-lifecycle; dual_write OFF · not Memory GA · PASS ≠ invent Connected · pull/analyze/repair ≠ invent Connected)
  /gtm [help|checklist|brief]  GTM draft-only guidance, checklist, or palace voc_brief / market_telling (aliases /gtm-draft /gtm-agent; no auto-send; human publish; palace SoR · source=agent-brief · tenant gtm/founder)
  /onboard [help|checklist|portal|status|next]  start here: portal MCP copy → TUI attach → /integrations list|plan → portal HITL (aliases /aion-onboard /agent-onboard; next wizard|journey|setup|portal-hitl|memory · operator notes /onboard next [plugins|gtm|memory|mesh|export|…])
  /plugins [help|list|validate|smoke|status]  residual-honest Agent Plugins soft offline smoke (alias /plugin; smoke aliases dogfood|soft|samples|offline; check→validate; Discover ≠ Connected · soft offline ≠ live smoke · ≠ invent Agent Plugins GA)
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

// integrationsHelp is bare /integrations and help/? copy (s1238/s1242/s1243/s1247 residual honesty).
// status is a separate operator pulse (IntegrationsStatus), not this help text.
func integrationsHelp() string {
	return agent.IntegrationsHelp()
}

// setupHelp is bare /setup and help/? copy (s1526 P3+P4 + s1530 P5 + s1534 P6 + s1538 P7 residual honesty).
// s1723: when IOMESH_PLATFORM_RESIDUAL is on, append PlatformResidualLabelNote (label only · never hides subcommands).
func setupHelp() string {
	base := strings.TrimSpace(`usage: /setup [init [profiles] [--stdio] [--print-only] [--plugins-dir path] [--memory-url URL] [--mesh-endpoint URL] [--mesh-tenant id] [--mesh-org id] [--platform-mcp-url URL] | preflight | portal | reload | pull … | analyze … | drift|maintain | repair …]
  init       write managed config fragment (profiles: local-memory|plugins|mesh|platform-mcp|all; default local-memory; mesh flags write hooks not /v7/mcp; --mesh-org persists [iomesh].org / IOMESH_ORG residual)
  preflight  residual-honest probe (aliases status|check) — inherits process --config / IOMESH_CONFIG unless slash --config; PASS ≠ invent Connected / Memory GA
  portal     browser HITL URLs (integrations + settings/agent)
  reload     hot-swap MCP + mesh + re-scan skills from process config (or slash --config; Wire · ReplaceSkills · NewMesh · ReplaceMesh · ConnectMCP + ReplaceMCP; package wire ≠ Connected · infer ≠ Connected)
  pull       continuous pull status|start|once|stop (s1530 P5 · opt-in · CLI iomesh memory pull still valid)
  analyze    analyze tick status|start|once|stop (s1534 P6 · opt-in · /memory digest still valid)
  drift      report-only config vs runtime drift (alias maintain · residual next steps)
  repair     guided plan from drift · apply safe steps only with --yes (s1538 P7 · no auto-repair without --yes)
  help|?     this residual-honest usage (also bare /setup)
aliases: /setup-lifecycle
honesty: ` + setup.SetupLifecycleHonestyOneLiner + `
  secrets via env names only · portal HITL for OAuth/install · continuous pull/analyze opt-in
  skill: read_skill setup-lifecycle-agent · system note <setup-lifecycle> on AttachMCP
  reload: dual_write OFF · skills re-scanned · package wire ≠ Connected · does not invent install green · skills re-scan ≠ invent Connected
  pull: dual_write OFF · not Memory GA · pull ≠ invent Connected · CLI iomesh memory pull still valid
  analyze: dual_write OFF · not Memory GA · analyze tick ≠ invent Connected · /memory digest still valid
  drift: dual_write OFF · not Memory GA · drift report ≠ invent install green · package wire ≠ Connected
  repair: dual_write OFF · not Memory GA · repair apply ≠ invent Connected · package wire ≠ Connected · portal HITL still human · safe steps only · no auto-repair without apply --yes`)
	if note := setup.PlatformResidualLabelNote(); note != "" {
		return base + "\n" + note
	}
	return base
}

// setupPullHonesty is printed on every /setup pull output (s1530 P5 residual honesty).
const setupPullHonesty = "honesty: dual_write OFF · not Memory GA · pull ≠ invent Connected · CLI iomesh memory pull still valid"

// handleSetupInit parses simple /setup init args and writes (or prints) managed fragment.
func handleSetupInit(out io.Writer, args []string) {
	opt := setup.DefaultInitOptions()
	printOnly := false
	var profileTokens []string
	var pluginDirs []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--stdio" || a == "-stdio":
			opt.UseStdioMemory = true
		case a == "--print-only" || a == "--print" || a == "-n":
			printOnly = true
		case a == "--plugins-dir" || a == "--plugins_dir":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(out, "setup init: --plugins-dir requires a path")
				return
			}
			i++
			pluginDirs = append(pluginDirs, args[i])
		case strings.HasPrefix(a, "--plugins-dir="):
			pluginDirs = append(pluginDirs, strings.TrimPrefix(a, "--plugins-dir="))
		case a == "--memory-url" || a == "--memory_url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(out, "setup init: --memory-url requires a URL")
				return
			}
			i++
			opt.MemoryHTTPURL = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--memory-url="):
			opt.MemoryHTTPURL = strings.TrimSpace(strings.TrimPrefix(a, "--memory-url="))
		case a == "--mesh-endpoint" || a == "--mesh_endpoint":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(out, "setup init: --mesh-endpoint requires a URL")
				return
			}
			i++
			opt.MeshEndpoint = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--mesh-endpoint="):
			opt.MeshEndpoint = strings.TrimSpace(strings.TrimPrefix(a, "--mesh-endpoint="))
		case a == "--mesh-tenant" || a == "--mesh_tenant":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(out, "setup init: --mesh-tenant requires a tenant")
				return
			}
			i++
			opt.MeshTenant = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--mesh-tenant="):
			opt.MeshTenant = strings.TrimSpace(strings.TrimPrefix(a, "--mesh-tenant="))
		case a == "--mesh-org" || a == "--mesh_org":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(out, "setup init: --mesh-org requires an org id")
				return
			}
			i++
			opt.MeshOrg = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--mesh-org="):
			opt.MeshOrg = strings.TrimSpace(strings.TrimPrefix(a, "--mesh-org="))
		case a == "--platform-mcp-url" || a == "--platform_mcp_url":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				fmt.Fprintln(out, "setup init: --platform-mcp-url requires a URL")
				return
			}
			i++
			opt.PlatformMCPURL = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--platform-mcp-url="):
			opt.PlatformMCPURL = strings.TrimSpace(strings.TrimPrefix(a, "--platform-mcp-url="))
		case a == "--config":
			// Accept and ignore with note — slash always uses user path unless print-only;
			// full --config path support stays on CLI iomesh setup init.
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				fmt.Fprintf(out, "setup init: note — slash writes user config path (CLI --config for custom path); got %q ignored for write target\n", args[i])
			}
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(out, "setup init: unknown flag %q\n%s\n", a, setupHelp())
			return
		default:
			profileTokens = append(profileTokens, a)
		}
	}
	opt.PluginsDirs = pluginDirs
	var profiles []setup.Profile
	if len(profileTokens) == 0 {
		profiles = []setup.Profile{setup.ProfileLocalMemory}
	} else {
		profiles = setup.ParseProfiles(strings.Join(profileTokens, ","))
	}
	frag, err := setup.BuildManagedFragment(profiles, opt)
	if err != nil {
		fmt.Fprintf(out, "setup init: %v\n", err)
		return
	}
	if printOnly {
		fmt.Fprint(out, frag)
		if !strings.HasSuffix(frag, "\n") {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out, "honesty: dual_write OFF · not Memory GA · catalog ≠ Connected · portal HITL · setup PASS ≠ invent install green")
		return
	}
	path, err := config.WriteSetupManagedUser(frag)
	if err != nil {
		fmt.Fprintf(out, "setup init: write: %v\n", err)
		return
	}
	fmt.Fprintf(out, "setup init: wrote managed fragment → %s\n", path)
	fmt.Fprintf(out, "profiles: %v\n", profiles)
	// s1686 residual-honest dual-path next-step (peer CLI iomesh setup init · free eng s1723 slash parity).
	for _, line := range setup.SetupInitNextStepLines() {
		fmt.Fprintln(out, line)
	}
	if setup.ProfilesWantMesh(profiles) {
		for _, line := range setup.SetupInitMeshNextStepLines() {
			fmt.Fprintln(out, line)
		}
	}
}

// handleSetupReload reloads skills catalog + MCP servers + mesh from config without process restart
// (s1526 P4 MCP · s1670 skills re-scan · s2055 mesh infer/hot-swap). Uses runtimewire.Wire +
// LoadWithBuiltin + ReplaceSkills, NewMesh + ReplaceMesh, then ConnectMCP + ReplaceMCP.
// Residual-honest: package wire ≠ Connected · infer ≠ Connected · skills re-scan ≠ invent Connected.
func handleSetupReload(out io.Writer, rt runtimeAdapter, args []string) {
	if rt.rt == nil {
		fmt.Fprintln(out, "setup reload: no agent runtime")
		return
	}
	cfgPath := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--config" && i+1 < len(args) {
			i++
			cfgPath = strings.TrimSpace(args[i])
			continue
		}
		if strings.HasPrefix(a, "--config=") {
			cfgPath = strings.TrimSpace(strings.TrimPrefix(a, "--config="))
			continue
		}
	}
	if cfgPath == "" {
		cfgPath = rt.processConfigPath()
	}
	var (
		cfg *config.Config
		err error
	)
	if cfgPath != "" {
		cfg, err = config.Load(cfgPath)
	} else {
		cfg, err = config.LoadUser()
	}
	if err != nil {
		fmt.Fprintf(out, "setup reload: config: %v\n", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	reloadRuntimeFromConfig(ctx, out, rt, cfg, true)
}

// reloadRuntimeFromConfig re-scans skills (when feature on) and hot-swaps MCP + mesh from cfg.
// Shared by /setup reload and setupRepairExecutor.ReloadMCP (s1670 · s2055 mesh).
// When print is true, writes residual-honest status lines to out (may be nil when silent).
// Residual honesty: dual_write OFF · package wire ≠ Connected · skills re-scan ≠ invent Connected ·
// infer ≠ Connected · catalog MCP ≠ hooks streams · not Memory GA · not Agent Plugins GA ·
// Discover/map ≠ install APPLY green.
func reloadRuntimeFromConfig(ctx context.Context, out io.Writer, rt runtimeAdapter, cfg *config.Config, print bool) {
	if rt.rt == nil {
		if print && out != nil {
			fmt.Fprintln(out, "setup reload: no agent runtime")
		}
		return
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	ws := ""
	if rt.rt.Workspace() != nil {
		ws = rt.rt.Workspace().Root()
	}
	logger := slog.Default()
	wired := runtimewire.Wire(cfg, ws, logger)

	// Skills re-scan when skills feature enabled (s1670).
	if runtimewire.SkillsFeatureOn(cfg) {
		cat, err := skills.LoadWithBuiltin(wired.SkillDirs...)
		if err != nil {
			if print && out != nil {
				fmt.Fprintf(out, "setup reload: skills load: %v (residual-honest · not invent green)\n", err)
			}
			// Leave prior skills alone on load error — fail-open residual, do not invent green.
		} else {
			rt.rt.ReplaceSkills(cat)
			if print && out != nil {
				n := 0
				if cat != nil {
					n = cat.Len()
				}
				fmt.Fprintf(out, "setup reload: skills re-scanned count=%d dirs=%d (package wire · LoadWithBuiltin)\n", n, len(wired.SkillDirs))
			}
		}
	} else {
		rt.rt.ReplaceSkills(nil) // detach when skills feature off
		if print && out != nil {
			fmt.Fprintln(out, "setup reload: skills feature off — catalog detached")
		}
	}

	// Hot-swap mesh from [iomesh] or inferred hooks (infer ≠ Connected).
	mesh, inf := runtimewire.NewMesh(cfg, logger)
	rt.rt.ReplaceMesh(mesh)
	if print && out != nil {
		if mesh != nil && mesh.Enabled() {
			src := "config [iomesh]"
			if inf.Endpoint != "" {
				src = "inferred from portal MCP"
			}
			fmt.Fprintf(out, "setup reload: mesh attached endpoint=%s (%s · infer ≠ Connected · catalog ≠ streams)\n",
				mesh.Endpoint(), src)
		} else {
			fmt.Fprintln(out, "setup reload: mesh detached — add [iomesh] or infer from portal MCP · do not invent consume")
		}
	}

	// ConnectMCP returns nil when MCP feature off or no servers — ReplaceMCP detaches.
	mgr := runtimewire.ConnectMCP(ctx, cfg, ws, logger)
	rt.rt.ReplaceMCP(mgr)
	if !print || out == nil {
		return
	}
	if mgr == nil {
		fmt.Fprintln(out, "setup reload: MCP feature off or no servers configured — detached")
		fmt.Fprintln(out, "honesty: dual_write OFF · package wire ≠ Connected · skills re-scan ≠ invent Connected · not Memory GA · not Agent Plugins GA · portal HITL for installs")
		for _, line := range setup.SetupReloadNextStepLines() {
			fmt.Fprintln(out, line)
		}
		return
	}
	nTools := 0
	for range mgr.Bindings() {
		nTools++
	}
	fmt.Fprintf(out, "setup reload: connected=%d tools=%d (package wire · fail-open per server)\n", mgr.Len(), nTools)
	fmt.Fprintln(out, "honesty: dual_write OFF · package wire ≠ Connected · skills re-scanned · Discover/map ≠ install APPLY green · skills re-scan ≠ invent Connected · not Memory GA · not Agent Plugins GA")
	fmt.Fprintln(out, "note: skills re-scanned on reload · continuous pull/analyze opt-in via /setup pull · /setup analyze · drift /setup drift · repair /setup repair · CLI iomesh memory pull · /memory digest still valid")
	for _, line := range setup.SetupReloadNextStepLines() {
		fmt.Fprintln(out, line)
	}
}

// handleSetupPull dispatches /setup pull [status|start|once|stop] (s1530 P5).
// Bare /setup pull → status. Residual-honest: dual_write OFF · not Memory GA ·
// pull ≠ invent Connected · CLI iomesh memory pull still valid.
func handleSetupPull(out io.Writer, rt runtimeAdapter, args []string) {
	sub := "status"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = strings.ToLower(strings.TrimSpace(args[0]))
		rest = args[1:]
	}
	switch sub {
	case "status", "st", "":
		handleSetupPullStatus(out, rt)
	case "start":
		handleSetupPullStart(out, rt, rest, false)
	case "once":
		handleSetupPullStart(out, rt, rest, true)
	case "stop":
		handleSetupPullStop(out, rt)
	case "help", "?":
		fmt.Fprintln(out, setupPullHelp())
	default:
		fmt.Fprintf(out, "setup pull: unknown subcommand %q\n%s\n", sub, setupPullHelp())
	}
}

func setupPullHelp() string {
	return strings.TrimSpace(`usage: /setup pull [status|start|once|stop]
  status   residual-honest continuous pull snapshot (default; bare /setup pull)
  start    start continuous pull (MaxLoops=0; loads [memory] pull_*; --once · --dry-run · --config path)
  once     single fetch cycle (MaxLoops=1; same knobs as start)
  stop     cancel in-session continuous pull (no-op when idle)
honesty: dual_write OFF · not Memory GA · pull ≠ invent Connected · CLI iomesh memory pull still valid
  opt-in only · pull_continuous=true is config opt-in · setup fragment defaults pull_continuous=false`)
}

// handleSetupPullStatus prints ContinuousMemoryPullStatus residual-honest (idle ≠ green).
func handleSetupPullStatus(out io.Writer, rt runtimeAdapter) {
	if rt.rt == nil {
		fmt.Fprintln(out, "setup pull status: no agent runtime")
		fmt.Fprintln(out, "running: false")
		fmt.Fprintln(out, setupPullHonesty)
		return
	}
	st := rt.rt.ContinuousMemoryPullStatus()
	fmt.Fprintln(out, "setup pull status (residual-honest · s1530 P5)")
	fmt.Fprintf(out, "running: %v\n", st.Running)
	if st.Stream != "" || st.Consumer != "" || st.Filter != "" {
		fmt.Fprintf(out, "stream: %s\n", st.Stream)
		fmt.Fprintf(out, "consumer: %s\n", st.Consumer)
		fmt.Fprintf(out, "filter: %s\n", st.Filter)
	} else if !st.Running {
		fmt.Fprintln(out, "stream: (idle · empty)")
		fmt.Fprintln(out, "consumer: (idle · empty)")
		fmt.Fprintln(out, "filter: (idle · empty)")
	}
	fmt.Fprintf(out, "stats: loops=%d fetched=%d ingested=%d skipped=%d acked=%d errors=%d\n",
		st.Stats.Loops, st.Stats.Fetched, st.Stats.Ingested, st.Stats.Skipped, st.Stats.Acked, st.Stats.Errors)
	if le := strings.TrimSpace(st.LastError); le != "" {
		fmt.Fprintf(out, "last_error: %s\n", le)
	} else if le := strings.TrimSpace(st.Stats.LastError); le != "" {
		fmt.Fprintf(out, "last_error: %s\n", le)
	} else {
		fmt.Fprintln(out, "last_error: (none)")
	}
	if !st.Running {
		fmt.Fprintln(out, "note: idle · not invent Connected / Memory GA · start with /setup pull start after mesh + pull_consumer")
	} else {
		fmt.Fprintln(out, "note: pull running ≠ invent Connected / Ops Pack GA / Memory GA")
	}
	fmt.Fprintln(out, setupPullHonesty)
	for _, line := range setup.SetupPullNextStepLines() {
		fmt.Fprintln(out, line)
	}
}

// handleSetupPullStart loads [memory] pull_* and starts continuous or once pull.
func handleSetupPullStart(out io.Writer, rt runtimeAdapter, args []string, once bool) {
	if rt.rt == nil {
		fmt.Fprintln(out, "setup pull start: no agent runtime")
		fmt.Fprintln(out, setupPullHonesty)
		return
	}
	cfgPath := ""
	forceOnce := once
	dryRun := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--config" && i+1 < len(args):
			i++
			cfgPath = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--config="):
			cfgPath = strings.TrimSpace(strings.TrimPrefix(a, "--config="))
		case a == "--once":
			forceOnce = true
		case a == "--dry-run" || a == "--dry_run":
			dryRun = true
		case a == "help" || a == "?":
			fmt.Fprintln(out, setupPullHelp())
			return
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(out, "setup pull start: unknown flag %q\n%s\n", a, setupPullHelp())
			return
		default:
			fmt.Fprintf(out, "setup pull start: unexpected arg %q\n%s\n", a, setupPullHelp())
			return
		}
	}
	if cfgPath == "" {
		cfgPath = rt.processConfigPath()
	}
	var (
		cfg *config.Config
		err error
	)
	if cfgPath != "" {
		cfg, err = config.Load(cfgPath)
	} else {
		cfg, err = config.LoadUser()
	}
	if err != nil {
		fmt.Fprintf(out, "setup pull start: config: %v\n", err)
		fmt.Fprintln(out, setupPullHonesty)
		return
	}
	pullCfg := continuousPullConfigFromMemory(cfg, dryRun)
	// Slash is explicit opt-in even when pull_continuous=false in config.
	pullCfg.Enabled = true

	label := "start"
	if forceOnce {
		label = "once"
		err = rt.rt.StartContinuousMemoryPullOnce(pullCfg)
	} else {
		err = rt.rt.StartContinuousMemoryPull(pullCfg)
	}
	if err != nil {
		fmt.Fprintf(out, "setup pull %s: %v\n", label, err)
		fmt.Fprintln(out, setupPullHonesty)
		return
	}
	mode := "continuous (MaxLoops=0)"
	if forceOnce {
		mode = "once (MaxLoops=1)"
	}
	if dryRun {
		mode += " · dry-run"
	}
	fmt.Fprintf(out, "setup pull %s: started %s\n", label, mode)
	fmt.Fprintf(out, "stream=%s consumer=%s filter=%q batch=%d max_wait_ms=%d server=%s\n",
		pullCfg.Stream, pullCfg.Consumer, pullCfg.Filter, pullCfg.Batch, pullCfg.MaxWaitMS, pullCfg.Server)
	fmt.Fprintln(out, "note: pull running ≠ invent Connected · dual_write OFF · not Memory GA")
	fmt.Fprintln(out, setupPullHonesty)
	for _, line := range setup.SetupPullNextStepLines() {
		fmt.Fprintln(out, line)
	}
}

// continuousPullConfigFromMemory maps [memory] pull_* into agent.ContinuousPullConfig.
// Default stream EVENTS when empty (matches CLI iomesh memory pull).
func continuousPullConfigFromMemory(cfg *config.Config, dryRun bool) agent.ContinuousPullConfig {
	if cfg == nil {
		return agent.ContinuousPullConfig{Stream: "EVENTS", DryRun: dryRun, Enabled: true}
	}
	stream := strings.TrimSpace(cfg.Memory.PullStream)
	if stream == "" {
		stream = "EVENTS"
	}
	server := strings.TrimSpace(cfg.Memory.Server)
	batch := cfg.Memory.PullBatch
	maxWait := cfg.Memory.PullMaxWaitMS
	return agent.ContinuousPullConfig{
		Enabled:   true, // slash path is explicit opt-in
		Stream:    stream,
		Consumer:  strings.TrimSpace(cfg.Memory.PullConsumer),
		Filter:    strings.TrimSpace(cfg.Memory.PullFilter),
		Batch:     batch,
		MaxWaitMS: maxWait,
		DryRun:    dryRun,
		Server:    server,
		Tenant:    strings.TrimSpace(cfg.Memory.Tenant),
	}
}

// handleSetupPullStop cancels in-session continuous pull (no-op when idle).
func handleSetupPullStop(out io.Writer, rt runtimeAdapter) {
	if rt.rt == nil {
		fmt.Fprintln(out, "setup pull stop: no agent runtime")
		fmt.Fprintln(out, setupPullHonesty)
		return
	}
	was := rt.rt.ContinuousMemoryPullStatus().Running
	rt.rt.StopContinuousMemoryPull()
	if was {
		fmt.Fprintln(out, "setup pull stop: stopped")
	} else {
		fmt.Fprintln(out, "setup pull stop: not running (no-op)")
	}
	fmt.Fprintln(out, setupPullHonesty)
	for _, line := range setup.SetupPullNextStepLines() {
		fmt.Fprintln(out, line)
	}
}

// setupAnalyzeHonesty is printed on every /setup analyze output (s1534 P6 residual honesty).
const setupAnalyzeHonesty = "honesty: dual_write OFF · not Memory GA · analyze tick ≠ invent Connected · /memory digest still valid"

// handleSetupAnalyze dispatches /setup analyze [status|start|once|stop] (s1534 P6).
// Bare /setup analyze → status. Residual-honest: dual_write OFF · not Memory GA ·
// analyze tick ≠ invent Connected · /memory digest still valid.
func handleSetupAnalyze(out io.Writer, rt runtimeAdapter, args []string) {
	sub := "status"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = strings.ToLower(strings.TrimSpace(args[0]))
		rest = args[1:]
	}
	switch sub {
	case "status", "st", "":
		handleSetupAnalyzeStatus(out, rt)
	case "start":
		handleSetupAnalyzeStart(out, rt, rest, false)
	case "once":
		handleSetupAnalyzeStart(out, rt, rest, true)
	case "stop":
		handleSetupAnalyzeStop(out, rt)
	case "help", "?":
		fmt.Fprintln(out, setupAnalyzeHelp())
	default:
		fmt.Fprintf(out, "setup analyze: unknown subcommand %q\n%s\n", sub, setupAnalyzeHelp())
	}
}

func setupAnalyzeHelp() string {
	return strings.TrimSpace(`usage: /setup analyze [status|start|once|stop]
  status   residual-honest analyze tick snapshot (default; bare /setup analyze)
  start    start continuous analyze ticks (loads [memory] analyze_*; --mode status|digest · --interval N · --window day|week · --config path)
  once     single analyze tick then exit (same knobs as start)
  stop     cancel in-session analyze tick loop (no-op when idle)
honesty: dual_write OFF · not Memory GA · analyze tick ≠ invent Connected · /memory digest still valid
  opt-in only · analyze_continuous=true is config opt-in · setup fragment defaults analyze_continuous=false`)
}

// handleSetupAnalyzeStatus prints AnalyzeTickStatus residual-honest (idle ≠ green).
func handleSetupAnalyzeStatus(out io.Writer, rt runtimeAdapter) {
	if rt.rt == nil {
		fmt.Fprintln(out, "setup analyze status: no agent runtime")
		fmt.Fprintln(out, "running: false")
		fmt.Fprintln(out, setupAnalyzeHonesty)
		return
	}
	st := rt.rt.AnalyzeTickStatus()
	fmt.Fprintln(out, "setup analyze status (residual-honest · s1534 P6)")
	fmt.Fprintf(out, "running: %v\n", st.Running)
	if st.Mode != "" || st.IntervalSec > 0 || st.TickCount > 0 {
		fmt.Fprintf(out, "mode: %s\n", st.Mode)
		fmt.Fprintf(out, "interval_sec: %d\n", st.IntervalSec)
		fmt.Fprintf(out, "tick_count: %d\n", st.TickCount)
	} else if !st.Running {
		fmt.Fprintln(out, "mode: (idle · empty)")
		fmt.Fprintln(out, "interval_sec: (idle · 0)")
		fmt.Fprintln(out, "tick_count: 0")
	}
	if !st.LastAt.IsZero() {
		fmt.Fprintf(out, "last_at: %s\n", st.LastAt.Format(time.RFC3339))
	} else {
		fmt.Fprintln(out, "last_at: (none)")
	}
	if sum := strings.TrimSpace(st.LastSummary); sum != "" {
		fmt.Fprintf(out, "last_summary: %s\n", sum)
	} else {
		fmt.Fprintln(out, "last_summary: (none)")
	}
	if le := strings.TrimSpace(st.LastError); le != "" {
		fmt.Fprintf(out, "last_error: %s\n", le)
	} else {
		fmt.Fprintln(out, "last_error: (none)")
	}
	if !st.Running {
		fmt.Fprintln(out, "note: idle · not invent Connected / Memory GA · start with /setup analyze start · /memory digest still valid")
	} else {
		fmt.Fprintln(out, "note: analyze running ≠ invent Connected / Ops Pack GA / Memory GA · /memory digest still valid")
	}
	fmt.Fprintln(out, setupAnalyzeHonesty)
	for _, line := range setup.SetupAnalyzeNextStepLines() {
		fmt.Fprintln(out, line)
	}
}

// handleSetupAnalyzeStart loads [memory] analyze_* + flags and starts continuous or once tick.
func handleSetupAnalyzeStart(out io.Writer, rt runtimeAdapter, args []string, once bool) {
	if rt.rt == nil {
		fmt.Fprintln(out, "setup analyze start: no agent runtime")
		fmt.Fprintln(out, setupAnalyzeHonesty)
		return
	}
	cfgPath := ""
	forceOnce := once
	modeFlag := ""
	intervalFlag := 0
	windowFlag := ""
	horizonFlag := ""
	limitFlag := 0
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--config" && i+1 < len(args):
			i++
			cfgPath = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--config="):
			cfgPath = strings.TrimSpace(strings.TrimPrefix(a, "--config="))
		case a == "--mode" && i+1 < len(args):
			i++
			modeFlag = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--mode="):
			modeFlag = strings.TrimSpace(strings.TrimPrefix(a, "--mode="))
		case a == "--interval" && i+1 < len(args):
			i++
			n, err := strconv.Atoi(strings.TrimSpace(args[i]))
			if err != nil {
				fmt.Fprintf(out, "setup analyze start: --interval requires int seconds: %v\n", err)
				fmt.Fprintln(out, setupAnalyzeHonesty)
				return
			}
			intervalFlag = n
		case strings.HasPrefix(a, "--interval="):
			n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(a, "--interval=")))
			if err != nil {
				fmt.Fprintf(out, "setup analyze start: --interval requires int seconds: %v\n", err)
				fmt.Fprintln(out, setupAnalyzeHonesty)
				return
			}
			intervalFlag = n
		case a == "--window" && i+1 < len(args):
			i++
			windowFlag = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--window="):
			windowFlag = strings.TrimSpace(strings.TrimPrefix(a, "--window="))
		case a == "--horizon" && i+1 < len(args):
			i++
			horizonFlag = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--horizon="):
			horizonFlag = strings.TrimSpace(strings.TrimPrefix(a, "--horizon="))
		case a == "--limit" && i+1 < len(args):
			i++
			n, err := strconv.Atoi(strings.TrimSpace(args[i]))
			if err != nil {
				fmt.Fprintf(out, "setup analyze start: --limit requires int: %v\n", err)
				fmt.Fprintln(out, setupAnalyzeHonesty)
				return
			}
			limitFlag = n
		case strings.HasPrefix(a, "--limit="):
			n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(a, "--limit=")))
			if err != nil {
				fmt.Fprintf(out, "setup analyze start: --limit requires int: %v\n", err)
				fmt.Fprintln(out, setupAnalyzeHonesty)
				return
			}
			limitFlag = n
		case a == "--once":
			forceOnce = true
		case a == "help" || a == "?":
			fmt.Fprintln(out, setupAnalyzeHelp())
			return
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(out, "setup analyze start: unknown flag %q\n%s\n", a, setupAnalyzeHelp())
			return
		default:
			fmt.Fprintf(out, "setup analyze start: unexpected arg %q\n%s\n", a, setupAnalyzeHelp())
			return
		}
	}
	if cfgPath == "" {
		cfgPath = rt.processConfigPath()
	}
	var (
		cfg *config.Config
		err error
	)
	if cfgPath != "" {
		cfg, err = config.Load(cfgPath)
	} else {
		cfg, err = config.LoadUser()
	}
	if err != nil {
		fmt.Fprintf(out, "setup analyze start: config: %v\n", err)
		fmt.Fprintln(out, setupAnalyzeHonesty)
		return
	}
	tickCfg := analyzeTickConfigFromMemory(cfg)
	// Slash is explicit opt-in even when analyze_continuous=false in config.
	tickCfg.Enabled = true
	if modeFlag != "" {
		tickCfg.Mode = modeFlag
	}
	if intervalFlag > 0 {
		tickCfg.IntervalSec = intervalFlag
	}
	if windowFlag != "" {
		tickCfg.Window = windowFlag
	}
	if horizonFlag != "" {
		tickCfg.Horizon = horizonFlag
	}
	if limitFlag > 0 {
		tickCfg.Limit = limitFlag
	}

	label := "start"
	if forceOnce {
		label = "once"
		err = rt.rt.StartAnalyzeTickOnce(tickCfg)
	} else {
		err = rt.rt.StartAnalyzeTick(tickCfg)
	}
	if err != nil {
		fmt.Fprintf(out, "setup analyze %s: %v\n", label, err)
		fmt.Fprintln(out, setupAnalyzeHonesty)
		return
	}
	mode := "continuous"
	if forceOnce {
		mode = "once"
	}
	fmt.Fprintf(out, "setup analyze %s: started %s mode=%s interval_sec=%d window=%q\n",
		label, mode, tickCfg.Mode, tickCfg.IntervalSec, tickCfg.Window)
	fmt.Fprintln(out, "note: analyze running ≠ invent Connected · dual_write OFF · not Memory GA · /memory digest still valid")
	fmt.Fprintln(out, setupAnalyzeHonesty)
	for _, line := range setup.SetupAnalyzeNextStepLines() {
		fmt.Fprintln(out, line)
	}
}

// analyzeTickConfigFromMemory maps [memory] analyze_* into agent.AnalyzeTickConfig.
// Default mode "status" when empty (matches Runtime startAnalyzeTick).
func analyzeTickConfigFromMemory(cfg *config.Config) agent.AnalyzeTickConfig {
	if cfg == nil {
		return agent.AnalyzeTickConfig{Enabled: true, Mode: "status"}
	}
	mode := strings.TrimSpace(cfg.Memory.AnalyzeMode)
	if mode == "" {
		mode = "status"
	}
	return agent.AnalyzeTickConfig{
		Enabled:     true, // slash path is explicit opt-in
		IntervalSec: cfg.Memory.AnalyzeIntervalSec,
		Mode:        mode,
	}
}

// handleSetupAnalyzeStop cancels in-session analyze ticks (no-op when idle).
func handleSetupAnalyzeStop(out io.Writer, rt runtimeAdapter) {
	if rt.rt == nil {
		fmt.Fprintln(out, "setup analyze stop: no agent runtime")
		fmt.Fprintln(out, setupAnalyzeHonesty)
		return
	}
	was := rt.rt.AnalyzeTickStatus().Running
	rt.rt.StopAnalyzeTick()
	if was {
		fmt.Fprintln(out, "setup analyze stop: stopped")
	} else {
		fmt.Fprintln(out, "setup analyze stop: not running (no-op)")
	}
	fmt.Fprintln(out, setupAnalyzeHonesty)
	for _, line := range setup.SetupAnalyzeNextStepLines() {
		fmt.Fprintln(out, line)
	}
}

// handleSetupDrift prints residual-honest FormatDriftText(BuildDriftReport(...)).
// /setup maintain is an alias (report-only residual next steps · no auto-repair).
func handleSetupDrift(out io.Writer, rt runtimeAdapter, args []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "help" || a == "?" {
			fmt.Fprintln(out, setupDriftHelp())
			return
		}
	}
	cfgPath := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--config" && i+1 < len(args):
			i++
			cfgPath = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--config="):
			cfgPath = strings.TrimSpace(strings.TrimPrefix(a, "--config="))
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(out, "setup drift: unknown flag %q\n%s\n", a, setupDriftHelp())
			return
		default:
			// tolerate unknown tokens lightly (aliases already consumed at switch)
		}
	}
	if cfgPath == "" {
		cfgPath = rt.processConfigPath()
	}
	var (
		cfg *config.Config
		err error
	)
	if cfgPath != "" {
		cfg, err = config.Load(cfgPath)
	} else {
		cfg, err = config.LoadUser()
	}
	// Fail-open: continue with nil cfg so report remains residual-honest (config absent).
	if err != nil {
		fmt.Fprintf(out, "setup drift: config note: %v (continuing with empty config intent)\n", err)
		cfg = nil
	}
	var snap setup.DriftSnapshot
	if rt.rt != nil {
		as := rt.rt.DriftSnapshot()
		snap = setup.DriftSnapshot{
			MCPAttached:    as.MCPAttached,
			MCPServerCount: as.MCPServerCount,
			MemoryServerOK: as.MemoryServerOK,
			MemoryServer:   as.MemoryServer,
			MeshEnabled:    as.MeshEnabled,
			PullRunning:    as.PullRunning,
			PullConsumer:   as.PullConsumer,
			AnalyzeRunning: as.AnalyzeRunning,
			DualWrite:      as.DualWrite,
			MemoryEnabled:  as.MemoryEnabled,
		}
	}
	// nil/no runtime → zero snap residual-honest (not invent green).
	rep := setup.BuildDriftReport(cfg, snap)
	// FormatDriftText always includes DriftHonestyFooter (dual_write OFF · package wire ≠ Connected).
	fmt.Fprint(out, setup.FormatDriftText(rep))
}

func setupDriftHelp() string {
	return strings.TrimSpace(`usage: /setup drift [--config path]
  report-only config intent vs runtime snapshot (alias: /setup maintain)
  residual next-steps notes · guided repair via /setup repair (plan · apply --yes)
honesty: dual_write OFF · not Memory GA · drift report ≠ invent install green · package wire ≠ Connected`)
}

// setupRepairHonesty is printed on every /setup repair output (s1538 P7 residual honesty).
const setupRepairHonesty = "honesty: dual_write OFF · not Memory GA · repair apply ≠ invent Connected · package wire ≠ Connected · portal HITL still human"

// handleSetupRepair dispatches /setup repair [plan|apply] (s1538 P7).
// Bare /setup repair → plan only (FormatRepairPlan from current drift).
// apply requires --yes; refuse without --yes (no auto-repair).
// Residual-honest: repair apply ≠ invent Connected · dual_write OFF · portal HITL still human.
func handleSetupRepair(out io.Writer, rt runtimeAdapter, args []string) {
	sub := "plan"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = strings.ToLower(strings.TrimSpace(args[0]))
		rest = args[1:]
	}
	switch sub {
	case "plan", "dry-run", "dryrun", "dry_run", "":
		handleSetupRepairPlan(out, rt, rest)
	case "apply":
		handleSetupRepairApply(out, rt, rest)
	case "help", "?":
		fmt.Fprintln(out, setupRepairHelp())
	default:
		fmt.Fprintf(out, "setup repair: unknown subcommand %q\n%s\n", sub, setupRepairHelp())
	}
}

func setupRepairHelp() string {
	return strings.TrimSpace(`usage: /setup repair [plan|apply] [--config path] [--yes]
  plan    residual-honest repair plan from current drift (default; bare /setup repair)
  apply   apply safe steps only; requires --yes (refuse without --yes · no auto-repair)
honesty: dual_write OFF · not Memory GA · repair apply ≠ invent Connected · package wire ≠ Connected · portal HITL still human
  safe steps only (reload_mcp · start_pull · start_analyze) · notes for human host/mesh/dual_write
  dual_write never auto-flipped ON · apply success ≠ invent Connected / Memory GA`)
}

// handleSetupRepairPlan prints FormatRepairPlan(PlanRepair(drift)) residual-honest (no side effects).
func handleSetupRepairPlan(out io.Writer, rt runtimeAdapter, args []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "help" || a == "?" {
			fmt.Fprintln(out, setupRepairHelp())
			return
		}
	}
	cfg, _, loadNote := loadSetupConfig(rt, args)
	if loadNote != "" {
		fmt.Fprintln(out, loadNote)
	}
	rep := buildSetupDriftReport(rt, cfg)
	plan := setup.PlanRepair(rep)
	// Plan path is no-side-effects; FormatRepairPlan always includes RepairHonestyFooter.
	fmt.Fprint(out, setup.FormatRepairPlan(plan))
}

// handleSetupRepairApply applies safe steps only when --yes is present.
// Without --yes: residual-honest refuse (no auto-repair).
func handleSetupRepairApply(out io.Writer, rt runtimeAdapter, args []string) {
	yes := false
	cfgPath := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "help" || a == "?":
			fmt.Fprintln(out, setupRepairHelp())
			return
		case a == "--yes" || a == "-y" || a == "--confirm":
			yes = true
		case a == "--config" && i+1 < len(args):
			i++
			cfgPath = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "--config="):
			cfgPath = strings.TrimSpace(strings.TrimPrefix(a, "--config="))
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(out, "setup repair apply: unknown flag %q\n%s\n", a, setupRepairHelp())
			return
		default:
			fmt.Fprintf(out, "setup repair apply: unexpected arg %q\n%s\n", a, setupRepairHelp())
			return
		}
	}
	if !yes {
		fmt.Fprintln(out, "setup repair apply: refuse without --yes (explicit opt-in · no auto-repair)")
		fmt.Fprintln(out, "hint: /setup repair plan to preview · /setup repair apply --yes for safe steps only")
		fmt.Fprintln(out, setupRepairHonesty)
		return
	}
	if cfgPath == "" {
		cfgPath = rt.processConfigPath()
	}
	// Load config (slash --config, else process path, else user).
	var (
		cfg *config.Config
		err error
	)
	if cfgPath != "" {
		cfg, err = config.Load(cfgPath)
	} else {
		cfg, err = config.LoadUser()
	}
	if err != nil {
		fmt.Fprintf(out, "setup repair apply: config note: %v (continuing with empty config intent)\n", err)
		cfg = nil
	}
	rep := buildSetupDriftReport(rt, cfg)
	plan := setup.PlanRepair(rep)
	ex := setupRepairExecutor{rt: rt, cfg: cfg}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	result := setup.ApplyRepairPlan(ctx, plan, ex, false /* dryRun */)
	fmt.Fprint(out, setup.FormatRepairResult(result))
	// Optionally re-print residual-honest drift summary after apply.
	after := buildSetupDriftReport(rt, cfg)
	fmt.Fprintln(out, "--- post-apply drift (residual-honest · ≠ invent install green) ---")
	fmt.Fprint(out, setup.FormatDriftText(after))
}

// loadSetupConfig parses optional --config from args and loads that path,
// else the process config path, else user/IOMESH_CONFIG. Fail-open: returns
// nil cfg + note on load error (residual-honest empty intent).
func loadSetupConfig(rt runtimeAdapter, args []string) (cfg *config.Config, cfgPath string, note string) {
	cfgPath = setupProbeConfigPath(args, rt.processConfigPath())
	var err error
	if cfgPath != "" {
		cfg, err = config.Load(cfgPath)
	} else {
		cfg, err = config.LoadUser()
	}
	if err != nil {
		return nil, cfgPath, fmt.Sprintf("setup repair: config note: %v (continuing with empty config intent)", err)
	}
	return cfg, cfgPath, ""
}

// buildSetupDriftReport maps runtime DriftSnapshot + cfg into setup.DriftReport.
// nil/no runtime → zero snap residual-honest (not invent green).
func buildSetupDriftReport(rt runtimeAdapter, cfg *config.Config) setup.DriftReport {
	var snap setup.DriftSnapshot
	if rt.rt != nil {
		as := rt.rt.DriftSnapshot()
		snap = setup.DriftSnapshot{
			MCPAttached:    as.MCPAttached,
			MCPServerCount: as.MCPServerCount,
			MemoryServerOK: as.MemoryServerOK,
			MemoryServer:   as.MemoryServer,
			MeshEnabled:    as.MeshEnabled,
			PullRunning:    as.PullRunning,
			PullConsumer:   as.PullConsumer,
			AnalyzeRunning: as.AnalyzeRunning,
			DualWrite:      as.DualWrite,
			MemoryEnabled:  as.MemoryEnabled,
		}
	}
	return setup.BuildDriftReport(cfg, snap)
}

// setupRepairExecutor implements setup.RepairExecutor wrapping runtimeAdapter.
// ReloadMCP mirrors handleSetupReload (skills re-scan + MCP hot-swap); StartPull/StartAnalyze use config knobs.
type setupRepairExecutor struct {
	rt  runtimeAdapter
	cfg *config.Config
}

func (e setupRepairExecutor) ReloadMCP(ctx context.Context) error {
	if e.rt.rt == nil {
		return fmt.Errorf("no agent runtime")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// Same path as /setup reload: Wire + ReplaceSkills + ConnectMCP + ReplaceMCP (s1670).
	// package wire ≠ Connected · skills re-scan ≠ invent Connected · silent (repair apply prints plan result).
	reloadRuntimeFromConfig(ctx, nil, e.rt, e.cfg, false)
	return nil
}

func (e setupRepairExecutor) StartPull(ctx context.Context) error {
	if e.rt.rt == nil {
		return fmt.Errorf("no agent runtime")
	}
	_ = ctx
	pullCfg := continuousPullConfigFromMemory(e.cfg, false)
	pullCfg.Enabled = true
	return e.rt.rt.StartContinuousMemoryPull(pullCfg)
}

func (e setupRepairExecutor) StartAnalyze(ctx context.Context) error {
	if e.rt.rt == nil {
		return fmt.Errorf("no agent runtime")
	}
	_ = ctx
	tickCfg := analyzeTickConfigFromMemory(e.cfg)
	tickCfg.Enabled = true
	return e.rt.rt.StartAnalyzeTick(tickCfg)
}

// handleSetupPreflight runs residual-honest setup.Preflight for /setup preflight|status|check.
// Bare /setup preflight inherits the process --config / IOMESH_CONFIG path; slash --config overrides.
func handleSetupPreflight(out io.Writer, rt runtimeAdapter, args []string) {
	cfgPath := setupProbeConfigPath(args, rt.processConfigPath())
	rep, err := setup.Preflight(context.Background(), cfgPath)
	if err != nil {
		fmt.Fprintf(out, "setup preflight: %v\n", err)
		return
	}
	fmt.Fprint(out, setup.FormatPreflightText(rep))
}

// parseIntegrationsListArgs extracts optional --layer for /integrations list (s1238).
// Supports: --layer operational|knowledge|analytical|all (also --mesh-layer / --mesh_layer).
// Returns errMsg when a flag is malformed or the layer value is invalid.
func parseIntegrationsListArgs(args []string) (layer string, errMsg string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--layer", "--mesh-layer", "--mesh_layer", "-l":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			v := strings.ToLower(strings.TrimSpace(val))
			switch v {
			case "operational", "knowledge", "analytical", "all", "":
				layer = v
			default:
				return "", "invalid --layer (operational|knowledge|analytical)"
			}
		default:
			if strings.HasPrefix(a, "-") {
				return "", "unknown flag " + a
			}
			// bare layer token: /integrations list knowledge
			v := strings.ToLower(strings.TrimSpace(a))
			switch v {
			case "operational", "knowledge", "analytical", "all":
				layer = v
			default:
				return "", "unexpected argument " + a
			}
		}
	}
	return layer, ""
}

// parseIntegrationsPlanArgs extracts connector_id for /integrations plan (s1238).
// Accepts: plan github | plan --id github | plan --connector-id=github
func parseIntegrationsPlanArgs(args []string) (connectorID string, errMsg string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--id", "--connector-id", "--connector_id":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			connectorID = strings.TrimSpace(val)
		default:
			if strings.HasPrefix(a, "-") {
				return "", "unknown flag " + a
			}
			if connectorID == "" {
				connectorID = strings.TrimSpace(a)
			} else {
				return "", "unexpected argument " + a
			}
		}
	}
	return connectorID, ""
}

// parseIntegrationsSigningArgs extracts optional mesh_layer or connector_id for
// /integrations signing (s1243). Accepts bare token, --layer, --id, --connector-id.
// Empty hint = full catalog. Layer values map to aion mesh_layer; other tokens are
// treated as connector_id client-side filters.
func parseIntegrationsSigningArgs(args []string) (hint string, errMsg string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--layer", "--mesh-layer", "--mesh_layer", "-l":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			v := strings.ToLower(strings.TrimSpace(val))
			switch v {
			case "operational", "knowledge", "analytical", "all", "":
				hint = v
			default:
				return "", "invalid --layer (operational|knowledge|analytical)"
			}
		case "--id", "--connector-id", "--connector_id":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			hint = strings.TrimSpace(val)
		default:
			if strings.HasPrefix(a, "-") {
				return "", "unknown flag " + a
			}
			if hint == "" {
				hint = strings.TrimSpace(a)
			} else {
				return "", "unexpected argument " + a
			}
		}
	}
	return hint, ""
}

// parseMemoryRecallArgs extracts optional temporal flags and the free-text query.
// Supports: --since RFC3339, --until RFC3339, --session-seq N (also --session_seq).
// Forms: --since=VALUE or --since VALUE. Remaining tokens join as the query (s1068).
func parseMemoryRecallArgs(args []string) (query string, opts agent.MemoryRecallOpts) {
	var qParts []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--since":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.Since = strings.TrimSpace(val)
		case "--until":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.Until = strings.TrimSpace(val)
		case "--session-seq", "--session_seq":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				opts.SessionSeq = n
				opts.SessionSeqSet = true
			}
		default:
			qParts = append(qParts, a)
		}
	}
	return strings.Join(qParts, " "), opts
}

// parseMemoryDigestArgs extracts ops digest flags (s1200 + #373 require-sources).
// Supports: --window day|week, --horizon ops|knowledge|analytical|all, --limit N,
// --as-of RFC3339, --require-sources mesh,private (cite-both or explicit miss).
// Returns errMsg when a flag is malformed or values are invalid.
func parseMemoryDigestArgs(args []string) (opts agent.MemoryOpsDigestOpts, errMsg string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--window", "-w":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			v := strings.ToLower(strings.TrimSpace(val))
			if v != "day" && v != "week" {
				return opts, "invalid --window (day|week)"
			}
			opts.Window = v
		case "--horizon", "-h":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			v := strings.ToLower(strings.TrimSpace(val))
			switch v {
			case "ops", "knowledge", "analytical", "all":
				opts.Horizon = v
			default:
				return opts, "invalid --horizon (ops|knowledge|analytical|all)"
			}
		case "--limit":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil || n < 0 {
				return opts, "invalid --limit"
			}
			opts.Limit = n
		case "--as-of", "--as_of":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.AsOf = strings.TrimSpace(val)
		case "--require-sources", "--require_sources":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			srcs, perr := agent.ParseRequireSourcesList(val)
			if perr != "" {
				return opts, perr
			}
			opts.RequireSources = srcs
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			return opts, "unexpected argument " + a
		}
	}
	return opts, ""
}

// parseMemorySemanticArgs extracts tier-4 semantic search flags (s1301).
// Supports: --query / -q, --limit. Remaining free tokens append to query.
// Returns errMsg when a flag is malformed.
func parseMemorySemanticArgs(args []string) (opts agent.MemorySemanticOpts, errMsg string) {
	var qParts []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--query", "-q":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			if v := strings.TrimSpace(val); v != "" {
				qParts = append(qParts, v)
			}
		case "--limit":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil || n < 0 {
				return opts, "invalid --limit"
			}
			opts.Limit = n
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			qParts = append(qParts, a)
		}
	}
	if len(qParts) > 0 {
		opts.Query = strings.Join(qParts, " ")
	}
	return opts, ""
}

// parseMemoryIngestEventArgs extracts ops/telemetry event ingest flags (s1301).
// Supports: --subject / -s (required by caller), --content / -c (required by caller),
// --event-time / --event_time, --session-id / --session_id, --session-seq / --session_seq,
// --severity, --source-stream / --source_stream.
// Rejects unknown flags / bare free tokens (no free-form content without --content).
func parseMemoryIngestEventArgs(args []string) (opts agent.MemoryIngestEventOpts, errMsg string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--subject", "-s":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.Subject = strings.TrimSpace(val)
		case "--content", "-c":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.Content = strings.TrimSpace(val)
		case "--event-time", "--event_time":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.EventTime = strings.TrimSpace(val)
		case "--session-id", "--session_id":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.SessionID = strings.TrimSpace(val)
		case "--session-seq", "--session_seq":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil {
				return opts, "invalid --session-seq"
			}
			opts.SessionSeq = n
		case "--severity":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.Severity = strings.TrimSpace(val)
		case "--source-stream", "--source_stream":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.SourceStream = strings.TrimSpace(val)
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			return opts, "unexpected argument " + a
		}
	}
	return opts, ""
}

// parseMemoryIngestDirArgs extracts folder ingest flags (#384).
// Supports: --dir / --path, --dry-run / --dry_run, --limit.
// First non-flag token is the directory path.
func parseMemoryIngestDirArgs(args []string) (opts agent.MemoryIngestDirOpts, errMsg string) {
	var pathParts []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--dir", "--path":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.Path = strings.TrimSpace(val)
		case "--dry-run", "--dry_run":
			opts.DryRun = true
		case "--limit":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil || n < 0 {
				return opts, "invalid --limit"
			}
			opts.Limit = n
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			pathParts = append(pathParts, a)
		}
	}
	if strings.TrimSpace(opts.Path) == "" && len(pathParts) > 0 {
		opts.Path = strings.Join(pathParts, " ")
	}
	return opts, ""
}

// parseMemoryTimelineArgs extracts temporal timeline flags (s1296).
// Supports: --since, --until, --session-id / --session_id, --query / -q, --limit.
// Remaining free tokens append to query. Returns errMsg when a flag is malformed.
func parseMemoryTimelineArgs(args []string) (opts agent.MemoryTimelineOpts, errMsg string) {
	var qParts []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--since":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.Since = strings.TrimSpace(val)
		case "--until":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.Until = strings.TrimSpace(val)
		case "--session-id", "--session_id":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.SessionID = strings.TrimSpace(val)
		case "--query", "-q":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			if v := strings.TrimSpace(val); v != "" {
				qParts = append(qParts, v)
			}
		case "--limit":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil || n < 0 {
				return opts, "invalid --limit"
			}
			opts.Limit = n
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			qParts = append(qParts, a)
		}
	}
	if len(qParts) > 0 {
		joined := strings.Join(qParts, " ")
		if opts.Query != "" {
			opts.Query = strings.TrimSpace(opts.Query + " " + joined)
		} else {
			opts.Query = joined
		}
	}
	return opts, ""
}

// parseMemoryPatternsArgs extracts ops-pulse patterns flags (s1287).
// Supports: --limit N. Rejects unknown flags / bare args.
func parseMemoryPatternsArgs(args []string) (opts agent.MemoryPatternsOpts, errMsg string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--limit":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil || n < 0 {
				return opts, "invalid --limit"
			}
			opts.Limit = n
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			return opts, "unexpected argument " + a
		}
	}
	return opts, ""
}

// parseMemoryAnomaliesArgs extracts ops-pulse anomalies flags (s1287).
// Supports: --limit N. Rejects unknown flags / bare args.
func parseMemoryAnomaliesArgs(args []string) (opts agent.MemoryAnomaliesOpts, errMsg string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--limit":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil || n < 0 {
				return opts, "invalid --limit"
			}
			opts.Limit = n
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			return opts, "unexpected argument " + a
		}
	}
	return opts, ""
}

// parseMemorySupersedeArgs extracts A3 lite supersede HITL flags (s1282).
// Supports: --entity / -e (required by caller), --as-of / --as_of (optional RFC3339),
// --i-confirm / --confirm / --yes → Confirm=true. Rejects unknown flags.
// Missing confirm parses cleanly with Confirm=false (MemorySupersede refuses residual-honestly).
func parseMemorySupersedeArgs(args []string) (opts agent.MemorySupersedeOpts, errMsg string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--entity", "-e":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.Entity = strings.TrimSpace(val)
		case "--as-of", "--as_of":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.AsOf = strings.TrimSpace(val)
		case "--i-confirm", "--confirm", "--yes":
			// Boolean HITL flags — value optional (=true form accepted, ignored).
			opts.Confirm = true
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			return opts, "unexpected argument " + a
		}
	}
	return opts, ""
}

// parseMemoryTriggerCompactArgs extracts RecMem trigger-compact HITL flags (s1311).
// Supports: --i-confirm / --confirm / --yes → Confirm=true. Rejects unknown flags / bare args.
// Missing confirm parses cleanly with Confirm=false (MemoryTriggerCompact refuses residual-honestly).
func parseMemoryTriggerCompactArgs(args []string) (opts agent.MemoryTriggerCompactOpts, errMsg string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, _, _ := splitFlagKV(a)
		switch key {
		case "--i-confirm", "--confirm", "--yes":
			// Boolean HITL flags — value optional (=true form accepted, ignored).
			opts.Confirm = true
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			return opts, "unexpected argument " + a
		}
	}
	return opts, ""
}

// parseMemoryFactsAsOfArgs extracts bi-temporal lite validity flags (s1276).
// Supports: --as-of / --as_of (required), --entity, --query / -q, --limit, --session-id / --session_id.
// Remaining free tokens append to query. Returns errMsg when a flag is malformed.
func parseMemoryFactsAsOfArgs(args []string) (opts agent.MemoryFactsAsOfOpts, errMsg string) {
	var qParts []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--as-of", "--as_of":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.AsOf = strings.TrimSpace(val)
		case "--entity", "-e":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.Entity = strings.TrimSpace(val)
		case "--query", "-q":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			if v := strings.TrimSpace(val); v != "" {
				qParts = append(qParts, v)
			}
		case "--limit":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil || n < 0 {
				return opts, "invalid --limit"
			}
			opts.Limit = n
		case "--session-id", "--session_id":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			opts.SessionID = strings.TrimSpace(val)
		default:
			if strings.HasPrefix(a, "-") {
				return opts, "unknown flag " + a
			}
			qParts = append(qParts, a)
		}
	}
	if len(qParts) > 0 {
		joined := strings.Join(qParts, " ")
		if opts.Query != "" {
			opts.Query = strings.TrimSpace(opts.Query + " " + joined)
		} else {
			opts.Query = joined
		}
	}
	return opts, ""
}

// parseMemoryRelatedArgs extracts multi-hop related flags (s1135 + s1281).
// Supports: --seed / --seed-entity, --query, --max-hops / --max_hops, --limit,
// --prefer-shorter-hops / --prefer_shorter_hops (optional true/false; bare = true),
// --no-prefer-shorter-hops / --legacy-sort → PreferShorterHops=false (legacy seed-first).
// Remaining free tokens append to query. Returns errMsg when a flag is malformed.
// Honesty: multi-hop lite ≠ full graph RAG · not Memory GA · dual_write OFF ·
// hop ranking path-aware lite · PreferShorterHops default true (nil = true).
func parseMemoryRelatedArgs(args []string) (seed, query string, opts agent.MemoryRelatedOpts, errMsg string) {
	var qParts []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		key, val, hasEq := splitFlagKV(a)
		switch key {
		case "--seed", "--seed-entity", "--seed_entity":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			seed = strings.TrimSpace(val)
		case "--query", "-q":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			if v := strings.TrimSpace(val); v != "" {
				qParts = append(qParts, v)
			}
		case "--max-hops", "--max_hops":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil || n < 0 {
				return "", "", opts, "invalid --max-hops"
			}
			opts.MaxHops = n
		case "--limit":
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					val = args[i]
				}
			}
			n, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil || n < 0 {
				return "", "", opts, "invalid --limit"
			}
			opts.Limit = n
		case "--prefer-shorter-hops", "--prefer_shorter_hops":
			// Optional value true/false; bare flag = true (s1281 / aion s1277).
			if !hasEq {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					// Only consume next token when it looks like a bool value.
					next := strings.ToLower(strings.TrimSpace(args[i+1]))
					if next == "true" || next == "false" || next == "1" || next == "0" {
						i++
						val = args[i]
						hasEq = true // treat as provided value below
					}
				}
			}
			if strings.TrimSpace(val) == "" {
				b := true
				opts.PreferShorterHops = &b
			} else {
				parsed, err := strconv.ParseBool(strings.TrimSpace(val))
				if err != nil {
					return "", "", opts, "invalid --prefer-shorter-hops (want true|false)"
				}
				opts.PreferShorterHops = &parsed
			}
		case "--no-prefer-shorter-hops", "--legacy-sort":
			// Legacy seed-first sort (PreferShorterHops=false).
			b := false
			opts.PreferShorterHops = &b
		default:
			qParts = append(qParts, a)
		}
	}
	return seed, strings.Join(qParts, " "), opts, ""
}

// splitFlagKV returns key and value for --flag=value forms; hasEq true when '=' present.
// For bare --flag, key is the token and val/hasEq are empty/false.
func splitFlagKV(tok string) (key, val string, hasEq bool) {
	if !strings.HasPrefix(tok, "--") && !strings.HasPrefix(tok, "-") {
		return tok, "", false
	}
	// Keep short -q as key for related parser; long flags use --.
	if i := strings.IndexByte(tok, '='); i > 0 {
		return tok[:i], tok[i+1:], true
	}
	return tok, "", false
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

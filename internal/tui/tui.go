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
			fmt.Fprintln(out, "usage: /memory [recall [--since|--until|--session-seq] [query] | related --seed <entity> [--query ...] [--max-hops N] [--prefer-shorter-hops|--legacy-sort] | digest [--window day|week] [--horizon ops|knowledge|analytical|all] [--limit N] | facts-as-of --as-of <RFC3339> [--entity ...] [--query ...] [--limit N] | timeline [--since|--until|--session-id|--query|--limit] | compact-status | trigger-compact --i-confirm | semantic [query|--query ...] [--limit N] | ingest-event --subject <id> --content <text> [--event-time|--session-id|--session-seq|--severity|--source-stream] | patterns [--limit N] | anomalies [--limit N] | supersede --entity <key> [--as-of RFC3339] --i-confirm | ingest <text> | status]")
			return false, nil
		}
		sub := strings.ToLower(parts[1])
		switch sub {
		case "status", "st":
			// s1311: base MemoryStatusLine + residual-honest advanced MCP inventory pulse.
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
			dopts, perr := parseMemoryDigestArgs(parts[2:])
			if perr != "" {
				fmt.Fprintf(out, "memory digest: %s\nusage: /memory digest [--window day|week] [--horizon ops|knowledge|analytical|all] [--limit N]\n", perr)
				return false, nil
			}
			text, err := rt.rt.MemoryOpsDigest(context.Background(), dopts)
			if err != nil {
				fmt.Fprintf(out, "memory digest: %v\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(empty digest)")
				return false, nil
			}
			fmt.Fprintln(out, text)
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
			// (also accepts --since/--until when first token is not a known subcommand)
			q, ropts := parseMemoryRecallArgs(parts[1:])
			text, err := rt.rt.MemoryRecallWithOpts(context.Background(), q, ropts)
			if err != nil {
				fmt.Fprintf(out, "memory: %v (try /memory status|recall|related|digest|facts-as-of|timeline|compact-status|trigger-compact|semantic|ingest-event|patterns|anomalies|supersede|ingest)\n", err)
				return false, nil
			}
			if strings.TrimSpace(text) == "" {
				fmt.Fprintln(out, "(no memories)")
				return false, nil
			}
			fmt.Fprintln(out, text)
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
			}
			// Unknown subcommand: still print guidance + usage hint for help/checklist.
			fmt.Fprintln(out, agent.GtmDraftOnlyAgentGuidanceNote())
			fmt.Fprintln(out, "— residual: drafts only · human publish · skill gtm-draft-only-agent via read_skill · dual_write OFF · not Memory GA")
			fmt.Fprintln(out, "usage: /gtm [help|checklist]  (aliases /gtm-draft /gtm-agent)")
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
		// next|after|continue|lanes → post-onboard operator lanes overview (plugins·gtm·memory·mesh·memory-pull·agentic·human-gates).
		// next <lane> (s1377): plugins|plugin|dogfood · gtm|drafts · memory|mcp|palace.
		// next mesh (s1402): mesh|stream|streams|heartbeat|heartbeats|pull (NOT pulse — pulse stays status board).
		// next memory-pull (s1407): memory-pull|ops-pack|pull-path|memorypull|ops_pack (NOT bare pull — pull stays mesh).
		// next agentic (s1417): agentic|agentic-integrations|integrations|portal-hitl|list-plan|hitl (NOT bare mcp/portal/pull).
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
				// s1417: agentic|agentic-integrations|integrations|portal-hitl|list-plan|hitl plane-3 agentic integrations;
				// s1413: human-gates|human|gates|apply-gates still-required vs offline residual.
				if len(parts) >= 3 {
					lane := strings.ToLower(parts[2])
					switch lane {
					case "plugins", "plugin", "dogfood":
						fmt.Fprintln(out, agent.AionAgentOnboardingNextPluginsLane())
						fmt.Fprintln(out, "— residual: plugins dogfood lane · dual_write OFF · not Memory GA · plugins dogfood ≠ Agent Plugins GA · residual PASS ≠ live dogfood · package load ≠ Memory GA · portal HITL")
						return false, nil
					case "gtm", "drafts":
						fmt.Fprintln(out, agent.AionAgentOnboardingNextGtmLane())
						fmt.Fprintln(out, "— residual: gtm draft-only lane · drafts only · no auto-send · human publish · GTM checklist ≠ invent GTM agent GA · dual_write OFF · not Memory GA")
						return false, nil
					case "memory", "mcp", "palace":
						fmt.Fprintln(out, agent.AionAgentOnboardingNextMemoryLane())
						fmt.Fprintln(out, "— residual: memory local lane · dual_write OFF · not Memory GA · package load ≠ Memory GA · ≠ freemium palace · mesh ≠ memory · portal HITL")
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
					case "agentic", "agentic-integrations", "integrations", "portal-hitl", "list-plan", "hitl":
						// s1417: product plane 3 agentic integrations (MCP list/plan + portal HITL).
						// s1422: optional 4th token dogfood|soft|samples|offline|list-plan-soft → soft offline list/plan dogfood.
						// s1427: optional 4th token dual-auth|candidacy|list-org|org-installs|dual_auth|dual-auth-candidacy → dual-auth candidacy board.
						// Bare /onboard next agentic stays board (not auto dogfood / dual-auth).
						// NOT bare mcp (memory) · NOT bare portal|agent-mcp (portal handoff) · NOT bare pull (mesh).
						// NOT bare dogfood under /onboard next (dogfood stays plugins lane).
						if len(parts) >= 4 {
							sub := strings.ToLower(parts[3])
							switch sub {
							case "dogfood", "soft", "samples", "offline", "list-plan-soft":
								fmt.Fprintln(out, agent.RunAgenticListPlanSoftDogfood())
								fmt.Fprintln(out, "— residual: agentic list/plan soft offline dogfood · s1422 · no MCP dial · soft offline list/plan ≠ live dogfood · ≠ invent Connected · portal HITL still · list_org fail-open ≠ empty-as-none · session soft ≠ live dogfood · dual_write OFF · not Memory GA · template= ≠ install APPLY · agent MCP cannot write installs")
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
						fmt.Fprintln(out, "— residual: agentic integrations lane · product plane 3 · dual_write OFF · not Memory GA · MCP list/plan residual-honest · plan deep links = browser HITL only · template= ≠ install APPLY · catalog ≠ Connected · list_org fail-open ≠ empty-as-none · list_plan_not_connected · portal_hitl_still · agent MCP cannot write installs · never invent Connected · rates ~$88/$119 optional · soft dogfood: /onboard next agentic dogfood · dual-auth: /onboard next agentic dual-auth")
						return false, nil
					case "human-gates", "human", "gates", "apply-gates":
						// s1413: residual-honest human-gates still-required vs offline residual.
						// Still human: Slack HMAC · Stripe Customers:Write · H1/H2 INSTALL_STORE · D1–D5 · book-demo OFF · ON_SIGNAL unset.
						// Offline residual ≠ invent APPLY · open boxes stay open · PASS ≠ invent human-gate green.
						fmt.Fprintln(out, agent.AionAgentHumanGatesHonestyBoard())
						fmt.Fprintln(out, "— residual: human-gates honesty board · dual_write OFF · not Memory GA · book-demo OFF · leave ON_SIGNAL unset · PASS ≠ invent human-gate green · PASS ≠ live APPLY · open boxes stay open · Knowledge Beta→GA cannot invent H1/H2 offline · never invent APPLY · local memory / dual_write OFF / agent MCP list/plan do not close human APPLY gates")
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
						fmt.Fprintln(out, "— residual: post-onboard next lanes · dual_write OFF · not Memory GA · plugins dogfood ≠ Agent Plugins GA · drafts only · no auto-send · package load ≠ Memory GA · mesh ≠ memory · portal HITL · board/export evidence ≠ invent Connected · pull_not_probed · list_plan_not_connected · PASS ≠ invent human-gate green")
						fmt.Fprintln(out, "usage: /onboard next [plugins|gtm|memory|mesh|memory-pull|agentic|status|export|human-gates]  (lane aliases: plugins→plugin|dogfood · gtm→drafts · memory→mcp|palace · mesh→stream|streams|heartbeat|heartbeats|pull · memory-pull→ops-pack|pull-path|memorypull|ops_pack · agentic→agentic-integrations|integrations|portal-hitl|list-plan|hitl · status→pulse|board · export→receipt|stamp|evidence · human-gates→human|gates|apply-gates; parent aliases after|continue|lanes; export json for JSON receipt; pulse stays status board; bare pull stays mesh; bare mcp stays memory; bare portal stays portal handoff)")
						return false, nil
					}
				}
				fmt.Fprintln(out, agent.AionAgentOnboardingNextLanes())
				fmt.Fprintln(out, "— residual: post-onboard next lanes · dual_write OFF · not Memory GA · plugins dogfood ≠ Agent Plugins GA · drafts only · no auto-send · package load ≠ Memory GA · mesh ≠ memory · portal HITL · board/export evidence ≠ invent Connected · pull_not_probed · list_plan_not_connected · PASS ≠ invent human-gate green")
				return false, nil
			}
			// Unknown subcommand: still print guidance + usage hint.
			fmt.Fprintln(out, agent.AionAgentOnboardingGuidanceNote())
			fmt.Fprintln(out, "— residual: TUI ↔ aion onboarding · dual_write OFF · not Memory GA · never invent Connected · portal HITL · skill aion-agent-onboarding via read_skill")
			fmt.Fprintln(out, "usage: /onboard [help|checklist|portal|status|next]  (aliases /aion-onboard /agent-onboard; portal aliases agent-mcp|mcp; next aliases after|continue|lanes; next lanes: plugins|gtm|memory|mesh|memory-pull|agentic|status|export|human-gates)")
			return false, nil
		}
		fmt.Fprintln(out, agent.AionAgentOnboardingGuidanceNote())
		fmt.Fprintln(out, "— residual: TUI ↔ aion onboarding · dual_write OFF · not Memory GA · never invent Connected · portal HITL · skill aion-agent-onboarding via read_skill")
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
		case "dogfood", "soft", "samples", "offline":
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
  /mesh                I/O Mesh status + usage
  /catalog [query]     list mesh data products (catalog plane)
  /memory [recall|related|digest|facts-as-of|timeline|compact-status|trigger-compact|semantic|ingest-event|patterns|anomalies|supersede|ingest|status]  Memory Palace (sync HTTP + MCP; related multi-hop · digest ops pulse · facts-as-of bi-temporal lite · timeline/compact-status · trigger-compact HITL · semantic tier-4 · ingest-event s138 T1 · patterns/anomalies ops pulse Beta · supersede A3 lite HITL · status advanced inventory)
  /integrations [list|plan|signing|status]  connector setup via MCP (catalog+plan+signing discovery+portal HITL; not install CRUD)
  /gtm [help|checklist]  GTM draft-only guidance or checklist (aliases /gtm-draft /gtm-agent; no auto-send; human publish)
  /onboard [help|checklist|portal|status|next]  TUI agent ↔ aion onboarding guidance, checklist, portal Agent/MCP handoff, offline status, or post-onboard next lanes (aliases /aion-onboard /agent-onboard; residual-honest · portal HITL · settings/agent; next [plugins|gtm|memory|mesh|status|export]; next mesh→stream|streams|heartbeat|heartbeats|pull; next status→pulse|board; next export→receipt|stamp|evidence [json]; next aliases after|continue|lanes; pulse stays status board)
  /plugins [help|list|validate|dogfood|status]  residual-honest Agent Plugins soft offline dogfood (alias /plugin; dogfood aliases soft|samples|offline; Discover ≠ Connected · soft offline ≠ live dogfood · ≠ invent Agent Plugins GA)
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
	return strings.TrimSpace(`usage: /integrations [list [--layer operational|knowledge|analytical] | plan <connector_id> | signing [layer|id] | status]
  list     MCP list_connector_catalog (v178 entries) → id · status · mesh_layer · oauth?
  plan     MCP plan_connector_setup → portal_url · oauth_mode_hint · signing_headers_tool · next_steps · honesty
  signing  MCP get_webhook_signing_headers → header parity (discovery only · not secret mint)
  status   residual-honest operator pulse: MCP path · tools present · catalog honesty counts (≠ install green)
honesty: ` + agent.IntegrationsHonestyOneLiner + `
  fail-open when MCP unavailable → portal HITL https://console.iome.sh/integrations
  aion MCP v178 list/plan + v30 signing · browser HITL for OAuth · never invent install green`)
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

// parseMemoryDigestArgs extracts ops digest flags (s1200).
// Supports: --window day|week, --horizon ops|knowledge|analytical|all, --limit N, --as-of RFC3339.
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

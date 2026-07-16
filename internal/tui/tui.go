// Package tui is the interactive terminal front-end.
//
// The full-screen Bubble Tea UI will land in a later PR; this scaffold provides
// a functional REPL so headless/router work is dogfoodable without a GUI.
package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/router"
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
	rt *agent.Runtime
}

func (a runtimeAdapter) RunTurn(ctx context.Context, userText string, onEvent func(agent.Event)) (string, error) {
	return a.rt.RunTurn(ctx, userText, onEvent)
}
func (a runtimeAdapter) Router() *router.Router { return a.rt.Router() }
func (a runtimeAdapter) Workspace() workspaceRoot {
	return a.rt.Workspace()
}

// Run starts the interactive REPL (scaffold).
func Run(ctx context.Context, rt *agent.Runtime, logger *slog.Logger) error {
	return runREPL(ctx, runtimeAdapter{rt: rt}, os.Stdin, os.Stdout, logger)
}

func runREPL(ctx context.Context, rt runtimeAdapter, in io.Reader, out io.Writer, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	fmt.Fprintf(out, "iomesh-tui REPL (scaffold) — workspace %s\n", rt.Workspace().Root())
	fmt.Fprintf(out, "default model: %s  |  /model <name>  /models  /subagents  /cost  /quit\n\n", rt.Router().DefaultModel())

	sc := bufio.NewScanner(in)
	// Allow long pastes.
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 4*1024*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		fmt.Fprint(out, "iomesh> ")
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				return err
			}
			fmt.Fprintln(out)
			return nil
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			if quit, err := handleSlash(out, rt, line); quit {
				return err
			}
			continue
		}

		_, err := rt.RunTurn(ctx, line, func(ev agent.Event) {
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
		}
		fmt.Fprintln(out)
	}
}

func handleSlash(out io.Writer, rt runtimeAdapter, line string) (quit bool, err error) {
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "/quit", "/exit", "/q":
		return true, nil
	case "/models":
		for _, m := range rt.Router().Models() {
			mark := " "
			if m.Name == rt.Router().DefaultModel() {
				mark = "*"
			}
			if ov := rt.Router().Override(); ov != "" && ov == m.Name {
				mark = ">"
			}
			fmt.Fprintf(out, "%s %s  (%s)  tier=%.1f  ctx=%d\n",
				mark, m.Name, m.ModelID, m.CostTier, m.MaxContext)
		}
	case "/model", "/m":
		if len(parts) < 2 {
			cur := rt.Router().Override()
			if cur == "" {
				cur = rt.Router().DefaultModel() + " (default)"
			}
			fmt.Fprintf(out, "current: %s\n", cur)
			return false, nil
		}
		name := parts[1]
		if name == "default" || name == "auto" {
			_ = rt.Router().SetOverride("")
			fmt.Fprintln(out, "model override cleared (auto cascade)")
			return false, nil
		}
		if err := rt.Router().SetOverride(name); err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			return false, nil
		}
		fmt.Fprintf(out, "model override → %s\n", name)
	case "/cost":
		// Sample estimate for 100k in / 4k out on default.
		name := rt.Router().Override()
		if name == "" {
			name = rt.Router().DefaultModel()
		}
		est := rt.Router().EstimateCostTokens(name, 100_000, 4_000, 0)
		fmt.Fprintf(out, "estimate for %s @ 100k in / 4k out: $%.5f\n", name, est.USD)
		estHit := rt.Router().EstimateCostTokens(name, 100_000, 4_000, 80_000)
		fmt.Fprintf(out, "with 80%% cache hit: $%.5f\n", estHit.USD)
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
		for _, rec := range list {
			fmt.Fprintf(out, "%s  %-12s  %-16s  %s\n", rec.ID, rec.Status, rec.Spec.SubagentType, rec.Spec.Description)
		}
	case "/help", "/?":
		fmt.Fprint(out, `commands:
  /models          list models
  /model <name>    pin model (or default|auto)
  /subagents       list spawned subagents
  /cost            sample cost estimate
  /quit            exit
`)
	default:
		fmt.Fprintf(out, "unknown command %s (try /help)\n", cmd)
	}
	return false, nil
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

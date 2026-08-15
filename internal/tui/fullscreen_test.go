package tui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

func fsTestRuntime(t *testing.T) *agent.Runtime {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		_ = json.NewEncoder(w).Encode(router.ChatResponse{
			Choices: []router.Choice{{
				Message:      router.Message{Role: "assistant", Content: "hello from model"},
				FinishReason: "stop",
			}},
			Usage: router.Usage{TotalTokens: 5},
		})
	}))
	t.Cleanup(srv.Close)
	models := []router.ModelConfig{{
		Name: "m", BaseURL: srv.URL, ModelID: "m", APIKey: "k",
		CostTier: 1, MaxContext: 10000, Capabilities: []string{"fast", "coding"}, Priority: 1,
	}}
	rtr, err := router.New(models, "m")
	if err != nil {
		t.Fatal(err)
	}
	rt, err := agent.New(agent.Config{
		Workspace: t.TempDir(), SubagentsEnabled: true, Yolo: true,
	}, rtr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

func TestFullscreenModel_WindowSizeAndView(t *testing.T) {
	rt := fsTestRuntime(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newFullscreenModel(ctx, cancel, rt, nil, nil, UIOptions{})

	mod, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	fm := mod.(*fullscreenModel)
	if !fm.ready {
		t.Fatal("expected ready after resize")
	}
	view := fm.View()
	if !strings.Contains(view, "iomesh") {
		t.Fatalf("view missing brand:\n%s", view)
	}
	if !strings.Contains(view, "workspace") && !strings.Contains(view, rt.Workspace().Root()) {
		// header/footer may truncate; transcript has workspace line
		if !strings.Contains(strings.Join(fm.lines, "\n"), "workspace") {
			t.Fatalf("missing workspace in transcript:\n%v", fm.lines)
		}
	}
}

func TestFullscreenModel_SlashQuit(t *testing.T) {
	rt := fsTestRuntime(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newFullscreenModel(ctx, cancel, rt, nil, nil, UIOptions{})
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fm := mod.(*fullscreenModel)

	cmd := fm.submitLine("/help")
	if cmd != nil {
		// /help returns nil cmd
		t.Log("help cmd", cmd)
	}
	// Direct slash
	var next tea.Cmd
	mod, next = fm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{}})
	_ = mod
	_ = next

	out, quit, err := captureSlash(rt, nil, "/quit")
	if err != nil || !quit {
		t.Fatalf("quit=%v err=%v out=%s", quit, err, out)
	}
}

func TestFullscreenModel_AgentEvents(t *testing.T) {
	rt := fsTestRuntime(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newFullscreenModel(ctx, cancel, rt, nil, nil, UIOptions{})
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fm := mod.(*fullscreenModel)

	mod, _ = fm.Update(agentEventMsg{Type: agent.EventModelSelected, Model: "m"})
	fm = mod.(*fullscreenModel)
	mod, _ = fm.Update(agentEventMsg{Type: agent.EventTextDelta, Text: "hi "})
	fm = mod.(*fullscreenModel)
	mod, _ = fm.Update(agentEventMsg{Type: agent.EventTextDelta, Text: "there"})
	fm = mod.(*fullscreenModel)
	mod, _ = fm.Update(agentEventMsg{Type: agent.EventToolStart, Tool: "read_file"})
	fm = mod.(*fullscreenModel)
	mod, _ = fm.Update(agentEventMsg{Type: agent.EventToolEnd, Tool: "read_file", Text: "ok"})
	fm = mod.(*fullscreenModel)
	mod, _ = fm.Update(agentEventMsg{
		Type: agent.EventLLMDone, Model: "m", Tokens: 5, CostUSD: 0.001, Duration: time.Millisecond,
	})
	fm = mod.(*fullscreenModel)

	joined := strings.Join(fm.lines, "\n")
	if !strings.Contains(joined, "hi there") {
		t.Fatalf("stream missing: %s", joined)
	}
	if !strings.Contains(joined, "read_file") {
		t.Fatalf("tool missing: %s", joined)
	}
	if fm.lastCost == "" {
		t.Fatal("expected lastCost")
	}
}

func TestFullscreenModel_ApprovalKeys(t *testing.T) {
	rt := fsTestRuntime(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newFullscreenModel(ctx, cancel, rt, nil, nil, UIOptions{})
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fm := mod.(*fullscreenModel)

	ch := make(chan agent.Approval, 1)
	mod, _ = fm.Update(approvalRequestMsg{tool: "write_file", args: `{"path":"x"}`, reply: ch})
	fm = mod.(*fullscreenModel)
	if fm.approval == nil {
		t.Fatal("expected approval pending")
	}
	mod, _ = fm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	fm = mod.(*fullscreenModel)
	select {
	case a := <-ch:
		if a != agent.ApprovalOnce {
			t.Fatalf("got %v", a)
		}
	default:
		t.Fatal("no approval reply")
	}
	if fm.approval != nil {
		t.Fatal("approval should clear")
	}

	// always
	ch2 := make(chan agent.Approval, 1)
	mod, _ = fm.Update(approvalRequestMsg{tool: "run_shell", args: "ls", reply: ch2})
	fm = mod.(*fullscreenModel)
	mod, _ = fm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	select {
	case a := <-ch2:
		if a != agent.ApprovalAlways {
			t.Fatalf("got %v", a)
		}
	default:
		t.Fatal("no always reply")
	}

	// deny
	ch3 := make(chan agent.Approval, 1)
	mod, _ = fm.Update(approvalRequestMsg{tool: "write_file", args: "{}", reply: ch3})
	fm = mod.(*fullscreenModel)
	_, _ = fm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	select {
	case a := <-ch3:
		if a != agent.ApprovalDeny {
			t.Fatalf("got %v", a)
		}
	default:
		t.Fatal("no deny reply")
	}
}

func TestFullscreenModel_SubmitTurn(t *testing.T) {
	rt := fsTestRuntime(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newFullscreenModel(ctx, cancel, rt, nil, nil, UIOptions{})
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fm := mod.(*fullscreenModel)

	cmd := fm.submitLine("say hi")
	if cmd == nil {
		t.Fatal("expected turn cmd")
	}
	if !fm.busy {
		t.Fatal("expected busy")
	}
	// Execute the cmd synchronously (agent turn).
	msg := cmd()
	done, ok := msg.(turnDoneMsg)
	if !ok {
		t.Fatalf("msg type %T", msg)
	}
	if done.err != nil {
		t.Fatal(done.err)
	}
	mod, _ = fm.Update(done)
	fm = mod.(*fullscreenModel)
	if fm.busy {
		t.Fatal("should not be busy after turnDone")
	}
}

func TestHandleAgentEvent_Denied(t *testing.T) {
	rt := fsTestRuntime(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newFullscreenModel(ctx, cancel, rt, nil, nil, UIOptions{})
	m.handleAgentEvent(agent.Event{Type: agent.EventToolDenied, Tool: "write_file", Text: "nope"})
	joined := strings.Join(m.lines, "\n")
	if !strings.Contains(joined, "write_file") || !strings.Contains(joined, "nope") {
		t.Fatal(joined)
	}
}

func TestFullscreenModel_ThemeSlash(t *testing.T) {
	rt := fsTestRuntime(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newFullscreenModel(ctx, cancel, rt, nil, nil, UIOptions{Theme: "default"})
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	fm := mod.(*fullscreenModel)
	_ = fm.handleThemeSlash([]string{"/theme", "mono"})
	if fm.theme.Name != "mono" {
		t.Fatalf("theme=%s", fm.theme.Name)
	}
	_ = fm.submitLine("hello\nworld")
	joined := strings.Join(fm.lines, "\n")
	if !strings.Contains(joined, "hello") || !strings.Contains(joined, "world") {
		t.Fatal(joined)
	}
}

func TestFullscreenModel_InputHeight(t *testing.T) {
	rt := fsTestRuntime(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newFullscreenModel(ctx, cancel, rt, nil, nil, UIOptions{})
	if m.inputHeight() != 3 {
		t.Fatalf("min height=%d", m.inputHeight())
	}
	m.input.SetValue("a\nb\nc\nd\ne")
	if m.inputHeight() != 5 {
		t.Fatalf("height=%d", m.inputHeight())
	}
	m.input.SetValue(strings.Repeat("x\n", 20))
	if m.inputHeight() != 8 {
		t.Fatalf("cap height=%d", m.inputHeight())
	}
}

func TestFullscreenModel_DashboardOverlay(t *testing.T) {
	rt := fsTestRuntime(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newFullscreenModel(ctx, cancel, rt, nil, nil, UIOptions{})
	mod, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 36})
	fm := mod.(*fullscreenModel)

	cmd := fm.submitLine("/dashboard")
	if fm.dash == nil {
		t.Fatal("expected dashboard overlay")
	}
	if cmd == nil {
		t.Fatal("expected tick cmd")
	}
	view := fm.View()
	if !strings.Contains(view, "context://mesh") || !strings.Contains(view, "P2 opened") {
		t.Fatalf("overlay missing live feed:\n%s", view)
	}
	if !strings.Contains(view, "catalog ≠ Connected") {
		t.Fatalf("overlay missing honesty:\n%s", view)
	}

	mod, _ = fm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	fm = mod.(*fullscreenModel)
	if fm.dash.Focus != "eng.ops" {
		t.Fatalf("key 2 focus=%s", fm.dash.Focus)
	}

	mod, _ = fm.Update(dashboardTickMsg{})
	fm = mod.(*fullscreenModel)
	if fm.dash == nil || fm.dash.Events[0].T == "14:02:11" && fm.dash.idx == 0 {
		t.Fatalf("tick did not advance: %+v", fm.dash.Events[0])
	}

	mod, _ = fm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	fm = mod.(*fullscreenModel)
	if fm.dash != nil {
		t.Fatal("esc should close dashboard")
	}
}

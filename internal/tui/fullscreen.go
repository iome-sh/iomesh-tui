package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/session"
)

// Styles for the full-screen TUI.
var (
	styleTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	styleStatus  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleUser    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	styleTool    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	styleErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleMesh    = lipgloss.NewStyle().Foreground(lipgloss.Color("45"))
	styleApprove = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226")).Background(lipgloss.Color("235"))
	styleHelp    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

// RunFullscreen starts the alt-screen Bubble Tea UI (scrollback + streaming + approvals).
func RunFullscreen(ctx context.Context, rt *agent.Runtime, store *session.Store, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	m := newFullscreenModel(ctx, cancel, rt, store, logger)
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)
	m.setProgram(p)

	// Interactive approval: agent goroutine Send()s a request; UI replies on ch.
	rt.SetApprover(func(ctx context.Context, tool, args string) (agent.Approval, error) {
		ch := make(chan agent.Approval, 1)
		p.Send(approvalRequestMsg{tool: tool, args: args, reply: ch})
		select {
		case a := <-ch:
			return a, nil
		case <-ctx.Done():
			return agent.ApprovalDeny, ctx.Err()
		}
	})

	final, err := p.Run()
	if err != nil {
		return err
	}
	if fm, ok := final.(*fullscreenModel); ok && fm.quitErr != nil {
		return fm.quitErr
	}
	return nil
}

// --- messages ---

type agentEventMsg agent.Event

type turnDoneMsg struct {
	err error
}

type approvalRequestMsg struct {
	tool  string
	args  string
	reply chan agent.Approval
}

type tickMsg time.Time

// --- model ---

type fullscreenModel struct {
	ctx    context.Context
	cancel context.CancelFunc
	rt     *agent.Runtime
	store  *session.Store
	logger *slog.Logger

	mu   sync.Mutex
	prog *tea.Program

	vp     viewport.Model
	input  textinput.Model
	ready  bool
	width  int
	height int

	// Transcript lines (raw, pre-viewport).
	lines []string
	// Streaming assistant buffer (not yet closed with newline).
	streamOpen bool

	busy      bool
	status    string
	lastCost  string
	quitErr   error
	helpShown bool

	// Approval overlay
	approval *approvalRequestMsg
}

func newFullscreenModel(ctx context.Context, cancel context.CancelFunc, rt *agent.Runtime, store *session.Store, logger *slog.Logger) *fullscreenModel {
	ti := textinput.New()
	ti.Placeholder = "message or /help  ·  ctrl+c quit"
	ti.Focus()
	ti.CharLimit = 32 * 1024
	ti.Width = 80
	ti.Prompt = "❯ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	m := &fullscreenModel{
		ctx:    ctx,
		cancel: cancel,
		rt:     rt,
		store:  store,
		logger: logger,
		vp:     viewport.New(80, 20),
		input:  ti,
		status: "ready",
	}
	m.appendLine(styleTitle.Render("iomesh-tui") + "  " + styleDim.Render("fullscreen · scroll · approvals"))
	m.appendLine(styleStatus.Render(fmt.Sprintf("workspace %s", rt.Workspace().Root())))
	if sid := rt.SessionID(); sid != "" {
		m.appendLine(styleStatus.Render("session " + sid))
	}
	m.appendLine(styleStatus.Render(fmt.Sprintf("model %s  ·  mutating tools prompt y/n/a unless --yolo", displayModel(rt.Router()))))
	m.appendLine(styleHelp.Render("keys: enter send · pgup/pgdn scroll · ctrl+c quit · /help"))
	m.appendLine("")
	return m
}

func (m *fullscreenModel) setProgram(p *tea.Program) {
	m.mu.Lock()
	m.prog = p
	m.mu.Unlock()
}

func (m *fullscreenModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *fullscreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerH := 2
		footerH := 3
		if m.approval != nil {
			footerH = 5
		}
		vpH := msg.Height - headerH - footerH
		if vpH < 3 {
			vpH = 3
		}
		m.vp.Width = msg.Width
		m.vp.Height = vpH
		m.input.Width = max(10, msg.Width-4)
		m.ready = true
		m.refreshViewport(true)
		return m, nil

	case tea.KeyMsg:
		// Approval takes keyboard focus.
		if m.approval != nil {
			return m.handleApprovalKey(msg)
		}
		if m.busy {
			// Allow scroll while busy; block send.
			switch msg.String() {
			case "ctrl+c":
				m.cancel()
				return m, tea.Quit
			case "pgup", "ctrl+u":
				m.vp.HalfViewUp()
				return m, nil
			case "pgdown", "ctrl+d":
				m.vp.HalfViewDown()
				return m, nil
			}
			// Still allow viewport mouse/keys via vp update below for up/down.
		}

		switch msg.String() {
		case "ctrl+c":
			m.cancel()
			return m, tea.Quit
		case "esc":
			if m.helpShown {
				m.helpShown = false
				return m, nil
			}
		case "enter":
			if m.busy {
				return m, nil
			}
			line := strings.TrimSpace(m.input.Value())
			m.input.SetValue("")
			if line == "" {
				return m, nil
			}
			return m, m.submitLine(line)
		case "pgup", "ctrl+u":
			m.vp.HalfViewUp()
			return m, nil
		case "pgdown", "ctrl+d":
			m.vp.HalfViewDown()
			return m, nil
		}

	case approvalRequestMsg:
		m.approval = &msg
		m.status = "awaiting approval"
		m.appendLine(styleApprove.Render(fmt.Sprintf(" ⚠ approve tool %s? ", msg.tool)))
		m.appendLine(styleDim.Render("  " + truncate(msg.args, 200)))
		m.appendLine(styleHelp.Render("  [y]es  [n]o  [a]lways this session"))
		m.refreshViewport(true)
		return m, nil

	case agentEventMsg:
		m.handleAgentEvent(agent.Event(msg))
		m.refreshViewport(true)
		return m, nil

	case turnDoneMsg:
		m.busy = false
		m.streamOpen = false
		if msg.err != nil {
			m.appendLine(styleErr.Render("error: " + msg.err.Error()))
			m.status = "error"
		} else {
			m.status = "ready"
			if m.store != nil {
				m.rt.AutoSaveAfterTurn(m.store)
				if id := m.rt.SessionID(); id != "" {
					m.appendLine(styleDim.Render(fmt.Sprintf("[session %s saved]", id)))
				}
			}
		}
		m.appendLine("")
		m.refreshViewport(true)
		return m, nil
	}

	// Forward to subcomponents.
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)
	m.vp, cmd = m.vp.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *fullscreenModel) handleApprovalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.approval == nil {
		return m, nil
	}
	var dec agent.Approval
	switch strings.ToLower(msg.String()) {
	case "y", "enter":
		dec = agent.ApprovalOnce
		m.appendLine(styleOK.Render("  → approved once"))
	case "a":
		dec = agent.ApprovalAlways
		m.appendLine(styleOK.Render("  → always allow " + m.approval.tool + " this session"))
	case "n", "esc":
		dec = agent.ApprovalDeny
		m.appendLine(styleErr.Render("  → denied"))
	case "ctrl+c":
		dec = agent.ApprovalDeny
		m.approval.reply <- dec
		m.approval = nil
		m.cancel()
		return m, tea.Quit
	default:
		return m, nil
	}
	ch := m.approval.reply
	m.approval = nil
	m.status = "running"
	m.refreshViewport(true)
	// Non-blocking send in case agent already cancelled.
	select {
	case ch <- dec:
	default:
	}
	return m, nil
}

func (m *fullscreenModel) submitLine(line string) tea.Cmd {
	if strings.HasPrefix(line, "/") {
		return m.runSlash(line)
	}
	m.appendLine(styleUser.Render("you › ") + line)
	m.appendLine("")
	m.busy = true
	m.status = "thinking…"
	m.streamOpen = false
	m.refreshViewport(true)

	rt := m.rt
	ctx := m.ctx
	return func() tea.Msg {
		_, err := rt.RunTurn(ctx, line, func(ev agent.Event) {
			m.mu.Lock()
			p := m.prog
			m.mu.Unlock()
			if p != nil {
				p.Send(agentEventMsg(ev))
			}
		})
		return turnDoneMsg{err: err}
	}
}

func (m *fullscreenModel) runSlash(line string) tea.Cmd {
	adapter := runtimeAdapter{rt: m.rt, store: m.store}
	var buf strings.Builder
	quit, err := handleSlash(&buf, adapter, line)
	out := strings.TrimRight(buf.String(), "\n")
	if out != "" {
		for _, ln := range strings.Split(out, "\n") {
			m.appendLine(styleDim.Render(ln))
		}
	}
	if err != nil {
		m.appendLine(styleErr.Render(err.Error()))
	}
	m.refreshViewport(true)
	if quit {
		m.cancel()
		return tea.Quit
	}
	return nil
}

func (m *fullscreenModel) handleAgentEvent(ev agent.Event) {
	switch ev.Type {
	case agent.EventModelSelected:
		m.appendLine(styleDim.Render("[model " + ev.Model + "]"))
	case agent.EventTextDelta:
		m.appendStream(ev.Text)
	case agent.EventThinkingDelta:
		m.appendStream(styleDim.Render(ev.Text))
	case agent.EventToolStart:
		m.closeStream()
		m.appendLine(styleTool.Render("→ tool " + ev.Tool))
	case agent.EventToolEnd:
		m.appendLine(styleOK.Render("✓ "+ev.Tool) + " " + styleDim.Render(truncate(ev.Text, 100)))
	case agent.EventToolError, agent.EventToolDenied:
		m.appendLine(styleErr.Render("✗ " + ev.Tool + " " + ev.Text))
	case agent.EventLLMDone:
		m.closeStream()
		m.lastCost = fmt.Sprintf("%s · %d tok · $%.5f · %s",
			ev.Model, ev.Tokens, ev.CostUSD, ev.Duration.Round(time.Millisecond))
		m.appendLine(styleDim.Render("— " + m.lastCost))
	case agent.EventMeshContext:
		m.appendLine(styleMesh.Render("[iomesh] " + ev.Text))
	}
}

func (m *fullscreenModel) appendStream(s string) {
	if s == "" {
		return
	}
	if !m.streamOpen {
		m.lines = append(m.lines, s)
		m.streamOpen = true
		return
	}
	// Append to last line; split on newlines.
	last := len(m.lines) - 1
	parts := strings.Split(s, "\n")
	m.lines[last] += parts[0]
	for i := 1; i < len(parts); i++ {
		m.lines = append(m.lines, parts[i])
	}
}

func (m *fullscreenModel) closeStream() {
	if m.streamOpen {
		m.streamOpen = false
	}
}

func (m *fullscreenModel) appendLine(s string) {
	m.closeStream()
	m.lines = append(m.lines, s)
}

func (m *fullscreenModel) refreshViewport(gotoBottom bool) {
	content := strings.Join(m.lines, "\n")
	m.vp.SetContent(content)
	if gotoBottom {
		m.vp.GotoBottom()
	}
}

func (m *fullscreenModel) View() string {
	if !m.ready {
		return "\n  initializing iomesh-tui…"
	}
	header := m.renderHeader()
	footer := m.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, m.vp.View(), footer)
}

func (m *fullscreenModel) renderHeader() string {
	left := styleTitle.Render("iomesh") + " " + styleDim.Render(displayModel(m.rt.Router()))
	right := styleStatus.Render(m.status)
	if m.lastCost != "" && !m.busy {
		right = styleDim.Render(m.lastCost) + "  " + right
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	sep := styleDim.Render(strings.Repeat("─", max(1, m.width)))
	return line + "\n" + sep
}

func (m *fullscreenModel) renderFooter() string {
	sep := styleDim.Render(strings.Repeat("─", max(1, m.width)))
	if m.approval != nil {
		hint := styleApprove.Render(" APPROVE ") + styleHelp.Render(" y=once  n=deny  a=always  ·  tool "+m.approval.tool)
		return sep + "\n" + hint + "\n" + styleDim.Render("keyboard focus: approval")
	}
	ws := ""
	if m.rt != nil {
		ws = truncate(m.rt.Workspace().Root(), 40)
	}
	meta := styleDim.Render(fmt.Sprintf("%s  ·  %s", ws, m.status))
	return sep + "\n" + m.input.View() + "\n" + meta
}

// ensure fullscreenModel implements tea.Model.
var _ tea.Model = (*fullscreenModel)(nil)

// captureSlash is used by tests for pure slash output without bubbletea.
func captureSlash(rt *agent.Runtime, store *session.Store, line string) (string, bool, error) {
	var buf strings.Builder
	quit, err := handleSlash(&buf, runtimeAdapter{rt: rt, store: store}, line)
	return buf.String(), quit, err
}

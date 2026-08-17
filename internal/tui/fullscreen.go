package tui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iome-sh/iomesh-tui/internal/agent"
	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/session"
)

// UIOptions configures full-screen presentation.
type UIOptions struct {
	// Theme is default|mono|high-contrast|dim (empty → default).
	Theme string
}

// RunFullscreen starts the alt-screen Bubble Tea UI (scrollback + streaming + approvals).
func RunFullscreen(ctx context.Context, rt *agent.Runtime, store *session.Store, logger *slog.Logger) error {
	return RunFullscreenOpts(ctx, rt, store, logger, UIOptions{})
}

// RunFullscreenOpts is RunFullscreen with theme / UI options.
func RunFullscreenOpts(ctx context.Context, rt *agent.Runtime, store *session.Store, logger *slog.Logger, opts UIOptions) error {
	if logger == nil {
		logger = slog.Default()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	m := newFullscreenModel(ctx, cancel, rt, store, logger, opts)
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

type dashboardTickMsg time.Time

type dashboardConsumeMsg struct {
	events []HeartbeatEvent
	names  []string
	reason string
}

func dashboardTick() tea.Cmd {
	return tea.Tick(dashboardTickEvery, func(t time.Time) tea.Msg {
		return dashboardTickMsg(t)
	})
}

func dashboardConsumeCmd(c *iomesh.Client) tea.Cmd {
	if c == nil || !c.Enabled() {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), dashboardConsumeTimeout)
		defer cancel()
		events, names, reason := probeDashboardConsume(ctx, c)
		return dashboardConsumeMsg{events: events, names: names, reason: reason}
	}
}

// --- model ---

type fullscreenModel struct {
	ctx    context.Context
	cancel context.CancelFunc
	rt     *agent.Runtime
	store  *session.Store
	logger *slog.Logger
	theme  Theme

	mu   sync.Mutex
	prog *tea.Program

	vp     viewport.Model
	input  textarea.Model
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

	// Landing-page heartbeat dashboard overlay (s1989).
	dash *dashboardState
}

func newFullscreenModel(ctx context.Context, cancel context.CancelFunc, rt *agent.Runtime, store *session.Store, logger *slog.Logger, opts UIOptions) *fullscreenModel {
	th, err := ParseTheme(opts.Theme)
	if err != nil {
		th = ThemeDefault()
	}

	ta := textarea.New()
	ta.Placeholder = "message or /help  ·  enter send · ctrl+j newline · /dashboard · /theme"
	ta.Focus()
	ta.CharLimit = 32 * 1024
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.Prompt = "❯ "
	// Enter is for send (handled in Update); newline via ctrl+j / alt+enter.
	ta.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("ctrl+j", "alt+enter", "shift+enter"),
		key.WithHelp("ctrl+j", "newline"),
	)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.FocusedStyle.Prompt = th.Prompt
	ta.BlurredStyle.Prompt = th.Prompt

	m := &fullscreenModel{
		ctx:    ctx,
		cancel: cancel,
		rt:     rt,
		store:  store,
		logger: logger,
		theme:  th,
		vp:     viewport.New(80, 20),
		input:  ta,
		status: "ready",
	}
	m.appendLine(m.theme.Title.Render("iomesh-tui") + "  " + m.theme.Dim.Render("fullscreen · multi-line · theme="+th.Name))
	m.appendLine(m.theme.Status.Render(fmt.Sprintf("workspace %s", rt.Workspace().Root())))
	if sid := rt.SessionID(); sid != "" {
		m.appendLine(m.theme.Status.Render("session " + sid))
	}
	m.appendLine(m.theme.Status.Render(fmt.Sprintf("model %s  ·  mutating tools prompt y/n/a unless --yolo", displayModel(rt.Router()))))
	m.appendLine(m.theme.Help.Render("keys: enter send · ctrl+j newline · pgup/pgdn · /dashboard · /theme · ctrl+c quit"))
	m.appendLine("")
	return m
}

func (m *fullscreenModel) setProgram(p *tea.Program) {
	m.mu.Lock()
	m.prog = p
	m.mu.Unlock()
}

func (m *fullscreenModel) applyTheme(th Theme) {
	m.theme = th
	m.input.FocusedStyle.Prompt = th.Prompt
	m.input.BlurredStyle.Prompt = th.Prompt
}

func (m *fullscreenModel) inputHeight() int {
	n := strings.Count(m.input.Value(), "\n") + 1
	if n < 3 {
		n = 3
	}
	if n > 8 {
		n = 8
	}
	return n
}

func (m *fullscreenModel) layout() {
	headerH := 2
	inH := m.inputHeight()
	footerH := inH + 2 // separator + meta
	if m.approval != nil {
		footerH = 5
	}
	if m.dash != nil {
		footerH = 3
	}
	vpH := m.height - headerH - footerH
	if vpH < 3 {
		vpH = 3
	}
	m.vp.Width = m.width
	m.vp.Height = vpH
	m.input.SetWidth(max(10, m.width-2))
	m.input.SetHeight(inH)
}

func (m *fullscreenModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m *fullscreenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		m.ready = true
		m.refreshViewport(true)
		return m, nil

	case tea.KeyMsg:
		// Approval takes keyboard focus.
		if m.approval != nil {
			return m.handleApprovalKey(msg)
		}
		if m.dash != nil {
			return m.handleDashboardKey(msg)
		}
		if m.busy {
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
		case "enter", "ctrl+m":
			// Send (newline is ctrl+j).
			if m.busy {
				return m, nil
			}
			line := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			m.input.SetHeight(3)
			m.layout()
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
		m.appendLine(m.theme.Approve.Render(fmt.Sprintf(" ⚠ approve tool %s? ", msg.tool)))
		m.appendLine(m.theme.Dim.Render("  " + truncate(msg.args, 200)))
		m.appendLine(m.theme.Help.Render("  [y]es  [n]o  [a]lways this session"))
		m.layout()
		m.refreshViewport(true)
		return m, nil

	case agentEventMsg:
		m.handleAgentEvent(agent.Event(msg))
		m.refreshViewport(true)
		return m, nil

	case dashboardTickMsg:
		if m.dash != nil {
			m.dash.Tick()
			return m, dashboardTick()
		}
		return m, nil

	case dashboardConsumeMsg:
		if m.dash != nil && !m.dash.Preview {
			m.dash.applyConsume(msg.events, msg.names, msg.reason)
		}
		return m, nil

	case turnDoneMsg:
		m.busy = false
		m.streamOpen = false
		if msg.err != nil {
			m.appendLine(m.theme.Err.Render("error: " + msg.err.Error()))
			m.status = "error"
		} else {
			m.status = "ready"
			if m.store != nil {
				m.rt.AutoSaveAfterTurn(m.store)
				if id := m.rt.SessionID(); id != "" {
					m.appendLine(m.theme.Dim.Render(fmt.Sprintf("[session %s saved]", id)))
				}
			}
		}
		m.appendLine("")
		m.refreshViewport(true)
		return m, nil
	}

	// Forward to subcomponents (textarea gets ctrl+j newline, typing, etc.).
	var cmd tea.Cmd
	prevH := m.input.Height()
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)
	if m.input.Height() != prevH || m.ready {
		// Grow/shrink with content when not approval-focused.
		if m.approval == nil && m.width > 0 {
			want := m.inputHeight()
			if m.input.Height() != want {
				m.input.SetHeight(want)
				m.layout()
			}
		}
	}
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
		m.appendLine(m.theme.OK.Render("  → approved once"))
	case "a":
		dec = agent.ApprovalAlways
		m.appendLine(m.theme.OK.Render("  → always allow " + m.approval.tool + " this session"))
	case "n", "esc":
		dec = agent.ApprovalDeny
		m.appendLine(m.theme.Err.Render("  → denied"))
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
	m.layout()
	m.refreshViewport(true)
	select {
	case ch <- dec:
	default:
	}
	return m, nil
}

func (m *fullscreenModel) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.dash == nil {
		return m, nil
	}
	switch msg.String() {
	case "ctrl+c":
		m.cancel()
		return m, tea.Quit
	case "esc", "q":
		m.dash = nil
		m.status = "ready"
		m.layout()
		return m, nil
	case "tab":
		m.dash.CycleFocus()
		return m, nil
	case "1":
		_ = m.dash.SetFocus("sre.incidents")
		return m, nil
	case "2":
		_ = m.dash.SetFocus("eng.ops")
		return m, nil
	case "3":
		_ = m.dash.SetFocus("cs.tickets")
		return m, nil
	case "4":
		_ = m.dash.SetFocus("gtm.pipeline")
		return m, nil
	}
	return m, nil
}

func (m *fullscreenModel) submitLine(line string) tea.Cmd {
	if strings.HasPrefix(line, "/") {
		return m.runSlash(line)
	}
	// Multi-line user bubbles: show first line with marker; indent rest.
	parts := strings.Split(line, "\n")
	m.appendLine(m.theme.User.Render("you › ") + parts[0])
	for _, p := range parts[1:] {
		m.appendLine(m.theme.User.Render("     ") + p)
	}
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
	parts := strings.Fields(line)
	if len(parts) > 0 && (parts[0] == "/theme" || parts[0] == "/themes") {
		return m.handleThemeSlash(parts)
	}
	if len(parts) > 0 && isDashboardSlash(parts[0]) {
		return m.handleDashboardSlash(parts)
	}
	adapter := runtimeAdapter{rt: m.rt, store: m.store}
	var buf strings.Builder
	quit, err := handleSlash(&buf, adapter, line)
	out := strings.TrimRight(buf.String(), "\n")
	if out != "" {
		for _, ln := range strings.Split(out, "\n") {
			m.appendLine(m.theme.Dim.Render(ln))
		}
	}
	if err != nil {
		m.appendLine(m.theme.Err.Render(err.Error()))
	}
	m.refreshViewport(true)
	if quit {
		m.cancel()
		return tea.Quit
	}
	return nil
}

func (m *fullscreenModel) handleDashboardSlash(parts []string) tea.Cmd {
	if len(parts) >= 2 {
		sub := strings.ToLower(parts[1])
		if sub == "help" || sub == "?" {
			m.appendLine(m.theme.Help.Render(dashboardHelp()))
			m.refreshViewport(true)
			return nil
		}
		if sub == "preview" {
			attached := false
			if m.rt != nil && m.rt.Mesh() != nil && m.rt.Mesh().Enabled() {
				attached = true
			}
			m.dash = newDashboardPreviewState(attached)
			m.status = "dashboard"
			m.layout()
			return dashboardTick()
		}
	}
	// Toggle overlay (landing MeshConsole). Help stays in transcript.
	if m.dash != nil && len(parts) < 2 {
		m.dash = nil
		m.status = "ready"
		m.layout()
		return nil
	}
	attached := false
	if m.rt != nil && m.rt.Mesh() != nil && m.rt.Mesh().Enabled() {
		attached = true
	}
	opened := m.dash == nil
	if m.dash == nil {
		m.dash = newDashboardState(attached)
	}
	if len(parts) >= 2 {
		want := parts[1]
		if strings.EqualFold(parts[1], "focus") && len(parts) > 2 {
			want = parts[2]
		}
		if !m.dash.SetFocus(want) {
			m.appendLine(m.theme.Err.Render("dashboard: unknown tenancy " + want))
			m.refreshViewport(true)
			return nil
		}
	}
	m.status = "dashboard"
	m.layout()
	if opened && attached && !m.dash.Preview {
		return tea.Batch(dashboardConsumeCmd(m.rt.Mesh()), dashboardTick())
	}
	return dashboardTick()
}

func (m *fullscreenModel) handleThemeSlash(parts []string) tea.Cmd {
	if len(parts) < 2 {
		m.appendLine(m.theme.Status.Render("themes: " + strings.Join(ThemeNames(), ", ")))
		m.appendLine(m.theme.Dim.Render("current: " + m.theme.Name + "  ·  usage: /theme <name>"))
		m.refreshViewport(true)
		return nil
	}
	th, err := ParseTheme(parts[1])
	if err != nil {
		m.appendLine(m.theme.Err.Render(err.Error()))
		m.refreshViewport(true)
		return nil
	}
	m.applyTheme(th)
	m.appendLine(m.theme.OK.Render("theme → " + th.Name))
	m.refreshViewport(true)
	return nil
}

func (m *fullscreenModel) handleAgentEvent(ev agent.Event) {
	switch ev.Type {
	case agent.EventModelSelected:
		m.appendLine(m.theme.Dim.Render("[model " + ev.Model + "]"))
	case agent.EventTextDelta:
		m.appendStream(ev.Text)
	case agent.EventThinkingDelta:
		m.appendStream(m.theme.Dim.Render(ev.Text))
	case agent.EventToolStart:
		m.closeStream()
		m.appendLine(m.theme.Tool.Render("→ tool " + ev.Tool))
	case agent.EventToolEnd:
		m.appendLine(m.theme.OK.Render("✓ "+ev.Tool) + " " + m.theme.Dim.Render(truncate(ev.Text, 100)))
	case agent.EventToolError, agent.EventToolDenied:
		m.appendLine(m.theme.Err.Render("✗ " + ev.Tool + " " + ev.Text))
	case agent.EventLLMDone:
		m.closeStream()
		m.lastCost = fmt.Sprintf("%s · %d tok · $%.5f · %s",
			ev.Model, ev.Tokens, ev.CostUSD, ev.Duration.Round(time.Millisecond))
		m.appendLine(m.theme.Dim.Render("— " + m.lastCost))
	case agent.EventMeshContext:
		m.appendLine(m.theme.Mesh.Render("[iomesh] " + ev.Text))
	case agent.EventMemoryRecall, agent.EventMemoryIngest:
		m.appendLine(m.theme.Dim.Render("[memory] " + ev.Text))
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
	body := m.vp.View()
	if m.dash != nil {
		inner := m.height - 2 - 3 // header + dashboard footer
		if inner < 8 {
			inner = 8
		}
		m.dash.Height = inner
		body = m.dash.Render(m.theme, max(40, m.width))
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *fullscreenModel) renderHeader() string {
	left := m.theme.Title.Render("iomesh") + " " + m.theme.Dim.Render(displayModel(m.rt.Router())+" · "+m.theme.Name)
	right := m.theme.Status.Render(m.status)
	if m.lastCost != "" && !m.busy {
		right = m.theme.Dim.Render(m.lastCost) + "  " + right
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	sep := m.theme.Dim.Render(strings.Repeat("─", max(1, m.width)))
	return line + "\n" + sep
}

func (m *fullscreenModel) renderFooter() string {
	sep := m.theme.Dim.Render(strings.Repeat("─", max(1, m.width)))
	if m.approval != nil {
		hint := m.theme.Approve.Render(" APPROVE ") + m.theme.Help.Render(" y=once  n=deny  a=always  ·  tool "+m.approval.tool)
		return sep + "\n" + hint + "\n" + m.theme.Dim.Render("keyboard focus: approval")
	}
	if m.dash != nil {
		hint := m.theme.Help.Render("esc/q close  ·  tab cycle tenancy  ·  1–4 jump  ·  eval template ≠ Connected")
		return sep + "\n" + hint
	}
	ws := ""
	if m.rt != nil {
		ws = truncate(m.rt.Workspace().Root(), 40)
	}
	meta := m.theme.Dim.Render(fmt.Sprintf("%s  ·  %s  ·  enter send · ctrl+j ⏎", ws, m.status))
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

package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Landing-page MeshConsole parity (iome.sh heartbeat live feed).
// Evaluation template — not a Connected workspace and not live APPLY.

const (
	dashboardDefaultFocus = "sre.incidents"
	dashboardMaxEvents    = 7
	dashboardStartRate    = 18
	dashboardMaxRate      = 42
	dashboardTickEvery    = 2600 * time.Millisecond
)

// DashboardHonestyOneLiner is the residual lock for /dashboard.
const DashboardHonestyOneLiner = "no mock live rows · /dashboard preview is eval template not your org · catalog ≠ Connected · dual_write OFF · knowledge/analytics Beta · not Memory GA · demo feed ≠ fleet-GA · not live APPLY · unacked brief ≠ known"

// DashboardBetaEmptyHonesty is shown when knowledge or analytics count is 0.
// Empty pillars stay Beta — not GA. Do not invent events.
const DashboardBetaEmptyHonesty = "knowledge Beta empty · analytics Beta empty · not GA"

// Compose already-shipped paths onto /dashboard (pulse, entitled pull, insights, human decision).
// No new backends. Pull uses iomesh memory pull. Insights use /memory digest.
// Decision stub never auto-applies. Brief ACK (#371): unread ≠ known; no send/pay/ship.
const (
	DashboardComposePulse    = "pulse: this feed · empty until consume"
	DashboardComposePull     = "pull: iomesh memory pull · entitled dept.* only"
	DashboardComposeInsights = "insights: /memory digest · existing agent path"
	DashboardComposeDecision = "decision stub: human owns apply · never auto-applies"
)

// Design Packet 4 visual leftover (#363): one first-run next action at ≥4.5:1
// pointing at hooks.iome.sh. Honesty stays Dim so it does not compete as CTAs.
// EMPTY until consume is expected. Not a functional consume-path change.
// REPL /dashboard always ThemeDefault() — /theme dim is fullscreen only.
const (
	dashboardEmptyPrimaryNext    = `next: add [iomesh] endpoint="https://hooks.iome.sh" · then consume GitHub`
	dashboardEmptyHonestyCatalog = `portal MCP (apiv1.iome.sh/v7/mcp) is catalog — streams are hooks.iome.sh`
	dashboardEmptyHonestyInfer   = `or infer from portal MCP · infer ≠ Connected · do not invent consume`
)

func dashboardHelp() string {
	return strings.TrimSpace(`usage: /dashboard [help|preview|focus <tenancy>|ack]
aliases: /heartbeat /mesh-console
  (no args)   empty until consume · probe if mesh attached (no mock rows)
  preview     opt-in eval template (iome.sh MeshConsole — not your org)
  focus       tenancy: sre.incidents | eng.ops | cs.tickets | gtm.pipeline
  ack         ACK today's morning brief (local ritual · unread ≠ known · no send/pay/ship)
fullscreen: esc/q close · tab cycle tenancy · 1-4 jump
probe:    iomesh mesh streams --messages / broker GET /v1/streams/{name}/messages
          (not portal GET /v52 — cookie-only, TUI must not call it)
setup:    add [iomesh] or infer hooks from portal MCP (catalog ≠ streams · infer ≠ Connected)
          then console Settings → Mesh routing → Streams (OPERATIONAL_EVENTS)
          or: iomesh mesh streams --create --yes  (create ≠ PULSE)
          then iomesh mesh streams --messages --name OPERATIONAL_EVENTS
honesty: ` + DashboardHonestyOneLiner)
}

// HeartbeatKind matches landing MeshConsole kinds.
type HeartbeatKind string

const (
	kindOps       HeartbeatKind = "ops"
	kindKnowledge HeartbeatKind = "knowledge"
	kindAnalytics HeartbeatKind = "analytics"
)

// HeartbeatEvent is one row in the landing-page heartbeat feed.
type HeartbeatEvent struct {
	T      string
	Dept   string
	Kind   HeartbeatKind
	Title  string
	Detail string
}

// MCPCall is a policy-gated tool chip (landing Agent tools column).
type MCPCall struct {
	Tool   string
	Policy string
	Dept   string
}

func landingTenancies() []string {
	return []string{"sre.incidents", "eng.ops", "cs.tickets", "gtm.pipeline"}
}

func landingSeedEvents() []HeartbeatEvent {
	return []HeartbeatEvent{
		{T: "14:02:11", Dept: "sre.incidents", Kind: kindOps, Title: "P2 opened — checkout p95 1.8s", Detail: "Signed from pager. Team tenancy: sre.oncall"},
		{T: "14:02:18", Dept: "eng.ops", Kind: kindOps, Title: "Deploy v2.14.3 → prod-eu", Detail: "Heartbeat linked to runbook RB-441"},
		{T: "14:02:24", Dept: "cs.tickets", Kind: kindOps, Title: "Enterprise ticket · latency on checkout", Detail: "Scoped to cs.enterprise — not a global index"},
		{T: "14:02:31", Dept: "sre.incidents", Kind: kindKnowledge, Title: "Recall: similar p95 in Mar", Detail: "Local memory · dual_write OFF"},
		{T: "14:02:39", Dept: "gtm.pipeline", Kind: kindAnalytics, Title: "Renewal risk +12% this week", Detail: "Pattern across heartbeats · Beta"},
	}
}

func landingMoreEvents() []HeartbeatEvent {
	return []HeartbeatEvent{
		{Dept: "eng.ops", Kind: kindOps, Title: "Canary 12% · error budget 97.4%", Detail: "dept.eng tenancy · policy ALLOW"},
		{Dept: "sre.incidents", Kind: kindKnowledge, Title: "Runbook RB-441 attached", Detail: "Institutional memory of last three pages"},
		{Dept: "cs.tickets", Kind: kindOps, Title: "CS linked incident INC-2041", Detail: "Same pulse, separate tenancy"},
		{Dept: "gtm.pipeline", Kind: kindOps, Title: "Stage change · Negotiation", Detail: "Signed CRM webhook · connector gtm"},
		{Dept: "sre.incidents", Kind: kindAnalytics, Title: "Checkout p95 reverting 14m", Detail: "Short-horizon stream history · GA"},
		{Dept: "eng.ops", Kind: kindOps, Title: "Feature flag checkout_v3 off", Detail: "Agent proposed · human confirmed"},
	}
}

func landingMCPCalls() []MCPCall {
	return []MCPCall{
		{Tool: "mesh.ops.pull", Policy: "ALLOW", Dept: "sre.incidents"},
		{Tool: "mesh.knowledge.search", Policy: "ALLOW", Dept: "sre.incidents"},
		{Tool: "mesh.gtm.forecast", Policy: "DENY", Dept: "sre.incidents"},
		{Tool: "mesh.ops.annotate", Policy: "ALLOW", Dept: "eng.ops"},
	}
}

// dashboardState is the landing-page MeshConsole, for REPL snapshot + fullscreen overlay.
type dashboardState struct {
	Focus         string
	Rate          int
	Events        []HeartbeatEvent
	idx           int
	phase         int
	MeshAttached  bool
	Preview       bool
	Width         int
	Height        int
	ConsumeReason string
	StreamNames   []string
	// BriefAck is today's morning brief state. Empty → load on render (fail-open unread).
	BriefAck BriefAckStatus
}

func newDashboardState(meshAttached bool) *dashboardState {
	return &dashboardState{
		Focus:         dashboardDefaultFocus,
		Rate:          0,
		Events:        nil,
		MeshAttached:  meshAttached,
		Preview:       false,
		ConsumeReason: consumeReasonMissing,
	}
}

func newDashboardPreviewState(meshAttached bool) *dashboardState {
	return &dashboardState{
		Focus:        dashboardDefaultFocus,
		Rate:         dashboardStartRate,
		Events:       landingSeedEvents(),
		MeshAttached: meshAttached,
		Preview:      true,
	}
}

func (d *dashboardState) SetFocus(name string) bool {
	name = strings.TrimSpace(name)
	for _, t := range landingTenancies() {
		if t == name {
			d.Focus = name
			return true
		}
	}
	return false
}

func (d *dashboardState) CycleFocus() {
	list := landingTenancies()
	for i, t := range list {
		if t == d.Focus {
			d.Focus = list[(i+1)%len(list)]
			return
		}
	}
	d.Focus = dashboardDefaultFocus
}

func (d *dashboardState) Calls() []MCPCall {
	var out []MCPCall
	for _, c := range landingMCPCalls() {
		if c.Dept == d.Focus || c.Dept == "eng.ops" {
			out = append(out, c)
		}
		if len(out) == 3 {
			break
		}
	}
	return out
}

func (d *dashboardState) Tick() {
	if d == nil || !d.Preview {
		return
	}
	more := landingMoreEvents()
	if len(more) == 0 {
		return
	}
	next := more[d.idx%len(more)]
	next.T = time.Now().Format("15:04:05")
	d.Events = append([]HeartbeatEvent{next}, d.Events...)
	if len(d.Events) > dashboardMaxEvents {
		d.Events = d.Events[:dashboardMaxEvents]
	}
	if d.idx%3 == 0 && d.Rate < dashboardMaxRate {
		d.Rate++
	}
	d.idx++
	d.phase++
}

func (d *dashboardState) kindCounts() (ops, knowledge, analytics int) {
	for _, e := range d.Events {
		switch e.Kind {
		case kindOps:
			ops++
		case kindKnowledge:
			knowledge++
		case kindAnalytics:
			analytics++
		}
	}
	return
}

func pulseSpark(width, phase int) string {
	if width < 8 {
		width = 8
	}
	cycle := []rune("▁▁▁▂▃▅█▅▃▂▁▁▁▂▃▆█▆▃▂")
	var b strings.Builder
	for i := 0; i < width; i++ {
		b.WriteRune(cycle[(i+phase)%len(cycle)])
	}
	return b.String()
}

func kindStyle(th Theme, k HeartbeatKind) lipgloss.Style {
	switch k {
	case kindOps:
		return th.OK
	case kindKnowledge:
		return th.Tool
	default:
		return th.Dim
	}
}

func (d *dashboardState) Render(th Theme, width int) string {
	if width < 40 {
		width = 40
	}
	d.Width = width
	badge := "EMPTY"
	if d.Preview {
		badge = "EVAL"
	} else if len(d.Events) > 0 {
		// PULSE only after ≥1 decoded broker message — never from eval or stream list.
		badge = "PULSE"
	} else if d.MeshAttached {
		badge = "CLIENT"
	}
	reason := strings.TrimSpace(d.ConsumeReason)
	if reason == "" {
		reason = consumeReasonMissing
	}
	ctx := "no live heartbeat · " + reason + " · " + d.Focus
	if d.Preview {
		ctx = "eval template preview · not your org · " + d.Focus
	} else if len(d.Events) > 0 {
		ctx = "consumed · " + reason + " · " + d.Focus
	} else if d.MeshAttached {
		ctx = "mesh client attached · " + reason + " · " + d.Focus
	}
	headLeft := th.Mesh.Render("●") + " " + th.Dim.Render("context://mesh · "+ctx)
	headRight := th.OK.Render(badge)
	gap := width - lipgloss.Width(headLeft) - lipgloss.Width(headRight)
	if gap < 1 {
		gap = 1
	}
	header := headLeft + strings.Repeat(" ", gap) + headRight
	rule := th.Dim.Render(strings.Repeat("─", width))
	spark := th.Mesh.Render(pulseSpark(width, d.phase))

	opsN, knN, anN := d.kindCounts()
	analysis := th.Dim.Render(fmt.Sprintf(
		"analysis  ops %d · knowledge %d · analytics %d  ·  knowledge/analytics Beta",
		opsN, knN, anN,
	))
	if knN == 0 || anN == 0 {
		analysis += "\n" + th.Dim.Render(DashboardBetaEmptyHonesty)
	}
	compose := th.Dim.Render("compose  " + DashboardComposePulse)
	compose += "\n" + th.Dim.Render(DashboardComposePull)
	compose += "\n" + th.Dim.Render(DashboardComposeInsights)
	compose += "\n" + th.Dim.Render(DashboardComposeDecision)
	briefStatus := d.BriefAck
	if briefStatus == "" {
		briefStatus = loadBriefAckStatus()
		d.BriefAck = briefStatus
	}
	briefLine := composeBriefLine(briefStatus)
	// Unacked must be visible (not Dim fake-handled / not OK green). ACKed stays Dim.
	if briefStatus == BriefAcked {
		compose += "\n" + th.Dim.Render(briefLine)
	} else {
		compose += "\n" + th.Status.Render(briefLine)
	}

	body := d.renderBody(th, width)
	honesty := th.Dim.Render(DashboardHonestyOneLiner)
	if d.Preview {
		honesty = th.Dim.Render("eval template preview · not your org · " + DashboardHonestyOneLiner)
	} else if d.MeshAttached {
		honesty = th.Dim.Render("mesh client attached · no mock live rows · /dashboard preview for eval template · " + DashboardHonestyOneLiner)
	}

	return strings.Join([]string{header, rule, spark, analysis, compose, rule, body, rule, honesty}, "\n")
}

func (d *dashboardState) renderBody(th Theme, width int) string {
	if !d.Preview {
		return d.renderFeed(th, width)
	}
	if width >= 88 {
		return d.renderWide(th, width)
	}
	return d.renderStacked(th, width)
}

func (d *dashboardState) renderWide(th Theme, width int) string {
	leftW := 18
	rightW := 24
	sepW := 1
	midW := width - leftW - rightW - 2*sepW
	if midW < 28 {
		return d.renderStacked(th, width)
	}
	col := func(s string, w int) string {
		return lipgloss.NewStyle().Width(w).MaxWidth(w).Render(s)
	}
	sep := th.Dim.Render("│")
	return lipgloss.JoinHorizontal(lipgloss.Top,
		col(d.renderTenancy(th, leftW), leftW),
		sep,
		col(d.renderFeed(th, midW), midW),
		sep,
		col(d.renderTools(th, rightW), rightW),
	)
}

func (d *dashboardState) renderStacked(th Theme, width int) string {
	var b strings.Builder
	b.WriteString(d.renderTenancy(th, width))
	b.WriteByte('\n')
	b.WriteString(d.renderFeed(th, width))
	b.WriteByte('\n')
	b.WriteString(d.renderTools(th, width))
	return b.String()
}

func (d *dashboardState) renderTenancy(th Theme, width int) string {
	var b strings.Builder
	b.WriteString(th.Help.Render("Tenancy"))
	b.WriteByte('\n')
	for _, t := range landingTenancies() {
		label := truncate(t, max(1, width-2))
		if t == d.Focus {
			b.WriteString(th.Title.Render("▸ " + label))
		} else {
			b.WriteString(th.Dim.Render("  " + label))
		}
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(th.Help.Render("Pulse"))
	b.WriteByte('\n')
	b.WriteString(th.Title.Render(fmt.Sprintf("%d", d.Rate)))
	b.WriteByte('\n')
	b.WriteString(th.Dim.Render("events / min"))
	return b.String()
}

func (d *dashboardState) renderFeed(th Theme, width int) string {
	var b strings.Builder
	title := th.Help.Render("Heartbeat")
	rate := th.Dim.Render(fmt.Sprintf("%d / min", d.Rate))
	gap := width - lipgloss.Width(title) - lipgloss.Width(rate)
	if gap < 1 {
		gap = 1
	}
	b.WriteString(title + strings.Repeat(" ", gap) + rate)
	b.WriteByte('\n')
	if len(d.Events) == 0 {
		b.WriteString(th.Dim.Render("no consumed messages · mock eval rows hidden"))
		b.WriteByte('\n')
		if !d.MeshAttached {
			// Visual leftover: one Status (ANSI 245) next action; honesty stays Dim.
			b.WriteString(th.Status.Render(dashboardEmptyPrimaryNext))
			b.WriteByte('\n')
			b.WriteString(th.Dim.Render(dashboardEmptyHonestyCatalog))
			b.WriteByte('\n')
			b.WriteString(th.Dim.Render(dashboardEmptyHonestyInfer))
		} else {
			switch strings.TrimSpace(d.ConsumeReason) {
			case consumeReasonEmptyStream:
				listed := strings.Join(d.StreamNames, ", ")
				if listed == "" {
					listed = "(named)"
				}
				b.WriteString(th.Dim.Render("empty_stream · listed " + listed + " · 0 messages"))
				b.WriteByte('\n')
				b.WriteString(th.Dim.Render("iomesh mesh streams --messages · broker GET /v1 (not /v52)"))
			case consumeReasonReplayDisabled:
				b.WriteString(th.Dim.Render("replay_disabled · GET /v1/streams/{name}/messages 403"))
				b.WriteByte('\n')
				b.WriteString(th.Dim.Render("needs X-IOMesh-Tenant or AION_MEMORY_REPLAY_ENABLED · not /v52"))
			case consumeReasonBrokerUnavailable:
				b.WriteString(th.Dim.Render("broker_unavailable · consume probe failed"))
				b.WriteByte('\n')
				b.WriteString(th.Dim.Render("probe uses mesh streams --messages / broker /v1 · not /v52"))
			default:
				// no_streams (mesh attached): create-stream CTA (create ≠ PULSE)
				b.WriteString(th.Dim.Render("create a mesh stream: console Settings → Mesh routing → Streams"))
				b.WriteByte('\n')
				b.WriteString(th.Dim.Render("or: iomesh mesh streams --create --yes"))
				b.WriteByte('\n')
				b.WriteString(th.Dim.Render("then iomesh mesh streams --messages --name OPERATIONAL_EVENTS"))
			}
		}
		b.WriteByte('\n')
		b.WriteString(th.Dim.Render("/dashboard preview · eval template on iome.sh (not your org)"))
		return strings.TrimRight(b.String(), "\n")
	}
	for _, e := range d.Events {
		b.WriteString(th.Dim.Render(e.T))
		b.WriteByte(' ')
		b.WriteString(kindStyle(th, e.Kind).Render(string(e.Kind)))
		b.WriteByte(' ')
		b.WriteString(th.Dim.Render(e.Dept))
		b.WriteByte('\n')
		b.WriteString(th.Status.Render(truncate(e.Title, width)))
		b.WriteByte('\n')
		b.WriteString(th.Dim.Render(truncate(e.Detail, width)))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func (d *dashboardState) renderTools(th Theme, width int) string {
	var b strings.Builder
	b.WriteString(th.Help.Render("Agent tools"))
	b.WriteByte('\n')
	for _, c := range d.Calls() {
		b.WriteString(th.Status.Render(truncate(c.Tool, width)))
		b.WriteByte('\n')
		pol := th.OK.Render(c.Policy)
		if c.Policy == "DENY" {
			pol = th.Err.Render(c.Policy)
		}
		b.WriteString(pol)
		b.WriteByte('\n')
	}
	b.WriteString(th.Dim.Render("Tools resolve under team tenancy."))
	b.WriteByte('\n')
	b.WriteString(th.Dim.Render("Cross-department invokes denied by default."))
	return strings.TrimRight(b.String(), "\n")
}

func formatDashboardSnapshot(meshAttached bool, focus string) string {
	return formatDashboardSnapshotMode(meshAttached, focus, false)
}

func formatDashboardSnapshotMode(meshAttached bool, focus string, preview bool) string {
	var d *dashboardState
	if preview {
		d = newDashboardPreviewState(meshAttached)
	} else {
		d = newDashboardState(meshAttached)
	}
	if focus != "" {
		_ = d.SetFocus(focus)
	}
	return d.Render(ThemeDefault(), 100)
}

func meshClientAttached(rt runtimeAdapter) bool {
	if rt.rt == nil {
		return false
	}
	m := rt.rt.Mesh()
	return m != nil && m.Enabled()
}

func handleDashboardSlash(out io.Writer, rt runtimeAdapter, parts []string) {
	if len(parts) < 2 {
		d := newDashboardState(meshClientAttached(rt))
		probeDashboardIfAttached(d, rt)
		fmt.Fprintln(out, d.Render(ThemeDefault(), 100))
		return
	}
	sub := strings.ToLower(parts[1])
	switch sub {
	case "help", "?":
		fmt.Fprintln(out, dashboardHelp())
	case "preview":
		fmt.Fprintln(out, formatDashboardSnapshotMode(meshClientAttached(rt), "", true))
	case "ack":
		msg, err := ackTodayBrief()
		if err != nil {
			fmt.Fprintf(out, "dashboard ack: %v · fail-open unread stays visible\n", err)
			return
		}
		fmt.Fprintln(out, msg)
		d := newDashboardState(meshClientAttached(rt))
		d.BriefAck = BriefAcked
		probeDashboardIfAttached(d, rt)
		fmt.Fprintln(out, d.Render(ThemeDefault(), 100))
	case "focus":
		want := ""
		if len(parts) > 2 {
			want = parts[2]
		}
		d := newDashboardState(meshClientAttached(rt))
		if want == "" || !d.SetFocus(want) {
			fmt.Fprintf(out, "dashboard: unknown tenancy %q (want %s)\n%s\n",
				want, strings.Join(landingTenancies(), " | "), dashboardHelp())
			return
		}
		probeDashboardIfAttached(d, rt)
		fmt.Fprintln(out, d.Render(ThemeDefault(), 100))
	default:
		// Bare tenancy token: /dashboard sre.incidents
		d := newDashboardState(meshClientAttached(rt))
		if d.SetFocus(parts[1]) {
			probeDashboardIfAttached(d, rt)
			fmt.Fprintln(out, d.Render(ThemeDefault(), 100))
			return
		}
		fmt.Fprintf(out, "dashboard: unknown subcommand %q\n%s\n", parts[1], dashboardHelp())
	}
}

func isDashboardSlash(cmd string) bool {
	switch strings.ToLower(strings.TrimSpace(cmd)) {
	case "/dashboard", "/heartbeat", "/mesh-console":
		return true
	default:
		return false
	}
}

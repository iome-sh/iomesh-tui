package tui

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestDashboardSnapshot_LandingParityAndHonesty(t *testing.T) {
	out := formatDashboardSnapshot(false, "")
	needles := []string{
		"context://mesh",
		"no live heartbeat",
		"EMPTY",
		"Heartbeat",
		"no consumed messages",
		"mock eval rows hidden",
		"add [iomesh]",
		"https://hooks.iome.sh",
		"then consume GitHub",
		"infer from portal MCP",
		"hooks.iome.sh",
		"infer ≠ Connected",
		"do not invent consume",
		"/dashboard preview",
		"dual_write OFF",
		"catalog ≠ Connected",
		"not Memory GA",
		"not live APPLY",
		"eval template",
		DashboardBetaEmptyHonesty,
		DashboardComposePulse,
		DashboardComposePull,
		DashboardComposeInsights,
		DashboardComposeDecision,
		"never auto-applies",
	}
	for _, n := range needles {
		if !strings.Contains(out, n) {
			t.Fatalf("snapshot missing %q\n%s", n, out)
		}
	}
	for _, bad := range []string{
		"Memory GA shipped",
		"Connected workspace",
		"live APPLY green",
		"fleet-GA on every surface",
		"INSTALL_STORE",
		"auto-apply green",
		"hosted Memory GA",
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("snapshot must not invent %q\n%s", bad, out)
		}
	}
}

func TestDashboardCompose_SmokeNeverAutoApplies(t *testing.T) {
	out := formatDashboardSnapshot(false, "")
	if !strings.Contains(out, DashboardComposePulse) {
		t.Fatalf("compose smoke missing pulse:\n%s", out)
	}
	if !strings.Contains(out, DashboardComposePull) {
		t.Fatalf("compose smoke missing pull:\n%s", out)
	}
	if !strings.Contains(out, DashboardComposeInsights) {
		t.Fatalf("compose smoke missing insights:\n%s", out)
	}
	if !strings.Contains(out, DashboardComposeDecision) || !strings.Contains(out, "never auto-applies") {
		t.Fatalf("compose smoke missing decision stub:\n%s", out)
	}
	if strings.Contains(out, "auto-apply green") {
		t.Fatalf("compose smoke must not auto-apply:\n%s", out)
	}
}

func TestDashboardSnapshot_MeshAttachedLabel(t *testing.T) {
	out := formatDashboardSnapshot(true, "eng.ops")
	if !strings.Contains(out, "CLIENT") {
		t.Fatalf("attached snapshot missing CLIENT badge:\n%s", out)
	}
	if !strings.Contains(out, "mesh client attached") {
		t.Fatalf("attached snapshot missing client honesty:\n%s", out)
	}
	if !strings.Contains(out, "CLIENT") {
		t.Fatalf("attached snapshot missing CLIENT:\n%s", out)
	}
	if !strings.Contains(out, "eval template") {
		t.Fatalf("attached view must keep eval-template honesty:\n%s", out)
	}
	if strings.Contains(out, "P2 opened") {
		t.Fatalf("default attached view must not show mock eval rows:\n%s", out)
	}
}

func TestDashboardSnapshot_NoMeshTellsInferNotCreate(t *testing.T) {
	out := formatDashboardSnapshot(false, "")
	if !strings.Contains(out, "add [iomesh]") || !strings.Contains(out, "infer from portal MCP") {
		t.Fatalf("unattached must tell operator to add [iomesh] or infer:\n%s", out)
	}
	if !strings.Contains(out, dashboardEmptyPrimaryNext) {
		t.Fatalf("unattached missing visual hooks.iome.sh next action:\n%s", out)
	}
	if strings.Contains(out, "create a mesh stream") {
		t.Fatalf("unattached must not jump to create-stream CTA:\n%s", out)
	}
	if strings.Contains(out, "PULSE") && !strings.Contains(out, "create ≠ PULSE") {
		t.Fatalf("unattached must not invent PULSE:\n%s", out)
	}
}

func TestDashboardEmpty_PrimaryNextActionContrast(t *testing.T) {
	th := ThemeDefault()
	if got := fmt.Sprint(th.Dim.GetForeground()); got != "241" {
		t.Fatalf("ThemeDefault Dim want ANSI 241, got %s", got)
	}
	if got := fmt.Sprint(th.Status.GetForeground()); got != "245" {
		t.Fatalf("ThemeDefault Status want ANSI 245, got %s", got)
	}

	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	out := formatDashboardSnapshot(false, "")
	primary := th.Status.Render(dashboardEmptyPrimaryNext)
	if !strings.Contains(out, primary) {
		t.Fatalf("primary next action must use Theme.Status (ANSI 245), not Dim:\n%s", out)
	}
	if strings.Contains(out, th.Dim.Render(dashboardEmptyPrimaryNext)) {
		t.Fatalf("primary next action must not use Theme.Dim (ANSI 241):\n%s", out)
	}
	if !strings.Contains(out, th.Dim.Render(dashboardEmptyHonestyCatalog)) {
		t.Fatalf("catalog honesty must stay Dim:\n%s", out)
	}
	if !strings.Contains(out, th.Dim.Render(dashboardEmptyHonestyInfer)) {
		t.Fatalf("infer honesty must stay Dim:\n%s", out)
	}
	if strings.Contains(out, th.Status.Render(dashboardEmptyHonestyCatalog)) ||
		strings.Contains(out, th.Status.Render(dashboardEmptyHonestyInfer)) {
		t.Fatalf("honesty lines must not compete as Status CTAs:\n%s", out)
	}
}

func TestThemeDefault_DashboardInkContrast(t *testing.T) {
	// Design Packet 4 visual leftover: ThemeDefault() on empty-pane ink #09090b.
	// Dim 241 (#626262) is the leftover FAIL; Status 245 (#8a8a8a) is the primary pass.
	// Does not probe consume or set IOMESH_* secrets.
	ink := [3]float64{0x09, 0x09, 0x0b}
	dim := ansi256GrayRGB(241)
	status := ansi256GrayRGB(245)
	before := contrastRatio(dim, ink)
	after := contrastRatio(status, ink)
	if before >= 4.5 {
		t.Fatalf("Dim 241 on #09090b unexpectedly passes AA: %.2f:1", before)
	}
	if after < 4.5 {
		t.Fatalf("Status 245 on #09090b must be ≥4.5:1, got %.2f:1", after)
	}
	if before < 3.2 || before > 3.4 {
		t.Fatalf("Dim 241 on #09090b want ~3.26:1, got %.2f:1", before)
	}
	if after < 5.6 || after > 5.9 {
		t.Fatalf("Status 245 on #09090b want ~5.73:1, got %.2f:1", after)
	}
}

// ansi256GrayRGB returns the xterm 256 grayscale cube (232–255).
func ansi256GrayRGB(code int) [3]float64 {
	v := float64(8 + 10*(code-232))
	return [3]float64{v, v, v}
}

func contrastRatio(fg, bg [3]float64) float64 {
	l1 := relativeLuminance(fg)
	l2 := relativeLuminance(bg)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func relativeLuminance(rgb [3]float64) float64 {
	lin := func(c float64) float64 {
		s := c / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(rgb[0]) + 0.7152*lin(rgb[1]) + 0.0722*lin(rgb[2])
}

func TestDashboardPreview_OptInEvalTemplate(t *testing.T) {
	out := formatDashboardSnapshotMode(false, "", true)
	if !strings.Contains(out, "EVAL") {
		t.Fatalf("preview missing EVAL:\n%s", out)
	}
	if !strings.Contains(out, "P2 opened — checkout p95 1.8s") {
		t.Fatalf("preview missing eval seed:\n%s", out)
	}
	if !strings.Contains(out, "not your org") {
		t.Fatalf("preview missing not-your-org honesty:\n%s", out)
	}
}

func TestDashboardFocus_FiltersMCPCalls(t *testing.T) {
	d := newDashboardState(false)
	if !d.SetFocus("sre.incidents") {
		t.Fatal("set sre.incidents")
	}
	calls := d.Calls()
	if len(calls) != 3 {
		t.Fatalf("want 3 calls, got %d", len(calls))
	}
	// Landing filter: focus dept or eng.ops.
	for _, c := range calls {
		if c.Dept != "sre.incidents" && c.Dept != "eng.ops" {
			t.Fatalf("unexpected call dept %s", c.Dept)
		}
	}
	if !d.SetFocus("gtm.pipeline") {
		t.Fatal("set gtm.pipeline")
	}
	// gtm focus still includes eng.ops annotate; forecast is sre.incidents-only.
	got := d.Calls()
	if len(got) == 0 {
		t.Fatal("gtm focus should still include eng.ops annotate")
	}
	if d.SetFocus("not.a.dept") {
		t.Fatal("unknown tenancy must fail")
	}
}

func TestDashboardTick_AdvancesFeed(t *testing.T) {
	empty := newDashboardState(false)
	empty.Tick()
	if len(empty.Events) != 0 {
		t.Fatalf("empty dashboard must not invent mock ticks: %+v", empty.Events)
	}
	d := newDashboardPreviewState(false)
	first := d.Events[0].Title
	d.Tick()
	if d.Events[0].Title == first && d.Events[0].T == "14:02:11" {
		t.Fatalf("tick should prepend a stamped event: %+v", d.Events[0])
	}
	if len(d.Events) > dashboardMaxEvents {
		t.Fatalf("feed grew past %d: %d", dashboardMaxEvents, len(d.Events))
	}
	if d.Events[0].T == "" {
		t.Fatal("tick event missing timestamp")
	}
}

func TestDashboardCycleFocus(t *testing.T) {
	d := newDashboardState(false)
	seen := map[string]bool{d.Focus: true}
	for i := 0; i < 4; i++ {
		d.CycleFocus()
		seen[d.Focus] = true
	}
	for _, tency := range landingTenancies() {
		if !seen[tency] {
			t.Fatalf("cycle missed %s", tency)
		}
	}
}

func TestPulseSpark_Width(t *testing.T) {
	s := pulseSpark(20, 0)
	if len([]rune(s)) != 20 {
		t.Fatalf("spark width=%d want 20 (%q)", len([]rune(s)), s)
	}
}

func TestReadmeDashboardShowcase(t *testing.T) {
	b, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	readme := string(b)
	for _, n := range []string{
		"docs/assets/dashboard-eval.svg",
		"/dashboard",
		"preview",
		"MeshConsole",
		"console.iome.sh",
		"EVAL",
		"EMPTY",
		"catalog ≠ Connected",
		"dual_write OFF",
		"not live APPLY",
		"eval template",
		"empty until consume",
		"knowledge Beta empty",
	} {
		if !strings.Contains(readme, n) {
			t.Fatalf("README showcase missing %q", n)
		}
	}
	if strings.Contains(readme, "TUI `/dashboard`") && strings.Contains(readme, "Same eval template") {
		t.Fatal("README must not claim TUI /dashboard = Same eval template")
	}
	if strings.Contains(readme, "tenant GIF") && strings.Contains(readme, "live tenant feed as proof") {
		t.Fatal("README must not sell a tenant GIF as Connected proof")
	}
}

func TestHandleSlash_Dashboard(t *testing.T) {
	rt := testRuntime(t)
	adapter := runtimeAdapter{rt: rt}

	var out bytes.Buffer
	quit, err := handleSlash(&out, adapter, "/dashboard")
	if quit || err != nil {
		t.Fatalf("quit=%v err=%v", quit, err)
	}
	if strings.Contains(out.String(), "P2 opened") {
		t.Fatalf("/dashboard must not show mock eval rows:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "no consumed messages") {
		t.Fatalf("/dashboard missing empty honesty:\n%s", out.String())
	}

	out.Reset()
	_, _ = handleSlash(&out, adapter, "/dashboard preview")
	if !strings.Contains(out.String(), "P2 opened") {
		t.Fatalf("/dashboard preview missing eval seed:\n%s", out.String())
	}

	out.Reset()
	_, _ = handleSlash(&out, adapter, "/heartbeat help")
	if !strings.Contains(out.String(), "usage: /dashboard") {
		t.Fatalf("/heartbeat help missing usage:\n%s", out.String())
	}

	out.Reset()
	_, _ = handleSlash(&out, adapter, "/mesh-console focus cs.tickets")
	if !strings.Contains(out.String(), "cs.tickets") {
		t.Fatalf("focus cs.tickets missing:\n%s", out.String())
	}

	out.Reset()
	_, _ = handleSlash(&out, adapter, "/dashboard nope")
	if !strings.Contains(out.String(), "unknown subcommand") {
		t.Fatalf("unknown subcommand:\n%s", out.String())
	}

	out.Reset()
	_, _ = handleSlash(&out, adapter, "/help")
	help := out.String()
	if !strings.Contains(help, "/dashboard") || !strings.Contains(help, "/heartbeat") {
		t.Fatalf("/help missing dashboard:\n%s", help)
	}
}

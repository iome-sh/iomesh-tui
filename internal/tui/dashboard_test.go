package tui

import (
	"bytes"
	"os"
	"strings"
	"testing"
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
		"/dashboard preview",
		"dual_write OFF",
		"catalog ≠ Connected",
		"not Memory GA",
		"not live APPLY",
		"eval template",
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
	} {
		if strings.Contains(out, bad) {
			t.Fatalf("snapshot must not invent %q\n%s", bad, out)
		}
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
		"MeshConsole",
		"console.iome.sh",
		"EVAL",
		"catalog ≠ Connected",
		"dual_write OFF",
		"not live APPLY",
		"eval template",
	} {
		if !strings.Contains(readme, n) {
			t.Fatalf("README showcase missing %q", n)
		}
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

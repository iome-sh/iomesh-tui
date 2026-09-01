package tui

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupBriefAck isolates ACK path + clock for this test only.
// Do not use package TestMain — that hijacks every internal/tui test.
func setupBriefAck(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("IOMESH_CONFIG", filepath.Join(dir, "config.toml"))
	path := filepath.Join(dir, briefAckFileName)

	prevPath := briefAckPathFn
	prevNow := briefAckNowFn
	briefAckPathFn = func() (string, error) { return path, nil }
	briefAckNowFn = func() time.Time {
		return time.Date(2026, 9, 1, 9, 0, 0, 0, time.Local)
	}
	t.Cleanup(func() {
		briefAckPathFn = prevPath
		briefAckNowFn = prevNow
	})
}

func assertDashboardHonesty(t *testing.T, out string) {
	t.Helper()
	for _, n := range []string{"dual_write OFF", "not Memory GA", "catalog ≠ Connected"} {
		if !strings.Contains(out, n) {
			t.Fatalf("honesty missing %q:\n%s", n, out)
		}
	}
}

func TestBriefAck_MissingIsUnreadFailOpen(t *testing.T) {
	setupBriefAck(t)

	if got := loadBriefAckStatus(); got != BriefUnread {
		t.Fatalf("missing ACK want unread, got %q", got)
	}
	out := formatDashboardSnapshot(false, "")
	if !strings.Contains(out, DashboardComposeBriefUnread) {
		t.Fatalf("missing ACK must render UNREAD:\n%s", out)
	}
	if strings.Contains(out, DashboardComposeBriefAcked) {
		t.Fatalf("missing ACK must not render ACKed:\n%s", out)
	}
	assertDashboardHonesty(t, out)
	for _, bad := range []string{"handled", "known green", "auto send", "auto-pay", "auto-ship"} {
		if strings.Contains(strings.ToLower(out), bad) {
			t.Fatalf("unacked must not look %q:\n%s", bad, out)
		}
	}
}

func TestBriefAck_CorruptAndStaleAreUnread(t *testing.T) {
	setupBriefAck(t)
	path, err := briefAckPathFn()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := loadBriefAckStatus(); got != BriefUnread {
		t.Fatalf("corrupt want unread, got %q", got)
	}

	stale := briefAckRecord{Day: "2020-01-01", AckedAt: "2020-01-01T00:00:00Z", Note: "stale"}
	raw, _ := json.Marshal(stale)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := loadBriefAckStatus(); got != BriefUnread {
		t.Fatalf("stale day want unread, got %q", got)
	}
}

func TestDashboardCompose_BriefAckUnreadVsAcked(t *testing.T) {
	setupBriefAck(t)

	unread := formatDashboardSnapshot(false, "")
	if !strings.Contains(unread, DashboardComposeBriefUnread) {
		t.Fatalf("compose unread missing:\n%s", unread)
	}
	if strings.Contains(unread, DashboardComposeBriefAcked) {
		t.Fatalf("unread must not render ACKed:\n%s", unread)
	}
	assertDashboardHonesty(t, unread)
	if strings.Contains(unread, "send/pay/ship") && strings.Contains(unread, "ACKed") {
		t.Fatalf("unread must not claim ACKed closed loop:\n%s", unread)
	}

	msg, err := ackTodayBrief()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "no send/pay/ship") {
		t.Fatalf("ack message must pin no closed loop: %s", msg)
	}
	if !strings.Contains(msg, "local palace write") {
		t.Fatalf("ack message must mention optional local palace write: %s", msg)
	}
	if got := loadBriefAckStatus(); got != BriefAcked {
		t.Fatalf("after ack want acked, got %q", got)
	}

	acked := formatDashboardSnapshot(false, "")
	if !strings.Contains(acked, DashboardComposeBriefAcked) {
		t.Fatalf("compose acked missing:\n%s", acked)
	}
	if strings.Contains(acked, DashboardComposeBriefUnread) {
		t.Fatalf("acked must not still say UNREAD:\n%s", acked)
	}
	if !strings.Contains(acked, "no send/pay/ship") {
		t.Fatalf("acked must still refuse closed loop:\n%s", acked)
	}
	assertDashboardHonesty(t, acked)
	for _, bad := range []string{"auto-apply green", "sent payment", "shipped order"} {
		if strings.Contains(acked, bad) {
			t.Fatalf("ACK must not invent closed loop %q:\n%s", bad, acked)
		}
	}
}

func TestDashboardSlash_AckRitual(t *testing.T) {
	setupBriefAck(t)

	var out bytes.Buffer
	handleDashboardSlash(&out, runtimeAdapter{}, []string{"/dashboard", "ack"})
	got := out.String()
	if !strings.Contains(got, "brief ACKed") {
		t.Fatalf("ack slash missing confirmation:\n%s", got)
	}
	if !strings.Contains(got, DashboardComposeBriefAcked) {
		t.Fatalf("ack slash must re-render ACKed brief:\n%s", got)
	}
	if !strings.Contains(got, "no send/pay/ship") {
		t.Fatalf("ack slash must pin no closed loop:\n%s", got)
	}
	if strings.Contains(got, "auto-apply green") {
		t.Fatalf("ack must not auto-apply:\n%s", got)
	}
	assertDashboardHonesty(t, got)
	path, err := briefAckPathFn()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("ack must write local palace marker: %v", err)
	}
}

func TestHandleSlash_DashboardAck(t *testing.T) {
	setupBriefAck(t)
	rt := testRuntime(t)
	adapter := runtimeAdapter{rt: rt}

	var out bytes.Buffer
	quit, err := handleSlash(&out, adapter, "/dashboard ack")
	if quit || err != nil {
		t.Fatalf("quit=%v err=%v", quit, err)
	}
	got := out.String()
	if !strings.Contains(got, "brief ACKed") {
		t.Fatalf("REPL /dashboard ack missing confirmation:\n%s", got)
	}
	if !strings.Contains(got, DashboardComposeBriefAcked) {
		t.Fatalf("REPL /dashboard ack must re-render ACKed brief:\n%s", got)
	}
	if !strings.Contains(got, "no send/pay/ship") {
		t.Fatalf("REPL /dashboard ack must pin no closed loop:\n%s", got)
	}
	assertDashboardHonesty(t, got)
	for _, bad := range []string{"auto-apply green", "sent payment", "shipped order", "auto send", "auto-pay", "auto-ship"} {
		if strings.Contains(strings.ToLower(got), bad) {
			t.Fatalf("REPL ack must not invent closed loop %q:\n%s", bad, got)
		}
	}
}

func TestDashboardHelp_MentionsAck(t *testing.T) {
	h := dashboardHelp()
	if !strings.Contains(h, "ack") || !strings.Contains(h, "unread ≠ known") {
		t.Fatalf("help must document ack ritual:\n%s", h)
	}
	if !strings.Contains(h, "no send/pay/ship") {
		t.Fatalf("help must pin no closed loop:\n%s", h)
	}
}

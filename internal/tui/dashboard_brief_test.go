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

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "iomesh-brief-ack-*")
	if err != nil {
		panic(err)
	}
	path := filepath.Join(dir, briefAckFileName)
	briefAckPathFn = func() (string, error) { return path, nil }
	briefAckNowFn = func() time.Time {
		return time.Date(2026, 9, 1, 9, 0, 0, 0, time.Local)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func clearBriefAckFile(t *testing.T) {
	t.Helper()
	path, err := briefAckPathFn()
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(path)
	t.Cleanup(func() { _ = os.Remove(path) })
}

func TestBriefAck_MissingIsUnreadFailOpen(t *testing.T) {
	clearBriefAckFile(t)

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
	for _, bad := range []string{"handled", "known green", "auto send", "auto-pay", "auto-ship"} {
		if strings.Contains(strings.ToLower(out), bad) {
			t.Fatalf("unacked must not look %q:\n%s", bad, out)
		}
	}
}

func TestBriefAck_CorruptAndStaleAreUnread(t *testing.T) {
	clearBriefAckFile(t)
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
	clearBriefAckFile(t)

	unread := formatDashboardSnapshot(false, "")
	if !strings.Contains(unread, DashboardComposeBriefUnread) {
		t.Fatalf("compose unread missing:\n%s", unread)
	}
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
	for _, bad := range []string{"auto-apply green", "sent payment", "shipped order"} {
		if strings.Contains(acked, bad) {
			t.Fatalf("ACK must not invent closed loop %q:\n%s", bad, acked)
		}
	}
}

func TestDashboardSlash_AckRitual(t *testing.T) {
	clearBriefAckFile(t)

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
	path, err := briefAckPathFn()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("ack must write local palace marker: %v", err)
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

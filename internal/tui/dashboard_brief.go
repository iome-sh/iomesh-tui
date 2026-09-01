package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/config"
)

// Morning brief ACK ritual on /dashboard (#371 / FR-33).
// Unacked does not count as known. Missing ACK is visible UNREAD (fail-open),
// never fake green. ACK is a human ritual only — no send/pay/ship closed loop.
// Optional local palace write: ~/.iomesh/brief-ack.json (same dir as config).

const (
	DashboardComposeBriefUnread = "brief: UNREAD · not known until /dashboard ack"
	DashboardComposeBriefAcked  = "brief: ACKed · confronted · no send/pay/ship"
	briefAckFileName            = "brief-ack.json"
)

// BriefAckStatus is unread vs ACKed for today's morning brief.
type BriefAckStatus string

const (
	BriefUnread BriefAckStatus = "unread"
	BriefAcked  BriefAckStatus = "acked"
)

type briefAckRecord struct {
	Day     string `json:"day"`
	AckedAt string `json:"acked_at"`
	Note    string `json:"note"`
}

// Overridable in tests. Production resolves beside user config.
var briefAckPathFn = defaultBriefAckPath

// Overridable clock for day-keyed ACK tests.
var briefAckNowFn = func() time.Time { return time.Now() }

func defaultBriefAckPath() (string, error) {
	cfgPath, err := config.UserConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(cfgPath), briefAckFileName), nil
}

func briefDayKey(t time.Time) string {
	return t.Local().Format("2006-01-02")
}

// loadBriefAckStatus returns today's ACK state. Missing/corrupt/stale → unread (fail-open).
func loadBriefAckStatus() BriefAckStatus {
	path, err := briefAckPathFn()
	if err != nil || strings.TrimSpace(path) == "" {
		return BriefUnread
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return BriefUnread
	}
	var rec briefAckRecord
	if json.Unmarshal(b, &rec) != nil {
		return BriefUnread
	}
	if strings.TrimSpace(rec.Day) != briefDayKey(briefAckNowFn()) {
		return BriefUnread
	}
	if strings.TrimSpace(rec.AckedAt) == "" {
		return BriefUnread
	}
	return BriefAcked
}

// ackTodayBrief writes today's local ACK marker. Does not send/pay/ship.
// Optional local palace write only — dual_write OFF · not Memory GA.
func ackTodayBrief() (string, error) {
	path, err := briefAckPathFn()
	if err != nil {
		return "", fmt.Errorf("brief ack path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("brief ack dir: %w", err)
	}
	now := briefAckNowFn()
	rec := briefAckRecord{
		Day:     briefDayKey(now),
		AckedAt: now.UTC().Format(time.RFC3339),
		Note:    "dashboard ritual · dual_write OFF · not Memory GA · no send/pay/ship",
	}
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("brief ack write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("brief ack rename: %w", err)
	}
	return fmt.Sprintf("brief ACKed for %s · local palace write %s · no send/pay/ship · unacked ≠ known",
		rec.Day, path), nil
}

func composeBriefLine(status BriefAckStatus) string {
	if status == BriefAcked {
		return DashboardComposeBriefAcked
	}
	return DashboardComposeBriefUnread
}

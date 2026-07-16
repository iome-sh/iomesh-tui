package agent

import (
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/session"
	"github.com/iome-sh/iomesh-tui/internal/subagent"
)

// SessionID returns the active persisted session id, if any.
func (rt *Runtime) SessionID() string {
	if rt == nil {
		return ""
	}
	return rt.sessionID
}

// SetSessionID pins the session id used for auto-save.
func (rt *Runtime) SetSessionID(id string) {
	if rt != nil {
		rt.sessionID = id
	}
}

// Snapshot builds a session snapshot from current runtime state.
func (rt *Runtime) Snapshot() *session.Snapshot {
	if rt == nil {
		return nil
	}
	snap := &session.Snapshot{
		Version:       session.SchemaVersion,
		ID:            rt.sessionID,
		Workspace:     rt.ws.Root(),
		ModelOverride: rt.router.Override(),
		Messages:      rt.Messages(),
	}
	if rt.subagents != nil {
		recs, seq := rt.subagents.ExportRegistry()
		snap.Subagents = recs
		snap.SubagentSeq = seq
	}
	return snap
}

// SaveSession writes the current state via store (creates id if empty).
func (rt *Runtime) SaveSession(store *session.Store, compactKeepUserTurns int) (*session.Snapshot, error) {
	if rt == nil || store == nil {
		return nil, fmt.Errorf("agent: nil runtime or store")
	}
	snap := rt.Snapshot()
	if compactKeepUserTurns > 0 {
		session.Compact(snap, compactKeepUserTurns)
		// Apply compacted messages back if compact ran.
		rt.messages = append([]router.Message(nil), snap.Messages...)
	}
	if err := store.Save(snap); err != nil {
		return nil, err
	}
	rt.sessionID = snap.ID
	return snap, nil
}

// LoadSession restores messages, model override, and subagent registry from snap.
func (rt *Runtime) LoadSession(snap *session.Snapshot) error {
	if rt == nil || snap == nil {
		return fmt.Errorf("agent: nil runtime or snapshot")
	}
	if snap.Workspace != "" && snap.Workspace != rt.ws.Root() {
		// Soft warning only — paths may differ by symlink resolution.
		rt.logger.Info("session workspace differs", "session", snap.Workspace, "current", rt.ws.Root())
	}
	if len(snap.Messages) == 0 {
		return fmt.Errorf("session: empty messages")
	}
	rt.messages = append([]router.Message(nil), snap.Messages...)
	rt.sessionID = snap.ID
	if snap.ModelOverride != "" {
		if err := rt.router.SetOverride(snap.ModelOverride); err != nil {
			rt.logger.Warn("session model override", "err", err)
		}
	}
	if rt.subagents != nil && len(snap.Subagents) > 0 {
		// Ensure cancelled status for any leftover running flags.
		recs := make([]subagent.Record, len(snap.Subagents))
		copy(recs, snap.Subagents)
		rt.subagents.ImportRegistry(recs, snap.SubagentSeq)
	}
	return nil
}

// AutoSaveAfterTurn saves after a successful turn when a store and session are configured.
func (rt *Runtime) AutoSaveAfterTurn(store *session.Store) {
	if rt == nil || store == nil || !rt.autoSave {
		return
	}
	if _, err := rt.SaveSession(store, 0); err != nil {
		rt.logger.Warn("session autosave failed", "err", err)
	}
}

// EnableAutoSave turns on persistence after each turn.
func (rt *Runtime) EnableAutoSave(v bool) {
	if rt != nil {
		rt.autoSave = v
	}
}

// LastUserPreview returns a short preview of the last user message (for listings).
func LastUserPreview(msgs []router.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			t := strings.TrimSpace(msgs[i].Content)
			t = strings.ReplaceAll(t, "\n", " ")
			if len(t) > 60 {
				return t[:60] + "…"
			}
			return t
		}
	}
	return ""
}

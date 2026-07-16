// Package session persists agent transcripts and subagent registry state
// under the workspace (.iomesh/sessions/) for continue/resume workflows.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/router"
	"github.com/iome-sh/iomesh-tui/internal/security"
	"github.com/iome-sh/iomesh-tui/internal/subagent"
)

const (
	// SchemaVersion is bumped on breaking snapshot shape changes.
	SchemaVersion = 1
	// DirName is relative to the workspace root.
	DirName = ".iomesh/sessions"
	// MaxStoredToolContent truncates large tool payloads on compact.
	MaxStoredToolContent = 8 * 1024
)

// Snapshot is a durable parent-session state (messages + subagent catalog).
type Snapshot struct {
	Version       int               `json:"version"`
	ID            string            `json:"id"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	Workspace     string            `json:"workspace"`
	ModelOverride string            `json:"model_override,omitempty"`
	Title         string            `json:"title,omitempty"`
	Messages      []router.Message  `json:"messages"`
	Subagents     []subagent.Record `json:"subagents,omitempty"`
	SubagentSeq   uint64            `json:"subagent_seq,omitempty"`
}

// Summary is a lightweight list row.
type Summary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  int       `json:"messages"`
	Subagents int       `json:"subagents"`
	Path      string    `json:"path"`
}

// Store reads/writes snapshots under workspace/.iomesh/sessions.
type Store struct {
	root string // absolute workspace root
	dir  string // absolute sessions directory
}

// Open creates a store rooted at workspace. Ensures the sessions directory exists.
func Open(workspace string) (*Store, error) {
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(abs, filepath.FromSlash(DirName))
	if !security.PathUnderRoot(abs, dir) {
		return nil, fmt.Errorf("session: invalid sessions dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{root: abs, dir: dir}, nil
}

// Dir returns the absolute sessions directory.
func (s *Store) Dir() string {
	if s == nil {
		return ""
	}
	return s.dir
}

// NewID allocates a new session id.
func NewID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("ses-%d-%s", time.Now().Unix(), hex.EncodeToString(b[:]))
}

// PathFor returns the JSON path for id (safe id only).
func (s *Store) PathFor(id string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("session: nil store")
	}
	id = strings.TrimSpace(id)
	if id == "" || strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") {
		return "", fmt.Errorf("session: invalid id %q", id)
	}
	// Allow ses-* and alphanumerics with dashes.
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return "", fmt.Errorf("session: invalid id char in %q", id)
	}
	p := filepath.Join(s.dir, id+".json")
	if !security.PathUnderRoot(s.dir, p) {
		return "", fmt.Errorf("session: path escapes store")
	}
	return p, nil
}

// Save writes snap (sets UpdatedAt, Version).
func (s *Store) Save(snap *Snapshot) error {
	if s == nil || snap == nil {
		return fmt.Errorf("session: nil")
	}
	if snap.ID == "" {
		snap.ID = NewID()
	}
	if snap.Version == 0 {
		snap.Version = SchemaVersion
	}
	now := time.Now().UTC()
	if snap.CreatedAt.IsZero() {
		snap.CreatedAt = now
	}
	snap.UpdatedAt = now
	if snap.Workspace == "" {
		snap.Workspace = s.root
	}
	if snap.Title == "" {
		snap.Title = deriveTitle(snap.Messages)
	}
	path, err := s.PathFor(snap.ID)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Load reads a snapshot by id.
func (s *Store) Load(id string) (*Snapshot, error) {
	path, err := s.PathFor(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("session: parse: %w", err)
	}
	if snap.Version > SchemaVersion {
		return nil, fmt.Errorf("session: unsupported version %d", snap.Version)
	}
	return &snap, nil
}

// Latest returns the most recently updated snapshot, or nil if none.
func (s *Store) Latest() (*Snapshot, error) {
	list, err := s.List()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return s.Load(list[0].ID)
}

// List returns session summaries newest first.
func (s *Store) List() ([]Summary, error) {
	if s == nil {
		return nil, fmt.Errorf("session: nil store")
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Summary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		snap, err := s.Load(id)
		if err != nil {
			continue
		}
		out = append(out, Summary{
			ID:        snap.ID,
			Title:     snap.Title,
			UpdatedAt: snap.UpdatedAt,
			Messages:  len(snap.Messages),
			Subagents: len(snap.Subagents),
			Path:      filepath.Join(s.dir, e.Name()),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

// Delete removes a session file.
func (s *Store) Delete(id string) error {
	path, err := s.PathFor(id)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// Compact truncates large tool results and drops intermediate system reminders
// beyond keepLast user/assistant turns (keeps system prompt). Mutates snap.
func Compact(snap *Snapshot, keepLastUserTurns int) {
	if snap == nil || keepLastUserTurns <= 0 {
		return
	}
	// Truncate tool payloads first.
	for i := range snap.Messages {
		m := &snap.Messages[i]
		if m.Role == "tool" && len(m.Content) > MaxStoredToolContent {
			m.Content = m.Content[:MaxStoredToolContent] + "\n…[compacted]"
		}
	}
	// Count user messages from the end; keep from that cut point, always keep first system.
	userCount := 0
	cut := 0
	for i := len(snap.Messages) - 1; i >= 0; i-- {
		if snap.Messages[i].Role == "user" {
			userCount++
			if userCount >= keepLastUserTurns {
				cut = i
				break
			}
		}
	}
	if cut <= 1 {
		return
	}
	var kept []router.Message
	if len(snap.Messages) > 0 && snap.Messages[0].Role == "system" {
		kept = append(kept, snap.Messages[0])
		// Optional compaction notice.
		kept = append(kept, router.Message{
			Role:    "system",
			Content: fmt.Sprintf("<session-compact retained_last_user_turns=%d dropped_messages=%d/>", keepLastUserTurns, cut-1),
		})
	}
	kept = append(kept, snap.Messages[cut:]...)
	snap.Messages = kept
}

func deriveTitle(msgs []router.Message) string {
	for _, m := range msgs {
		if m.Role == "user" && strings.TrimSpace(m.Content) != "" {
			t := strings.TrimSpace(m.Content)
			t = strings.ReplaceAll(t, "\n", " ")
			if len(t) > 72 {
				return t[:72] + "…"
			}
			return t
		}
	}
	return "session"
}

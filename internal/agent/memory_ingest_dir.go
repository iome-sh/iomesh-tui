package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/iome-sh/iomesh-tui/internal/workspace"
)

// ingest-dir caps (private overlay folder ingest · #384).
const (
	DefaultIngestDirLimit    = 32
	MaxIngestDirFileBytes    = 32 << 10 // 32 KiB per file
	maxIngestDirSkipReported = 24
)

// MemoryIngestDirOpts is the slash/CLI folder ingest plan (#384).
// DryRun lists files without calling MCP. dual_write stays OFF.
type MemoryIngestDirOpts struct {
	Path   string
	DryRun bool
	Limit  int // 0 = DefaultIngestDirLimit
}

// IngestDirFile is one workspace-jailed text file selected for overlay ingest.
type IngestDirFile struct {
	Rel  string
	Size int
	Text string
}

// IngestDirPlan is the residual-honest folder ingest inventory.
type IngestDirPlan struct {
	Dir     string
	Files   []IngestDirFile
	Skipped []string
}

// ListIngestDirFiles walks a workspace-jailed directory for UTF-8 text files.
// Skips .git / vendor / binaries / empty / oversize. Path jail via Workspace.Resolve.
func ListIngestDirFiles(ws *workspace.Workspace, dir string, limit int) (IngestDirPlan, error) {
	var plan IngestDirPlan
	if ws == nil {
		return plan, fmt.Errorf("workspace required for ingest-dir")
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return plan, fmt.Errorf("ingest-dir path required")
	}
	if limit <= 0 {
		limit = DefaultIngestDirLimit
	}
	root, err := ws.Resolve(dir)
	if err != nil {
		return plan, fmt.Errorf("ingest-dir: %w", err)
	}
	fi, err := os.Stat(root)
	if err != nil {
		return plan, fmt.Errorf("ingest-dir: %w", err)
	}
	if !fi.IsDir() {
		return plan, fmt.Errorf("ingest-dir: not a directory: %s", dir)
	}
	plan.Dir = dir
	wsRoot := ws.Root()
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			plan.Skipped = appendSkip(plan.Skipped, relOrBase(wsRoot, path)+": "+err.Error())
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if skipIngestDirName(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if len(plan.Files) >= limit {
			plan.Skipped = appendSkip(plan.Skipped, relOrBase(wsRoot, path)+": limit "+fmt.Sprintf("%d", limit))
			return nil
		}
		rel := relOrBase(wsRoot, path)
		if skipIngestDirName(d.Name()) {
			plan.Skipped = appendSkip(plan.Skipped, rel+": skipped name")
			return nil
		}
		info, err := d.Info()
		if err != nil {
			plan.Skipped = appendSkip(plan.Skipped, rel+": "+err.Error())
			return nil
		}
		if info.Size() <= 0 {
			plan.Skipped = appendSkip(plan.Skipped, rel+": empty")
			return nil
		}
		if info.Size() > MaxIngestDirFileBytes {
			plan.Skipped = appendSkip(plan.Skipped, rel+": exceeds "+fmt.Sprintf("%d", MaxIngestDirFileBytes)+" bytes")
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			plan.Skipped = appendSkip(plan.Skipped, rel+": "+err.Error())
			return nil
		}
		if !isIngestDirText(data) {
			plan.Skipped = appendSkip(plan.Skipped, rel+": not utf-8 text")
			return nil
		}
		plan.Files = append(plan.Files, IngestDirFile{
			Rel:  rel,
			Size: len(data),
			Text: string(data),
		})
		return nil
	})
	if err != nil {
		return plan, fmt.Errorf("ingest-dir: %w", err)
	}
	return plan, nil
}

func skipIngestDirName(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", ".iomesh", ".cursor", "bin", "dist":
		return true
	}
	return false
}

func isIngestDirText(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	n := len(data)
	if n > 512 {
		n = 512
	}
	for i := 0; i < n; i++ {
		if data[i] == 0 {
			return false
		}
	}
	return utf8.Valid(data)
}

func relOrBase(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.ToSlash(rel)
}

func appendSkip(skipped []string, line string) []string {
	if len(skipped) >= maxIngestDirSkipReported {
		return skipped
	}
	return append(skipped, line)
}

// FormatIngestDirPlan is the residual-honest dry-run / inventory text.
// Always names ingest-dir, session_id, dual_write=off. Catalog list ≠ consume.
func FormatIngestDirPlan(plan IngestDirPlan, sid string, minted, dryRun bool) string {
	var b strings.Builder
	mode := "ingest-dir"
	if dryRun {
		mode = "ingest-dir dry-run"
	}
	fmt.Fprintf(&b, "%s: dir=%s files=%d skipped=%d session_id=%s",
		mode, plan.Dir, len(plan.Files), len(plan.Skipped), sid)
	if minted {
		b.WriteString(" (minted · operator had none)")
	}
	b.WriteString(" dual_write=off · not Memory GA · catalog list ≠ consume · private overlay\n")
	for _, f := range plan.Files {
		fmt.Fprintf(&b, "  %s (%d bytes)\n", f.Rel, f.Size)
	}
	for _, s := range plan.Skipped {
		fmt.Fprintf(&b, "  skip %s\n", s)
	}
	return strings.TrimRight(b.String(), "\n")
}

// MemoryIngestDir ingests workspace-jailed folder text into the local palace
// via memory_ingest_turn (same session mint as /memory ingest). dual_write OFF
// unless the operator already enabled DualWrite (default false).
func (rt *Runtime) MemoryIngestDir(ctx context.Context, opts MemoryIngestDirOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	ws := rt.Workspace()
	if ws == nil {
		return "", fmt.Errorf("workspace required for ingest-dir")
	}
	plan, err := ListIngestDirFiles(ws, opts.Path, opts.Limit)
	if err != nil {
		return "", err
	}
	sid := rt.memoryIngestSessionID()
	minted := rt.memoryIngestSessionMinted()
	if opts.DryRun {
		return FormatIngestDirPlan(plan, sid, minted, true), nil
	}
	if len(plan.Files) == 0 {
		return FormatIngestDirPlan(plan, sid, minted, false) + "\n(no files ingested · empty ≠ invent overlay)", nil
	}

	var parts []string
	ingested := 0
	failed := 0
	for _, f := range plan.Files {
		content := "file: " + f.Rel + "\n\n" + f.Text
		out, ierr := rt.MemoryIngestTurn(ctx, "user", content)
		if ierr != nil {
			failed++
			parts = append(parts, f.Rel+": "+ierr.Error())
			continue
		}
		ingested++
		if s := strings.TrimSpace(out); s != "" {
			parts = append(parts, f.Rel+": "+s)
		} else {
			parts = append(parts, f.Rel+": ok")
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ingest-dir: dir=%s ingested=%d failed=%d skipped=%d session_id=%s",
		plan.Dir, ingested, failed, len(plan.Skipped), sid)
	if minted {
		b.WriteString(" (minted · operator had none)")
	}
	fmt.Fprintf(&b, " dual_write=%v · not Memory GA · catalog list ≠ consume · private overlay\n", rt.memory.DualWrite)
	for _, p := range parts {
		fmt.Fprintf(&b, "  %s\n", p)
	}
	for _, s := range plan.Skipped {
		fmt.Fprintf(&b, "  skip %s\n", s)
	}
	msg := strings.TrimRight(b.String(), "\n")
	if ingested == 0 && failed > 0 {
		return msg, fmt.Errorf("%s", msg)
	}
	return msg, nil
}

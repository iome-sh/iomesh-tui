package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iome-sh/iomesh-tui/internal/workspace"
)

// ingest-dir caps (private overlay folder ingest · V1.6 D2).
const (
	DefaultIngestDirLimit    = 128
	MaxIngestDirFileBytes    = 64 << 10 // 64 KiB per file
	maxIngestDirSkipReported = 64

	IngestDirSourceHintPrivate = "private"
)

var ingestDirTagIDRe = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

// MemoryIngestDirOpts is the slash/CLI folder ingest plan (#384 · V1.6 D2).
// DryRun lists files without calling MCP. dual_write stays OFF.
// SourceHint is private only (empty → private). Never stamp mesh on overlay.
type MemoryIngestDirOpts struct {
	Path       string
	DryRun     bool
	Limit      int // 0 = DefaultIngestDirLimit
	SourceHint string
	Department string
	Scenario   string
	SessionID  string // explicit --session-id; empty + department → local-overlay:{dept}
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

// NormalizeMemoryIngestDirOpts lowercases department/scenario, defaults
// source_hint to private, and rejects mesh/catalog/grant (never stamp mesh).
func NormalizeMemoryIngestDirOpts(opts *MemoryIngestDirOpts) error {
	if opts == nil {
		return fmt.Errorf("ingest-dir opts required")
	}
	hint, err := normalizeIngestDirSourceHint(opts.SourceHint)
	if err != nil {
		return err
	}
	opts.SourceHint = hint
	dept, err := normalizeIngestDirTagID("department", opts.Department)
	if err != nil {
		return err
	}
	opts.Department = dept
	scen, err := normalizeIngestDirTagID("scenario", opts.Scenario)
	if err != nil {
		return err
	}
	opts.Scenario = scen
	opts.SessionID = strings.TrimSpace(opts.SessionID)
	opts.Path = strings.TrimSpace(opts.Path)
	return nil
}

func normalizeIngestDirSourceHint(s string) (string, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == IngestDirSourceHintPrivate {
		return IngestDirSourceHintPrivate, nil
	}
	switch s {
	case "mesh", "catalog", "grant":
		return "", fmt.Errorf("source-hint %q rejected (ingest-dir is private overlay · never stamp mesh)", s)
	default:
		return "", fmt.Errorf("source-hint %q rejected (only private)", s)
	}
}

func normalizeIngestDirTagID(kind, s string) (string, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "", nil
	}
	if !ingestDirTagIDRe.MatchString(s) {
		return "", fmt.Errorf("%s %q invalid (lowercase [a-z0-9_-]{1,32})", kind, s)
	}
	return s, nil
}

// NormalizeMemoryDepartmentFilter validates a department id for facts-as-of / recall.
// Empty is no extra filter (honest empty ≠ invent). Does not lowercase: MESH is invalid.
func NormalizeMemoryDepartmentFilter(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if !ingestDirTagIDRe.MatchString(s) {
		return "", fmt.Errorf("department %q invalid (lowercase [a-z0-9_-]{1,32})", s)
	}
	return s, nil
}

// IngestDirTags returns dept:{id} / scenario:{kit} when set.
func IngestDirTags(opts MemoryIngestDirOpts) []string {
	var tags []string
	if opts.Department != "" {
		tags = append(tags, "dept:"+opts.Department)
	}
	if opts.Scenario != "" {
		tags = append(tags, "scenario:"+opts.Scenario)
	}
	return tags
}

// ResolveIngestDirSessionID mints local-overlay:{dept} when department is set
// and no --session-id. Otherwise configured/runtime, then local-overlay.
func ResolveIngestDirSessionID(opts MemoryIngestDirOpts, configured, runtime string) (sid string, minted bool) {
	return ResolvePalaceSessionID(opts.SessionID, configured, runtime, opts.Department)
}

func ingestDirSourceHint(opts MemoryIngestDirOpts) string {
	if s := strings.TrimSpace(opts.SourceHint); s != "" {
		return s
	}
	return IngestDirSourceHintPrivate
}

// FormatIngestDirFileContent prefixes overlay metadata so older MCP hosts that
// ignore unknown tags still persist dept/scenario in the turn body.
func FormatIngestDirFileContent(f IngestDirFile, opts MemoryIngestDirOpts) string {
	var b strings.Builder
	fmt.Fprintf(&b, "file: %s\n", f.Rel)
	fmt.Fprintf(&b, "source_hint: %s\n", ingestDirSourceHint(opts))
	if tags := IngestDirTags(opts); len(tags) > 0 {
		fmt.Fprintf(&b, "tags: %s\n", strings.Join(tags, ", "))
	}
	b.WriteString("\n")
	b.WriteString(f.Text)
	return b.String()
}

// IngestDirTurnArgs builds memory_ingest_turn arguments for one overlay file.
// Always sets source_hint=private. Sends tags when non-empty (sibling MCP PR).
func IngestDirTurnArgs(f IngestDirFile, sid, tenant string, opts MemoryIngestDirOpts) map[string]any {
	args := map[string]any{
		"role":        "user",
		"content":     FormatIngestDirFileContent(f, opts),
		"session_id":  sid,
		"source_hint": IngestDirSourceHintPrivate,
	}
	if t := strings.TrimSpace(tenant); t != "" {
		args["tenant"] = t
	}
	if tags := IngestDirTags(opts); len(tags) > 0 {
		args["tags"] = tags
	}
	return args
}

// CallIngestDirMCP calls memory_ingest_turn. If the host rejects unknown tags,
// retries once without tags (source_hint=private + content header remain).
func CallIngestDirMCP(ctx context.Context, call func(context.Context, string, map[string]any) (string, error), args map[string]any) (string, error) {
	if call == nil {
		return "", fmt.Errorf("ingest-dir MCP call required")
	}
	out, err := call(ctx, "memory_ingest_turn", args)
	if err == nil {
		return out, nil
	}
	if _, hasTags := args["tags"]; hasTags && mcpUnknownProperty(err) {
		delete(args, "tags")
		return call(ctx, "memory_ingest_turn", args)
	}
	return "", err
}

func mcpUnknownProperty(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	needles := []string{
		"additional propert",
		"unknown field",
		"unknown argument",
		"unknown parameter",
		"unknown key",
		"unexpected propert",
		"not allowed",
		"unrecognized",
		"extra field",
		"extra argument",
	}
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func ingestDirHonestyMeta(opts MemoryIngestDirOpts) string {
	var b strings.Builder
	fmt.Fprintf(&b, " source_hint=%s", ingestDirSourceHint(opts))
	if opts.Department != "" {
		fmt.Fprintf(&b, " dept:%s", opts.Department)
	}
	if opts.Scenario != "" {
		fmt.Fprintf(&b, " scenario:%s", opts.Scenario)
	}
	return b.String()
}

// ListIngestDirFiles walks a workspace-jailed directory for allowlisted UTF-8
// text files. Skips .git / vendor / binaries / empty / oversize / non-allowlist.
// Path jail via Workspace.Resolve. PDF: export text first (no OCR).
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
		if reason := ingestDirFileSkipReason(d.Name()); reason != "" {
			plan.Skipped = appendSkip(plan.Skipped, rel+": "+reason)
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

func ingestDirFileSkipReason(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".pdf" {
		return "export text first (no OCR)"
	}
	switch ext {
	case ".md", ".txt", ".json", ".jsonl", ".csv", ".html":
		return ""
	}
	if ext == "" {
		return "skipped extension (none)"
	}
	return "skipped extension (" + ext + ")"
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
// Always names ingest-dir, session_id, source_hint=private, dual_write=off.
// Catalog list ≠ consume. Never stamps mesh.
func FormatIngestDirPlan(plan IngestDirPlan, sid string, minted, dryRun bool, opts MemoryIngestDirOpts) string {
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
	b.WriteString(ingestDirHonestyMeta(opts))
	b.WriteString(" dual_write=off · catalog list ≠ consume · private overlay\n")
	for _, f := range plan.Files {
		fmt.Fprintf(&b, "  %s (%d bytes)\n", f.Rel, f.Size)
	}
	for _, s := range plan.Skipped {
		fmt.Fprintf(&b, "  skip %s\n", s)
	}
	return strings.TrimRight(b.String(), "\n")
}

// MemoryIngestDir ingests workspace-jailed folder text into the local palace
// via memory_ingest_turn (source_hint=private). dual_write OFF unless the
// operator already enabled DualWrite (default false). Never stamps mesh.
func (rt *Runtime) MemoryIngestDir(ctx context.Context, opts MemoryIngestDirOpts) (string, error) {
	if rt == nil || !rt.memory.Enabled {
		return "", fmt.Errorf("memory hooks disabled")
	}
	if err := NormalizeMemoryIngestDirOpts(&opts); err != nil {
		return "", err
	}
	ws := rt.Workspace()
	if ws == nil {
		return "", fmt.Errorf("workspace required for ingest-dir")
	}
	plan, err := ListIngestDirFiles(ws, opts.Path, opts.Limit)
	if err != nil {
		return "", err
	}
	sid, minted := ResolveIngestDirSessionID(opts, rt.memory.SessionID, rt.sessionID)
	if opts.DryRun {
		return rt.withPalaceProvenance(FormatIngestDirPlan(plan, sid, minted, true, opts), sid, nil), nil
	}
	if len(plan.Files) == 0 {
		empty := FormatIngestDirPlan(plan, sid, minted, false, opts) + "\n(no files ingested · empty ≠ invent overlay)"
		return rt.withPalaceProvenance(empty, sid, nil), nil
	}

	var parts []string
	var ids []string
	ingested := 0
	failed := 0
	for _, f := range plan.Files {
		out, ierr := rt.memoryIngestDirTurn(ctx, f, sid, opts)
		if ierr != nil {
			failed++
			parts = append(parts, f.Rel+": "+ierr.Error())
			continue
		}
		ingested++
		ids = append(ids, extractMemoryIDsFromWire(out)...)
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
	b.WriteString(ingestDirHonestyMeta(opts))
	fmt.Fprintf(&b, " dual_write=%v · catalog list ≠ consume · private overlay · %s\n", rt.memory.DualWrite, rt.PalaceVisibilityLine())
	for _, p := range parts {
		fmt.Fprintf(&b, "  %s\n", p)
	}
	for _, s := range plan.Skipped {
		fmt.Fprintf(&b, "  skip %s\n", s)
	}
	msg := strings.TrimRight(b.String(), "\n")
	msg = rt.withPalaceProvenance(msg, sid, uniqueMemoryIDs(ids))
	if IngestDirFailClosed(failed) {
		msg = msg + "\n" + IngestDirHalfWriteLine
		return msg, fmt.Errorf("%s", msg)
	}
	return msg, nil
}

func (rt *Runtime) memoryIngestDirTurn(ctx context.Context, f IngestDirFile, sid string, opts MemoryIngestDirOpts) (string, error) {
	mcpReady := rt.mcpMemoryReady()
	dualReady := rt.dualWriteReady()
	if !mcpReady && !dualReady {
		return "", fmt.Errorf("mcp server %q not connected (and dual_write unavailable)", rt.memory.Server)
	}
	content := FormatIngestDirFileContent(f, opts)
	eventTime := time.Now().UTC().Format(time.RFC3339)
	var parts []string
	ok := false

	if mcpReady {
		c := rt.mcp.ClientByName(rt.memory.Server)
		args := IngestDirTurnArgs(f, sid, rt.memoryTenant(), opts)
		out, err := CallIngestDirMCP(ctx, c.CallTool, args)
		if err != nil {
			if rt.logger != nil {
				rt.logger.Debug("memory MCP ingest-dir", "err", err)
			}
			parts = append(parts, "mcp failed: "+err.Error())
		} else {
			ok = true
			if s := strings.TrimSpace(out); s != "" {
				parts = append(parts, s)
			} else {
				parts = append(parts, "mcp ingest ok")
			}
		}
	}

	if dualReady {
		if err := rt.publishMemoryDualWrite(ctx, "user", content, eventTime); err != nil {
			if rt.logger != nil {
				rt.logger.Debug("memory dual_write ingest-dir", "err", err)
			}
			parts = append(parts, "dual_write failed: "+err.Error())
		} else {
			ok = true
			parts = append(parts, "dual_write MEMORY_INGEST ok")
		}
	}

	msg := strings.Join(parts, "; ")
	if ok {
		return msg, nil
	}
	if msg == "" {
		msg = "memory ingest-dir failed"
	}
	return msg, fmt.Errorf("%s", msg)
}

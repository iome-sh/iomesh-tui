package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

// IngestDirHalfWriteLine is the fail-closed copy when any memory_ingest_turn fails.
// Skip-list (PDF/oversize) is skip, not failed. Half-write is not a completed ingest.
const IngestDirHalfWriteLine = "half-write is not a completed ingest · do not cite the walk as landed"

// IngestDirFailClosed is the V2-B host contract: any memory_ingest_turn failure
// (failed>0) is not a completed folder ingest. Skip-list is not failed.
func IngestDirFailClosed(failed int) bool {
	return failed > 0
}

// ResolvePalaceSessionID returns explicit --session-id, else local-overlay:{dept}
// when department is set, else configured/runtime, else local-overlay.
// Same mint as ingest-dir so retrieve/facts-as-of find the private overlay.
func ResolvePalaceSessionID(explicit, configured, runtime, department string) (sid string, minted bool) {
	if s := strings.TrimSpace(explicit); s != "" {
		return s, false
	}
	if d := strings.TrimSpace(department); d != "" {
		return LocalOverlaySessionID + ":" + d, true
	}
	return ResolveMemoryIngestSessionID(configured, runtime), strings.TrimSpace(configured) == "" && strings.TrimSpace(runtime) == ""
}

func (rt *Runtime) palaceSessionID(explicit, department string) (sid string, minted bool) {
	if rt == nil {
		return ResolvePalaceSessionID(explicit, "", "", department)
	}
	return ResolvePalaceSessionID(explicit, rt.memory.SessionID, rt.sessionID, department)
}

// FormatPalaceProvenanceLine is one operator-visible footer from wire/config only:
// palace path, session_id, memory ids if present. Never invents model/tenant/mesh.
func FormatPalaceProvenanceLine(palacePath, sessionID string, memoryIDs []string) string {
	path := strings.TrimSpace(palacePath)
	if path == "" {
		path = ExpandPalaceRoot(DefaultPalaceRoot)
	}
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		sid = "-"
	}
	ids := uniqueMemoryIDs(memoryIDs)
	idPart := "(none)"
	if len(ids) > 0 {
		idPart = strings.Join(ids, ",")
	}
	return fmt.Sprintf("provenance: palace=%s session_id=%s memory_ids=%s", path, sid, idPart)
}

// ExtractMemoryIDsFromWire is the exported JSON memory_id reader for CLI ingest-dir.
func ExtractMemoryIDsFromWire(raw string) []string {
	return extractMemoryIDsFromWire(raw)
}

func appendPalaceProvenance(text, palacePath, sessionID string, memoryIDs []string) string {
	line := FormatPalaceProvenanceLine(palacePath, sessionID, memoryIDs)
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return line
	}
	return text + "\n" + line
}

func (rt *Runtime) withPalaceProvenance(text, sessionID string, memoryIDs []string) string {
	path := ""
	if rt != nil {
		path = rt.PalacePath()
	}
	return appendPalaceProvenance(text, path, sessionID, memoryIDs)
}

func memoryIDsFromHits(hits []iomesh.MemoryHit) []string {
	var ids []string
	seen := map[string]bool{}
	for _, h := range hits {
		s := strings.TrimSpace(h.ID)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		ids = append(ids, s)
	}
	return ids
}

// extractMemoryIDsFromWire reads memory_id / memories|facts|hits[].id from JSON only.
// Unknown or non-JSON payloads yield no ids (never invent).
func extractMemoryIDsFromWire(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if i := strings.IndexByte(raw, '{'); i >= 0 {
		raw = raw[i:]
	} else if i := strings.IndexByte(raw, '['); i >= 0 {
		raw = raw[i:]
	} else {
		return nil
	}
	var ids []string
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err == nil {
		collectWireMemoryIDs(obj, &ids)
		return uniqueMemoryIDs(ids)
	}
	var arr []any
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				collectWireMemoryIDs(m, &ids)
			}
		}
		return uniqueMemoryIDs(ids)
	}
	return nil
}

func collectWireMemoryIDs(obj map[string]any, ids *[]string) {
	if obj == nil || ids == nil {
		return
	}
	if s := wireStringField(obj, "memory_id"); s != "" {
		*ids = append(*ids, s)
	} else if s := wireStringField(obj, "id"); s != "" {
		if _, hasMemories := obj["memories"]; !hasMemories {
			if _, hasFacts := obj["facts"]; !hasFacts {
				if _, hasHits := obj["hits"]; !hasHits {
					*ids = append(*ids, s)
				}
			}
		}
	}
	for _, k := range []string{"memories", "facts", "hits"} {
		arr, _ := obj[k].([]any)
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if s := wireStringField(m, "memory_id"); s != "" {
				*ids = append(*ids, s)
			} else if s := wireStringField(m, "id"); s != "" {
				*ids = append(*ids, s)
			}
		}
	}
}

func wireStringField(obj map[string]any, key string) string {
	if obj == nil {
		return ""
	}
	s, _ := obj[key].(string)
	return strings.TrimSpace(s)
}

func uniqueMemoryIDs(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

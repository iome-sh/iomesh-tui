package agent

import (
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

func TestIngestDirFailClosed(t *testing.T) {
	if IngestDirFailClosed(0) {
		t.Fatal("failed=0 must not fail-closed")
	}
	if !IngestDirFailClosed(1) {
		t.Fatal("failed=1 must fail-closed (half-write is not a completed ingest)")
	}
	if !IngestDirFailClosed(2) {
		t.Fatal("failed>0 must fail-closed")
	}
}

func TestResolvePalaceSessionID(t *testing.T) {
	sid, minted := ResolvePalaceSessionID("", "", "", "")
	if sid != LocalOverlaySessionID || !minted {
		t.Fatalf("empty mint: sid=%q minted=%v", sid, minted)
	}
	sid, minted = ResolvePalaceSessionID("explicit", "cfg", "rt", "sales")
	if sid != "explicit" || minted {
		t.Fatalf("explicit wins: sid=%q minted=%v", sid, minted)
	}
	sid, minted = ResolvePalaceSessionID("", "cfg", "rt", "")
	if sid != "cfg" || minted {
		t.Fatalf("configured: sid=%q minted=%v", sid, minted)
	}
	sid, minted = ResolvePalaceSessionID("", "", "rt-sess", "")
	if sid != "rt-sess" || minted {
		t.Fatalf("runtime: sid=%q minted=%v", sid, minted)
	}
	sid, minted = ResolvePalaceSessionID("", "cfg", "rt", "sales")
	if sid != "local-overlay:sales" || !minted {
		t.Fatalf("department mint matches ingest-dir: sid=%q minted=%v", sid, minted)
	}
}

func TestFormatPalaceProvenanceLine(t *testing.T) {
	got := FormatPalaceProvenanceLine("/tmp/palace", "local-overlay", []string{"mem_a", "mem_b"})
	for _, want := range []string{"provenance:", "palace=/tmp/palace", "session_id=local-overlay", "memory_ids=mem_a,mem_b"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q: %s", want, got)
		}
	}
	empty := FormatPalaceProvenanceLine("/tmp/palace", "local-overlay", nil)
	if !strings.Contains(empty, "memory_ids=(none)") {
		t.Fatalf("empty ids: %s", empty)
	}
	for _, bad := range []string{
		"Memory GA",
		"leftover_is_bind",
		"CRM GET",
		"MTTR",
		"churn",
		"LME",
	} {
		if strings.Contains(got, bad) || strings.Contains(empty, bad) {
			t.Fatalf("provenance must not invent %q: %s", bad, got)
		}
	}
}

func TestExtractMemoryIDsFromWire(t *testing.T) {
	ids := extractMemoryIDsFromWire(`{"memory_id":"mem_test","tier":1,"tenant":"default"}`)
	if len(ids) != 1 || ids[0] != "mem_test" {
		t.Fatalf("ingest wire ids=%v", ids)
	}
	ids = extractMemoryIDsFromWire(`{"memories":[{"id":"1","summary":"a"},{"id":"2","full":"b"}]}`)
	if len(ids) != 2 || ids[0] != "1" || ids[1] != "2" {
		t.Fatalf("retrieve wire ids=%v", ids)
	}
	ids = extractMemoryIDsFromWire(`{"as_of":"2026-02-28T18:00:00Z","facts":[{"id":"f1","summary":"12 list-units"}]}`)
	if len(ids) != 1 || ids[0] != "f1" {
		t.Fatalf("facts wire ids=%v", ids)
	}
	if got := extractMemoryIDsFromWire("not json"); got != nil {
		t.Fatalf("non-json must not invent ids: %v", got)
	}
	if got := extractMemoryIDsFromWire(""); got != nil {
		t.Fatalf("empty must not invent ids: %v", got)
	}
}

func TestMemoryIDsFromHits(t *testing.T) {
	ids := memoryIDsFromHits([]iomesh.MemoryHit{{ID: "1"}, {ID: "1"}, {ID: "2"}, {ID: ""}})
	if len(ids) != 2 || ids[0] != "1" || ids[1] != "2" {
		t.Fatalf("ids=%v", ids)
	}
}

func TestPalaceProvenance_LeftoverIsBindOpenDefaultDNE(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(EnvMemoryPalaceRoot, "")
	t.Setenv(EnvPalaceRoot, "")
	got := FormatPalaceProvenanceLine("", LocalOverlaySessionID, nil)
	if !strings.Contains(got, "palace=-") {
		t.Fatalf("empty path must not invent default palace: %s", got)
	}
	def := ExpandPalaceRoot(DefaultPalaceRoot)
	if strings.Contains(got, def) {
		t.Fatalf("leftover_is_bind OPEN: must not present DNE default: %s", got)
	}
	if path := PalaceProvenancePath("", nil); path != "" {
		t.Fatalf("default DNE provenance path must be empty, got %q", path)
	}
	explicit := "/no/such/palace-root"
	if path := PalaceProvenancePath(explicit, nil); path != explicit {
		t.Fatalf("explicit DNE still prints: %q", path)
	}
	rt := &Runtime{memory: MemoryConfig{Enabled: true, DualWrite: false}}
	line := rt.withPalaceProvenance("ok", LocalOverlaySessionID, nil)
	if strings.Contains(line, def) {
		t.Fatalf("runtime provenance must not invent default DNE: %s", line)
	}
	if !strings.Contains(line, "palace=-") {
		t.Fatalf("want palace=-: %s", line)
	}
}

func TestPalaceProvenanceForbidsInventedClaims(t *testing.T) {
	line := FormatPalaceProvenanceLine("~/.iomesh/palace", LocalOverlaySessionID, nil)
	low := strings.ToLower(line)
	for _, bad := range []string{
		"memory ga",
		"leftover_is_bind close",
		"crm get",
		"mttr",
		"churn",
		"lme",
	} {
		if strings.Contains(low, bad) {
			t.Fatalf("forbid %q in provenance: %s", bad, line)
		}
	}
	if !strings.Contains(line, "palace=") || !strings.Contains(line, "session_id=") {
		t.Fatalf("needles palace/session: %s", line)
	}
}

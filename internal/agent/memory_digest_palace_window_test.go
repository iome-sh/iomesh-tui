package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

func TestScanPalaceCiteClasses_PinsNewestStampedAnyAge(t *testing.T) {
	asOf := time.Date(2026, 9, 24, 2, 14, 8, 0, time.UTC)
	root := t.TempDir()
	tenant := "org_ybq5j16vgr3b7uagfzkeoy7l"
	writePalaceTurn(t, root, tenant, "tier-1-working", "mesh-in-week.json", palaceTurn{
		ID: "mesh-in-week", Summary: "durable mesh pull",
		Timestamp: asOf.Add(-48 * time.Hour).Format(time.RFC3339),
		Prov:      "mesh", Tags: []string{"source_hint:mesh"},
	})
	writePalaceTurn(t, root, tenant, "tier-1-working", "mesh-old.json", palaceTurn{
		ID: "mesh-old", Summary: "ancient mesh",
		Timestamp: asOf.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		Prov:      "mesh", Tags: []string{"source_hint:mesh"},
	})
	writePalaceTurn(t, root, tenant, "tier-1-working", "private-only.json", palaceTurn{
		ID: "priv-unstamped", Summary: "palace timeline only",
		Timestamp: asOf.Add(-time.Hour).Format(time.RFC3339),
	})
	writePalaceTurn(t, root, "other-org", "tier-1-working", "foreign-mesh.json", palaceTurn{
		ID: "foreign-mesh", Summary: "other org mesh",
		Timestamp: asOf.Add(-time.Hour).Format(time.RFC3339),
		Prov:      "mesh", Tags: []string{"source_hint:mesh"},
	})

	pins := scanPalaceCiteClasses(root, tenant, []string{"mesh"})
	if len(pins) != 1 || pins[0].ID != "mesh-in-week" {
		t.Fatalf("want newest stamped mesh pin, got %+v", pins)
	}
	if ClassifyDigestReceipt(pins[0]) != DigestSourceMesh {
		t.Fatalf("pinned class=%q", ClassifyDigestReceipt(pins[0]))
	}
	if pins[0].Summary != "durable mesh pull" {
		t.Fatalf("summary=%q", pins[0].Summary)
	}

	oldRoot := t.TempDir()
	older := asOf.Add(-30 * 24 * time.Hour)
	newestOld := time.Date(2026, 9, 10, 6, 49, 4, 0, time.UTC)
	writePalaceTurn(t, oldRoot, tenant, "tier-2-contextual", "mesh-old.json", palaceTurn{
		ID: "mesh-old", Summary: "ancient mesh",
		Timestamp: older.Format(time.RFC3339),
		Prov:      "mesh",
	})
	writePalaceTurn(t, oldRoot, tenant, "tier-3-archival", "mesh-sep.json", palaceTurn{
		ID: "mesh-sep10", Summary: "newest old mesh",
		Timestamp: newestOld.Format(time.RFC3339),
		Prov:      "mesh", Tags: []string{"source_hint:mesh"},
	})
	pins = scanPalaceCiteClasses(oldRoot, tenant, []string{"mesh"})
	if len(pins) != 1 || pins[0].ID != "mesh-sep10" {
		t.Fatalf("mesh older than week must still pin newest stamp, got %+v", pins)
	}
	if pins[0].EventTime != newestOld.Format(time.RFC3339) {
		t.Fatalf("newest event_time=%q", pins[0].EventTime)
	}

	onlyPrivate := t.TempDir()
	writePalaceTurn(t, onlyPrivate, tenant, "tier-1-working", "private-only.json", palaceTurn{
		ID: "priv-unstamped", Summary: "palace timeline only",
		Timestamp: asOf.Add(-time.Hour).Format(time.RFC3339),
	})
	pins = scanPalaceCiteClasses(onlyPrivate, tenant, []string{"mesh"})
	if len(pins) != 0 {
		t.Fatalf("unstamped palace_timeline must not invent mesh pins=%+v", pins)
	}

	untimed := t.TempDir()
	writePalaceTurn(t, untimed, tenant, "tier-1-working", "mesh-notime.json", palaceTurn{
		ID: "mesh-notime", Summary: "mesh without time",
		Prov: "mesh", Tags: []string{"source_hint:mesh"},
	})
	pins = scanPalaceCiteClasses(untimed, tenant, []string{"mesh"})
	if len(pins) != 1 || pins[0].ID != "mesh-notime" {
		t.Fatalf("stamped mesh with no event time must still pin, got %+v", pins)
	}
}

func TestFormatRequireSourcesCheck_NamesMeshOnPalaceOutsideWindow(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Window:     "day",
		Since:      "2026-09-23T02:14:08Z",
		AsOf:       "2026-09-24T02:14:08Z",
		FetchLimit: 50,
		FetchedN:   2,
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "p1", EventTime: "2026-09-24T02:14:04Z", Summary: "private RCA", SourceHint: "palace_timeline"},
		},
		PalaceOutsideWindow: []iomesh.MemoryOpsDigestPalaceOutside{
			{Class: "mesh", Count: 257, Newest: "2026-08-01T00:00:00Z"},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	for _, want := range []string{
		"require-sources: miss",
		"cited=private",
		"missing=mesh",
		"mesh not in this receipt set",
		"mesh on palace outside window",
		"mesh_on_disk=257",
		"newest_mesh=2026-08-01T00:00:00Z",
		"dual_write OFF",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "require-sources: ok") || strings.Contains(out, "Connected") {
		t.Fatalf("must not invent cite-both or Connected: %q", out)
	}
}

func TestMemoryOpsDigest_WeekExportCitesMeshOutsideDay(t *testing.T) {
	asOf := time.Date(2026, 9, 24, 2, 14, 8, 0, time.UTC)
	var windows []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		win, _ := body["window"].(string)
		windows = append(windows, win)
		receipts := []map[string]any{
			{
				"id": "p1", "event_time": asOf.Add(-4 * time.Second).Format(time.RFC3339),
				"summary": "private RCA", "source_hint": "palace_timeline",
			},
		}
		if win == "week" {
			receipts = append(receipts, map[string]any{
				"id": "mesh-week", "event_time": asOf.Add(-48 * time.Hour).Format(time.RFC3339),
				"summary": "durable mesh pull", "source_hint": "palace_timeline",
				"tags":       []string{"source_hint:mesh"},
				"provenance": map[string]any{"source_hint": "mesh"},
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": win, "horizon": "ops",
			"since": asOf.Add(-24 * time.Hour).Format(time.RFC3339),
			"as_of": asOf.Format(time.RFC3339),
			"honesty": map[string]any{
				"ops_pulse": "ga_path", "never_invent_ga": true, "dual_write_default": "off",
			},
			"patterns": []any{},
			"receipts": receipts,
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "org_test"}, nil)
	rt := &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, Tenant: "org_test", Server: "memory", DualWrite: false,
		},
	}
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) < 2 || windows[0] != "day" || windows[1] != "week" {
		t.Fatalf("cite-both must widen day→week when mesh is missing, windows=%v", windows)
	}
	assertCiteBothOK(t, out, "durable mesh pull")
}

func TestMemoryOpsDigest_PalaceMeshInsideWeekCitesBoth(t *testing.T) {
	asOf := time.Date(2026, 9, 24, 2, 14, 8, 0, time.UTC)
	tenant := "org_ybq5j16vgr3b7uagfzkeoy7l"
	root := t.TempDir()
	writePalaceTurn(t, root, tenant, "tier-1-working", "mesh.json", palaceTurn{
		ID: "mesh-disk", Summary: "durable mesh pull",
		Timestamp: asOf.Add(-48 * time.Hour).Format(time.RFC3339),
		Prov:      "mesh", Tags: []string{"source_hint:mesh"},
	})
	srv := privateOnlyDigestServer(t, asOf)
	defer srv.Close()

	rt := digestRuntime(srv.URL, tenant, root)
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertCiteBothOK(t, out, "durable mesh pull")
	if !strings.Contains(out, "cite=mesh") {
		t.Fatalf("active receipt set must cite mesh from palace: %q", out)
	}
	if strings.Contains(out, "mesh on palace outside window") {
		t.Fatalf("in-week palace mesh must not be named outside: %q", out)
	}
}

func TestMemoryOpsDigest_PalaceMeshOutsideWeekCitesBoth(t *testing.T) {
	asOf := time.Date(2026, 9, 24, 3, 47, 32, 0, time.UTC)
	tenant := "org_ybq5j16vgr3b7uagfzkeoy7l"
	newest := time.Date(2026, 9, 10, 6, 49, 4, 0, time.UTC)
	root := t.TempDir()
	writePalaceTurn(t, root, tenant, "tier-1-working", "mesh-a.json", palaceTurn{
		ID: "mesh-a", Summary: "older mesh",
		Timestamp: newest.Add(-24 * time.Hour).Format(time.RFC3339),
		Prov:      "mesh", Tags: []string{"source_hint:mesh"},
	})
	writePalaceTurn(t, root, tenant, "tier-3-archival", "mesh-b.json", palaceTurn{
		ID: "mesh-b", Summary: "newest old mesh",
		Timestamp: newest.Format(time.RFC3339),
		Prov:      "mesh", Tags: []string{"source_hint:mesh"},
	})
	srv := privateOnlyDigestServer(t, asOf)
	defer srv.Close()

	rt := digestRuntime(srv.URL, tenant, root)
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertCiteBothOK(t, out, "newest old mesh")
	if strings.Contains(out, "mesh on palace outside window") || strings.Contains(out, "mesh_on_disk=") || strings.Contains(out, "missing=mesh") {
		t.Fatalf("outside-week stamped mesh must cite, not name a window miss: %q", out)
	}
	if strings.Contains(out, "Connected") || strings.Contains(out, "Memory GA") {
		t.Fatalf("must not invent Connected / Memory GA: %q", out)
	}
	if rt.memory.DualWrite {
		t.Fatal("dual_write must remain OFF")
	}
}

func TestMemoryOpsDigest_OtherOrgAndUnstampedDoNotInventMesh(t *testing.T) {
	asOf := time.Date(2026, 9, 24, 2, 14, 8, 0, time.UTC)
	tenant := "org_ybq5j16vgr3b7uagfzkeoy7l"
	root := t.TempDir()
	writePalaceTurn(t, root, "other-org", "tier-1-working", "foreign.json", palaceTurn{
		ID: "foreign-mesh", Summary: "other org mesh",
		Timestamp: asOf.Add(-2 * time.Hour).Format(time.RFC3339),
		Prov:      "mesh", Tags: []string{"source_hint:mesh"},
	})
	writePalaceTurn(t, root, tenant, "tier-1-working", "private.json", palaceTurn{
		ID: "local-private", Summary: "palace timeline only",
		Timestamp: asOf.Add(-2 * time.Hour).Format(time.RFC3339),
	})
	srv := privateOnlyDigestServer(t, asOf)
	defer srv.Close()

	rt := digestRuntime(srv.URL, tenant, root)
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"require-sources: miss",
		"missing=mesh",
		"miss_class=no_mesh_pulse",
		"mesh not in this receipt set",
		"cited=private",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if strings.Contains(out, "mesh on palace outside window") || strings.Contains(out, "mesh_on_disk=") {
		t.Fatalf("must not claim mesh on this org palace: %q", out)
	}
	if strings.Contains(out, "require-sources: ok") || strings.Contains(out, "other org mesh") || strings.Contains(out, "Connected") {
		t.Fatalf("must not invent mesh from another org: %q", out)
	}
}

func TestMemoryOpsDigest_PalacePrivateOutsideWeekCitesBoth(t *testing.T) {
	asOf := time.Date(2026, 9, 24, 3, 47, 32, 0, time.UTC)
	tenant := "org_ybq5j16vgr3b7uagfzkeoy7l"
	root := t.TempDir()
	writePalaceTurn(t, root, tenant, "tier-2-contextual", "private-old.json", palaceTurn{
		ID: "priv-old", Summary: "old private RCA",
		Timestamp: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Prov:      "private", Tags: []string{"source_hint:private"},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": "day", "horizon": "ops",
			"since": asOf.Add(-24 * time.Hour).Format(time.RFC3339),
			"as_of": asOf.Format(time.RFC3339),
			"honesty": map[string]any{
				"ops_pulse": "ga_path", "never_invent_ga": true, "dual_write_default": "off",
			},
			"patterns": []any{},
			"receipts": []map[string]any{
				{
					"id": "m1", "event_time": asOf.Add(-time.Hour).Format(time.RFC3339),
					"summary": "durable mesh pull", "source_hint": "mesh",
				},
			},
		})
	}))
	defer srv.Close()

	rt := digestRuntime(srv.URL, tenant, root)
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertCiteBothOK(t, out, "durable mesh pull")
	if !strings.Contains(out, "private=old private RCA") {
		t.Fatalf("want outside-week private pin: %q", out)
	}
}

type palaceTurn struct {
	ID        string
	Summary   string
	Timestamp string
	Prov      string
	Tags      []string
}

func writePalaceTurn(t *testing.T, root, tenant, tier, name string, turn palaceTurn) {
	t.Helper()
	dir := filepath.Join(root, tenant, tier)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{
		"id":        turn.ID,
		"type":      "turn",
		"timestamp": turn.Timestamp,
		"content": map[string]any{
			"summary": turn.Summary,
			"tags":    turn.Tags,
		},
		"provenance": map[string]any{
			"source_hint": turn.Prov,
			"source_step": "memory_pull",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func privateOnlyDigestServer(t *testing.T, asOf time.Time) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": "day", "horizon": "ops",
			"since": asOf.Add(-24 * time.Hour).Format(time.RFC3339),
			"as_of": asOf.Format(time.RFC3339),
			"honesty": map[string]any{
				"ops_pulse": "ga_path", "never_invent_ga": true, "dual_write_default": "off",
			},
			"patterns": []any{},
			"receipts": []map[string]any{
				{
					"id": "p1", "event_time": asOf.Add(-4 * time.Second).Format(time.RFC3339),
					"summary": "private RCA", "source_hint": "palace_timeline",
				},
			},
		})
	}))
}

func digestRuntime(endpoint, tenant, palace string) *Runtime {
	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: endpoint, Tenant: tenant}, nil)
	return &Runtime{
		mesh: mesh,
		memory: MemoryConfig{
			Enabled: true, Tenant: tenant, Server: "memory", DualWrite: false,
			PalaceRoot: palace,
		},
	}
}

func assertCiteBothOK(t *testing.T, out, meshCite string) {
	t.Helper()
	if !strings.HasPrefix(strings.TrimSpace(out), "require-sources: ok") {
		t.Fatalf("want cite-both ok: %q", out)
	}
	if !strings.Contains(out, "cited=mesh,private") {
		t.Fatalf("want both cited: %q", out)
	}
	if !strings.Contains(out, "mesh="+meshCite) || !strings.Contains(out, "private=") {
		t.Fatalf("want mesh and private cites: %q", out)
	}
	if strings.Contains(out, "missing=mesh") || strings.Contains(out, "require-sources: miss") {
		t.Fatalf("must not miss mesh: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") {
		t.Fatalf("dual_write pin missing: %q", out)
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Connected") || strings.Contains(out, "Memory GA") {
		t.Fatalf("must not invent dual_write ON / Connected / Memory GA: %q", out)
	}
}

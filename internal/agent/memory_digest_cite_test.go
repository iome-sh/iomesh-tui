package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/mcp"
)

func TestClassifyDigestReceipt_PalaceTimelineMeshProvenance(t *testing.T) {
	meshFromProv := iomesh.MemoryOpsDigestReceipt{
		ID:         "m1",
		EventTime:  "2026-09-10T06:46:00Z",
		Summary:    "dept.*.events pull",
		SourceHint: "palace_timeline",
		Provenance: iomesh.MemoryOpsDigestProvenance{SourceHint: "mesh"},
	}
	if got := ClassifyDigestReceipt(meshFromProv); got != DigestSourceMesh {
		t.Fatalf("provenance mesh: got=%q", got)
	}
	meshFromTag := iomesh.MemoryOpsDigestReceipt{
		ID:         "m2",
		EventTime:  "2026-09-10T06:46:11Z",
		Summary:    "durable consume",
		SourceHint: "palace_timeline",
		Tags:       []string{"source_hint:mesh", "dept.engineering"},
	}
	if got := ClassifyDigestReceipt(meshFromTag); got != DigestSourceMesh {
		t.Fatalf("tag mesh: got=%q", got)
	}
	priv := iomesh.MemoryOpsDigestReceipt{
		ID:         "p1",
		EventTime:  "2026-09-10T16:10:00Z",
		Summary:    "private RCA",
		SourceHint: "palace_timeline",
	}
	if got := ClassifyDigestReceipt(priv); got != DigestSourcePrivate {
		t.Fatalf("palace_timeline alone must stay private: got=%q", got)
	}
	if ClassifyDigestSourceHint("palace_timeline") != DigestSourcePrivate {
		t.Fatal("token classifier must still map palace_timeline → private")
	}
}

func TestFormatRequireSourcesCheck_MeshProvenanceAlongsideNewerPrivate(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Window: "day",
		Since:  "2026-09-09T16:10:00Z",
		AsOf:   "2026-09-10T16:10:00Z",
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{
				ID: "p1", EventTime: "2026-09-10T16:10:12Z",
				Summary: "private RCA", SourceHint: "palace_timeline",
			},
			{
				ID: "m1", EventTime: "2026-09-10T06:46:00Z",
				Summary:    "mesh pull INC-9",
				SourceHint: "palace_timeline",
				Provenance: iomesh.MemoryOpsDigestProvenance{SourceHint: "mesh"},
				Tags:       []string{"source_hint:mesh"},
			},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if !strings.Contains(out, "require-sources: ok") {
		t.Fatalf("want cite-both ok when mesh provenance exists beside newer private: %q", out)
	}
	if !strings.Contains(out, "cited=mesh,private") {
		t.Fatalf("want both cited: %q", out)
	}
	if !strings.Contains(out, "mesh=mesh pull INC-9") || !strings.Contains(out, "private=private RCA") {
		t.Fatalf("want cites: %q", out)
	}
	if strings.Contains(out, "missing=mesh") || strings.Contains(out, "require-sources: miss") {
		t.Fatalf("must not miss mesh: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") {
		t.Fatalf("honesty pin missing: %q", out)
	}
}

func TestFormatRequireSourcesCheck_MissingMeshPrintsWindowReason(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Window:        "day",
		Since:         "2026-09-09T16:10:00Z",
		AsOf:          "2026-09-10T16:10:00Z",
		FetchLimit:    20,
		FetchedN:      20,
		FetchedNewest: "2026-09-10T16:10:12Z",
		FetchedOldest: "2026-09-10T15:01:00Z",
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "p1", EventTime: "2026-09-10T16:10:12Z", Summary: "private RCA", SourceHint: "palace_timeline"},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if !strings.Contains(out, "require-sources: miss") || !strings.Contains(out, "missing=mesh") {
		t.Fatalf("want mesh miss: %q", out)
	}
	if !strings.Contains(out, "cited=private") {
		t.Fatalf("want private cited: %q", out)
	}
	for _, want := range []string{
		"receipt window newest-first",
		"limit=20",
		"n=20",
		"since=2026-09-09T16:10:00Z",
		"as_of=2026-09-10T16:10:00Z",
		"newest=2026-09-10T16:10:12Z",
		"oldest=2026-09-10T15:01:00Z",
		"mesh not in this receipt set",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("window reason missing %q: %q", want, out)
		}
	}
	if strings.Contains(out, "require-sources: ok") {
		t.Fatalf("must not invent cite-both: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") {
		t.Fatalf("dual_write pin missing: %q", out)
	}
}

func TestPinRequiredDigestReceipts_KeepsOlderMeshWithNewerPrivate(t *testing.T) {
	var receipts []iomesh.MemoryOpsDigestReceipt
	for i := 0; i < 25; i++ {
		receipts = append(receipts, iomesh.MemoryOpsDigestReceipt{
			ID:         "p" + strconv.Itoa(i),
			EventTime:  "2026-09-10T16:10:" + twoDigit(i) + "Z",
			Summary:    "private RCA",
			SourceHint: "palace_timeline",
		})
	}
	receipts = append(receipts, iomesh.MemoryOpsDigestReceipt{
		ID:         "mesh-old",
		EventTime:  "2026-09-10T06:46:00Z",
		Summary:    "mesh pull",
		SourceHint: "palace_timeline",
		Tags:       []string{"source_hint:mesh"},
	})
	got := pinRequiredDigestReceipts(receipts, []string{"mesh", "private"}, 20)
	if len(got) != 20 {
		t.Fatalf("display n=%d want 20", len(got))
	}
	var haveMesh, havePrivate bool
	for _, r := range got {
		switch ClassifyDigestReceipt(r) {
		case DigestSourceMesh:
			haveMesh = true
			if r.ID != "mesh-old" {
				t.Fatalf("pinned mesh id=%q", r.ID)
			}
		case DigestSourcePrivate:
			havePrivate = true
		}
	}
	if !haveMesh || !havePrivate {
		t.Fatalf("pin must keep mesh+private in the display set (mesh=%v private=%v) ids=%v", haveMesh, havePrivate, receiptIDs(got))
	}
}

func TestDigestCiteLimits_RequireSourcesFetchesExportCap(t *testing.T) {
	fetch, display := digestCiteLimits(MemoryOpsDigestOpts{RequireSources: []string{"mesh", "private"}})
	if fetch != opsDigestLimitCiteBoth || display != opsDigestLimitDefault {
		t.Fatalf("sticky cite limits fetch=%d display=%d", fetch, display)
	}
	fetch2, display2 := digestCiteLimits(MemoryOpsDigestOpts{})
	if fetch2 != opsDigestLimitDefault || display2 != opsDigestLimitDefault {
		t.Fatalf("default digest limits fetch=%d display=%d", fetch2, display2)
	}
	fetch3, display3 := digestCiteLimits(MemoryOpsDigestOpts{Limit: 7, RequireSources: []string{"mesh", "private"}})
	if fetch3 != 7 || display3 != 7 {
		t.Fatalf("explicit limit must win: fetch=%d display=%d", fetch3, display3)
	}
}

func TestMemoryOpsDigest_RequireSourcesCitesOlderMeshWithNewerPrivate(t *testing.T) {
	var gotLimit any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotLimit = body["limit"]
		receipts := make([]map[string]any, 0, 26)
		for i := 0; i < 25; i++ {
			receipts = append(receipts, map[string]any{
				"id":          "p" + strconv.Itoa(i),
				"event_time":  "2026-09-10T16:10:" + twoDigit(i) + "Z",
				"summary":     "private RCA " + strconv.Itoa(i),
				"source_hint": "palace_timeline",
			})
		}
		receipts = append(receipts, map[string]any{
			"id":          "mesh-old",
			"event_time":  "2026-09-10T06:46:00Z",
			"summary":     "mesh pull INC-9",
			"source_hint": "palace_timeline",
			"tags":        []string{"source_hint:mesh"},
			"provenance":  map[string]any{"source_hint": "mesh"},
		})
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": "day", "horizon": "ops",
			"since": "2026-09-09T16:10:00Z",
			"as_of": "2026-09-10T16:10:00Z",
			"honesty": map[string]any{
				"ops_pulse": "ga_path", "never_invent_ga": true, "dual_write_default": "off",
			},
			"patterns": []any{},
			"receipts": receipts,
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, Tenant: "t", Server: "memory", DualWrite: false},
	}
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotLimit != float64(opsDigestLimitCiteBoth) {
		t.Fatalf("sticky cite-both must fetch limit=%d, got %v", opsDigestLimitCiteBoth, gotLimit)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "require-sources: ok") {
		t.Fatalf("want ok prefix: %q", out)
	}
	if !strings.Contains(out, "cited=mesh,private") {
		t.Fatalf("want both cited: %q", out)
	}
	if !strings.Contains(out, "cite=mesh") && !strings.Contains(out, "source=mesh") {
		t.Fatalf("want mesh visible on a receipt line: %q", out)
	}
	if strings.Contains(out, "missing=mesh") {
		t.Fatalf("must not miss mesh when stamped turn is in the export: %q", out)
	}
	if rt.memory.DualWrite || strings.Contains(out, "dual_write ON") {
		t.Fatal("dual_write must remain OFF")
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Connected") {
		t.Fatalf("must not invent GA / Connected: %q", out)
	}
}

func TestMemoryOpsDigest_RequireSourcesDoesNotInventMesh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": "day", "horizon": "ops",
			"since": "2026-09-09T16:10:00Z",
			"as_of": "2026-09-10T16:10:00Z",
			"honesty": map[string]any{
				"ops_pulse": "ga_path", "never_invent_ga": true, "dual_write_default": "off",
			},
			"patterns": []any{},
			"receipts": []map[string]any{
				{"id": "p1", "event_time": "2026-09-10T16:10:12Z", "summary": "private RCA", "source_hint": "palace_timeline"},
			},
		})
	}))
	defer srv.Close()

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		memory: MemoryConfig{Enabled: true, Tenant: "t", Server: "memory", DualWrite: false},
	}
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "require-sources: miss") || !strings.Contains(out, "missing=mesh") {
		t.Fatalf("palace_timeline alone must not invent mesh: %q", out)
	}
	if !strings.Contains(out, "receipt window newest-first") || !strings.Contains(out, "mesh not in this receipt set") {
		t.Fatalf("want honest window reason: %q", out)
	}
	if strings.Contains(out, "require-sources: ok") {
		t.Fatalf("must not invent cite-both: %q", out)
	}
}

func TestParseOpsDigestJSON_ExtractsFencedAndProvenance(t *testing.T) {
	raw := "here is the pack:\n```json\n" + `{
		"window": "day",
		"horizon": "ops",
		"receipts": [
			{"id": "p1", "summary": "private RCA", "source_hint": "palace_timeline", "event_time": "2026-09-10T16:10:12Z"},
			{"id": "m1", "summary": "mesh pull", "source_hint": "palace_timeline", "event_time": "2026-09-10T06:46:00Z",
			 "provenance": {"source_hint": "mesh"}, "tags": ["source_hint:mesh"]}
		]
	}` + "\n```\n"
	res, formatted := parseOpsDigestJSON(raw, 6000)
	if res == nil || formatted == "" {
		t.Fatalf("parse failed res=%v formatted=%q", res, formatted)
	}
	ok := applyRequireSources(formatted, res, []string{"mesh", "private"})
	if !strings.HasPrefix(strings.TrimSpace(ok), "require-sources: ok") {
		t.Fatalf("fenced mesh provenance must cite-both: %q", ok)
	}
}

func TestClassifyDigestReceipt_NestedPalaceTagsDespitePalaceTimeline(t *testing.T) {
	raw := `{
		"id": 42,
		"event_time": "2026-09-10T06:46:00Z",
		"summary": "dept.*.events pull",
		"source_hint": "palace_timeline",
		"entry": {
			"content": {"tags": ["source_hint:mesh", "dept.engineering"]},
			"provenance": {"source_hint": "mesh"}
		}
	}`
	var r iomesh.MemoryOpsDigestReceipt
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.SourceHint != "palace_timeline" {
		t.Fatalf("export origin=%q", r.SourceHint)
	}
	if got := ClassifyDigestReceipt(r); got != DigestSourceMesh {
		t.Fatalf("nested provenance/tags must classify mesh: got=%q tags=%v prov=%+v", got, r.Tags, r.Provenance)
	}
}

func TestFormatRequireSourcesCheck_PrivatePresentNotCitedNone(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Window:     "day",
		Since:      "2026-09-09T16:10:00Z",
		AsOf:       "2026-09-10T16:10:00Z",
		FetchLimit: 50,
		FetchedN:   2,
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "p1", EventTime: "2026-09-10T16:10:12Z", Summary: "private RCA", SourceHint: "palace_timeline"},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if strings.Contains(out, "cited=(none)") {
		t.Fatalf("private RCA present must not cite=(none): %q", out)
	}
	if !strings.Contains(out, "cited=private") || !strings.Contains(out, "missing=mesh") {
		t.Fatalf("want private cited / mesh miss: %q", out)
	}
	for _, want := range []string{"receipt window newest-first", "limit=50", "n=2", "since=", "as_of="} {
		if !strings.Contains(out, want) {
			t.Fatalf("window reason missing %q: %q", want, out)
		}
	}
	if strings.Contains(out, "require-sources: ok") {
		t.Fatalf("must not invent cite-both: %q", out)
	}
}

func TestSalvageDigestReceiptsJSON_TruncatedPrettyPack(t *testing.T) {
	full := `{
  "window": "day",
  "since": "2026-09-09T16:10:00Z",
  "as_of": "2026-09-10T16:10:00Z",
  "receipts": [
    {"id": "p1", "event_time": "2026-09-10T16:10:12Z", "summary": "private RCA", "source_hint": "palace_timeline"},
    {"id": "m1", "event_time": "2026-09-10T06:46:00Z", "summary": "mesh pull", "source_hint": "palace_timeline", "tags": ["source_hint:mesh"], "provenance": {"source_hint": "mesh"}},
    {"id": "p2", "event_time": "2026-09-10T16:09:00Z", "summary": "cut off`
	got := salvageDigestReceiptsJSON(full)
	if len(got) < 2 {
		t.Fatalf("salvage n=%d want ≥2 complete receipts: %+v", len(got), got)
	}
	res := &iomesh.MemoryOpsDigestResult{Receipts: got, FetchLimit: 50}
	fillDigestWindowFromRaw(res, full)
	annotateDigestFetchWindow(res, 50)
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if strings.Contains(out, "cited=(none)") {
		t.Fatalf("salvaged private+mesh must not cite=(none): %q", out)
	}
	if !strings.Contains(out, "require-sources: ok") || !strings.Contains(out, "cited=mesh,private") {
		t.Fatalf("want cite-both from salvaged provenance/tags: %q", out)
	}
}

func TestParseOpsDigestFromTool_PrefersStructuredWhenTextTruncated(t *testing.T) {
	structured := json.RawMessage(`{
		"window": "day", "horizon": "ops",
		"since": "2026-09-09T16:10:00Z",
		"as_of": "2026-09-10T16:10:00Z",
		"receipts": [
			{"id": "p1", "summary": "private RCA", "source_hint": "palace_timeline", "event_time": "2026-09-10T16:10:12Z"},
			{"id": "m1", "summary": "mesh pull", "source_hint": "palace_timeline", "event_time": "2026-09-10T06:46:00Z",
			 "provenance": {"source_hint": "mesh"}, "tags": ["source_hint:mesh"]}
		]
	}`)
	truncated := `{"window":"day","receipts":[{"id":"p1","source_hint":"palace_timeline","summary":"private RCA"}` // mid-object
	res := parseOpsDigestFromTool(truncated, structured, 6000)
	if res == nil || len(res.Receipts) != 2 {
		t.Fatalf("structured must win over truncated text: %+v", res)
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if !strings.Contains(out, "require-sources: ok") {
		t.Fatalf("want cite-both from structuredContent: %q", out)
	}
	if strings.Contains(out, "cited=(none)") {
		t.Fatalf("must not cited=(none): %q", out)
	}
}

func TestMergeDigestReceipts_MCPTagsWinOverPalaceTimeline(t *testing.T) {
	httpSide := []iomesh.MemoryOpsDigestReceipt{
		{ID: "m1", Summary: "dept pull", SourceHint: "palace_timeline"},
		{ID: "p1", Summary: "private RCA", SourceHint: "palace_timeline"},
	}
	mcpSide := []iomesh.MemoryOpsDigestReceipt{
		{
			ID: "m1", Summary: "dept pull", SourceHint: "palace_timeline",
			Tags:       []string{"source_hint:mesh"},
			Provenance: iomesh.MemoryOpsDigestProvenance{SourceHint: "mesh"},
		},
	}
	got := mergeDigestReceipts(httpSide, mcpSide)
	if len(got) != 2 {
		t.Fatalf("n=%d ids=%v", len(got), receiptIDs(got))
	}
	var mesh, priv bool
	for _, r := range got {
		switch ClassifyDigestReceipt(r) {
		case DigestSourceMesh:
			mesh = true
			if r.SourceHint != "palace_timeline" {
				t.Fatalf("must keep export origin palace_timeline: %+v", r)
			}
		case DigestSourcePrivate:
			priv = true
		}
	}
	if !mesh || !priv {
		t.Fatalf("merge must cite mesh+private: mesh=%v private=%v", mesh, priv)
	}
	out := FormatRequireSourcesCheck(&iomesh.MemoryOpsDigestResult{Receipts: got}, []string{"mesh", "private"})
	if !strings.Contains(out, "require-sources: ok") {
		t.Fatalf("want ok: %q", out)
	}
}

func TestMemoryOpsDigest_HTTPPalaceTimelineMergesMCPProvenance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"window": "day", "horizon": "ops",
			"since": "2026-09-09T16:10:00Z",
			"as_of": "2026-09-10T16:10:00Z",
			"honesty": map[string]any{
				"ops_pulse": "ga_path", "never_invent_ga": true, "dual_write_default": "off",
			},
			"patterns": []any{},
			"receipts": []map[string]any{
				{"id": "p1", "event_time": "2026-09-10T16:10:12Z", "summary": "private RCA", "source_hint": "palace_timeline"},
				{"id": "m1", "event_time": "2026-09-10T06:46:00Z", "summary": "mesh pull INC-9", "source_hint": "palace_timeline"},
			},
		})
	}))
	defer srv.Close()

	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockMCPOpsDigestExport(cOutW, cInR, `{
		"window": "day", "horizon": "ops",
		"since": "2026-09-09T16:10:00Z",
		"as_of": "2026-09-10T16:10:00Z",
		"patterns": [],
		"receipts": [
			{"id": "m1", "event_time": "2026-09-10T06:46:00Z", "summary": "mesh pull INC-9",
			 "source_hint": "palace_timeline", "tags": ["source_hint:mesh"],
			 "provenance": {"source_hint": "mesh"}},
			{"id": "p1", "event_time": "2026-09-10T16:10:12Z", "summary": "private RCA", "source_hint": "palace_timeline"}
		]
	}`)

	mut := false
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "memory", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)

	mesh := iomesh.New(iomesh.Config{Enabled: true, Endpoint: srv.URL, Tenant: "t"}, nil)
	rt := &Runtime{
		mesh:   mesh,
		mcp:    mgr,
		memory: MemoryConfig{Enabled: true, Tenant: "t", Server: "memory", DualWrite: false},
	}
	out, err := rt.MemoryOpsDigest(context.Background(), MemoryOpsDigestOpts{
		RequireSources: []string{"mesh", "private"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "cited=(none)") {
		t.Fatalf("HTTP+MCP merge must not cited=(none): %q", out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "require-sources: ok") {
		t.Fatalf("want cite-both from MCP provenance/tags: %q", out)
	}
	if !strings.Contains(out, "cited=mesh,private") {
		t.Fatalf("want both cited: %q", out)
	}
	if rt.memory.DualWrite || strings.Contains(out, "dual_write ON") {
		t.Fatal("dual_write must remain OFF")
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "Connected") {
		t.Fatalf("must not invent GA / Connected: %q", out)
	}
}

func TestApplyRequireSources_NilResultStillNamesWindow(t *testing.T) {
	stub := digestResultOrStub(nil, opsDigestLimitCiteBoth)
	out := applyRequireSources("ops digest window=day horizon=ops\nreceipts: (none)\n", stub, []string{"mesh", "private"})
	if !strings.Contains(out, "require-sources: miss") {
		t.Fatalf("want miss: %q", out)
	}
	if !strings.Contains(out, "receipt window newest-first") || !strings.Contains(out, "limit=50") {
		t.Fatalf("sticky miss must name window: %q", out)
	}
}

func mockMCPOpsDigestExport(w io.WriteCloser, r io.Reader, payload string) {
	defer w.Close()
	dec := json.NewDecoder(r)
	var structured any
	_ = json.Unmarshal([]byte(payload), &structured)
	for {
		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			return
		}
		id := req["id"]
		method, _ := req["method"].(string)
		if method == "notifications/initialized" || id == nil {
			continue
		}
		var result any
		switch method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{"name": "memory", "version": "1"}}
		case "tools/list":
			result = map[string]any{"tools": []map[string]any{{
				"name": "ops_digest_export", "description": "ops digest",
				"inputSchema": map[string]any{"type": "object"},
			}}}
		case "tools/call":
			result = map[string]any{
				"content":           []map[string]any{{"type": "text", "text": payload}},
				"structuredContent": structured,
			}
		}
		line, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
		_, _ = w.Write(append(line, '\n'))
	}
}

func twoDigit(n int) string {
	if n < 0 {
		n = 0
	}
	if n > 59 {
		n = 59
	}
	return string([]byte{'0' + byte(n/10), '0' + byte(n%10)})
}

func receiptIDs(rs []iomesh.MemoryOpsDigestReceipt) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.ID
	}
	return out
}

package agent

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/mcp"
)

func TestMemoryExtractFacts_DisabledAndRequired(t *testing.T) {
	rt := &Runtime{memory: DefaultMemoryConfig()}
	_, err := rt.MemoryExtractFacts(context.Background(), "mem_abc")
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled err=%v", err)
	}
	rt2 := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}}
	_, err2 := rt2.MemoryExtractFacts(context.Background(), "")
	if err2 == nil || !strings.Contains(err2.Error(), "memory_id required") {
		t.Fatalf("memory_id required err=%v", err2)
	}
	_, err3 := rt2.MemoryExtractFacts(context.Background(), "   ")
	if err3 == nil || !strings.Contains(err3.Error(), "memory_id required") {
		t.Fatalf("blank memory_id err=%v", err3)
	}
}

func TestMemoryExtractFacts_OfflineFailOpen(t *testing.T) {
	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research"},
		mcp:    mcp.NewManagerEmpty(nil),
	}
	out, err := rt.MemoryExtractFacts(context.Background(), "mem_abc")
	if err != nil {
		t.Fatalf("expected fail-open nil err, got %v", err)
	}
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "not connected") {
		t.Fatalf("offline: %q", out)
	}
	if !strings.Contains(out, "not NLP") || !strings.Contains(out, "not Memory GA") {
		t.Fatalf("honesty: %q", out)
	}
	if !strings.Contains(out, "dual_write OFF") {
		t.Fatalf("dual_write pin: %q", out)
	}
	if !strings.Contains(out, "do not invent facts") {
		t.Fatalf("empty≠invent pin: %q", out)
	}
	if strings.Contains(out, "facts: (none)") && !strings.Contains(out, "unavailable") {
		t.Fatalf("must not invent empty-success: %q", out)
	}
}

func TestMemoryExtractFacts_ToolNotOnHost(t *testing.T) {
	var called []string
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockMCPExtractFacts(cOutW, cInR, []string{"memory_ingest_turn"}, &called, nil, `{"memory_id":"invented","facts":["nope"]}`)

	mut := true
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "memory", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)

	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "default", DualWrite: false},
		mcp:    mgr,
	}
	out, err := rt.MemoryExtractFacts(context.Background(), "mem_abc")
	if err != nil {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if !strings.Contains(out, "not on host") || !strings.Contains(out, memoryExtractFactsTool) {
		t.Fatalf("want tool-not-on-host residual: %q", out)
	}
	if !strings.Contains(out, "do not invent facts") {
		t.Fatalf("invent pin: %q", out)
	}
	if strings.Contains(out, "nope") || strings.Contains(out, "invented") {
		t.Fatalf("must not invent facts from missing tool: %q", out)
	}
	for _, name := range called {
		if name == memoryExtractFactsTool {
			t.Fatalf("must not CallTool when tool not listed: %v", called)
		}
	}
}

func TestMemoryExtractFacts_CallToolArgs(t *testing.T) {
	var called []string
	var gotArgs map[string]any
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	payload := `{"memory_id":"mem_abc","facts":["Alice owns Project alpha","Ship date is Friday"]}`
	go mockMCPExtractFacts(cOutW, cInR, []string{memoryExtractFactsTool}, &called, &gotArgs, payload)

	mut := true
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "memory", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)

	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "dept.research", DualWrite: false},
		mcp:    mgr,
	}
	out, err := rt.MemoryExtractFacts(context.Background(), "mem_abc")
	if err != nil {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if len(called) != 1 || called[0] != memoryExtractFactsTool {
		t.Fatalf("called=%v", called)
	}
	if gotArgs["memory_id"] != "mem_abc" {
		t.Fatalf("memory_id=%v args=%v", gotArgs["memory_id"], gotArgs)
	}
	if gotArgs["tenant"] != "dept.research" {
		t.Fatalf("tenant=%v", gotArgs["tenant"])
	}
	if v, ok := gotArgs["dual_write"]; ok {
		switch tval := v.(type) {
		case bool:
			if tval {
				t.Fatalf("dual_write must not be ON: %v", v)
			}
		case string:
			if strings.EqualFold(tval, "on") || strings.EqualFold(tval, "true") {
				t.Fatalf("dual_write must not be ON: %v", v)
			}
		}
	}
	if !strings.Contains(out, "Alice owns Project alpha") || !strings.Contains(out, "Ship date is Friday") {
		t.Fatalf("facts: %q", out)
	}
	if !strings.Contains(out, "not NLP") || !strings.Contains(out, "dual_write OFF") {
		t.Fatalf("honesty: %q", out)
	}
}

func TestMemoryExtractFacts_OmitsTenantWhenUnset(t *testing.T) {
	var gotArgs map[string]any
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockMCPExtractFacts(cOutW, cInR, []string{memoryExtractFactsTool}, nil, &gotArgs, `{"memory_id":"m1","facts":[]}`)

	mut := true
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "memory", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)

	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", DualWrite: false},
		mcp:    mgr,
	}
	out, err := rt.MemoryExtractFacts(context.Background(), "m1")
	if err != nil {
		t.Fatalf("err=%v out=%q", err, out)
	}
	if _, ok := gotArgs["tenant"]; ok {
		t.Fatalf("empty tenant must omit tenant arg; got %v", gotArgs["tenant"])
	}
	if !strings.Contains(out, "facts: (none)") {
		t.Fatalf("empty facts: %q", out)
	}
}

func TestMemoryIngestTurn_DoesNotCallExtractFacts(t *testing.T) {
	var called []string
	var gotArgs map[string]any
	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	go mockMCPExtractFacts(cOutW, cInR, []string{"memory_ingest_turn", memoryExtractFactsTool}, &called, &gotArgs, `{"memory_id":"mem_test","tier":1,"tenant":"default","audited":false,"dual_write":"off"}`)

	mut := true
	cl := mcp.NewClientForTest(mcp.ServerConfig{Name: "memory", Command: "x", Mutating: &mut}, cInW, cOutR, nil)
	defer cl.Close()
	if err := cl.InitForTest(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)

	rt := &Runtime{
		memory: MemoryConfig{Enabled: true, Server: "memory", Tenant: "default", DualWrite: false, PalaceRoot: t.TempDir()},
		mcp:    mgr,
	}
	out, err := rt.MemoryIngestTurn(context.Background(), "user", "Demo note: overlay needle alpha")
	if err != nil {
		t.Fatalf("err=%v out=%q", err, out)
	}
	for _, name := range called {
		if name == memoryExtractFactsTool {
			t.Fatalf("ingest must not auto-extract; called=%v", called)
		}
	}
	foundIngest := false
	for _, name := range called {
		if name == "memory_ingest_turn" {
			foundIngest = true
		}
	}
	if !foundIngest {
		t.Fatalf("expected memory_ingest_turn; called=%v", called)
	}
}

func TestFormatExtractFactsJSON_StringsAndObjects(t *testing.T) {
	out := formatExtractFactsJSON(`{"memory_id":"m1","facts":["alpha","beta"]}`, "fallback", 6000)
	if !strings.Contains(out, "memory_id=m1") || !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
		t.Fatalf("strings: %q", out)
	}
	if !strings.Contains(out, extractFactsHonestyFooter) {
		t.Fatalf("honesty: %q", out)
	}
	out2 := formatExtractFactsJSON(`{"extracted_facts":[{"summary":"from object"}]}`, "m2", 6000)
	if !strings.Contains(out2, "from object") || !strings.Contains(out2, "memory_id=m2") {
		t.Fatalf("objects: %q", out2)
	}
	out3 := formatExtractFactsJSON(`{"memory_id":"m3","facts":[]}`, "m3", 6000)
	if !strings.Contains(out3, "facts: (none)") {
		t.Fatalf("empty: %q", out3)
	}
	if got := formatExtractFactsJSON("not json", "m", 100); got != "" {
		t.Fatalf("non-json: %q", got)
	}
}

func mockMCPExtractFacts(w io.WriteCloser, r io.Reader, tools []string, called *[]string, got *map[string]any, payload string) {
	defer w.Close()
	dec := json.NewDecoder(r)
	listed := make([]map[string]any, 0, len(tools))
	for _, name := range tools {
		listed = append(listed, map[string]any{
			"name":        name,
			"description": name,
			"inputSchema": map[string]any{"type": "object"},
		})
	}
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
			result = map[string]any{"tools": listed}
		case "tools/call":
			if params, _ := req["params"].(map[string]any); params != nil {
				if name, _ := params["name"].(string); name != "" && called != nil {
					*called = append(*called, name)
				}
				if args, ok := params["arguments"].(map[string]any); ok && got != nil {
					*got = args
				}
			}
			result = map[string]any{"content": []map[string]any{{"type": "text", "text": payload}}}
		}
		line, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
		_, _ = w.Write(append(line, '\n'))
	}
}

package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/workspace"
)

func TestDefaultTools_ReadAndFilter(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hi.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	reg := DefaultTools(ws)
	if !reg.IsMutating("run_shell") || !reg.IsMutating("write_file") {
		t.Fatal("mutating flags")
	}
	if reg.IsMutating("read_file") {
		t.Fatal("read should not mutate")
	}
	out, err := reg.Execute(context.Background(), "read_file", `{"path":"hi.txt"}`)
	if err != nil || !strings.Contains(out, "hello") {
		t.Fatalf("read=%q err=%v", out, err)
	}

	// Path escape
	if _, err := reg.Execute(context.Background(), "read_file", `{"path":"../secret"}`); err == nil {
		t.Fatal("expected path jail")
	}

	// Filter to read-only
	ro := reg.FilterTools([]string{"read_file", "list_dir", "grep"})
	if _, err := ro.Execute(context.Background(), "write_file", `{"path":"x","content":"y"}`); err == nil {
		t.Fatal("write should be filtered out")
	}
	if len(ro.Schemas()) != 3 {
		t.Fatalf("schemas=%d", len(ro.Schemas()))
	}
}

func TestRunShell_PolicyAndEnvScrub(t *testing.T) {
	dir := t.TempDir()
	ws, err := workspace.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	reg := DefaultTools(ws)

	// Dangerous command blocked
	_, err = reg.Execute(context.Background(), "run_shell", `{"command":"rm -rf /"}`)
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("expected block, err=%v", err)
	}

	// Secret scrub: child must not see DEEPSEEK_API_KEY
	t.Setenv("DEEPSEEK_API_KEY", "sk-should-not-leak-12345678")
	out, err := reg.Execute(context.Background(), "run_shell", `{"command":"echo DEEPSEEK=$DEEPSEEK_API_KEY; echo OK"}`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "sk-should-not-leak") {
		t.Fatalf("api key leaked into shell output: %q", out)
	}
	if !strings.Contains(out, "OK") {
		t.Fatalf("unexpected out=%q", out)
	}
}

func TestWriteFile_RequiresValidJSON(t *testing.T) {
	dir := t.TempDir()
	ws, _ := workspace.Open(dir)
	reg := DefaultTools(ws)
	_, err := reg.Execute(context.Background(), "write_file", `not-json`)
	if err == nil {
		t.Fatal("expected json error")
	}
	_, err = reg.Execute(context.Background(), "nope", `{}`)
	if err == nil {
		t.Fatal("unknown tool")
	}
}

func TestSchemas_ValidJSONParameters(t *testing.T) {
	dir := t.TempDir()
	ws, _ := workspace.Open(dir)
	for _, s := range DefaultTools(ws).Schemas() {
		if s.Function.Name == "" {
			t.Fatal("empty name")
		}
		if len(s.Function.Parameters) > 0 {
			var m map[string]any
			if err := json.Unmarshal(s.Function.Parameters, &m); err != nil {
				t.Fatalf("%s params: %v", s.Function.Name, err)
			}
		}
	}
}

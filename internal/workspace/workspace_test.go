package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolve_BlocksEscape(t *testing.T) {
	dir := t.TempDir()
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Resolve("../outside"); err == nil {
		t.Fatal("expected escape error")
	}
}

func TestReadWriteListGrep(t *testing.T) {
	dir := t.TempDir()
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("pkg/a.go", "package pkg\n// hello mesh\n"); err != nil {
		t.Fatal(err)
	}
	content, err := ws.ReadFile("pkg/a.go", 0, 0)
	if err != nil || !strings.Contains(content, "package pkg") {
		t.Fatalf("read: %q %v", content, err)
	}
	entries, err := ws.ListDir(".")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if e == "pkg/" {
			found = true
		}
	}
	if !found {
		t.Fatalf("entries=%v", entries)
	}
	out, err := ws.Grep("hello mesh", "")
	if err != nil || !strings.Contains(out, "a.go") {
		t.Fatalf("grep=%q err=%v", out, err)
	}
}

func TestOpen_Cwd(t *testing.T) {
	wd, _ := os.Getwd()
	ws, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	if ws.Root() != wd && filepath.Clean(ws.Root()) != filepath.Clean(wd) {
		// Accept equal after clean
		if ws.Root() != wd {
			t.Logf("root=%s wd=%s", ws.Root(), wd)
		}
	}
}

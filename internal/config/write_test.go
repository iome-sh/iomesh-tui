package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteToPath_RoundTripDualWriteOff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := Default()
	cfg.Memory.Enabled = true
	cfg.Memory.Server = "iomesh-memory-mcp"
	cfg.Memory.DualWrite = false
	cfg.MCP.Enabled = true
	if err := WriteToPath(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Memory.DualWrite {
		t.Fatal("dual_write must remain false after write/load")
	}
	if !loaded.Memory.Enabled {
		t.Fatal("memory enabled lost")
	}
}

func TestWriteSetupManagedFragment_ReplaceAndPreserve(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	user := "# user top\n[ui]\ntheme = \"mono\"\n"
	if err := os.WriteFile(path, []byte(user), 0o600); err != nil {
		t.Fatal(err)
	}
	frag1 := "[memory]\nenabled = true\ndual_write = false\n"
	if err := WriteSetupManagedFragment(path, frag1); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	s := string(data)
	if !strings.Contains(s, "theme = \"mono\"") {
		t.Fatalf("user content lost: %s", s)
	}
	if !strings.Contains(s, ManagedBegin) || !strings.Contains(s, "dual_write = false") {
		t.Fatalf("managed missing: %s", s)
	}
	frag2 := "[memory]\nenabled = true\nserver = \"iomesh-memory-mcp\"\ndual_write = false\n"
	if err := WriteSetupManagedFragment(path, frag2); err != nil {
		t.Fatal(err)
	}
	data2, _ := os.ReadFile(path)
	s2 := string(data2)
	if strings.Count(s2, ManagedBegin) != 1 {
		t.Fatalf("expected one managed block: %s", s2)
	}
	if !strings.Contains(s2, "iomesh-memory-mcp") {
		t.Fatalf("replace failed: %s", s2)
	}
	if !strings.Contains(s2, "theme = \"mono\"") {
		t.Fatal("user theme lost on replace")
	}
}

func TestWriteSetupManagedFragment_RejectDualWriteTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	err := WriteSetupManagedFragment(path, "dual_write = true\n")
	if err == nil {
		t.Fatal("expected error for dual_write true")
	}
}

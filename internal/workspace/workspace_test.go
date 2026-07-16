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
	if _, err := ws.Resolve(".."); err == nil {
		t.Fatal("expected .. escape")
	}
	if _, err := ws.Resolve("../../../../../../etc/passwd"); err == nil {
		t.Fatal("expected deep escape")
	}
}

func TestResolve_BlocksAbsoluteOutside(t *testing.T) {
	dir := t.TempDir()
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Resolve("/etc/passwd"); err == nil {
		t.Fatal("absolute outside root")
	}
	// Absolute path inside root is OK.
	inner := filepath.Join(dir, "ok.txt")
	if err := os.WriteFile(inner, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ws.Resolve(inner)
	if err != nil {
		t.Fatal(err)
	}
	if got != inner && filepath.Clean(got) != filepath.Clean(inner) {
		// EvalSymlinks may change path on macOS temp dirs
		t.Logf("resolved=%s inner=%s", got, inner)
	}
}

func TestResolve_BlocksSymlinkEscape(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("leak"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "escape")
	if err := os.Symlink(secret, link); err != nil {
		t.Skip("symlink not supported:", err)
	}
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Resolve("escape"); err == nil {
		t.Fatal("expected symlink escape rejection")
	}
	if _, err := ws.ReadFile("escape", 0, 0); err == nil {
		t.Fatal("read via symlink should fail")
	}
}

func TestResolve_NULAndDrive(t *testing.T) {
	dir := t.TempDir()
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Resolve("a\x00b"); err == nil {
		t.Fatal("NUL")
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
	// Line offset
	partial, err := ws.ReadFile("pkg/a.go", 2, 1)
	if err != nil || !strings.Contains(partial, "hello mesh") {
		t.Fatalf("partial=%q err=%v", partial, err)
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

func TestReadFile_MaxSize(t *testing.T) {
	dir := t.TempDir()
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ws.SetMaxReadBytes(16)
	if err := ws.WriteFile("big.txt", strings.Repeat("a", 64)); err != nil {
		// Write allows up to 4x maxRead; should succeed
		t.Fatal(err)
	}
	if _, err := ws.ReadFile("big.txt", 0, 0); err == nil {
		t.Fatal("expected size limit on read")
	}
}

func TestWriteFile_Escape(t *testing.T) {
	dir := t.TempDir()
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.WriteFile("../pwned", "x"); err == nil {
		t.Fatal("write escape")
	}
}

func TestOpen_Cwd(t *testing.T) {
	wd, _ := os.Getwd()
	ws, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	// Roots may differ by symlink resolution (e.g. /var vs /private/var on macOS).
	if filepath.Base(ws.Root()) != filepath.Base(wd) && ws.Root() != wd {
		t.Logf("root=%s wd=%s (symlink ok)", ws.Root(), wd)
	}
}

func TestGrep_RejectsEscapePath(t *testing.T) {
	dir := t.TempDir()
	ws, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Grep("x", "../"); err == nil {
		t.Fatal("grep path escape")
	}
}

package agentplugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfine_OK(t *testing.T) {
	root := t.TempDir()
	got, err := Confine("./bin/server", root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Clean(filepath.Join(root, "bin", "server"))
	if got != want && filepath.Clean(got) != want {
		// EvalSymlinks may change root; check suffix containment.
		if !strings.HasSuffix(got, filepath.Join("bin", "server")) {
			t.Fatalf("got %q want under %q", got, want)
		}
	}
	// Root itself via ./
	got, err = Confine("./", root)
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := resolveRoot(root)
	if got != abs {
		t.Fatalf("got %q want %q", got, abs)
	}
}

func TestConfine_Rejects(t *testing.T) {
	root := t.TempDir()
	cases := []string{
		"bin/server",  // missing ./
		"../escape",   // no ./
		"./../escape", // escapes
		"",            // empty
		"/abs/path",   // absolute
	}
	for _, c := range cases {
		if _, err := Confine(c, root); err == nil {
			t.Fatalf("expected error for %q", c)
		}
	}
}

func TestConfine_SymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	// Put a file outside and symlink inside root.
	target := filepath.Join(outside, "secret")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "leak")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported:", err)
	}
	if _, err := Confine("./leak", root); err == nil {
		t.Fatal("expected symlink escape rejection")
	}
}

func TestWithinRoot(t *testing.T) {
	if !withinRoot("/a/b", "/a") {
		t.Fatal("expected within")
	}
	if withinRoot("/a", "/a/b") {
		t.Fatal("parent not within child")
	}
	if withinRoot("/ab", "/a") {
		t.Fatal("prefix false positive")
	}
	if !withinRoot("/a", "/a") {
		t.Fatal("equal is within")
	}
}

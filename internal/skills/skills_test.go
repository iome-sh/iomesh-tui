package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse_Frontmatter(t *testing.T) {
	raw := `---
name: check-work
description: >
  Verify changes with a subagent
metadata:
  short-description: "x"
---

# Body

Do the thing.
`
	sk, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if sk.Name != "check-work" {
		t.Fatalf("name=%q", sk.Name)
	}
	if !strings.Contains(sk.Description, "Verify") {
		t.Fatalf("desc=%q", sk.Description)
	}
	if !strings.Contains(sk.Body, "Do the thing") {
		t.Fatalf("body=%q", sk.Body)
	}
}

func TestLoadDirs_SkillMDLayout(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "help")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: help\ndescription: Help skill\n---\n\n# Help\n\nBe helpful.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// second skill
	dir2 := filepath.Join(root, "review")
	_ = os.MkdirAll(dir2, 0o755)
	_ = os.WriteFile(filepath.Join(dir2, "SKILL.md"), []byte("---\nname: review\ndescription: Review code\n---\n\nReview well.\n"), 0o644)

	cat, err := LoadDirs(root)
	if err != nil {
		t.Fatal(err)
	}
	if cat.Len() != 2 {
		t.Fatalf("len=%d names=%v", cat.Len(), cat.Names())
	}
	sk, ok := cat.Get("help")
	if !ok || !strings.Contains(sk.Body, "helpful") {
		t.Fatalf("%+v ok=%v", sk, ok)
	}
	block := cat.PromptBlock()
	if !strings.Contains(block, "help") || !strings.Contains(block, "review") {
		t.Fatal(block)
	}
}

func TestLoadDirs_MissingOK(t *testing.T) {
	cat, err := LoadDirs(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if cat.Len() != 0 {
		t.Fatal(cat.Len())
	}
}

func TestSanitizeAndDefaultDirs(t *testing.T) {
	if sanitizeName("Hello World!") != "hello-world" {
		t.Fatal(sanitizeName("Hello World!"))
	}
	dirs := DefaultDirs(t.TempDir())
	if len(dirs) < 1 {
		t.Fatal(dirs)
	}
}

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

// s1251: builtin connector-integrations-setup always loads via go:embed.
func TestLoadBuiltin_ConnectorIntegrationsSetup(t *testing.T) {
	cat, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	if cat.Len() == 0 {
		t.Fatal("builtin catalog empty")
	}
	sk, ok := cat.Get("connector-integrations-setup")
	if !ok {
		t.Fatalf("missing connector-integrations-setup; names=%v", cat.Names())
	}
	if sk.Name != "connector-integrations-setup" {
		t.Fatalf("name=%q", sk.Name)
	}
	if strings.TrimSpace(sk.Description) == "" {
		t.Fatal("description empty")
	}
	// Body honesty needles
	for _, want := range []string{
		"list_connector_catalog",
		"plan_connector_setup",
		"portal HITL",
		"never invent install green",
	} {
		if !strings.Contains(sk.Body, want) && !strings.Contains(sk.Description, want) {
			// Body preferred; description may not have all needles
			if !strings.Contains(sk.Body, want) {
				t.Fatalf("body missing %q:\n%s", want, sk.Body)
			}
		}
	}
	// Extra residual locks in body
	for _, want := range []string{
		"browser HITL",
		"dual_write OFF",
		"book-demo OFF",
		"get_webhook_signing_headers",
	} {
		if !strings.Contains(sk.Body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
	// Source marks as builtin
	if !strings.Contains(sk.Path, "builtin") && sk.SourceDir != "builtin" {
		t.Fatalf("path/source not builtin: path=%q source=%q", sk.Path, sk.SourceDir)
	}
}

// s1251: LoadWithBuiltin always includes builtin even when dirs empty/missing.
func TestLoadWithBuiltin_EmptyDirsStillBuiltin(t *testing.T) {
	cat, err := LoadWithBuiltin(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cat.Get("connector-integrations-setup"); !ok {
		t.Fatalf("builtin missing when dirs empty; names=%v", cat.Names())
	}
}

// s1251: user skill overrides builtin on name collision; new names merge.
func TestLoadWithBuiltin_UserOverrides(t *testing.T) {
	root := t.TempDir()
	// Override builtin name with a stub user skill
	dir := filepath.Join(root, "connector-integrations-setup")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: connector-integrations-setup\ndescription: User override\n---\n\n# Override\n\nUser body.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	// Extra user skill
	dir2 := filepath.Join(root, "my-skill")
	_ = os.MkdirAll(dir2, 0o755)
	_ = os.WriteFile(filepath.Join(dir2, "SKILL.md"), []byte("---\nname: my-skill\ndescription: Custom\n---\n\nCustom body.\n"), 0o644)

	cat, err := LoadWithBuiltin(root)
	if err != nil {
		t.Fatal(err)
	}
	sk, ok := cat.Get("connector-integrations-setup")
	if !ok {
		t.Fatal("missing skill")
	}
	if !strings.Contains(sk.Body, "User body") {
		t.Fatalf("want user override body: %s", sk.Body)
	}
	if sk.Description != "User override" {
		t.Fatalf("desc=%q", sk.Description)
	}
	if _, ok := cat.Get("my-skill"); !ok {
		t.Fatalf("my-skill missing: %v", cat.Names())
	}
}

func TestCatalogMerge_NilSafe(t *testing.T) {
	var c *Catalog
	out := c.Merge(nil)
	if out == nil || out.Len() != 0 {
		t.Fatalf("%+v", out)
	}
	builtin, err := LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	out2 := (*Catalog)(nil).Merge(builtin)
	if out2.Len() != builtin.Len() {
		t.Fatalf("merge from nil: %d vs %d", out2.Len(), builtin.Len())
	}
}

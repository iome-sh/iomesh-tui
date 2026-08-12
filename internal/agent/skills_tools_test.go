package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/skills"
)

// TestSkillsNextStepLines_HonestyNeedles pins s1837 residual-honest next-step
// after list_skills / read_skill (peer of PluginsNextStepLines s1829 · MemoryNextStepLines s1831).
func TestSkillsNextStepLines_HonestyNeedles(t *testing.T) {
	lines := SkillsNextStepLines()
	if len(lines) == 0 {
		t.Fatal("empty skills next-step lines")
	}
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"dual path residual-honest after skills list/read or skills reload",
		"TUI/session running",
		"/setup preflight",
		"/setup reload",
		"skills re-scan",
		"package wire ≠ Connected",
		"list_skills tool",
		"/onboard next setup",
		"cold start",
		"restart iomesh",
		"iomesh setup preflight",
		"skills re-scan ≠ invent Connected",
		"dual_write OFF",
		"not Agent Plugins GA",
		"not Memory GA",
		"s1837",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("skills next-step missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Memory GA shipped") {
		t.Fatalf("must not invent dual_write ON / Memory GA shipped:\n%s", out)
	}
	if strings.Contains(out, "Connected: yes") || strings.Contains(out, "Agent Plugins GA shipped") {
		t.Fatalf("must not invent Connected green / Agent Plugins GA shipped:\n%s", out)
	}
}

// TestListSkillsReadSkill_S1837NextStepFooter pins residual-honest footers on
// successful list_skills / read_skill tool results (never invent success on error).
func TestListSkillsReadSkill_S1837NextStepFooter(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".iomesh", "skills", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: demo\ndescription: Demo skill\n---\n\n# Demo\n\nHello skill.\n"), 0o644)

	cat, err := skills.LoadDirs(filepath.Join(root, ".iomesh", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	rt := testRT(t, root)
	rt.AttachSkills(cat)

	needles := []string{
		"s1837",
		"/setup preflight",
		"/setup reload",
		"skills re-scan",
		"package wire ≠ Connected",
		"dual_write OFF",
		"not Agent Plugins GA",
		"not Memory GA",
		"list_skills tool",
		"/onboard next setup",
		"restart iomesh",
	}

	listOut, err := rt.tools.Execute(context.Background(), "list_skills", `{}`)
	if err != nil {
		t.Fatalf("list_skills: %v", err)
	}
	if !strings.Contains(listOut, "demo") {
		t.Fatalf("list_skills missing demo:\n%s", listOut)
	}
	for _, want := range needles {
		if !strings.Contains(listOut, want) {
			t.Fatalf("list_skills missing s1837 next-step %q in:\n%s", want, listOut)
		}
	}

	readOut, err := rt.tools.Execute(context.Background(), "read_skill", `{"name":"demo"}`)
	if err != nil {
		t.Fatalf("read_skill: %v", err)
	}
	if !strings.Contains(readOut, "Hello skill") {
		t.Fatalf("read_skill missing body:\n%s", readOut)
	}
	for _, want := range needles {
		if !strings.Contains(readOut, want) {
			t.Fatalf("read_skill missing s1837 next-step %q in:\n%s", want, readOut)
		}
	}

	// Unknown skill stays error — no invent success footer.
	_, err = rt.tools.Execute(context.Background(), "read_skill", `{"name":"missing-skill"}`)
	if err == nil {
		t.Fatal("expected unknown skill error")
	}
	if strings.Contains(err.Error(), "s1837") || strings.Contains(err.Error(), "dual_write OFF") {
		t.Fatalf("error path must not invent next-step success footer: %v", err)
	}
}

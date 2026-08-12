package agentplugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSamplePluginRelPaths(t *testing.T) {
	rels := SamplePluginRelPaths()
	if len(rels) != 2 {
		t.Fatalf("want 2 sample paths, got %v", rels)
	}
	if !strings.Contains(rels[0], "hello-iome") {
		t.Fatalf("first sample: %s", rels[0])
	}
	if !strings.Contains(rels[1], "iomesh-memory-mcp") {
		t.Fatalf("second sample: %s", rels[1])
	}
}

func TestDefaultSamplePluginDirs(t *testing.T) {
	got := DefaultSamplePluginDirs("/mod")
	if len(got) != 2 {
		t.Fatalf("%v", got)
	}
	if got[0] != filepath.Join("/mod", "examples", "agent-plugins", "hello-iome") {
		t.Fatal(got[0])
	}
	if got[1] != filepath.Join("/mod", "examples", "agent-plugins", "iomesh-memory-mcp") {
		t.Fatal(got[1])
	}
	// Empty module root → relative paths only.
	rel := DefaultSamplePluginDirs("")
	if len(rel) != 2 || filepath.IsAbs(rel[0]) {
		t.Fatalf("%v", rel)
	}
}

func TestFindModuleRoot(t *testing.T) {
	root := moduleRoot(t)
	// From package dir under module.
	found, err := FindModuleRoot(filepath.Join(root, "internal", "agentplugins"))
	if err != nil {
		t.Fatal(err)
	}
	if found != root {
		// EvalSymlinks may differ on some systems; compare cleaned abs.
		want, _ := filepath.EvalSymlinks(root)
		got, _ := filepath.EvalSymlinks(found)
		if want != got {
			t.Fatalf("found=%q want=%q", found, root)
		}
	}
	// Empty start uses cwd — run from known under-module path via chdir in subtest.
	t.Run("cwd", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(filepath.Join(root, "examples")); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(wd) })
		found, err := FindModuleRoot("")
		if err != nil {
			t.Fatal(err)
		}
		want, _ := filepath.EvalSymlinks(root)
		got, _ := filepath.EvalSymlinks(found)
		if want != got {
			t.Fatalf("found=%q want=%q", found, root)
		}
	})
	// Above any go.mod → error.
	tmp := t.TempDir()
	if _, err := FindModuleRoot(tmp); err == nil {
		t.Fatal("expected error above go.mod")
	}
}

// TestDogfoodSamples_BothOK pins s1357+s1478 offline residual-honest dogfood of both
// product samples when run from module root. Does not require iomesh-memory-mcp
// binary on PATH (PATH residual; connect skip · Discover ≠ Connected).
func TestDogfoodSamples_BothOK(t *testing.T) {
	root := moduleRoot(t)
	outcomes, warns, err := DogfoodSamples(root)
	if err != nil {
		t.Fatal(err)
	}
	if !DogfoodPass(outcomes) {
		t.Fatalf("dogfood fail: outcomes=%+v warns=%v", outcomes, warns)
	}
	if ValidateOKCount(outcomes) != 2 {
		t.Fatalf("ok count: %+v", outcomes)
	}
	if ValidateHasFatal(outcomes) {
		t.Fatalf("unexpected fatal: %+v", outcomes)
	}
	byName := map[string]ValidateOutcome{}
	for _, o := range outcomes {
		if o.OK {
			byName[o.Name] = o
		}
	}
	hi, ok := byName["hello-iome"]
	if !ok || hi.Skills != 1 || hi.MCP != 0 {
		t.Fatalf("hello-iome: %+v", hi)
	}
	am, ok := byName["iomesh-memory-mcp"]
	if !ok || am.Skills != 1 || am.MCP != 1 {
		t.Fatalf("iomesh-memory-mcp: %+v", am)
	}
	// PATH residual warning present; never a fatal.
	sawPATH := false
	for _, w := range warns {
		if strings.Contains(w, "PATH residual") {
			sawPATH = true
		}
	}
	if !sawPATH {
		t.Fatalf("want PATH residual warning; got %v", warns)
	}
	sum := FormatDogfoodSummary(outcomes)
	if !strings.Contains(sum, "PASS") || !strings.Contains(sum, "no MCP dial") {
		t.Fatal(sum)
	}
	if !strings.Contains(ResidualDogfoodHonesty, "PATH residual") {
		t.Fatal(ResidualDogfoodHonesty)
	}
	if !strings.Contains(ResidualDogfoodHonesty, "dual_write OFF") {
		t.Fatal(ResidualDogfoodHonesty)
	}
}

func TestDogfoodSamples_MissingModule(t *testing.T) {
	// Nonexistent module root → missing sample FAILs (not module-resolve error).
	tmp := filepath.Join(t.TempDir(), "not-a-module")
	outcomes, warns, err := DogfoodSamples(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if DogfoodPass(outcomes) {
		t.Fatal("expected fail")
	}
	if !ValidateHasFatal(outcomes) {
		t.Fatalf("%+v", outcomes)
	}
	if ValidateOKCount(outcomes) != 0 {
		t.Fatalf("%+v", outcomes)
	}
	if len(warns) == 0 {
		t.Fatal("want missing/PATH warnings")
	}
	sum := FormatDogfoodSummary(outcomes)
	if !strings.Contains(sum, "FAIL") {
		t.Fatal(sum)
	}
}

func TestSamplesSoftState(t *testing.T) {
	// Module root with both samples → samples_ok.
	root := moduleRoot(t)
	if got := SamplesSoftState(root); got != "samples_ok" {
		t.Fatalf("module root: got %q want samples_ok", got)
	}
	// Missing root → samples_missing.
	tmp := filepath.Join(t.TempDir(), "no-samples")
	if got := SamplesSoftState(tmp); got != "samples_missing" {
		t.Fatalf("missing: got %q want samples_missing", got)
	}
	// ResidualSlashHonesty pins (s1392).
	for _, want := range []string{
		"soft offline dogfood ≠ invent Agent Plugins GA",
		"dual_write OFF",
		"Discover ≠ Connected",
		"not Memory GA",
		"residual PASS ≠ live dogfood",
		"package load ≠ Memory GA",
	} {
		if !strings.Contains(ResidualSlashHonesty, want) {
			t.Fatalf("ResidualSlashHonesty missing %q: %s", want, ResidualSlashHonesty)
		}
	}
}

// TestPluginsNextStepLines_HonestyNeedles pins s1829 residual-honest next-step
// after /plugins list|validate|smoke|status (peer of IntegrationsNextStepLines s1727 ·
// OnboardNextStepLines s1825).
func TestPluginsNextStepLines_HonestyNeedles(t *testing.T) {
	lines := PluginsNextStepLines()
	if len(lines) == 0 {
		t.Fatal("empty plugins next-step lines")
	}
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"dual path residual-honest after plugins discover/validate/smoke",
		"TUI/session running",
		"/setup preflight",
		"/setup reload",
		"package wire ≠ Connected",
		"/onboard next plugins",
		"cold start",
		"restart iomesh",
		"iomesh setup preflight",
		"iomesh plugins smoke",
		"Discover ≠ Connected",
		"Agent Plugins GA",
		"package load ≠ Memory GA",
		"dual_write OFF",
		"s1829",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("plugins next-step missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "dual_write ON") || strings.Contains(out, "Memory GA shipped") {
		t.Fatalf("must not invent dual_write ON / Memory GA shipped:\n%s", out)
	}
	if strings.Contains(out, "Connected: yes") || strings.Contains(out, "Agent Plugins GA shipped") {
		t.Fatalf("must not invent Connected green / Agent Plugins GA shipped:\n%s", out)
	}
}

func TestDogfoodSamples_EmptyRootResolves(t *testing.T) {
	// Empty moduleRoot uses FindModuleRoot(cwd). Chdir to module so both samples OK.
	root := moduleRoot(t)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	outcomes, _, err := DogfoodSamples("")
	if err != nil {
		t.Fatal(err)
	}
	if !DogfoodPass(outcomes) {
		t.Fatalf("%+v", outcomes)
	}
}

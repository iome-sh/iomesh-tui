package setup

import (
	"strings"
	"testing"
)

func TestBuildManagedFragment_LocalMemoryDualWriteOff(t *testing.T) {
	frag, err := BuildManagedFragment([]Profile{ProfileLocalMemory}, DefaultInitOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(frag, "dual_write = false") {
		t.Fatalf("missing dual_write false:\n%s", frag)
	}
	if strings.Contains(frag, "dual_write = true") {
		t.Fatal("must not set dual_write true")
	}
	if !strings.Contains(frag, "iomesh-memory-mcp") {
		t.Fatal("missing memory server")
	}
	if !strings.Contains(frag, "[mcp]") || !strings.Contains(frag, "enabled = true") {
		t.Fatal("mcp not enabled")
	}
	// s1530 P5: pull_continuous default false (in-session opt-in · CLI still valid).
	if !strings.Contains(frag, "pull_continuous = false") {
		t.Fatalf("missing pull_continuous = false:\n%s", frag)
	}
	if strings.Contains(frag, "pull_continuous = true") {
		t.Fatal("must not set pull_continuous true by default")
	}
	// s1534 P6: analyze_continuous default false (opt-in analyze ticks · drift report-only).
	if !strings.Contains(frag, "analyze_continuous = false") {
		t.Fatalf("missing analyze_continuous = false:\n%s", frag)
	}
	if strings.Contains(frag, "analyze_continuous = true") {
		t.Fatal("must not set analyze_continuous true by default")
	}
}

func TestBuildManagedFragment_All(t *testing.T) {
	opt := DefaultInitOptions()
	opt.MeshEndpoint = "https://mesh.example"
	opt.PlatformMCPURL = "https://mcp.example/mcp"
	opt.PluginsDirs = []string{"/tmp/plugins"}
	frag, err := BuildManagedFragment([]Profile{ProfileAll}, opt)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"[iomesh]",
		"aion-platform",
		"[plugins]",
		"dirs = [\"/tmp/plugins\"]",
		"dual_write = false",
		"portal HITL",
	} {
		if !strings.Contains(frag, needle) {
			t.Fatalf("missing %q in:\n%s", needle, frag)
		}
	}
}

func TestParseProfiles(t *testing.T) {
	p := ParseProfiles("local-memory,plugins")
	if len(p) != 2 {
		t.Fatalf("%v", p)
	}
}

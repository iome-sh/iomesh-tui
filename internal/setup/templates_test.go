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

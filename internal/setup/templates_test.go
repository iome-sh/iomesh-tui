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
		`api_key_env = "IOMESH_TOKEN"`,
		"# org =",
		"IOMESH_ORG",
		"fail-open",
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

func TestBuildManagedFragment_PlatformMCPInfersHooks(t *testing.T) {
	opt := DefaultInitOptions()
	opt.PlatformMCPURL = "https://apiv1.iome.sh/v7/mcp"
	opt.MeshTenant = "dept.engineering"
	frag, err := BuildManagedFragment([]Profile{ProfilePlatformMCP}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(frag, "[iomesh]") || !strings.Contains(frag, "https://hooks.iome.sh") {
		t.Fatalf("want inferred broker in:\n%s", frag)
	}
	if !strings.Contains(frag, "not portal /v7/mcp") {
		t.Fatalf("want honesty comment in:\n%s", frag)
	}
}

func TestBuildManagedFragment_MeshOrgResidualEmpty(t *testing.T) {
	opt := DefaultInitOptions()
	opt.MeshEndpoint = "https://hooks.iome.sh"
	frag, err := BuildManagedFragment([]Profile{ProfileMesh}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(frag, "[iomesh]") {
		t.Fatalf("missing [iomesh]:\n%s", frag)
	}
	if !strings.Contains(frag, `api_key_env = "IOMESH_TOKEN"`) {
		t.Fatalf("missing api_key_env:\n%s", frag)
	}
	if !strings.Contains(frag, "# org =") {
		t.Fatalf("empty org must write commented residual:\n%s", frag)
	}
	if strings.Contains(frag, "\norg = ") {
		t.Fatalf("empty org must not persist a live org field:\n%s", frag)
	}
	for _, needle := range []string{
		"IOMESH_ORG",
		"X-IOMesh-Org",
		"fail-open",
		"aion #2721",
		"never invent Connected",
	} {
		if !strings.Contains(frag, needle) {
			t.Fatalf("missing org residual honesty %q in:\n%s", needle, frag)
		}
	}
	if strings.Contains(frag, "dual_write = true") {
		t.Fatal("must not set dual_write true")
	}
}

func TestBuildManagedFragment_MeshOrgPersistsWhenSet(t *testing.T) {
	opt := DefaultInitOptions()
	opt.MeshEndpoint = "https://hooks.iome.sh"
	opt.MeshTenant = "dept.engineering"
	opt.MeshOrg = "org_a"
	frag, err := BuildManagedFragment([]Profile{ProfileMesh}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(frag, `org = "org_a"`) {
		t.Fatalf("want persisted org field:\n%s", frag)
	}
	if !strings.Contains(frag, "IOMESH_ORG") || !strings.Contains(frag, "X-IOMesh-Org") {
		t.Fatalf("want IOMESH_ORG / X-IOMesh-Org honesty:\n%s", frag)
	}
	if strings.Contains(frag, "secret") && strings.Contains(frag, "org_a") && strings.Contains(frag, "token=") {
		t.Fatalf("must not inline secrets next to org:\n%s", frag)
	}
}

func TestParseProfiles(t *testing.T) {
	p := ParseProfiles("local-memory,plugins")
	if len(p) != 2 {
		t.Fatalf("%v", p)
	}
}

func TestProfilesWantMesh(t *testing.T) {
	if ProfilesWantMesh([]Profile{ProfileLocalMemory}) {
		t.Fatal("local-memory is not mesh")
	}
	if !ProfilesWantMesh([]Profile{ProfileMesh}) || !ProfilesWantMesh([]Profile{ProfilePlatformMCP}) || !ProfilesWantMesh([]Profile{ProfileAll}) {
		t.Fatal("mesh / platform-mcp / all should want mesh")
	}
}

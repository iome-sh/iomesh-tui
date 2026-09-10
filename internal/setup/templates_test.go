package setup

import (
	"os"
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
	if !strings.Contains(frag, `palace_root = "~/.iomesh/palace"`) {
		t.Fatalf("HTTP local-memory must write palace_root:\n%s", frag)
	}
	for _, needle := range []string{
		"HTTP MCP URL-only has no stdio -palace-root args",
		"IOMESH_MEMORY_PALACE_ROOT",
		"match the MCP process -palace-root",
		"never invent Connected",
		"not Memory GA",
	} {
		if !strings.Contains(frag, needle) {
			t.Fatalf("HTTP palace_root honesty missing %q:\n%s", needle, frag)
		}
	}
}

func TestBuildManagedFragment_LocalMemoryPalaceRootKnown(t *testing.T) {
	opt := DefaultInitOptions()
	opt.MemoryPalaceRoot = "/workspace/data/memory-palaces"
	frag, err := BuildManagedFragment([]Profile{ProfileLocalMemory}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(frag, `palace_root = "/workspace/data/memory-palaces"`) {
		t.Fatalf("must write known palace_root:\n%s", frag)
	}
	if strings.Contains(frag, "dual_write = true") {
		t.Fatal("must not set dual_write true")
	}
}

func TestBuildManagedFragment_LocalMemoryStdioPalaceRoot(t *testing.T) {
	opt := DefaultInitOptions()
	opt.UseStdioMemory = true
	opt.MemoryPalaceRoot = "/tmp/stdio-palace"
	frag, err := BuildManagedFragment([]Profile{ProfileLocalMemory}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(frag, `args = ["-palace-root", "/tmp/stdio-palace"`) {
		t.Fatalf("stdio must keep -palace-root args:\n%s", frag)
	}
	if !strings.Contains(frag, `palace_root = "/tmp/stdio-palace"`) {
		t.Fatalf("stdio must write matching palace_root:\n%s", frag)
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
		"iomesh-platform",
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
	if !strings.Contains(frag, "broker streams (hooks.*)") || !strings.Contains(frag, "catalog CP") {
		t.Fatalf("want honesty comment in:\n%s", frag)
	}
	if strings.Contains(frag, `endpoint = "https://apiv1`) {
		t.Fatalf("must not stamp portal CP as [iomesh] endpoint:\n%s", frag)
	}
}

func TestBuildManagedFragment_MeshEndpointAPIv1Honesty(t *testing.T) {
	opt := DefaultInitOptions()
	opt.MeshEndpoint = "https://apiv1.staging.iome.sh"
	frag, err := BuildManagedFragment([]Profile{ProfileMesh}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(frag, `endpoint = "https://apiv1.staging.iome.sh"`) {
		t.Fatalf("explicit mesh-endpoint is residual-honest (no silent rewrite):\n%s", frag)
	}
	if !strings.Contains(frag, "portal/catalog CP") || !strings.Contains(frag, "not broker streams") {
		t.Fatalf("apiv1 endpoint must not be labeled broker streams:\n%s", frag)
	}
	if !strings.Contains(frag, "hooks.staging.iome.sh") {
		t.Fatalf("want hooks residual in comment:\n%s", frag)
	}
	if strings.Contains(frag, `endpoint = "https://apiv1.staging.iome.sh"  # broker streams`) {
		t.Fatalf("must not stamp apiv1 as broker streams:\n%s", frag)
	}
	if strings.Contains(frag, "Connected: yes") || strings.Contains(frag, "dual_write = true") {
		t.Fatalf("must not invent Connected / dual_write ON:\n%s", frag)
	}
}

func TestBuildManagedFragment_MeshEndpointHooksHonesty(t *testing.T) {
	opt := DefaultInitOptions()
	opt.MeshEndpoint = "https://hooks.staging.iome.sh"
	frag, err := BuildManagedFragment([]Profile{ProfileMesh}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(frag, `endpoint = "https://hooks.staging.iome.sh"`) {
		t.Fatalf("want hooks endpoint:\n%s", frag)
	}
	if !strings.Contains(frag, "broker streams (hooks.*)") {
		t.Fatalf("hooks endpoint may be labeled broker streams:\n%s", frag)
	}
	if !strings.Contains(frag, "portal apiv1.* is catalog CP") {
		t.Fatalf("want CP vs broker contrast:\n%s", frag)
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
		"broker empty-org fail-open",
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

func TestMeshEndpointHonestyComment(t *testing.T) {
	apiv1 := meshEndpointHonestyComment("https://apiv1.staging.iome.sh")
	if strings.Contains(apiv1, "broker streams") && !strings.Contains(apiv1, "not broker streams") {
		t.Fatalf("apiv1 comment must not call CP a broker: %s", apiv1)
	}
	if !strings.Contains(apiv1, "portal/catalog CP") || !strings.Contains(apiv1, "hooks.") {
		t.Fatalf("apiv1 comment want CP vs hooks: %s", apiv1)
	}
	hooks := meshEndpointHonestyComment("https://hooks.staging.iome.sh")
	if !strings.Contains(hooks, "broker streams (hooks.*)") || !strings.Contains(hooks, "catalog CP") {
		t.Fatalf("hooks comment want broker vs CP: %s", hooks)
	}
}

func TestExampleConfig_IOMeshEndpointHonesty(t *testing.T) {
	b, err := os.ReadFile("../../configs/config.example.toml")
	if err != nil {
		t.Fatal(err)
	}
	txt := string(b)
	for _, want := range []string{
		"hooks.iome.sh",
		"hooks.staging.iome.sh",
		"apiv1.* is portal/catalog CP",
		"not a broker streams endpoint",
		"Catalog ≠ Connected",
	} {
		if !strings.Contains(txt, want) {
			t.Fatalf("example config missing honesty needle %q", want)
		}
	}
	if strings.Contains(txt, `endpoint = "https://apiv1`) {
		t.Fatal("example config must not set [iomesh].endpoint to apiv1")
	}
	if strings.Contains(txt, `endpoint = "https://iomesh.example.com"`) {
		t.Fatal("example config must not use a generic host that hides hooks vs apiv1")
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

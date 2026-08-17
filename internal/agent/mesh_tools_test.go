package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
)

func TestReplaceMesh_HotSwapAndDetach(t *testing.T) {
	rt := testRT(t, t.TempDir())
	rt.ReplaceMesh(nil)
	if rt.Mesh() != nil {
		t.Fatal("expected nil mesh after ReplaceMesh(nil)")
	}
	sys := rt.Messages()[0].Content
	if !strings.Contains(sys, "detached") || !strings.Contains(sys, "infer ≠ Connected") {
		t.Fatalf("detached note: %s", sys)
	}

	on := iomesh.New(iomesh.Config{
		Enabled: true, Endpoint: "https://hooks.iome.sh", CatalogPlane: true,
	}, nil)
	rt.ReplaceMesh(on)
	if rt.Mesh() != on {
		t.Fatal("mesh not swapped")
	}
	out, err := rt.tools.Execute(context.Background(), "mesh_status", `{}`)
	if err != nil {
		t.Fatalf("mesh_status after attach: %v", err)
	}
	if !strings.Contains(out, "mesh: enabled") {
		t.Fatalf("status: %s", out)
	}

	off := iomesh.New(iomesh.Config{}, nil)
	rt.ReplaceMesh(off)
	if rt.Mesh() == nil || rt.Mesh().Enabled() {
		t.Fatal("expected disabled mesh after replace")
	}
	if _, err := rt.tools.Execute(context.Background(), "mesh_status", `{}`); err == nil {
		t.Fatal("mesh_status should unregister when catalog off")
	}
	if strings.Count(rt.Messages()[0].Content, "<iomesh-tools>") != 1 {
		t.Fatalf("iomesh-tools note duplicated: %s", rt.Messages()[0].Content)
	}
}

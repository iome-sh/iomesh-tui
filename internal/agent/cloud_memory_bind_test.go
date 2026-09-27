package agent

import "testing"

func TestSessionPalaceRefusesAreNotABind(t *testing.T) {
	cases := []SessionPalaceInput{
		{HostedPalaceEnabled: true},
		{SharedMemoryURL: "https://cfg.example/mcp"},
		{WorkspaceMemoryURL: "https://cfg.example/mcp", SharedMemoryURL: "https://cfg.example/mcp"},
		{WorkspaceMemoryURL: "https://aion-mem-acme.internal"},
		{WorkspaceMemoryURL: ""},
		{WorkspaceMemoryURL: "~/.iomesh/palace"},
		{CatalogRow: true, WorkspacePatched: true, OpenGates: []string{}},
	}
	for _, in := range cases {
		if got := DisplaySessionPalace(in); got != "palace=-" {
			t.Fatalf("got %q for %+v", got, in)
		}
	}
	in := SessionPalaceInput{
		WorkspaceMemoryURL:  "https://mem.customer.example/mcp",
		SharedMemoryURL:     "https://cfg.example/mcp",
		HostedPalaceEnabled: true,
		CatalogRow:          true,
		WorkspacePatched:    true,
	}
	if got := SelectSessionPalace(in); got != in.WorkspaceMemoryURL {
		t.Fatalf("workspace URL got %q", got)
	}
	if got := DisplaySessionPalace(in); got != "palace="+in.WorkspaceMemoryURL {
		t.Fatalf("display %q", got)
	}
}

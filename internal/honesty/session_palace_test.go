package honesty

import (
	"strings"
	"testing"
)

func TestHostedPalaceEnabledIsNotABind(t *testing.T) {
	cases := []SessionPalaceInput{
		{HostedPalaceEnabled: true},
		{HostedPalaceEnabled: true, SharedMemoryURL: "https://shared.example/mcp"},
		{HostedPalaceEnabled: true, WorkspaceMemoryURL: "https://aion-mem-acme.internal"},
		{HostedPalaceEnabled: true, CatalogRow: true, WorkspacePatched: true},
	}
	for _, in := range cases {
		if got := SelectSessionPalace(in); got != "" {
			t.Fatalf("HostedPalaceEnabled must not bind, got %q for %+v", got, in)
		}
		if got := DisplaySessionPalace(in); got != UnboundPalaceToken {
			t.Fatalf("display %q, want %s", got, UnboundPalaceToken)
		}
	}
	// A real workspace URL still selects when the flag is also set.
	// The flag is not the reason.
	in := SessionPalaceInput{
		WorkspaceMemoryURL:  "https://mem.customer.example/mcp",
		HostedPalaceEnabled: true,
	}
	if got := SelectSessionPalace(in); got != in.WorkspaceMemoryURL {
		t.Fatalf("real workspace URL must still select, got %q", got)
	}
	off := in
	off.HostedPalaceEnabled = false
	if got := SelectSessionPalace(off); got != in.WorkspaceMemoryURL {
		t.Fatalf("flag false must not be required, got %q", got)
	}
}

func TestSharedConfigMemoryURLIsNotWorkspacePalace(t *testing.T) {
	shared := "https://shared.example/mcp"
	cases := []SessionPalaceInput{
		{SharedMemoryURL: shared},
		{WorkspaceMemoryURL: shared, SharedMemoryURL: shared},
		{WorkspaceMemoryURL: "https://shared.example/mcp/", SharedMemoryURL: shared},
		{WorkspaceMemoryURL: "https://shared.example:443/mcp", SharedMemoryURL: shared},
		{WorkspaceMemoryURL: "HTTPS://Shared.Example/mcp", SharedMemoryURL: shared, HostedPalaceEnabled: true},
	}
	for _, in := range cases {
		if got := DisplaySessionPalace(in); got != UnboundPalaceToken {
			t.Fatalf("shared cfg.MemoryURL must not bind: got %q for %+v", got, in)
		}
	}
	in := SessionPalaceInput{
		WorkspaceMemoryURL: "https://mem.customer.example/mcp",
		SharedMemoryURL:    shared,
	}
	if got := SelectSessionPalace(in); got != in.WorkspaceMemoryURL {
		t.Fatalf("distinct workspace URL got %q", got)
	}
}

func TestSyntheticOneLabelInternalIsNotCustomerURL(t *testing.T) {
	for _, raw := range []string{
		"https://aion-mem-acme.internal",
		"https://aion-mem-acme.internal/mcp",
		"http://aion-mem-foo.internal",
		"https://AION-MEM-Bar.internal",
		"https://aion-mem-acme.internal:8443/v1",
		"https://aion-mem-.internal",
		"https://aion-mem-acme.internal.",
	} {
		in := SessionPalaceInput{WorkspaceMemoryURL: raw, HostedPalaceEnabled: true}
		if got := DisplaySessionPalace(in); got != UnboundPalaceToken {
			t.Fatalf("synthetic %q displayed %q", raw, got)
		}
	}
	// Two labels before .internal is not the one-label placeholder.
	raw := "https://aion-mem-acme.prod.internal/mcp"
	in := SessionPalaceInput{WorkspaceMemoryURL: raw}
	if got := SelectSessionPalace(in); got != raw {
		t.Fatalf("two-label host got %q", got)
	}
}

func TestEmptyMemoryURLStaysPalaceDash(t *testing.T) {
	for _, raw := range []string{"", "   ", "mem.customer.example", "://missing-host"} {
		in := SessionPalaceInput{WorkspaceMemoryURL: raw}
		if got := DisplaySessionPalace(in); got != "palace=-" {
			t.Fatalf("empty/unusable %q displayed %q", raw, got)
		}
		if got := SelectSessionPalace(in); got != "" {
			t.Fatalf("select %q for %q", got, raw)
		}
	}
}

func TestCatalogPatchAndEmptyOpenGatesDoNotBind(t *testing.T) {
	cases := []SessionPalaceInput{
		{CatalogRow: true},
		{WorkspacePatched: true},
		{OpenGates: nil},
		{OpenGates: []string{}},
		{OpenGates: []string{"https://mem.customer.example/mcp"}, CatalogRow: true, WorkspacePatched: true},
	}
	for _, in := range cases {
		if got := DisplaySessionPalace(in); got != UnboundPalaceToken {
			t.Fatalf("non-bind signal displayed %q for %+v", got, in)
		}
	}
}

func TestLaptopDefaultPalaceIsNotCloudMemoryBind(t *testing.T) {
	for _, raw := range []string{
		"~/.iomesh/palace",
		"/Users/ego/.iomesh/palace",
		"file:///Users/ego/.iomesh/palace",
		".iomesh/palace",
	} {
		in := SessionPalaceInput{WorkspaceMemoryURL: raw, HostedPalaceEnabled: true}
		if got := DisplaySessionPalace(in); got != UnboundPalaceToken {
			t.Fatalf("laptop default %q displayed %q", raw, got)
		}
	}
}

func TestWorkspaceMemoryURLSelectsSessionPalace(t *testing.T) {
	raw := "https://mem.customer.example/mcp"
	in := SessionPalaceInput{
		WorkspaceMemoryURL:  raw,
		SharedMemoryURL:     "https://cfg.example/mcp",
		HostedPalaceEnabled: false,
		CatalogRow:          true,
		WorkspacePatched:    true,
		OpenGates:           nil,
	}
	if got := SelectSessionPalace(in); got != raw {
		t.Fatalf("select %q", got)
	}
	if got := DisplaySessionPalace(in); got != "palace="+raw {
		t.Fatalf("display %q", got)
	}
	if strings.Contains(UnboundPalaceToken, "https://") || UnboundPalaceToken != "palace=-" {
		t.Fatalf("unbound token %q", UnboundPalaceToken)
	}
}

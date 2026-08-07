package agent

import (
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/skills"
)

// s1347: GtmDraftOnlyAgentGuidanceNote residual-honest needles.
func TestGtmDraftOnlyAgentGuidanceNote_HonestyNeedles(t *testing.T) {
	out := GtmDraftOnlyAgentGuidanceNote()
	if out == "" {
		t.Fatal("empty guidance note")
	}
	for _, want := range []string{
		"drafts only",
		"no auto-send",
		"human publish",
		"human CRM commercial",
		"Salesforce",
		"GA CRM",
		"HubSpot",
		"Beta multi-tenant",
		"guerrilla",
		"global",
		"portal HITL",
		"dual_write OFF",
		"not Memory GA",
		"book-demo OFF",
		"gtm-draft-only-agent",
		"read_skill",
		"never invent",
		"Connected",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("guidance missing %q in:\n%s", want, out)
		}
	}
	// Must not invent auto-send / suite GA product success language.
	if strings.Contains(out, "auto-send ON") || strings.Contains(out, "suite ops GA shipped") {
		t.Fatalf("must not invent auto-send/suite GA claim: %s", out)
	}
	if strings.Contains(out, "Memory GA shipped") || strings.Contains(out, "dual_write ON") {
		t.Fatalf("must not invent Memory GA / dual_write ON: %s", out)
	}
}

// s1347: AttachSkills injects <gtm-draft-only> system note when skills catalog attaches.
// Mirrors TestAttachMCP_InjectsMemoryAdvancedGuidance (s1291).
func TestAttachSkills_InjectsGtmDraftOnlyGuidance(t *testing.T) {
	cat, err := skills.LoadBuiltin()
	if err != nil {
		t.Fatal(err)
	}
	if cat == nil || cat.Len() == 0 {
		t.Fatal("expected builtin skills catalog")
	}
	// Builtin always includes gtm-draft-only-agent when skills load.
	if _, ok := cat.Get("gtm-draft-only-agent"); !ok {
		t.Fatal("want builtin gtm-draft-only-agent skill")
	}

	rt := testRT(t, t.TempDir())
	rt.AttachSkills(cat)

	sys := rt.Messages()[0].Content
	if !strings.Contains(sys, "<gtm-draft-only>") {
		t.Fatalf("want <gtm-draft-only> system note: %s", sys)
	}
	if !strings.Contains(sys, "</gtm-draft-only>") {
		t.Fatalf("want closed gtm-draft-only tag: %s", sys)
	}
	for _, want := range []string{
		"drafts only",
		"no auto-send",
		"human publish",
		"human CRM commercial",
		"dual_write OFF",
		"not Memory GA",
		"book-demo OFF",
		"Salesforce",
		"gtm-draft-only-agent",
	} {
		if !strings.Contains(sys, want) {
			t.Fatalf("gtm-draft-only note missing %q: %s", want, sys)
		}
	}
	// Skills catalog note still present (co-inject with guidance).
	if !strings.Contains(sys, "<skills>") {
		t.Fatalf("want <skills> note too: %s", sys)
	}
}

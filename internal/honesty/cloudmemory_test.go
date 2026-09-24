package honesty

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDigestChrome_GapPartialWithoutConnectedTheater(t *testing.T) {
	s := DigestChrome()
	for _, want := range []string{
		WritePathPin,
		WritePathChip,
		"One write path — not mirrored to a second store.",
		"Local and Cloud Memory stay on separate paths",
		"Cloud Memory GA",
		"optional beside TTFH",
		"TTFH/heartbeat is the SoR",
		"Empty until consume",
		"GAP · B5 host bind · Partial",
		"Gap until QA evidence",
		"Console entitlement is the primary attach",
		"Entitlement ≠ live bind",
		"Do not invent a live host URL",
		"US-CM-JOURNEY-05",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("digest chrome missing %q\n%s", want, s)
		}
	}
	assertNoBannedBindClaims(t, s)
	// Cite-both digest tests reject this substring outright.
	if strings.Contains(s, "Connected") {
		t.Fatalf("digest chrome must not contain Connected:\n%s", s)
	}
}

func TestHostBindGap_GapPartialNotExists(t *testing.T) {
	s := HostBindGap()
	for _, want := range []string{
		"GAP / Partial",
		"US-CM-JOURNEY-05",
		"not an Exists Connected bind",
		WritePathChip,
		"Cloud Memory GA",
		"optional beside TTFH",
		"not required for heartbeat",
		"Catalog ≠ Connected",
		"workspace-as-principal",
		"Multi-human palace read/write stays Gap",
		"B5 · TUI host bind",
		"C4 · SDK palace URL bind",
		"Entitlement ≠ live bind",
		"Do not invent a Connected host URL",
		"No Connected badge",
		"This TUI does not ship that bind",
		"Empty until consume",
		"stop before any host URL",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("host bind gap missing %q\n%s", want, s)
		}
	}
	assertNoBannedBindClaims(t, s)
	if strings.Contains(s, "Exists Connected bind") && !strings.Contains(s, "not an Exists Connected bind") {
		t.Fatalf("must not claim an Exists Connected bind:\n%s", s)
	}
}

func TestShippedDocs_NameTheGap(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, rel := range []string{
		"README.md",
		"CHANGELOG.md",
		"docs/architecture/memory-mcp.md",
		"docs/architecture/edge-user-journey.md",
		"internal/skills/builtin/mesh-agent-onboarding/SKILL.md",
	} {
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		text := string(b)
		if !strings.Contains(text, "US-CM-JOURNEY-05") || !strings.Contains(text, "Gap / Partial") {
			t.Fatalf("%s missing Gap / Partial stamp", rel)
		}
		low := strings.ToLower(text)
		for _, bad := range []string{"soft" + "r", "not memory ga", "ga-path", "path-to-ga"} {
			if strings.Contains(low, bad) {
				t.Fatalf("%s contains banned claim %q", rel, bad)
			}
		}
	}
}

func TestRepo_BannedBrandNounAbsent(t *testing.T) {
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "vendor" || base == "bin" || base == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".go", ".md", ".toml", ".yml", ".yaml", ".json", ".sh", ".txt":
		default:
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(strings.ToLower(string(b)), "soft"+"r") {
			t.Errorf("banned brand noun in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func assertNoBannedBindClaims(t *testing.T, s string) {
	t.Helper()
	low := strings.ToLower(s)
	for _, bad := range []string{
		"soft" + "r",
		"not memory ga",
		"ga-path",
		"path-to-ga",
		"https://",
		"http://",
		"connected: yes",
		"dual_write",
	} {
		if strings.Contains(low, bad) {
			t.Fatalf("banned claim %q in:\n%s", bad, s)
		}
	}
}

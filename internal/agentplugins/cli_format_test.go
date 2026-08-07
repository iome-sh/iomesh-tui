package agentplugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDirFlagValue(t *testing.T) {
	if got := ParseDirFlagValue(""); len(got) != 0 {
		t.Fatalf("empty: %v", got)
	}
	got := ParseDirFlagValue(" /a , /b, ,/c ")
	if len(got) != 3 || got[0] != "/a" || got[1] != "/b" || got[2] != "/c" {
		t.Fatalf("%v", got)
	}
}

func TestMergePluginDirs(t *testing.T) {
	got := MergePluginDirs([]string{"  /cfg ", "", "/cfg2"}, []string{"/cli", "  "})
	if len(got) != 3 || got[0] != "/cfg" || got[1] != "/cfg2" || got[2] != "/cli" {
		t.Fatalf("%v", got)
	}
}

func TestDirFlag_SetRepeatable(t *testing.T) {
	var d DirFlag
	if err := d.Set("/a,/b"); err != nil {
		t.Fatal(err)
	}
	if err := d.Set("/c"); err != nil {
		t.Fatal(err)
	}
	if len(d) != 3 || d[0] != "/a" || d[2] != "/c" {
		t.Fatalf("%v", d)
	}
	if d.String() != "/a,/b,/c" {
		t.Fatal(d.String())
	}
}

func TestFormatListRowAndHeader(t *testing.T) {
	h := FormatListHeader()
	if !strings.Contains(h, "NAME") || !strings.Contains(h, "ROOT") {
		t.Fatal(h)
	}
	row := FormatListRow(PluginListRow{
		Name: "my-plugin", Version: "1.0.0", Skills: 2, MCP: 1, Warn: 0, Root: "/tmp/p",
	})
	if !strings.Contains(row, "my-plugin") || !strings.Contains(row, "1.0.0") || !strings.Contains(row, "/tmp/p") {
		t.Fatal(row)
	}
}

func TestPluginToListRow_NilAndEmptyVersion(t *testing.T) {
	r := PluginToListRow(nil)
	if r.Name != "-" || r.Version != "-" {
		t.Fatalf("%+v", r)
	}
	r = PluginToListRow(&Plugin{Root: "/r", Manifest: PluginManifest{Name: "n"}})
	if r.Version != "-" || r.Name != "n" || r.Root != "/r" {
		t.Fatalf("%+v", r)
	}
}

func TestFormatListEmptyFooter_Honesty(t *testing.T) {
	msg := FormatListEmptyFooter(false, false)
	if !strings.Contains(msg, "opt-in") || !strings.Contains(msg, "Agent Plugins GA") {
		t.Fatal(msg)
	}
	msg = FormatListEmptyFooter(true, false)
	if !strings.Contains(msg, "dirs empty") {
		t.Fatal(msg)
	}
	msg = FormatListEmptyFooter(true, true)
	if !strings.Contains(msg, "no plugins discovered") {
		t.Fatal(msg)
	}
	if !strings.Contains(ResidualCLIHonesty, "dual_write OFF") {
		t.Fatal(ResidualCLIHonesty)
	}
}

func TestFormatValidateOKFail(t *testing.T) {
	ok := FormatValidateOK(ValidateOutcome{
		Path: "/p", Name: "n", Version: "1", Skills: 1, MCP: 0, Warnings: []string{"w"},
	})
	if !strings.HasPrefix(ok, "OK") || !strings.Contains(ok, "skills=1") || !strings.Contains(ok, "warnings=1") {
		t.Fatal(ok)
	}
	fail := FormatValidateFail("/bad", "invalid name")
	if !strings.HasPrefix(fail, "FAIL") || !strings.Contains(fail, "invalid name") {
		t.Fatal(fail)
	}
}

func TestValidateDirs_OKAndFail(t *testing.T) {
	base := t.TempDir()
	good := filepath.Join(base, "good")
	if err := os.MkdirAll(good, 0o755); err != nil {
		t.Fatal(err)
	}
	writePlugin(t, good, "good-plug")
	sk := filepath.Join(good, "skills", "s1")
	if err := os.MkdirAll(sk, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sk, "SKILL.md"), []byte("# s"), 0o644); err != nil {
		t.Fatal(err)
	}

	bad := filepath.Join(base, "bad")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	// Invalid name in plugin.json → fatal Discover.
	if err := os.WriteFile(filepath.Join(bad, "plugin.json"), []byte(`{"name":"BAD NAME"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(base, "missing-nope")

	outcomes, _ := ValidateDirs([]string{good, bad, missing})
	if ValidateOKCount(outcomes) != 1 {
		t.Fatalf("ok count: %+v", outcomes)
	}
	if !ValidateHasFatal(outcomes) {
		t.Fatal("expected fatals")
	}
	var sawOK, sawFailBad, sawFailMissing bool
	for _, o := range outcomes {
		if o.OK {
			sawOK = true
			if o.Name != "good-plug" || o.Skills != 1 {
				t.Fatalf("ok outcome: %+v", o)
			}
			line := FormatValidateOK(o)
			if !strings.Contains(line, "good-plug") {
				t.Fatal(line)
			}
		} else {
			if strings.Contains(o.Path, "bad") || strings.Contains(o.Error, "name") {
				sawFailBad = true
			}
			if strings.Contains(o.Path, "missing") || strings.Contains(o.Error, "no such") || strings.Contains(o.Error, "not exist") {
				sawFailMissing = true
			}
		}
	}
	if !sawOK || !sawFailBad || !sawFailMissing {
		t.Fatalf("sawOK=%v sawFailBad=%v sawFailMissing=%v outcomes=%+v", sawOK, sawFailBad, sawFailMissing, outcomes)
	}
}

func TestValidateDirs_ParentScan(t *testing.T) {
	parent := t.TempDir()
	a := filepath.Join(parent, "a")
	b := filepath.Join(parent, "b")
	if err := os.MkdirAll(a, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(b, 0o755); err != nil {
		t.Fatal(err)
	}
	writePlugin(t, a, "alpha")
	writePlugin(t, b, "beta")

	outcomes, _ := ValidateDirs([]string{parent})
	if ValidateOKCount(outcomes) != 2 {
		t.Fatalf("%+v", outcomes)
	}
	if ValidateHasFatal(outcomes) {
		t.Fatalf("unexpected fatal: %+v", outcomes)
	}
}

func TestValidateDirs_EmptyParent(t *testing.T) {
	empty := t.TempDir()
	outcomes, _ := ValidateDirs([]string{empty})
	if len(outcomes) != 1 || outcomes[0].OK {
		t.Fatalf("%+v", outcomes)
	}
	if !strings.Contains(outcomes[0].Error, "no plugin.json") {
		t.Fatal(outcomes[0].Error)
	}
}

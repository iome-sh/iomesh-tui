package tui

import (
	"strings"
	"testing"
)

func TestParseTheme(t *testing.T) {
	for _, name := range ThemeNames() {
		th, err := ParseTheme(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if th.Name == "" {
			t.Fatal(name)
		}
	}
	th, err := ParseTheme("")
	if err != nil || th.Name != "default" {
		t.Fatalf("%+v %v", th, err)
	}
	_, err = ParseTheme("neon-xyz")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "default") {
		t.Fatal(err)
	}
}

func TestThemeAliases(t *testing.T) {
	th, err := ParseTheme("hc")
	if err != nil || th.Name != "high-contrast" {
		t.Fatalf("%+v %v", th, err)
	}
	th, err = ParseTheme("bw")
	if err != nil || th.Name != "mono" {
		t.Fatalf("%+v %v", th, err)
	}
}

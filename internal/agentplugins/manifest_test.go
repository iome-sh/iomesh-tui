package agentplugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePluginName(t *testing.T) {
	valid := []string{"my-plugin", "acme.tools", "lint3r", "a", "x1", "a.b-c1"}
	for _, n := range valid {
		if err := ValidatePluginName(n); err != nil {
			t.Fatalf("valid %q: %v", n, err)
		}
	}
	invalid := []string{
		"",
		"My-Plugin",
		"-start",
		"end-",
		"has--double",
		"too.many..dots",
		".dotstart",
		"end.",
		"has space",
		"under_score",
		strings.Repeat("a", 65),
	}
	for _, n := range invalid {
		if err := ValidatePluginName(n); err == nil {
			t.Fatalf("expected invalid for %q", n)
		}
	}
}

func TestValidatePluginJSON_Minimal(t *testing.T) {
	raw := []byte(`{
		"$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json",
		"name": "minimal-plugin"
	}`)
	res, err := ValidatePluginJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if res.Manifest.Name != "minimal-plugin" {
		t.Fatal(res.Manifest.Name)
	}
	if res.Manifest.Schema != PluginSchemaID {
		t.Fatal(res.Manifest.Schema)
	}
}

func TestValidatePluginJSON_Full(t *testing.T) {
	raw := []byte(`{
		"$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json",
		"name": "plugin-name",
		"version": "1.2.0",
		"description": "Brief",
		"author": {"name": "A", "email": "a@b.c", "url": "https://ex.com"},
		"homepage": "https://docs.example.com",
		"repository": "https://github.com/ex/p",
		"license": "MIT",
		"keywords": ["k1", "k2"],
		"extensions": {"com.example.client": {"setting": true}}
	}`)
	res, err := ValidatePluginJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	m := res.Manifest
	if m.Version != "1.2.0" || m.Description != "Brief" || m.License != "MIT" {
		t.Fatalf("%+v", m)
	}
	if m.Author == nil || m.Author.Name != "A" {
		t.Fatalf("author=%+v", m.Author)
	}
	if len(m.Keywords) != 2 || m.Extensions["com.example.client"] == nil {
		t.Fatalf("%+v", m)
	}
}

func TestValidatePluginJSON_UnknownKeyWarn(t *testing.T) {
	raw := []byte(`{
		"$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json",
		"name": "ok-plugin",
		"skills": ["nope"],
		"extra": 1
	}`)
	res, err := ValidatePluginJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) < 2 {
		t.Fatalf("want unknown key warnings, got %v", res.Warnings)
	}
}

func TestValidatePluginJSON_FatalCases(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"bad json", `{`},
		{"missing name", `{"$schema":"https://agent-plugins.org/schemas/1.0.0/plugin.schema.json"}`},
		{"bad name", `{"name":"Bad_Name"}`},
		{"wrong schema", `{"$schema":"https://example.com/other.json","name":"ok"}`},
		{"name not string", `{"name":1}`},
		{"author unknown field", `{"name":"ok","author":{"twitter":"x"}}`},
	}
	for _, c := range cases {
		if _, err := ValidatePluginJSON([]byte(c.raw)); err == nil {
			t.Fatalf("%s: expected error", c.name)
		}
	}
}

func TestValidatePluginJSON_NonObjectExtensionsIgnored(t *testing.T) {
	raw := []byte(`{"name":"ok-plugin","extensions":["not","object"]}`)
	res, err := ValidatePluginJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if res.Manifest.Extensions != nil {
		t.Fatalf("extensions should be ignored: %v", res.Manifest.Extensions)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "extensions") {
			found = true
		}
	}
	if !found {
		t.Fatalf("want extensions warning: %v", res.Warnings)
	}
}

func TestLoadManifest(t *testing.T) {
	root := t.TempDir()
	content := `{"$schema":"https://agent-plugins.org/schemas/1.0.0/plugin.schema.json","name":"loaded"}`
	if err := os.WriteFile(filepath.Join(root, "plugin.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if res.Manifest.Name != "loaded" {
		t.Fatal(res.Manifest.Name)
	}
	// Missing plugin.json is fatal.
	if _, err := LoadManifest(t.TempDir()); err == nil {
		t.Fatal("expected missing plugin.json error")
	}
}

func TestValidatePluginJSON_MissingSchemaOK(t *testing.T) {
	// Spec requires $schema; this client treats missing as non-fatal per residual slice
	// (when present must be exact). Name alone is enough to load.
	raw := []byte(`{"name":"schema-optional"}`)
	res, err := ValidatePluginJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if res.Manifest.Name != "schema-optional" {
		t.Fatal(res.Manifest)
	}
}

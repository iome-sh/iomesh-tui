package agentplugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMCPJSON_Full(t *testing.T) {
	root := t.TempDir()
	// Create bundled binary path for confinement.
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{
		"$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json",
		"mcpServers": {
			"local-validator": {
				"type": "stdio",
				"command": "./bin/validator",
				"args": ["--data", "${PLUGIN_DATA}/validator"],
				"env": {"CONFIG": "${PLUGIN_ROOT}/config.json"},
				"cwd": "${PLUGIN_ROOT}"
			},
			"deployment-api": {
				"type": "streamable-http",
				"url": "https://deploy.example.com/mcp",
				"headers": {"X-Tenant": "public-tenant"}
			},
			"legacy-events": {
				"type": "sse",
				"url": "https://legacy.example.com/sse"
			},
			"bare-cmd": {
				"type": "stdio",
				"command": "npx",
				"args": ["-y", "pkg"]
			}
		}
	}`)
	res, err := ParseMCPJSON(raw, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Servers) != 4 {
		t.Fatalf("servers=%d warnings=%v", len(res.Servers), res.Warnings)
	}
	byName := map[string]MCPServerRef{}
	for _, s := range res.Servers {
		byName[s.Name] = s
	}
	if byName["local-validator"].Command != "./bin/validator" {
		t.Fatal(byName["local-validator"])
	}
	if byName["deployment-api"].Type != TransportStreamableHTTP {
		t.Fatal(byName["deployment-api"])
	}
	if byName["legacy-events"].Type != TransportSSE {
		t.Fatal(byName["legacy-events"])
	}
}

func TestParseMCPJSON_SkipInvalidEntries(t *testing.T) {
	root := t.TempDir()
	raw := []byte(`{
		"$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json",
		"mcpServers": {
			"good": {"type": "stdio", "command": "npx"},
			"bad-type": {"type": "websocket", "url": "wss://x"},
			"no-cmd": {"type": "stdio"},
			"secret-env": {"type": "stdio", "command": "x", "env": {"PLUGIN_ROOT": "/evil"}},
			"bad-cwd": {"type": "stdio", "command": "x", "cwd": "data"},
			"escape-cmd": {"type": "stdio", "command": "../bin/x"},
			"http-no-https": {"type": "streamable-http", "url": "http://example.com/mcp"},
			"http-localhost": {"type": "streamable-http", "url": "http://localhost:8080/mcp"},
			"userinfo": {"type": "streamable-http", "url": "https://u:p@example.com/mcp"},
			"placeholder-cmd": {"type": "stdio", "command": "${PLUGIN_ROOT}/bin/x"}
		}
	}`)
	res, err := ParseMCPJSON(raw, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Servers) != 2 {
		t.Fatalf("want good+http-localhost, got %d: %+v warnings=%v", len(res.Servers), res.Servers, res.Warnings)
	}
	names := map[string]bool{}
	for _, s := range res.Servers {
		names[s.Name] = true
	}
	if !names["good"] || !names["http-localhost"] {
		t.Fatalf("%v", names)
	}
	if len(res.Warnings) < 5 {
		t.Fatalf("want skip warnings, got %v", res.Warnings)
	}
}

func TestParseMCPJSON_TopLevelDisable(t *testing.T) {
	root := t.TempDir()
	// Invalid JSON
	res, err := ParseMCPJSON([]byte(`{`), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Servers) != 0 || len(res.Warnings) == 0 {
		t.Fatal(res)
	}
	// Unknown top-level key
	res, err = ParseMCPJSON([]byte(`{
		"$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json",
		"mcpServers": {},
		"extra": true
	}`), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Servers) != 0 {
		t.Fatal(res.Servers)
	}
	// Wrong schema
	res, err = ParseMCPJSON([]byte(`{
		"$schema": "https://agent-plugins.org/schemas/9.9.9/mcp.schema.json",
		"mcpServers": {"a": {"type":"stdio","command":"npx"}}
	}`), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Servers) != 0 {
		t.Fatal("wrong schema should disable MCP")
	}
}

func TestLoadMCPJSON_MissingOK(t *testing.T) {
	root := t.TempDir()
	res, err := LoadMCPJSON(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Servers) != 0 {
		t.Fatal(res.Servers)
	}
}

func TestLoadMCPJSON_Present(t *testing.T) {
	root := t.TempDir()
	body := `{
		"$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json",
		"mcpServers": {
			"s": {"type": "stdio", "command": "npx"}
		}
	}`
	if err := os.WriteFile(filepath.Join(root, "mcp.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadMCPJSON(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Servers) != 1 || res.Servers[0].Name != "s" {
		t.Fatal(res.Servers)
	}
}

func TestValidateMCPURL(t *testing.T) {
	if err := validateMCPURL("https://example.com/mcp"); err != nil {
		t.Fatal(err)
	}
	if err := validateMCPURL("http://127.0.0.1:9/mcp"); err != nil {
		t.Fatal(err)
	}
	if err := validateMCPURL("http://example.com/mcp"); err == nil {
		t.Fatal("non-loopback http")
	}
	if err := validateMCPURL("https://user:pass@example.com/mcp"); err == nil {
		t.Fatal("userinfo")
	}
	if err := validateMCPURL("https://example.com/mcp#frag"); err == nil {
		t.Fatal("fragment")
	}
}

func TestExpandMCPServer_NoCommandExpand(t *testing.T) {
	ref := MCPServerRef{
		Name:    "s",
		Type:    TransportStdio,
		Command: "./bin/x",
		Args:    []string{"${PLUGIN_ROOT}/a"},
		Env:     map[string]string{"D": "${PLUGIN_DATA}/d"},
		Cwd:     "${PLUGIN_ROOT}",
	}
	out := ExpandMCPServer(ref, "/p", "/d")
	if out.Command != "./bin/x" {
		t.Fatal("command must not expand")
	}
	if out.Args[0] != "/p/a" || out.Env["D"] != "/d/d" || out.Cwd != "/p" {
		t.Fatalf("%+v", out)
	}
	// Secrets honesty: headers/env from package are opaque package data.
	if strings.Contains(out.Command, "${") {
		t.Fatal("unexpected")
	}
}

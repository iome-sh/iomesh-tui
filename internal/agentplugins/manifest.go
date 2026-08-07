package agentplugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"
)

// Canonical schema identifiers for Agent Plugins 1.0.0.
const (
	PluginSchemaID = "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json"
	MCPSchemaID    = "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json"
)

// Closed top-level keys for plugin.json (unknown keys are warned + ignored).
var pluginAllowedKeys = map[string]struct{}{
	"$schema":     {},
	"name":        {},
	"version":     {},
	"description": {},
	"author":      {},
	"homepage":    {},
	"repository":  {},
	"license":     {},
	"keywords":    {},
	"extensions":  {},
}

// Author is optional plugin.json author metadata.
type Author struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

// PluginManifest is the closed plugin.json object after validation.
type PluginManifest struct {
	Schema      string                     `json:"$schema,omitempty"`
	Name        string                     `json:"name"`
	Version     string                     `json:"version,omitempty"`
	Description string                     `json:"description,omitempty"`
	Author      *Author                    `json:"author,omitempty"`
	Homepage    string                     `json:"homepage,omitempty"`
	Repository  string                     `json:"repository,omitempty"`
	License     string                     `json:"license,omitempty"`
	Keywords    []string                   `json:"keywords,omitempty"`
	Extensions  map[string]json.RawMessage `json:"extensions,omitempty"`
}

// ManifestResult is a validated manifest plus non-fatal warnings.
type ManifestResult struct {
	Manifest PluginManifest
	Warnings []string
}

// LoadManifest reads and validates plugin.json at pluginRoot/plugin.json.
// Fatal only on missing/unreadable file, invalid JSON, missing/bad name, or
// unsupported $schema when present. Unknown top-level keys are warned + ignored.
func LoadManifest(pluginRoot string) (*ManifestResult, error) {
	absRoot, err := resolveRoot(pluginRoot)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(absRoot, "plugin.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("agentplugins: read plugin.json: %w", err)
	}
	return ValidatePluginJSON(data)
}

// ValidatePluginJSON validates raw plugin.json bytes (closed keys, name rules).
func ValidatePluginJSON(data []byte) (*ManifestResult, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("agentplugins: plugin.json: %w", err)
	}

	var warnings []string
	for k := range raw {
		if _, ok := pluginAllowedKeys[k]; !ok {
			warnings = append(warnings, fmt.Sprintf("unknown top-level key %q ignored", k))
			delete(raw, k)
		}
	}

	// $schema when present must be the exact 1.0.0 identifier.
	if rawSchema, ok := raw["$schema"]; ok {
		var schema string
		if err := json.Unmarshal(rawSchema, &schema); err != nil {
			return nil, fmt.Errorf("agentplugins: plugin.json: $schema must be a string")
		}
		if schema != PluginSchemaID {
			return nil, fmt.Errorf("agentplugins: unsupported $schema %q (want %s)", schema, PluginSchemaID)
		}
	}

	// name is required.
	rawName, ok := raw["name"]
	if !ok {
		return nil, fmt.Errorf("agentplugins: plugin.json: missing required field name")
	}
	var name string
	if err := json.Unmarshal(rawName, &name); err != nil {
		return nil, fmt.Errorf("agentplugins: plugin.json: name must be a string")
	}
	if err := ValidatePluginName(name); err != nil {
		return nil, err
	}

	m := PluginManifest{Name: name}
	if rawSchema, ok := raw["$schema"]; ok {
		_ = json.Unmarshal(rawSchema, &m.Schema)
	}

	if v, ok := raw["version"]; ok {
		if err := json.Unmarshal(v, &m.Version); err != nil {
			return nil, fmt.Errorf("agentplugins: plugin.json: version must be a string")
		}
	}
	if v, ok := raw["description"]; ok {
		if err := json.Unmarshal(v, &m.Description); err != nil {
			return nil, fmt.Errorf("agentplugins: plugin.json: description must be a string")
		}
	}
	if v, ok := raw["homepage"]; ok {
		if err := json.Unmarshal(v, &m.Homepage); err != nil {
			return nil, fmt.Errorf("agentplugins: plugin.json: homepage must be a string")
		}
	}
	if v, ok := raw["repository"]; ok {
		if err := json.Unmarshal(v, &m.Repository); err != nil {
			return nil, fmt.Errorf("agentplugins: plugin.json: repository must be a string")
		}
	}
	if v, ok := raw["license"]; ok {
		if err := json.Unmarshal(v, &m.License); err != nil {
			return nil, fmt.Errorf("agentplugins: plugin.json: license must be a string")
		}
	}
	if v, ok := raw["keywords"]; ok {
		if err := json.Unmarshal(v, &m.Keywords); err != nil {
			return nil, fmt.Errorf("agentplugins: plugin.json: keywords must be an array of strings")
		}
	}
	if v, ok := raw["author"]; ok {
		author, warn, err := parseAuthor(v)
		if err != nil {
			return nil, err
		}
		if warn != "" {
			warnings = append(warnings, warn)
		} else {
			m.Author = author
		}
	}
	if v, ok := raw["extensions"]; ok {
		ext, warn, err := parseExtensions(v)
		if err != nil {
			return nil, err
		}
		if warn != "" {
			warnings = append(warnings, warn)
		} else {
			m.Extensions = ext
		}
	}

	return &ManifestResult{Manifest: m, Warnings: warnings}, nil
}

// ValidatePluginName enforces Agent Plugins 1.0.0 name constraints:
// 1–64 runes, [a-z0-9.-], start/end alphanumeric, no "--" or "..".
func ValidatePluginName(name string) error {
	if name == "" {
		return fmt.Errorf("agentplugins: plugin name is empty")
	}
	n := utf8.RuneCountInString(name)
	if n < 1 || n > 64 {
		return fmt.Errorf("agentplugins: plugin name length %d not in 1–64", n)
	}
	// Single-char: must be alnum.
	if n == 1 {
		c := name[0]
		if !isAlnumByte(c) {
			return fmt.Errorf("agentplugins: invalid plugin name %q (need [a-z0-9.-], start/end alnum, no -- or ..)", name)
		}
		return nil
	}
	// Start/end alphanumeric.
	if !isAlnumByte(name[0]) || !isAlnumByte(name[len(name)-1]) {
		return fmt.Errorf("agentplugins: invalid plugin name %q (need [a-z0-9.-], start/end alnum, no -- or ..)", name)
	}
	// Character set + no consecutive -- or ..
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !(isAlnumByte(c) || c == '-' || c == '.') {
			return fmt.Errorf("agentplugins: invalid plugin name %q (need [a-z0-9.-], start/end alnum, no -- or ..)", name)
		}
		if i+1 < len(name) {
			if c == '-' && name[i+1] == '-' {
				return fmt.Errorf("agentplugins: invalid plugin name %q (need [a-z0-9.-], start/end alnum, no -- or ..)", name)
			}
			if c == '.' && name[i+1] == '.' {
				return fmt.Errorf("agentplugins: invalid plugin name %q (need [a-z0-9.-], start/end alnum, no -- or ..)", name)
			}
		}
	}
	return nil
}

func isAlnumByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

func parseAuthor(raw json.RawMessage) (*Author, string, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, "", fmt.Errorf("agentplugins: plugin.json: author must be an object")
	}
	allowed := map[string]struct{}{"name": {}, "email": {}, "url": {}}
	for k := range obj {
		if _, ok := allowed[k]; !ok {
			return nil, "", fmt.Errorf("agentplugins: plugin.json: author has unknown field %q", k)
		}
	}
	a := &Author{}
	if v, ok := obj["name"]; ok {
		if err := json.Unmarshal(v, &a.Name); err != nil {
			return nil, "", fmt.Errorf("agentplugins: plugin.json: author.name must be a string")
		}
	}
	if v, ok := obj["email"]; ok {
		if err := json.Unmarshal(v, &a.Email); err != nil {
			return nil, "", fmt.Errorf("agentplugins: plugin.json: author.email must be a string")
		}
	}
	if v, ok := obj["url"]; ok {
		if err := json.Unmarshal(v, &a.URL); err != nil {
			return nil, "", fmt.Errorf("agentplugins: plugin.json: author.url must be a string")
		}
	}
	return a, "", nil
}

func parseExtensions(raw json.RawMessage) (map[string]json.RawMessage, string, error) {
	// Non-object extensions: report and ignore (non-fatal per §8.1).
	var probe any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, "extensions is not valid JSON; ignored", nil
	}
	obj, ok := probe.(map[string]any)
	if !ok {
		return nil, "extensions is not an object; ignored", nil
	}
	out := make(map[string]json.RawMessage, len(obj))
	for k, v := range obj {
		// Namespace values should be objects; ignore non-objects without failing the plugin.
		switch v.(type) {
		case map[string]any:
			b, err := json.Marshal(v)
			if err != nil {
				continue
			}
			out[k] = b
		default:
			// Keep raw only if object; non-object namespace values are ignored by clients.
			// Spec: ignore unimplemented namespaces without validating contents.
			// We still store objects; skip non-objects.
		}
	}
	// Preserve original raw for object values more faithfully.
	var typed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &typed); err == nil {
		clean := make(map[string]json.RawMessage, len(typed))
		for k, v := range typed {
			var m map[string]json.RawMessage
			if json.Unmarshal(v, &m) == nil {
				clean[k] = v
			}
		}
		return clean, "", nil
	}
	return out, "", nil
}

// PluginNameOK is a convenience bool wrapper around ValidatePluginName.
func PluginNameOK(name string) bool {
	return ValidatePluginName(name) == nil
}

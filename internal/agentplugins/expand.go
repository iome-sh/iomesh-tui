package agentplugins

import "strings"

// ExpandPlaceholders performs a single, non-recursive textual replacement of
// ${PLUGIN_ROOT} and ${PLUGIN_DATA} in s. Unrecognized placeholder-like text
// remains literal. Expansion applies to args/env values/cwd only — never command.
func ExpandPlaceholders(s, pluginRoot, pluginData string) string {
	if s == "" || (!strings.Contains(s, "${PLUGIN_ROOT}") && !strings.Contains(s, "${PLUGIN_DATA}")) {
		return s
	}
	// Non-recursive: replace exact placeholders once; do not rescan introduced text.
	// Order is defined but replacements cannot introduce new placeholders from paths.
	var b strings.Builder
	b.Grow(len(s) + len(pluginRoot) + len(pluginData))
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], "${PLUGIN_ROOT}") {
			b.WriteString(pluginRoot)
			i += len("${PLUGIN_ROOT}")
			continue
		}
		if strings.HasPrefix(s[i:], "${PLUGIN_DATA}") {
			b.WriteString(pluginData)
			i += len("${PLUGIN_DATA}")
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// ExpandStringSlice expands each element with ExpandPlaceholders.
func ExpandStringSlice(in []string, pluginRoot, pluginData string) []string {
	if len(in) == 0 {
		return in
	}
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = ExpandPlaceholders(v, pluginRoot, pluginData)
	}
	return out
}

// ExpandEnvMap expands each value with ExpandPlaceholders (keys unchanged).
func ExpandEnvMap(in map[string]string, pluginRoot, pluginData string) map[string]string {
	if len(in) == 0 {
		return in
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = ExpandPlaceholders(v, pluginRoot, pluginData)
	}
	return out
}

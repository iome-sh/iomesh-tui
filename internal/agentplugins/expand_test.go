package agentplugins

import "testing"

func TestExpandPlaceholders(t *testing.T) {
	root := "/plugins/demo"
	data := "/data/demo"
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"plain", "plain"},
		{"${PLUGIN_ROOT}", root},
		{"${PLUGIN_DATA}", data},
		{"${PLUGIN_ROOT}/bin", root + "/bin"},
		{"${PLUGIN_DATA}/cache", data + "/cache"},
		{"x=${PLUGIN_ROOT};y=${PLUGIN_DATA}", "x=" + root + ";y=" + data},
		// Unrecognized stays literal.
		{"${OTHER}", "${OTHER}"},
		{"$PLUGIN_ROOT", "$PLUGIN_ROOT"},
		// Non-recursive: if root itself contained a placeholder string, no rescan —
		// our impl replaces exact tokens only once per occurrence in input.
		{"${PLUGIN_ROOT}${PLUGIN_ROOT}", root + root},
	}
	for _, c := range cases {
		got := ExpandPlaceholders(c.in, root, data)
		if got != c.want {
			t.Fatalf("ExpandPlaceholders(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestExpandStringSliceAndEnv(t *testing.T) {
	args := ExpandStringSlice([]string{"--cfg", "${PLUGIN_ROOT}/c.json"}, "/r", "/d")
	if args[1] != "/r/c.json" {
		t.Fatal(args)
	}
	env := ExpandEnvMap(map[string]string{"DATA": "${PLUGIN_DATA}/x"}, "/r", "/d")
	if env["DATA"] != "/d/x" {
		t.Fatal(env)
	}
	if ExpandStringSlice(nil, "/r", "/d") != nil {
		t.Fatal("nil slice")
	}
	if ExpandEnvMap(nil, "/r", "/d") != nil {
		t.Fatal("nil map")
	}
}

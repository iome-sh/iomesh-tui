package router

import (
	"os"
	"strings"
)

// getenv is indirection for tests.
var getenv = os.Getenv

// expandEnvPlaceholders replaces ${VAR} and $VAR in s using the process env.
// Unset variables expand to empty strings.
func expandEnvPlaceholders(s string) string {
	if s == "" || !strings.ContainsAny(s, "$") {
		return s
	}
	return os.Expand(s, getenv)
}

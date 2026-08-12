package agentplugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResidualDogfoodHonesty is the residual-honest footer for `iomesh plugins dogfood` (s1357).
// dogfood PASS ≠ invent Agent Plugins GA · dual_write OFF · Discover ≠ Connected ·
// not Memory GA · PATH residual for binary · book-demo OFF.
const ResidualDogfoodHonesty = "honesty: dogfood PASS ≠ invent Agent Plugins GA · dual_write OFF · Discover ≠ Connected · not Memory GA · PATH residual for binary · book-demo OFF"

// SamplePluginRelPaths returns the in-repo sample package paths relative to module root (s1357+s1478).
// Order is stable: skills-only hello-iome, then product stdio-map iomesh-memory-mcp.
// PATH residual: iomesh-memory-mcp binary is not required for discover/validate dogfood.
func SamplePluginRelPaths() []string {
	return []string{
		filepath.Join("examples", "agent-plugins", "hello-iome"),
		filepath.Join("examples", "agent-plugins", "iomesh-memory-mcp"),
	}
}

// DefaultSamplePluginDirs joins moduleRoot with SamplePluginRelPaths.
// moduleRoot may be empty (returns relative paths only).
func DefaultSamplePluginDirs(moduleRoot string) []string {
	rels := SamplePluginRelPaths()
	out := make([]string, 0, len(rels))
	root := strings.TrimSpace(moduleRoot)
	for _, rel := range rels {
		if root == "" {
			out = append(out, rel)
			continue
		}
		out = append(out, filepath.Join(root, rel))
	}
	return out
}

// FindModuleRoot walks up from start looking for a directory containing go.mod.
// When start is empty, uses the process working directory.
// Residual honesty: module root discovery is for offline sample paths only —
// not install Connected / marketplace / Agent Plugins GA.
func FindModuleRoot(start string) (string, error) {
	dir := strings.TrimSpace(start)
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("agentplugins: find module root: getwd: %w", err)
		}
		dir = cwd
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("agentplugins: find module root: abs: %w", err)
	}
	dir = abs
	for {
		gomod := filepath.Join(dir, "go.mod")
		if st, err := os.Stat(gomod); err == nil && !st.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("agentplugins: find module root: go.mod not found above %s", abs)
		}
		dir = parent
	}
}

// DogfoodSamples validates both in-repo product sample packages offline (s1357+s1478).
// Discover/validate only — does not Dial MCP, spawn processes, or require
// iomesh-memory-mcp on PATH (PATH residual; connect skip).
//
// When moduleRoot is empty, FindModuleRoot("") is used (cwd walk-up).
// Returns ValidateDirs outcomes for each sample path; missing roots surface as FAIL.
// err is only for module-root resolution failure (caller should exit non-zero).
func DogfoodSamples(moduleRoot string) (outcomes []ValidateOutcome, warnings []string, err error) {
	root := strings.TrimSpace(moduleRoot)
	if root == "" {
		root, err = FindModuleRoot("")
		if err != nil {
			return nil, nil, err
		}
	}
	dirs := DefaultSamplePluginDirs(root)
	// Pre-check: surface missing sample trees as explicit FAIL before ValidateDirs
	// so operators see which sample path is absent (offline residual honesty).
	var present []string
	for _, d := range dirs {
		st, statErr := os.Stat(d)
		if statErr != nil || !st.IsDir() {
			msg := "sample package missing"
			if statErr != nil {
				msg = statErr.Error()
			} else {
				msg = "not a directory"
			}
			outcomes = append(outcomes, ValidateOutcome{
				Path:  d,
				OK:    false,
				Error: msg,
			})
			warnings = append(warnings, fmt.Sprintf("dogfood: missing sample %s", d))
			continue
		}
		present = append(present, d)
	}
	if len(present) > 0 {
		okOutcomes, scanWarns := ValidateDirs(present)
		outcomes = append(outcomes, okOutcomes...)
		warnings = append(warnings, scanWarns...)
	}
	// PATH residual note is a warning only — never a fatal for dogfood discover.
	warnings = append(warnings,
		"dogfood: PATH residual — iomesh-memory-mcp binary not required for discover/validate (connect skip)")
	return outcomes, warnings, nil
}

// DogfoodPass reports whether dogfood is residual-honest PASS: both product samples OK,
// no fatals, expected sample count present.
func DogfoodPass(outcomes []ValidateOutcome) bool {
	if ValidateHasFatal(outcomes) {
		return false
	}
	if ValidateOKCount(outcomes) != len(SamplePluginRelPaths()) {
		return false
	}
	// Ensure expected names when OK (stable product sample pins · s1478).
	want := map[string]bool{"hello-iome": false, "iomesh-memory-mcp": false}
	for _, o := range outcomes {
		if o.OK {
			if _, ok := want[o.Name]; ok {
				want[o.Name] = true
			}
		}
	}
	for _, ok := range want {
		if !ok {
			return false
		}
	}
	return true
}

// FormatDogfoodSummary returns a one-line dogfood summary for stdout.
func FormatDogfoodSummary(outcomes []ValidateOutcome) string {
	ok := ValidateOKCount(outcomes)
	total := len(outcomes)
	want := len(SamplePluginRelPaths())
	status := "PASS"
	if !DogfoodPass(outcomes) {
		status = "FAIL"
	}
	return fmt.Sprintf("dogfood %s: ok=%d/%d want=%d (discover/validate only · no MCP dial · PATH residual)",
		status, ok, total, want)
}

// SamplesSoftState soft-checks in-repo sample package dirs (s1382/s1392/s1478).
// Returns "samples_ok" when both hello-iome + iomesh-memory-mcp dirs exist under
// moduleRoot; "samples_missing" otherwise (including when module root cannot
// be resolved). When moduleRoot is empty, FindModuleRoot("") is used.
// Soft path check only — ≠ dogfood run · ≠ invent Agent Plugins GA · ≠ Connected · ≠ live dogfood.
func SamplesSoftState(moduleRoot string) string {
	root := strings.TrimSpace(moduleRoot)
	if root == "" {
		found, err := FindModuleRoot("")
		if err != nil {
			return "samples_missing"
		}
		root = found
	}
	for _, d := range DefaultSamplePluginDirs(root) {
		st, statErr := os.Stat(d)
		if statErr != nil || !st.IsDir() {
			return "samples_missing"
		}
	}
	return "samples_ok"
}

// ResidualSlashHonesty is the residual-honest footer for TUI /plugins slash (s1392).
// Soft offline dogfood ≠ invent Agent Plugins GA · dual_write OFF · Discover ≠ Connected ·
// not Memory GA · residual PASS ≠ live dogfood · package load ≠ Memory GA · book-demo OFF.
const ResidualSlashHonesty = "honesty: soft offline dogfood ≠ invent Agent Plugins GA · dual_write OFF · Discover ≠ Connected · not Memory GA · residual PASS ≠ live dogfood · package load ≠ Memory GA · book-demo OFF · never invent install green / Connected / INSTALL_STORE APPLY"

// PluginsNextStepLines residual-honest post /plugins list|validate|smoke|status (s1829).
// Dual path after discover/validate/smoke: in-session setup continuum vs cold start.
// Peer of IntegrationsNextStepLines (s1727) · OnboardNextStepLines (s1825).
// dual_write OFF · Discover ≠ Connected · soft offline smoke ≠ invent Agent Plugins GA ·
// package load ≠ Memory GA · free eng s1829.
func PluginsNextStepLines() []string {
	return []string{
		"next: dual path residual-honest after plugins discover/validate/smoke",
		"      if TUI/session running → /setup preflight · /setup reload (skills/MCP re-scan · package wire ≠ Connected) · optional /onboard next plugins|status",
		"      else cold start → restart iomesh · iomesh setup preflight · optional iomesh plugins smoke",
		"note: Discover ≠ Connected · soft offline smoke ≠ invent Agent Plugins GA · package load ≠ Memory GA · dual_write OFF · free eng s1829",
	}
}

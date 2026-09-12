package agent

import (
	"os"
	"strings"
)

// Mode A TTFH polish (#399) — Collision demo chrome only.
// Cite-both (#373) and brief ACK (#371) already exist; this package names
// the sticky command, prints the palace path, and keeps air-gap / pin honesty.
// dual_write OFF · not Memory GA · catalog ≠ Connected · do not invent GA.

const (
	// Published tip class (do not invent a newer GA tag).
	// Trio must stay compatible: TUI v1.3.6 extract HITL needs MCP v0.4.1
	// (persist companion + memory_extract_facts) over kernel v1.5.11. persist-onnx-vec is opt-in default OFF.
	ModeATUIPin    = "v1.3.6"
	ModeAMCPPin    = "v0.4.1"
	ModeAMemoryPin = "v1.5.11"

	// DefaultPalaceRoot is the setup-template local palace (buyer can ls).
	DefaultPalaceRoot = "~/.iomesh/palace"

	// EnvMemoryPalaceRoot is the TUI-named palace root (#402 leftover_is_bind).
	// HTTP MCP URL-only has no stdio -palace-root args; set this to match MCP.
	EnvMemoryPalaceRoot = "IOMESH_MEMORY_PALACE_ROOT"

	// EnvPalaceRoot is the MCP-host fallback already used by ResolvePalaceRoot.
	EnvPalaceRoot = "PALACE_ROOT"

	// ModeAPalaceRootResidualHint is printed when the resolved path is unset or DNE.
	// Never invent Connected / Memory GA. HTTP MCP needs palace_root to match -palace-root.
	ModeAPalaceRootResidualHint = "unset or DNE · set [memory] palace_root / IOMESH_MEMORY_PALACE_ROOT to match MCP -palace-root · never invent Connected · not hosted Memory GA"

	// ModeADigestStickyCommand is the Mode A cite-both walk.
	ModeADigestStickyCommand = "/memory digest --require-sources mesh,private"

	// ModeADigestMissAckLine coordinates digest miss with /dashboard ack (#371).
	// Unacked miss is not known. No send/pay/ship.
	ModeADigestMissAckLine = "digest miss ≠ known · ACK via /dashboard ack (local ritual · no send/pay/ship) · local RCA stays on disk"

	// ModeADigestStickyHelp is slash-help / talk-track chrome.
	ModeADigestStickyHelp = "Mode A sticky: /memory digest --require-sources mesh,private — cite-both or explicit miss · miss ≠ known until /dashboard ack (no send/pay/ship)"
)

// ModeAPinHonestyLine names current published TUI / MCP / memory tags.
func ModeAPinHonestyLine() string {
	return "Published pins (tip class · do not invent GA): TUI " + ModeATUIPin +
		" · MCP " + ModeAMCPPin + " · memory kernel " + ModeAMemoryPin
}

// ModeAAirGapFallbackLine is the local path when portal HITL is blocked.
// No fake Connected. Local RCA + digest miss stay honest.
func ModeAAirGapFallbackLine() string {
	return "Air-gap fallback (portal HITL blocked): stay local — local RCA on palace disk · " +
		ModeADigestStickyCommand + " miss is honest · ACK via /dashboard ack · no fake Connected"
}

// ModeAPalaceVisibilityLine prints an ls-able palace path after attach/ingest.
func ModeAPalaceVisibilityLine(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		p = ExpandPalaceRoot(DefaultPalaceRoot)
	}
	return "palace: " + p + " · ls this path · local disk · not hosted Memory GA"
}

// ModeAPalaceRootResidualLine is the honest line when the path is unset or DNE.
// Do not present a missing default as the ls target. Never invent Connected / Memory GA.
func ModeAPalaceRootResidualLine(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		p = ExpandPalaceRoot(DefaultPalaceRoot)
	}
	return "palace: " + p + " " + ModeAPalaceRootResidualHint
}

// palaceDirExists reports whether p is an existing directory (ls-able palace root).
func palaceDirExists(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// ParsePalaceRootFromArgs reads -palace-root / --palace-root from MCP host args.
func ParsePalaceRootFromArgs(args []string) string {
	for i := 0; i < len(args); i++ {
		a := strings.TrimSpace(args[i])
		switch {
		case a == "-palace-root" || a == "--palace-root":
			if i+1 < len(args) {
				return strings.TrimSpace(args[i+1])
			}
		case strings.HasPrefix(a, "-palace-root="):
			return strings.TrimSpace(strings.TrimPrefix(a, "-palace-root="))
		case strings.HasPrefix(a, "--palace-root="):
			return strings.TrimSpace(strings.TrimPrefix(a, "--palace-root="))
		}
	}
	return ""
}

// ExpandPalaceRoot expands a leading ~/ to the operator home directory.
func ExpandPalaceRoot(raw string) string {
	p := strings.TrimSpace(raw)
	if p == "" {
		p = DefaultPalaceRoot
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return home + p[1:]
		}
	}
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return home
		}
	}
	return p
}

// ResolvePalaceRoot picks configured → MCP args → IOMESH_MEMORY_PALACE_ROOT → PALACE_ROOT → setup default.
// HTTP MCP URL-only has empty args; operators must set [memory] palace_root or env to match -palace-root.
func ResolvePalaceRoot(configured string, mcpArgs []string) string {
	path, _ := resolvePalaceRoot(configured, mcpArgs)
	return path
}

func resolvePalaceRoot(configured string, mcpArgs []string) (path string, explicit bool) {
	if p := strings.TrimSpace(configured); p != "" {
		return ExpandPalaceRoot(p), true
	}
	if p := ParsePalaceRootFromArgs(mcpArgs); p != "" {
		return ExpandPalaceRoot(p), true
	}
	if p := strings.TrimSpace(os.Getenv(EnvMemoryPalaceRoot)); p != "" {
		return ExpandPalaceRoot(p), true
	}
	if p := strings.TrimSpace(os.Getenv(EnvPalaceRoot)); p != "" {
		return ExpandPalaceRoot(p), true
	}
	return ExpandPalaceRoot(DefaultPalaceRoot), false
}

// mcpPalaceArgs returns -palace-root args from the attached memory MCP client (stdio only).
// HTTP URL-only clients have empty Args — TUI cannot see the process -palace-root.
func (rt *Runtime) mcpPalaceArgs() []string {
	if rt == nil || rt.mcp == nil {
		return nil
	}
	name := strings.TrimSpace(rt.memory.Server)
	if c := rt.mcp.ClientByName(name); c != nil {
		return c.Config().Args
	}
	if clients := rt.mcp.Clients(); len(clients) == 1 {
		return clients[0].Config().Args
	}
	return nil
}

// PalacePath is the operator-visible local palace directory (#399/#402).
// Empty runtime falls back to the setup-template default so the buyer can ls
// when that directory is the real root.
func (rt *Runtime) PalacePath() string {
	if rt == nil {
		return ExpandPalaceRoot(DefaultPalaceRoot)
	}
	return ResolvePalaceRoot(rt.memory.PalaceRoot, rt.mcpPalaceArgs())
}

// PalaceVisibilityLine is the attach/ingest chrome line.
// If the resolved default path DNE and no config/env/args named a root, print residual
// (path unset or DNE · set palace_root to match MCP -palace-root). Never invent Connected / Memory GA.
func (rt *Runtime) PalaceVisibilityLine() string {
	if rt == nil {
		def := ExpandPalaceRoot(DefaultPalaceRoot)
		if !palaceDirExists(def) {
			return ModeAPalaceRootResidualLine(def)
		}
		return ModeAPalaceVisibilityLine(def)
	}
	path, explicit := resolvePalaceRoot(rt.memory.PalaceRoot, rt.mcpPalaceArgs())
	if palaceDirExists(path) {
		return ModeAPalaceVisibilityLine(path)
	}
	if explicit {
		// Operator named a root (config/env/args). Print it even if DNE so they can ls / fix.
		return ModeAPalaceVisibilityLine(path)
	}
	// Default fallback DNE (typical HTTP MCP with no args and no palace_root).
	return ModeAPalaceRootResidualLine(path)
}

// modeAHappyPathAionRE matches residual product naming on demo chrome.
// Tests use this; happy-path strings must stay clean after #396.
func modeAHappyPathHasAion(s string) bool {
	low := strings.ToLower(s)
	if strings.Contains(low, "aion_") {
		return true
	}
	// Word-boundary product name (not iomesh / iome.sh).
	for _, tok := range []string{" aion ", " aion.", " aion/", " aion-", "\naion ", "(aion"} {
		if strings.Contains(low, tok) {
			return true
		}
	}
	if strings.HasPrefix(low, "aion ") || strings.HasPrefix(low, "aion_") {
		return true
	}
	return false
}

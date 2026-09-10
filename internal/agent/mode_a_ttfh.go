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
	ModeATUIPin    = "v1.3.0"
	ModeAMCPPin    = "v0.1.1+post-pin" // published v0.1.1; tip may include post-pin commits
	ModeAMemoryPin = "v1.5.8"

	// DefaultPalaceRoot is the setup-template local palace (buyer can ls).
	DefaultPalaceRoot = "~/.iomesh/palace"

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

// ResolvePalaceRoot picks configured → MCP args → PALACE_ROOT → setup default.
func ResolvePalaceRoot(configured string, mcpArgs []string) string {
	if p := strings.TrimSpace(configured); p != "" {
		return ExpandPalaceRoot(p)
	}
	if p := ParsePalaceRootFromArgs(mcpArgs); p != "" {
		return ExpandPalaceRoot(p)
	}
	if p := strings.TrimSpace(os.Getenv("PALACE_ROOT")); p != "" {
		return ExpandPalaceRoot(p)
	}
	return ExpandPalaceRoot(DefaultPalaceRoot)
}

// PalacePath is the operator-visible local palace directory (#399).
// Empty runtime falls back to the setup-template default so the buyer can ls.
func (rt *Runtime) PalacePath() string {
	if rt == nil {
		return ExpandPalaceRoot(DefaultPalaceRoot)
	}
	var args []string
	if rt.mcp != nil {
		name := strings.TrimSpace(rt.memory.Server)
		if c := rt.mcp.ClientByName(name); c != nil {
			args = c.Config().Args
		} else if clients := rt.mcp.Clients(); len(clients) == 1 {
			args = clients[0].Config().Args
		}
	}
	return ResolvePalaceRoot(rt.memory.PalaceRoot, args)
}

// PalaceVisibilityLine is the attach/ingest chrome line.
func (rt *Runtime) PalaceVisibilityLine() string {
	return ModeAPalaceVisibilityLine(rt.PalacePath())
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

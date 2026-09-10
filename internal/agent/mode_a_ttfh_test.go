package agent

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/mcp"
)

func TestModeAPins_PublishedTipClass(t *testing.T) {
	if ModeATUIPin != "v1.2.0" {
		t.Fatalf("TUI pin=%q", ModeATUIPin)
	}
	if ModeAMCPPin != "v0.1.1+post-pin" {
		t.Fatalf("MCP pin=%q want v0.1.1+post-pin (published tip class · do not invent GA)", ModeAMCPPin)
	}
	if ModeAMemoryPin != "v1.5.8" {
		t.Fatalf("memory pin=%q", ModeAMemoryPin)
	}
	line := ModeAPinHonestyLine()
	for _, want := range []string{"v1.2.0", "v0.1.1+post-pin", "v1.5.8", "do not invent GA", "tip class"} {
		if !strings.Contains(line, want) {
			t.Fatalf("pin line missing %q: %s", want, line)
		}
	}
	if strings.Contains(line, "Memory GA") && !strings.Contains(line, "do not invent") {
		t.Fatalf("must not invent Memory GA: %s", line)
	}
}

func TestModeADigestSticky_HelpAndMissACK(t *testing.T) {
	if ModeADigestStickyCommand != "/memory digest --require-sources mesh,private" {
		t.Fatalf("sticky=%q", ModeADigestStickyCommand)
	}
	if !strings.Contains(ModeADigestStickyHelp, ModeADigestStickyCommand) {
		t.Fatalf("help missing sticky command: %s", ModeADigestStickyHelp)
	}
	for _, want := range []string{"cite-both", "explicit miss", "/dashboard ack", "no send/pay/ship"} {
		if !strings.Contains(ModeADigestStickyHelp, want) {
			t.Fatalf("sticky help missing %q: %s", want, ModeADigestStickyHelp)
		}
	}
	for _, want := range []string{"digest miss ≠ known", "/dashboard ack", "no send/pay/ship", "local RCA"} {
		if !strings.Contains(ModeADigestMissAckLine, want) {
			t.Fatalf("miss ACK missing %q: %s", want, ModeADigestMissAckLine)
		}
	}
	if strings.Contains(ModeADigestMissAckLine, "send/pay/ship") &&
		!strings.Contains(ModeADigestMissAckLine, "no send/pay/ship") {
		t.Fatalf("must not invent send/pay/ship: %s", ModeADigestMissAckLine)
	}
}

func TestFormatRequireSourcesCheck_MissPrintsVisibleACK(t *testing.T) {
	res := &iomesh.MemoryOpsDigestResult{
		Receipts: []iomesh.MemoryOpsDigestReceipt{
			{ID: "p1", Summary: "local RCA", SourceHint: "palace_timeline"},
		},
	}
	out := FormatRequireSourcesCheck(res, []string{"mesh", "private"})
	if !strings.Contains(out, "require-sources: miss") || !strings.Contains(out, "missing=mesh") {
		t.Fatalf("want mesh miss: %q", out)
	}
	if !strings.Contains(out, ModeADigestMissAckLine) {
		t.Fatalf("digest miss must print visible ACK:\n%s", out)
	}
	if !strings.Contains(out, "/dashboard ack") {
		t.Fatalf("must coordinate with /dashboard ack:\n%s", out)
	}
	if strings.Contains(out, "require-sources: ok") {
		t.Fatalf("must not ok: %q", out)
	}
}

func TestParsePalaceRootFromArgs(t *testing.T) {
	if got := ParsePalaceRootFromArgs([]string{"-palace-root", "/tmp/demo-palace", "-tenant", "default"}); got != "/tmp/demo-palace" {
		t.Fatalf("short flag=%q", got)
	}
	if got := ParsePalaceRootFromArgs([]string{"--palace-root=/var/palace"}); got != "/var/palace" {
		t.Fatalf("equals=%q", got)
	}
	if got := ParsePalaceRootFromArgs([]string{"-http-addr", ":8080"}); got != "" {
		t.Fatalf("missing should be empty, got %q", got)
	}
}

func TestExpandAndResolvePalaceRoot(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home dir")
	}
	got := ExpandPalaceRoot("~/.iomesh/palace")
	want := filepath.Join(home, ".iomesh", "palace")
	if got != want {
		t.Fatalf("expand=%q want %q", got, want)
	}
	if got := ResolvePalaceRoot("/explicit/palace", nil); got != "/explicit/palace" {
		t.Fatalf("configured wins: %q", got)
	}
	if got := ResolvePalaceRoot("", []string{"-palace-root", "/from/args"}); got != "/from/args" {
		t.Fatalf("args: %q", got)
	}
	t.Setenv("PALACE_ROOT", "/from/env")
	if got := ResolvePalaceRoot("", nil); got != "/from/env" {
		t.Fatalf("env: %q", got)
	}
	t.Setenv("PALACE_ROOT", "")
	if got := ResolvePalaceRoot("", nil); got != want {
		t.Fatalf("default: %q want %q", got, want)
	}
}

func TestRuntimePalacePath_FromConfigAndMCPArgs(t *testing.T) {
	rt := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory", PalaceRoot: "/cfg/palace"}}
	if got := rt.PalacePath(); got != "/cfg/palace" {
		t.Fatalf("config=%q", got)
	}
	line := rt.PalaceVisibilityLine()
	if !strings.Contains(line, "palace: /cfg/palace") || !strings.Contains(line, "ls this path") {
		t.Fatalf("visibility=%q", line)
	}
	if strings.Contains(line, "hosted Memory GA") && !strings.Contains(line, "not hosted") {
		t.Fatalf("must not claim hosted Memory GA: %s", line)
	}

	cInR, cInW := io.Pipe()
	cOutR, cOutW := io.Pipe()
	t.Cleanup(func() {
		_ = cInR.Close()
		_ = cInW.Close()
		_ = cOutR.Close()
		_ = cOutW.Close()
	})
	mut := true
	cl := mcp.NewClientForTest(mcp.ServerConfig{
		Name: "memory", Command: "iomesh-memory-mcp",
		Args:     []string{"-palace-root", "/mcp/palace", "-tenant", "default"},
		Mutating: &mut,
	}, cInW, cOutR, nil)
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)
	rt2 := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}, mcp: mgr}
	if got := rt2.PalacePath(); got != "/mcp/palace" {
		t.Fatalf("mcp args=%q", got)
	}
}

func TestModeAAirGapFallback_NoFakeConnected(t *testing.T) {
	line := ModeAAirGapFallbackLine()
	for _, want := range []string{
		"Air-gap fallback",
		"portal HITL blocked",
		"stay local",
		"local RCA",
		ModeADigestStickyCommand,
		"miss is honest",
		"/dashboard ack",
		"no fake Connected",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("air-gap missing %q: %s", want, line)
		}
	}
	if strings.Contains(line, "Connected: yes") || strings.Contains(line, "fake Connected") && !strings.Contains(line, "no fake") {
		t.Fatalf("must not invent Connected: %s", line)
	}
}

func TestModeAHappyPath_NoAionProductName(t *testing.T) {
	blobs := []string{
		ModeADigestStickyHelp,
		ModeADigestMissAckLine,
		ModeAPinHonestyLine(),
		ModeAAirGapFallbackLine(),
		ModeAPalaceVisibilityLine("/tmp/palace"),
		ModeADigestStickyCommand,
		MeshAgentOnboardingNextMarketingDemoLane(),
		MeshAgentOnboardingNextPortalHITLLane(),
		strings.Join(MemoryNextStepLines(), "\n"),
	}
	for _, blob := range blobs {
		if modeAHappyPathHasAion(blob) {
			t.Fatalf("Mode A happy path must not leak aion product naming:\n%s", blob)
		}
		if strings.Contains(blob, "AION_") {
			t.Fatalf("Mode A happy path must not teach AION_*:\n%s", blob)
		}
	}
}

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
	if ModeATUIPin != "v1.3.5" {
		t.Fatalf("TUI pin=%q", ModeATUIPin)
	}
	if ModeAMCPPin != "v0.4.0" {
		t.Fatalf("MCP pin=%q want v0.4.0 (extract companion · do not invent GA)", ModeAMCPPin)
	}
	if ModeAMemoryPin != "v1.5.11" {
		t.Fatalf("memory pin=%q", ModeAMemoryPin)
	}
	line := ModeAPinHonestyLine()
	for _, want := range []string{"v1.3.5", "v0.4.0", "v1.5.11", "do not invent GA", "tip class"} {
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
	t.Setenv(EnvMemoryPalaceRoot, "")
	t.Setenv(EnvPalaceRoot, "")
	if got := ResolvePalaceRoot("/explicit/palace", nil); got != "/explicit/palace" {
		t.Fatalf("configured wins: %q", got)
	}
	if got := ResolvePalaceRoot("", []string{"-palace-root", "/from/args"}); got != "/from/args" {
		t.Fatalf("args: %q", got)
	}
	t.Setenv(EnvMemoryPalaceRoot, "/from/iomesh-env")
	if got := ResolvePalaceRoot("", nil); got != "/from/iomesh-env" {
		t.Fatalf("IOMESH_MEMORY_PALACE_ROOT: %q", got)
	}
	t.Setenv(EnvMemoryPalaceRoot, "")
	t.Setenv(EnvPalaceRoot, "/from/env")
	if got := ResolvePalaceRoot("", nil); got != "/from/env" {
		t.Fatalf("PALACE_ROOT: %q", got)
	}
	t.Setenv(EnvPalaceRoot, "")
	if got := ResolvePalaceRoot("", nil); got != want {
		t.Fatalf("default: %q want %q", got, want)
	}
	// Stdio args still beat env (live MCP process is the source of truth).
	t.Setenv(EnvMemoryPalaceRoot, "/from/iomesh-env")
	if got := ResolvePalaceRoot("", []string{"-palace-root", "/from/args"}); got != "/from/args" {
		t.Fatalf("stdio args beat IOMESH_MEMORY_PALACE_ROOT: %q", got)
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

func httpMemoryMCPForTest(t *testing.T) *mcp.Manager {
	t.Helper()
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
		Name: "memory", URL: "http://127.0.0.1:8080/mcp",
		Mutating: &mut,
	}, cInW, cOutR, nil)
	mgr := mcp.NewManagerEmpty(nil)
	mgr.Attach(cl)
	return mgr
}

func TestRuntimePalacePath_HTTPNoArgsUsesConfigAndEnv(t *testing.T) {
	t.Setenv(EnvMemoryPalaceRoot, "")
	t.Setenv(EnvPalaceRoot, "")
	mgr := httpMemoryMCPForTest(t)

	rtCfg := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory", PalaceRoot: "/workspace/data/memory-palaces"}, mcp: mgr}
	if got := rtCfg.PalacePath(); got != "/workspace/data/memory-palaces" {
		t.Fatalf("config HTTP-no-args=%q", got)
	}
	line := rtCfg.PalaceVisibilityLine()
	if !strings.Contains(line, "palace: /workspace/data/memory-palaces") || !strings.Contains(line, "ls this path") {
		t.Fatalf("configured HTTP visibility=%q", line)
	}
	if strings.Contains(line, ModeAPalaceRootResidualHint) {
		t.Fatalf("explicit config must not residual: %s", line)
	}

	rtEnv := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}, mcp: mgr}
	if got := rtEnv.PalacePath(); got == "/workspace/data/memory-palaces" {
		t.Fatalf("HTTP-no-args without env/config must not invent MCP root: %q", got)
	}
	t.Setenv(EnvMemoryPalaceRoot, "/workspace/data/memory-palaces")
	if got := rtEnv.PalacePath(); got != "/workspace/data/memory-palaces" {
		t.Fatalf("IOMESH_MEMORY_PALACE_ROOT HTTP-no-args=%q", got)
	}
	t.Setenv(EnvMemoryPalaceRoot, "")
	t.Setenv(EnvPalaceRoot, "/workspace/data/memory-palaces")
	if got := rtEnv.PalacePath(); got != "/workspace/data/memory-palaces" {
		t.Fatalf("PALACE_ROOT HTTP-no-args=%q", got)
	}
}

func TestRuntimePalaceVisibility_DefaultDNEResidual(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(EnvMemoryPalaceRoot, "")
	t.Setenv(EnvPalaceRoot, "")
	mgr := httpMemoryMCPForTest(t)
	rt := &Runtime{memory: MemoryConfig{Enabled: true, Server: "memory"}, mcp: mgr}
	path := rt.PalacePath()
	if palaceDirExists(path) {
		t.Fatalf("temp HOME must not have default palace %q", path)
	}
	line := rt.PalaceVisibilityLine()
	if !strings.Contains(line, "palace: "+path) {
		t.Fatalf("residual must name path: %s", line)
	}
	if !strings.Contains(line, ModeAPalaceRootResidualHint) {
		t.Fatalf("HTTP-no-args default DNE must residual:\n%s", line)
	}
	if strings.Contains(line, "ls this path") {
		t.Fatalf("must not present DNE default as ls target:\n%s", line)
	}
	if strings.Contains(line, "Connected") && !strings.Contains(line, "never invent Connected") {
		t.Fatalf("must not invent Connected: %s", line)
	}
	if strings.Contains(line, "Memory GA") && !strings.Contains(line, "not hosted Memory GA") {
		t.Fatalf("must not invent Memory GA: %s", line)
	}
	status := rt.MemoryStatusLine()
	if !strings.Contains(status, ModeAPalaceRootResidualHint) {
		t.Fatalf("status must carry residual:\n%s", status)
	}
}

func TestModeAPalaceRootResidual_Honesty(t *testing.T) {
	line := ModeAPalaceRootResidualLine("/home/box/.iomesh/palace")
	for _, want := range []string{
		"palace: /home/box/.iomesh/palace",
		"unset or DNE",
		"[memory] palace_root",
		"IOMESH_MEMORY_PALACE_ROOT",
		"MCP -palace-root",
		"never invent Connected",
		"not hosted Memory GA",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("residual missing %q: %s", want, line)
		}
	}
	if strings.Contains(line, "ls this path") {
		t.Fatalf("residual must not tell operator to ls a DNE path: %s", line)
	}
	if strings.Contains(line, "Connected: yes") || strings.Contains(line, "Memory GA shipped") {
		t.Fatalf("must not invent Connected / Memory GA: %s", line)
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
		ModeAPalaceRootResidualLine("/tmp/missing-palace"),
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

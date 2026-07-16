package security

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Shell policy limits for the run_shell tool.
const (
	// MaxShellCommandBytes is the maximum accepted command length.
	MaxShellCommandBytes = 32 * 1024
	// MaxShellOutputBytes truncates combined stdout/stderr returned to the model.
	MaxShellOutputBytes = 512 * 1024
)

// dangerousShell patterns are hard-denied even with --yolo. These are not a
// complete sandbox — they are a best-effort guardrail for accidental catastrophe
// and common supply-chain footguns. Operators still must treat --yolo as full trust.
var dangerousShell = []string{
	"rm -rf /",
	"rm -rf/*",
	"rm -fr /",
	":(){ :|:& };:",
	"mkfs.",
	"dd if=/dev/zero of=/dev/",
	"dd if=/dev/random of=/dev/",
	"> /dev/sda",
	"chmod -R 777 /",
	"curl | sh",
	"curl|sh",
	"curl | bash",
	"curl|bash",
	"wget | sh",
	"wget|sh",
	"wget | bash",
	"wget|bash",
	"| sh -",
	"| bash -",
	"npm install -g",     // global install without review
	"pip install --user", // heuristic; still allow pip install local
}

// ValidateShellCommand rejects empty, oversized, non-UTF8, or hard-denied commands.
func ValidateShellCommand(cmd string) error {
	if strings.TrimSpace(cmd) == "" {
		return fmt.Errorf("shell: empty command")
	}
	if len(cmd) > MaxShellCommandBytes {
		return fmt.Errorf("shell: command exceeds %d bytes", MaxShellCommandBytes)
	}
	if !utf8.ValidString(cmd) {
		return fmt.Errorf("shell: command is not valid UTF-8")
	}
	// Null bytes can confuse shells / C APIs.
	if strings.ContainsRune(cmd, 0) {
		return fmt.Errorf("shell: command contains NUL byte")
	}
	lower := strings.ToLower(strings.Join(strings.Fields(cmd), " "))
	// Collapse spaces for pattern checks.
	compact := strings.ReplaceAll(lower, " ", "")
	for _, d := range dangerousShell {
		dNorm := strings.ToLower(strings.Join(strings.Fields(d), " "))
		dCompact := strings.ReplaceAll(dNorm, " ", "")
		if strings.Contains(lower, dNorm) || strings.Contains(compact, dCompact) {
			return fmt.Errorf("shell: command blocked by safety policy (%q)", d)
		}
	}
	// Pipe-to-interpreter without space variants already covered; also block
	// `curl ... | sudo bash` style loosely.
	if strings.Contains(lower, "curl ") && (strings.Contains(lower, "| sh") || strings.Contains(lower, "| bash") || strings.Contains(lower, "|sudo sh") || strings.Contains(lower, "| sudo bash")) {
		return fmt.Errorf("shell: pipe-to-shell download pattern blocked by safety policy")
	}
	if strings.Contains(lower, "wget ") && (strings.Contains(lower, "| sh") || strings.Contains(lower, "| bash")) {
		return fmt.Errorf("shell: pipe-to-shell download pattern blocked by safety policy")
	}
	return nil
}

// TruncateOutput limits tool output size returned to the LLM.
func TruncateOutput(b []byte, max int) string {
	if max <= 0 {
		max = MaxShellOutputBytes
	}
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + fmt.Sprintf("\n…[truncated %d bytes]", len(b)-max)
}

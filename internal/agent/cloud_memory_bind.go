package agent

import "github.com/iome-sh/iomesh-tui/internal/honesty"

// CloudMemoryBindDigestChrome is the digest secondary frame (B5).
// It does not contain the substring "Connected".
func CloudMemoryBindDigestChrome() string { return honesty.DigestChrome() }

// CloudMemoryBindGapText is the operator Gap / Partial stamp (B5 + C4).
func CloudMemoryBindGapText() string { return honesty.HostBindGap() }

// SessionPalaceInput is the workspace view for session palace selection.
type SessionPalaceInput = honesty.SessionPalaceInput

// SelectSessionPalace returns a real workspace MemoryURL, or "" when the
// displayed palace stays palace=-. HostedPalaceEnabled, cfg.MemoryURL, a
// catalog row, a workspace PATCH, and OpenGates do not bind.
func SelectSessionPalace(in SessionPalaceInput) string {
	return honesty.SelectSessionPalace(in)
}

// DisplaySessionPalace is palace=<url> or the exact token palace=-.
func DisplaySessionPalace(in SessionPalaceInput) string {
	return honesty.DisplaySessionPalace(in)
}

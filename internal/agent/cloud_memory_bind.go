package agent

import "github.com/iome-sh/iomesh-tui/internal/honesty"

// CloudMemoryBindDigestChrome is the digest secondary frame (B5).
// It does not contain the substring "Connected".
func CloudMemoryBindDigestChrome() string { return honesty.DigestChrome() }

// CloudMemoryBindGapText is the operator Gap / Partial stamp (B5 + C4).
func CloudMemoryBindGapText() string { return honesty.HostBindGap() }

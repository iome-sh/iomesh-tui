// Package honesty holds buyer-facing stamps that must stay free of invented
// product claims. Cloud Memory palace bind is Gap / Partial only.
package honesty

import "strings"

// WritePathPin is the short buyer pin. It does not name internal flags.
const WritePathPin = "memory pin · one write path"

// WritePathChip is the buyer sentence for a single write path.
const WritePathChip = "One write path — not mirrored to a second store."

// SeparatePaths states local disk and Cloud Memory stay apart.
// Cloud Memory GA is optional beside TTFH. TTFH/heartbeat is the SoR.
const SeparatePaths = "Local and Cloud Memory stay on separate paths. Local private on disk. Cloud Memory GA and optional beside TTFH — not a substitute. TTFH/heartbeat is the SoR. Empty until consume."

// HostBindGapDigest is B5 chrome appended to /memory digest.
// Digest output must not contain the substring "Connected" (cite tests).
const HostBindGapDigest = "GAP · B5 host bind · Partial — Cloud Memory as a remote palace target is Gap until QA evidence. Console entitlement is the primary attach. Entitlement ≠ live bind. Do not invent a live host URL. US-CM-JOURNEY-05 after QA."

// DigestChrome is the digest secondary frame: write-path pin plus B5 gap.
func DigestChrome() string {
	return WritePathPin + "\n" + WritePathChip + "\n" + SeparatePaths + "\n" + HostBindGapDigest
}

// HostBindGap is the operator stamp for slash, onboard, preflight, and help.
// Status is Gap / Partial. It does not claim an Exists Connected bind and
// does not invent a live host URL.
func HostBindGap() string {
	return strings.TrimSpace(`GAP / Partial · US-CM-JOURNEY-05 · not an Exists Connected bind
` + WritePathPin + `
` + WritePathChip + `
` + SeparatePaths + `
Cloud Memory is not required for heartbeat. Catalog ≠ Connected. workspace-as-principal. Multi-human palace read/write stays Gap.
B5 · TUI host bind · Gap until QA evidence. Console entitlement is the primary attach. Entitlement ≠ live bind. Do not invent a Connected host URL. No Connected badge.
C4 · SDK palace URL bind · Gap until QA evidence. Entitlement ≠ live bind. Empty until consume. Do not invent a live host URL. This TUI does not ship that bind.
Entitled path: confirm Console entitlement for the workspace (primary attach · not a live host) · keep private notes on the local palace · run TTFH/heartbeat as the system of record · cite-both or an honest miss · stop before any host URL.`)
}

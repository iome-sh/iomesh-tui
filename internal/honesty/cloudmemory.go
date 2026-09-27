// Package honesty holds buyer-facing stamps that must stay free of invented
// product claims. Cloud Memory palace bind is Gap / Partial only.
package honesty

import "strings"

// WritePathPin is the short buyer pin. It does not name internal flags.
const WritePathPin = "memory pin · one write path"

// WritePathChip is the buyer sentence for a single write path.
const WritePathChip = "One write path — not mirrored to a second store."

// SeparatePaths states local disk and Cloud Memory stay apart.
// Cloud Memory is available, optional beside TTFH. TTFH/heartbeat is the SoR.
const SeparatePaths = "Local and Cloud Memory stay on separate paths. Local private on disk. Available, optional beside TTFH — not a substitute. TTFH/heartbeat is the SoR. Empty until consume."

// CloudMemoryOffer is the buyer and operator price line.
// It does not say the product is GA, and entitlement is not a bind.
const CloudMemoryOffer = "Cloud Memory is the $199 add-on, available beside TTFH."

// UnboundPalaceToken is the exact displayed palace when no bind has succeeded.
const UnboundPalaceToken = "palace=-"

// UnboundPalaceLine names that default. It is not a host URL.
const UnboundPalaceLine = "Default when no bind has succeeded: palace=-."

// NotBindSignals is operator copy. Digest chrome must not include it:
// cite tests reject the substring Connected.
const NotBindSignals = "A catalog row, a workspace PATCH, or an empty OpenGates list is not Connected."

// LaptopPalaceNotBind keeps the local default off a Cloud Memory bind.
const LaptopPalaceNotBind = "Laptop default ~/.iomesh/palace is not a Cloud Memory bind."

// HostBindGapDigest is B5 chrome appended to /memory digest.
// Digest output must not contain the substring "Connected" (cite tests).
const HostBindGapDigest = "B5 host bind · Partial — 2026-09-27 the laptop retrieved qq14l3dmqgu5sz8hu7wchom8 from the stage dedicated disk. Local palace was not used. Entitlement ≠ live bind. Unbound workspaces stay palace=-."

// DigestChrome is the digest secondary frame: write-path pin plus B5 gap.
// No Connected substring. The catalog / PATCH / OpenGates sentence stays on
// the operator stamp.
func DigestChrome() string {
	return WritePathPin + "\n" + WritePathChip + "\n" + SeparatePaths + "\n" + CloudMemoryOffer + "\n" + UnboundPalaceLine + "\n" + HostBindGapDigest
}

// HostBindGap is the operator stamp for slash, onboard, preflight, and help.
// Status is Gap / Partial. It does not claim an Exists Connected bind and
// does not invent a live host URL.
func HostBindGap() string {
	return strings.TrimSpace(`GAP / Partial · US-CM-JOURNEY-05 · not an Exists Connected bind
` + WritePathPin + `
` + WritePathChip + `
` + SeparatePaths + `
` + CloudMemoryOffer + `
` + UnboundPalaceLine + `
` + NotBindSignals + `
` + LaptopPalaceNotBind + `
Entitlement is not Connected. Cloud Memory is not required for heartbeat. Catalog ≠ Connected. workspace-as-principal. Multi-human palace read/write stays Gap.
B5 · TUI host bind · 2026-09-27 laptop retrieve qq14l3dmqgu5sz8hu7wchom8 from the stage dedicated disk. Local palace was not used. Entitlement ≠ live bind. No Connected badge.
C4 · SDK palace URL bind · that sitting used the dedicated host, not HostedPalaceEnabled, not a shared cfg.MemoryURL, and not the one-label aion-mem-*.internal placeholder. This TUI process was not the client.
Entitled path: confirm Console entitlement for the workspace (primary attach · not a live host) · keep private notes on the local palace · run TTFH/heartbeat as the system of record · cite-both or an honest miss · stop before any host URL.`)
}

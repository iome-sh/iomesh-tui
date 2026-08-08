package agentplugins

import "sync"

// Soft offline dogfood session marker (s1392 set path · s1397 onboard next status/export).
// Session-only: default dogfood_not_run. Soft offline pass/fail ≠ invent Agent Plugins GA ·
// ≠ live dogfood · board/export evidence ≠ invent Connected.
//
// Lives in agentplugins so both TUI /plugins slash and agent aion_onboarding can share
// state without import cycles (agent cannot import tui; tui already imports agentplugins).

var softDogfoodSession struct {
	mu   sync.Mutex
	ran  bool
	pass bool // residual soft offline pass only
}

// Soft dogfood session state labels (honest vocabulary only).
const (
	// SoftDogfoodSessionNotRun is the default when /plugins dogfood has not run this session.
	SoftDogfoodSessionNotRun = "dogfood_not_run"
	// SoftDogfoodSessionPass is session soft offline dogfood residual PASS (≠ live dogfood · ≠ GA).
	SoftDogfoodSessionPass = "soft_offline_dogfood_session_pass"
	// SoftDogfoodSessionFail is session soft offline dogfood residual FAIL (≠ invent red product).
	SoftDogfoodSessionFail = "soft_offline_dogfood_session_fail"
)

// SetSoftDogfoodSessionState records that soft offline dogfood ran this session.
// pass is residual soft offline only — never invents Agent Plugins GA / live dogfood / Connected.
func SetSoftDogfoodSessionState(pass bool) {
	softDogfoodSession.mu.Lock()
	defer softDogfoodSession.mu.Unlock()
	softDogfoodSession.ran = true
	softDogfoodSession.pass = pass
}

// GetSoftDogfoodSessionState returns whether soft dogfood ran this session and residual pass.
// Default: ran=false, pass=false → SoftDogfoodSessionLabel returns dogfood_not_run.
func GetSoftDogfoodSessionState() (ran bool, pass bool) {
	softDogfoodSession.mu.Lock()
	defer softDogfoodSession.mu.Unlock()
	return softDogfoodSession.ran, softDogfoodSession.pass
}

// SoftDogfoodSessionLabel returns the honest session dogfood vocabulary token:
// dogfood_not_run | soft_offline_dogfood_session_pass | soft_offline_dogfood_session_fail.
// Session soft marker ≠ live dogfood · ≠ invent Agent Plugins GA · board ≠ invent Connected.
func SoftDogfoodSessionLabel() string {
	ran, pass := GetSoftDogfoodSessionState()
	if !ran {
		return SoftDogfoodSessionNotRun
	}
	if pass {
		return SoftDogfoodSessionPass
	}
	return SoftDogfoodSessionFail
}

// ResetSoftDogfoodSessionState clears the session marker (tests only).
func ResetSoftDogfoodSessionState() {
	softDogfoodSession.mu.Lock()
	defer softDogfoodSession.mu.Unlock()
	softDogfoodSession.ran = false
	softDogfoodSession.pass = false
}

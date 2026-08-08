package agentplugins

import (
	"strings"
	"testing"
)

func TestSoftDogfoodSessionState_DefaultNotRun(t *testing.T) {
	ResetSoftDogfoodSessionState()
	t.Cleanup(ResetSoftDogfoodSessionState)

	ran, pass := GetSoftDogfoodSessionState()
	if ran || pass {
		t.Fatalf("default: ran=%v pass=%v want false/false", ran, pass)
	}
	if got := SoftDogfoodSessionLabel(); got != SoftDogfoodSessionNotRun {
		t.Fatalf("label: got %q want %q", got, SoftDogfoodSessionNotRun)
	}
}

func TestSoftDogfoodSessionState_PassFail(t *testing.T) {
	ResetSoftDogfoodSessionState()
	t.Cleanup(ResetSoftDogfoodSessionState)

	SetSoftDogfoodSessionState(true)
	ran, pass := GetSoftDogfoodSessionState()
	if !ran || !pass {
		t.Fatalf("after pass: ran=%v pass=%v", ran, pass)
	}
	if got := SoftDogfoodSessionLabel(); got != SoftDogfoodSessionPass {
		t.Fatalf("pass label: got %q want %q", got, SoftDogfoodSessionPass)
	}

	SetSoftDogfoodSessionState(false)
	ran, pass = GetSoftDogfoodSessionState()
	if !ran || pass {
		t.Fatalf("after fail: ran=%v pass=%v", ran, pass)
	}
	if got := SoftDogfoodSessionLabel(); got != SoftDogfoodSessionFail {
		t.Fatalf("fail label: got %q want %q", got, SoftDogfoodSessionFail)
	}

	// Honesty: labels never invent GA / Connected / live dogfood product language.
	for _, label := range []string{SoftDogfoodSessionNotRun, SoftDogfoodSessionPass, SoftDogfoodSessionFail} {
		if strings.Contains(label, "Connected") || strings.Contains(label, "GA") || strings.Contains(label, "live") {
			t.Fatalf("label must not invent Connected/GA/live: %q", label)
		}
	}
}

func TestSoftDogfoodSessionState_Reset(t *testing.T) {
	SetSoftDogfoodSessionState(true)
	ResetSoftDogfoodSessionState()
	if got := SoftDogfoodSessionLabel(); got != SoftDogfoodSessionNotRun {
		t.Fatalf("after reset: got %q want %q", got, SoftDogfoodSessionNotRun)
	}
}

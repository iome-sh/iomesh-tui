package setup

import (
	"strings"
	"testing"
)

// s1723: IOMESH_PLATFORM_RESIDUAL env truthy values (1 · true · yes, case-insensitive).
func TestPlatformResidualEnvOn(t *testing.T) {
	t.Run("off when unset", func(t *testing.T) {
		t.Setenv("IOMESH_PLATFORM_RESIDUAL", "")
		if PlatformResidualEnvOn() {
			t.Fatal("expected false when env empty")
		}
	})
	t.Run("off when garbage", func(t *testing.T) {
		t.Setenv("IOMESH_PLATFORM_RESIDUAL", "maybe")
		if PlatformResidualEnvOn() {
			t.Fatal("expected false for non-truthy value")
		}
	})
	for _, v := range []string{"1", "true", "TRUE", "Yes", "yes", "True"} {
		v := v
		t.Run("on_"+v, func(t *testing.T) {
			t.Setenv("IOMESH_PLATFORM_RESIDUAL", v)
			if !PlatformResidualEnvOn() {
				t.Fatalf("expected true for %q", v)
			}
		})
	}
	for _, v := range []string{"0", "false", "no", "off"} {
		v := v
		t.Run("off_"+v, func(t *testing.T) {
			t.Setenv("IOMESH_PLATFORM_RESIDUAL", v)
			if PlatformResidualEnvOn() {
				t.Fatalf("expected false for %q", v)
			}
		})
	}
}

// s1723: label note only when env on; labeling only — never product "hide" lanes.
func TestPlatformResidualLabelNote(t *testing.T) {
	t.Run("empty when off", func(t *testing.T) {
		t.Setenv("IOMESH_PLATFORM_RESIDUAL", "")
		if got := PlatformResidualLabelNote(); got != "" {
			t.Fatalf("expected empty label when off, got %q", got)
		}
	})
	t.Run("non-empty when on", func(t *testing.T) {
		t.Setenv("IOMESH_PLATFORM_RESIDUAL", "1")
		got := PlatformResidualLabelNote()
		if got == "" {
			t.Fatal("expected non-empty label when on")
		}
		for _, want := range []string{
			"IOMESH_PLATFORM_RESIDUAL",
			"labels only",
			"lanes not hidden",
			"control plane",
			"s1723",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("label missing %q in:\n%s", want, got)
			}
		}
		// Never claims product behavior of hiding lanes.
		if strings.Contains(got, "hide lanes") || strings.Contains(got, "hides lanes") {
			t.Fatalf("must not claim hide-lanes product behavior:\n%s", got)
		}
	})
}

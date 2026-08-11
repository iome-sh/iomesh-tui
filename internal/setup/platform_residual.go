package setup

import (
	"os"
	"strings"
)

// PlatformResidualEnvOn reports whether IOMESH_PLATFORM_RESIDUAL is set truthy
// (1 · true · yes, case-insensitive). Residual-honest optional label only —
// MUST NOT hide lanes · labeling only · does not invent control plane · free eng s1723.
func PlatformResidualEnvOn() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("IOMESH_PLATFORM_RESIDUAL")))
	switch v {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

// PlatformResidualLabelNote returns an optional one-liner when
// IOMESH_PLATFORM_RESIDUAL is on; empty when off. Labels only — lanes not
// hidden · residual PASS ≠ invent control plane · free eng s1723.
func PlatformResidualLabelNote() string {
	if !PlatformResidualEnvOn() {
		return ""
	}
	return "label: IOMESH_PLATFORM_RESIDUAL=1 · platform residual honesty labels only · lanes not hidden · residual PASS ≠ invent control plane · free eng s1723"
}

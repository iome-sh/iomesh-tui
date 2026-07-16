package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme is a named color palette for the full-screen TUI.
type Theme struct {
	Name string

	Title   lipgloss.Style
	Status  lipgloss.Style
	Dim     lipgloss.Style
	User    lipgloss.Style
	Tool    lipgloss.Style
	OK      lipgloss.Style
	Err     lipgloss.Style
	Mesh    lipgloss.Style
	Approve lipgloss.Style
	Help    lipgloss.Style
	Prompt  lipgloss.Style
}

// ThemeNames returns built-in theme identifiers.
func ThemeNames() []string {
	return []string{"default", "mono", "high-contrast", "dim"}
}

// ParseTheme resolves a theme by name (case-insensitive). Empty → default.
func ParseTheme(name string) (Theme, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "default", "cyan":
		return ThemeDefault(), nil
	case "mono", "monochrome", "bw":
		return ThemeMono(), nil
	case "high-contrast", "hc", "highcontrast":
		return ThemeHighContrast(), nil
	case "dim", "muted", "soft":
		return ThemeDim(), nil
	default:
		return Theme{}, fmt.Errorf("unknown theme %q (want: %s)", name, strings.Join(ThemeNames(), ", "))
	}
}

// ThemeDefault is the original cyan/amber palette.
func ThemeDefault() Theme {
	return Theme{
		Name:    "default",
		Title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")),
		Status:  lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Dim:     lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		User:    lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
		Tool:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		OK:      lipgloss.NewStyle().Foreground(lipgloss.Color("78")),
		Err:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		Mesh:    lipgloss.NewStyle().Foreground(lipgloss.Color("45")),
		Approve: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226")).Background(lipgloss.Color("235")),
		Help:    lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		Prompt:  lipgloss.NewStyle().Foreground(lipgloss.Color("86")),
	}
}

// ThemeMono is grayscale for minimal distraction / accessibility.
func ThemeMono() Theme {
	return Theme{
		Name:    "mono",
		Title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")),
		Status:  lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		Dim:     lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		User:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Underline(true),
		Tool:    lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		OK:      lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
		Err:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Reverse(true),
		Mesh:    lipgloss.NewStyle().Foreground(lipgloss.Color("248")),
		Approve: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("232")).Background(lipgloss.Color("255")),
		Help:    lipgloss.NewStyle().Foreground(lipgloss.Color("246")),
		Prompt:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")),
	}
}

// ThemeHighContrast emphasizes errors and approvals for bright terminals.
func ThemeHighContrast() Theme {
	return Theme{
		Name:    "high-contrast",
		Title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226")),
		Status:  lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
		Dim:     lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		User:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51")),
		Tool:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208")),
		OK:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46")),
		Err:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Background(lipgloss.Color("52")),
		Mesh:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("87")),
		Approve: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("226")),
		Help:    lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
		Prompt:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226")),
	}
}

// ThemeDim is a softer, lower-saturation palette.
func ThemeDim() Theme {
	return Theme{
		Name:    "dim",
		Title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("109")),
		Status:  lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		Dim:     lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
		User:    lipgloss.NewStyle().Foreground(lipgloss.Color("67")).Bold(true),
		Tool:    lipgloss.NewStyle().Foreground(lipgloss.Color("136")),
		OK:      lipgloss.NewStyle().Foreground(lipgloss.Color("65")),
		Err:     lipgloss.NewStyle().Foreground(lipgloss.Color("131")),
		Mesh:    lipgloss.NewStyle().Foreground(lipgloss.Color("73")),
		Approve: lipgloss.NewStyle().Foreground(lipgloss.Color("178")).Background(lipgloss.Color("236")),
		Help:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Prompt:  lipgloss.NewStyle().Foreground(lipgloss.Color("109")),
	}
}

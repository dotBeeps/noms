package shared

import (
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// NewSpinner returns a standard accent-colored dot spinner for local/general operations.
func NewSpinner() spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.ColorAccent)),
	)
}

// NewNetworkSpinner returns a globe spinner for network/fetch operations.
func NewNetworkSpinner() spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.Globe),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.ColorAccent)),
	)
}

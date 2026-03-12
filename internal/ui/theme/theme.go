package theme

import "charm.land/lipgloss/v2"

// Color palette constants
var (
	ColorPrimary   = lipgloss.Color("62")  // purple-ish
	ColorSecondary = lipgloss.Color("243") // grey
	ColorAccent    = lipgloss.Color("205") // pink
	ColorError     = lipgloss.Color("196") // red
	ColorSuccess   = lipgloss.Color("78")  // green
	ColorMuted     = lipgloss.Color("241") // dim grey
	ColorHighlight = lipgloss.Color("229") // bright yellow
)

// Layout constants
const (
	// TabBarHeight is the height of the tab bar
	TabBarHeight = 1
	// StatusBarHeight is the height of the status bar
	StatusBarHeight = 1
)

// Reusable styles
var (
	StylePost = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	StyleHeader = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	StyleSelected = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorError)

	StyleHeaderSubtle = lipgloss.NewStyle().
				Foreground(ColorPrimary)

	StyleTabActive = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorAccent).
			Padding(0, 1)

	StyleTabInactive = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Padding(0, 1)
)

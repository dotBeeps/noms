package theme

import "charm.land/lipgloss/v2"

// NotifUnreadDotStyle returns the style for the unread indicator dot.
func NotifUnreadDotStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorAccent)
}

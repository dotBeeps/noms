package components

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dotBeeps/noms/internal/ui/theme"
)

// Style factory functions — constructed on call so they always reflect the active theme.

func tabActiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorOnPrimary).Background(theme.ColorPrimary).Bold(true).Padding(0, 2)
}
func tabInactiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorMuted).Background(theme.ColorSurfaceAlt).Padding(0, 2)
}

type Tab int

const (
	TabFeed Tab = iota
	TabNotifications
	TabProfile
	TabSearch
	TabVoresky
	TabVoreskyNotifications
	TabCount
)

var tabNames = map[Tab]string{
	TabFeed:                 "Feed",
	TabNotifications:        "Notifications",
	TabProfile:              "Profile",
	TabSearch:               "Search",
	TabVoresky:              "Voresky",
	TabVoreskyNotifications: "V-Notifs",
}

type TabBar struct {
	Width         int
	ActiveTab     Tab
	VoreskyActive bool
}

func NewTabBar() TabBar {
	return TabBar{
		ActiveTab: TabFeed,
	}
}

func (m TabBar) Init() tea.Cmd {
	return nil
}

func (m TabBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
	}
	return m, nil
}

func (m TabBar) View() tea.View {
	var tabs []string

	for i := Tab(0); i < TabCount; i++ {
		if !m.VoreskyActive && (i == TabVoresky || i == TabVoreskyNotifications) {
			continue
		}
		key := fmt.Sprintf("[%d]", i+1)
		name := tabNames[i]

		var rendered string
		if i == m.ActiveTab {
			rendered = tabActiveStyle().Render(key + " " + name)
		} else {
			rendered = tabInactiveStyle().Render(key + " " + name)
		}
		tabs = append(tabs, rendered)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	row = lipgloss.Place(m.Width, 1, lipgloss.Left, lipgloss.Top, row,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(theme.ColorSurfaceAlt)))

	return tea.NewView(row)
}

func (m *TabBar) SetActiveTab(tab Tab) {
	if tab >= 0 && tab < TabCount {
		m.ActiveTab = tab
	}
}

func (m TabBar) ActiveTabName() string {
	return tabNames[m.ActiveTab]
}

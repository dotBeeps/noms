package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	tabActiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 2)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("246")).
				Background(lipgloss.Color("238")).
				Padding(0, 2)

	tabKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
)

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
			rendered = tabActiveStyle.Render(key + " " + name)
		} else {
			rendered = tabInactiveStyle.Render(key + " " + name)
		}
		tabs = append(tabs, rendered)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	remaining := m.Width - lipgloss.Width(row)
	if remaining > 0 {
		filler := lipgloss.NewStyle().
			Background(lipgloss.Color("238")).
			Render(strings.Repeat(" ", remaining))
		row = row + filler
	}

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

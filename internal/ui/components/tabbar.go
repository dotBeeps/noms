package components

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
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

type tabBarTickMsg struct{}

func tabBarTick() tea.Cmd {
	return tea.Tick(time.Second/30, func(time.Time) tea.Msg { return tabBarTickMsg{} })
}

type TabBar struct {
	Width         int
	ActiveTab     Tab
	VoreskyActive bool

	// Underline slide animation
	prevTab       Tab
	animStartTime time.Time
	animActive    bool
}

func NewTabBar() TabBar {
	return TabBar{
		ActiveTab: TabFeed,
		prevTab:   TabFeed,
	}
}

func (m TabBar) Init() tea.Cmd {
	return nil
}

func (m TabBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.(tea.WindowSizeMsg).Width
	case tabBarTickMsg:
		if !m.animActive {
			return m, nil
		}
		progress := shared.AnimProgress(m.animStartTime, 150*time.Millisecond)
		if progress >= 1 {
			m.animActive = false
			return m, nil
		}
		return m, tabBarTick()
	}
	return m, nil
}

func (m TabBar) View() tea.View {
	var visibleTabs []Tab
	var tabPositions []int
	var tabWidths []int
	var tabLabels []string

	pos := 0
	for i := Tab(0); i < TabCount; i++ {
		if !m.VoreskyActive && (i == TabVoresky || i == TabVoreskyNotifications) {
			continue
		}
		visibleTabs = append(visibleTabs, i)
		label := "  " + tabNames[i] + "  "
		w := len([]rune(label))
		tabPositions = append(tabPositions, pos)
		tabWidths = append(tabWidths, w)
		tabLabels = append(tabLabels, label)
		pos += w
	}

	pillPos, pillWidth := m.interpolatePill(visibleTabs, tabPositions, tabWidths)

	activeStyle := lipgloss.NewStyle().Foreground(theme.ColorOnPrimary).Background(theme.ColorPrimary).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(theme.ColorMuted).Background(theme.ColorSurfaceAlt)

	type seg struct {
		text   string
		active bool
	}
	var segs []seg

	charPos := 0
	for _, label := range tabLabels {
		for _, ch := range label {
			isActive := charPos >= pillPos && charPos < pillPos+pillWidth
			s := string(ch)
			if len(segs) > 0 && segs[len(segs)-1].active == isActive {
				segs[len(segs)-1].text += s
			} else {
				segs = append(segs, seg{s, isActive})
			}
			charPos++
		}
	}

	var b strings.Builder
	for _, s := range segs {
		if s.active {
			b.WriteString(activeStyle.Render(s.text))
		} else {
			b.WriteString(inactiveStyle.Render(s.text))
		}
	}

	row := lipgloss.Place(m.Width, 1, lipgloss.Left, lipgloss.Top, b.String(),
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(theme.ColorSurfaceAlt)))

	return tea.NewView(row)
}

func (m TabBar) interpolatePill(visibleTabs []Tab, positions, widths []int) (int, int) {
	activePos, activeWidth := 0, 0
	prevPos, prevWidth := 0, 0
	for i, tab := range visibleTabs {
		if tab == m.ActiveTab {
			activePos = positions[i]
			activeWidth = widths[i]
		}
		if tab == m.prevTab {
			prevPos = positions[i]
			prevWidth = widths[i]
		}
	}

	progress := float64(1)
	if m.animActive {
		progress = shared.EaseOutQuad(shared.AnimProgress(m.animStartTime, 150*time.Millisecond))
	}

	currentPos := int(float64(prevPos) + float64(activePos-prevPos)*progress)
	currentWidth := int(float64(prevWidth) + float64(activeWidth-prevWidth)*progress)
	if currentWidth < 1 {
		currentWidth = 1
	}
	return currentPos, currentWidth
}

func (m *TabBar) SetActiveTab(tab Tab) tea.Cmd {
	if tab >= 0 && tab < TabCount {
		if tab != m.ActiveTab {
			m.prevTab = m.ActiveTab
			m.animStartTime = time.Now()
			m.animActive = true
			m.ActiveTab = tab
			return tabBarTick()
		}
		m.ActiveTab = tab
	}
	return nil
}

func (m TabBar) ActiveTabName() string {
	return tabNames[m.ActiveTab]
}

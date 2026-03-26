package components

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"

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

	// Spring-driven pill slide animation
	animActive bool
	pillPos    float64
	pillPosVel float64
	pillWid    float64
	pillWidVel float64
	spring     harmonica.Spring
}

func NewTabBar() TabBar {
	return TabBar{
		ActiveTab: TabFeed,
		spring:    harmonica.NewSpring(harmonica.FPS(30), 7.0, 0.7),
	}
}

func (m TabBar) Init() tea.Cmd {
	return nil
}

func (m TabBar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
	case tabBarTickMsg:
		if !m.animActive {
			return m, nil
		}
		// Compute target position and width for the active tab
		targetPos, targetWid := m.tabTarget()
		m.pillPos, m.pillPosVel = m.spring.Update(m.pillPos, m.pillPosVel, float64(targetPos))
		m.pillWid, m.pillWidVel = m.spring.Update(m.pillWid, m.pillWidVel, float64(targetWid))

		// Settle when close enough
		posDiff := m.pillPos - float64(targetPos)
		widDiff := m.pillWid - float64(targetWid)
		if posDiff < 0.5 && posDiff > -0.5 && widDiff < 0.5 && widDiff > -0.5 {
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
		w := lipgloss.Width(label)
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
	if !m.animActive {
		// No animation — snap to active tab
		for i, tab := range visibleTabs {
			if tab == m.ActiveTab {
				return positions[i], widths[i]
			}
		}
		return 0, 1
	}

	pos := int(m.pillPos)
	wid := int(m.pillWid)
	if wid < 1 {
		wid = 1
	}
	return pos, wid
}

// tabTarget returns the character position and width for the current active tab.
func (m TabBar) tabTarget() (int, int) {
	pos := 0
	for i := Tab(0); i < TabCount; i++ {
		if !m.VoreskyActive && (i == TabVoresky || i == TabVoreskyNotifications) {
			continue
		}
		label := "  " + tabNames[i] + "  "
		w := lipgloss.Width(label)
		if i == m.ActiveTab {
			return pos, w
		}
		pos += w
	}
	return 0, 1
}

func (m *TabBar) SetActiveTab(tab Tab) tea.Cmd {
	if tab >= 0 && tab < TabCount {
		if tab != m.ActiveTab {
			// Snapshot current pill position as spring start
			prevPos, prevWid := m.tabTarget()
			m.ActiveTab = tab
			m.pillPos = float64(prevPos)
			m.pillWid = float64(prevWid)
			m.pillPosVel = 0
			m.pillWidVel = 0
			m.animActive = true
			return tabBarTick()
		}
		m.ActiveTab = tab
	}
	return nil
}

func (m TabBar) ActiveTabName() string {
	return tabNames[m.ActiveTab]
}

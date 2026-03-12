package vtab

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	voresky "github.com/dotBeeps/noms/internal/api/voresky"
	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// CharactersLoadedMsg is emitted when the character list is fetched successfully.
type CharactersLoadedMsg struct {
	Characters      []voresky.Character
	MainCharacterID string
}

// CharactersErrorMsg is emitted when the character list fetch fails.
type CharactersErrorMsg struct {
	Err error
}

// NavigateToCharacterMsg is emitted when the user selects a character.
type NavigateToCharacterMsg struct {
	CharacterID string
}

var (
	nameStyle = lipgloss.NewStyle().
			Foreground(theme.ColorPrimary).
			Bold(true)

	selectedNameStyle = lipgloss.NewStyle().
				Foreground(theme.ColorAccent).
				Bold(true)

	mainCharacterStyle = lipgloss.NewStyle().
				Foreground(theme.ColorAccent)

	statusActiveStyle = lipgloss.NewStyle().
				Foreground(theme.ColorSuccess)

	statusInactiveStyle = lipgloss.NewStyle().
				Foreground(theme.ColorMuted)

	universeStyle = lipgloss.NewStyle().
			Foreground(theme.ColorSecondary)

	descriptionStyle = lipgloss.NewStyle().
				Foreground(theme.ColorMuted)
)

// VoreskyModel is the BubbleTea model for the Voresky character list tab.
type VoreskyModel struct {
	client          *voresky.VoreskyClient
	characters      []voresky.Character
	mainCharacterID string
	selectedIndex   int
	loading         bool
	err             error
	width           int
	height          int
	offset          int
	spinner         spinner.Model
}

func NewVoreskyModel(client *voresky.VoreskyClient, width, height int) VoreskyModel {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.ColorAccent)),
	)
	return VoreskyModel{
		client:  client,
		width:   width,
		height:  height,
		loading: true,
		spinner: sp,
	}
}

// Init implements tea.Model.
func (m VoreskyModel) Init() tea.Cmd {
	return tea.Batch(m.fetchCharactersCmd, m.spinner.Tick)
}

func (m VoreskyModel) fetchCharactersCmd() tea.Msg {
	if m.client == nil {
		return CharactersErrorMsg{Err: fmt.Errorf("client not initialized")}
	}
	result, err := m.client.GetMyCharacters(context.Background())
	if err != nil {
		return CharactersErrorMsg{Err: err}
	}
	return CharactersLoadedMsg{
		Characters:      result.Characters,
		MainCharacterID: result.MainCharacterID,
	}
}

// Update implements tea.Model.
func (m VoreskyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureSelectedVisible()
		return m, nil

	case CharactersLoadedMsg:
		m.loading = false
		m.characters = msg.Characters
		m.mainCharacterID = msg.MainCharacterID
		return m, nil

	case CharactersErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case tea.MouseWheelMsg:
		if len(m.characters) == 0 {
			return m, nil
		}
		switch msg.Button {
		case tea.MouseWheelDown:
			m.selectedIndex = min(m.selectedIndex+3, len(m.characters)-1)
		case tea.MouseWheelUp:
			m.selectedIndex = max(m.selectedIndex-3, 0)
		}
		m.ensureSelectedVisible()
		return m, nil
	}

	return m, nil
}

func (m VoreskyModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.selectedIndex < len(m.characters)-1 {
			m.selectedIndex++
			m.ensureSelectedVisible()
		}
		return m, nil

	case "k", "up":
		if m.selectedIndex > 0 {
			m.selectedIndex--
			m.ensureSelectedVisible()
		}
		return m, nil

	case "enter":
		if m.selectedIndex < len(m.characters) {
			id := m.characters[m.selectedIndex].ID
			return m, func() tea.Msg { return NavigateToCharacterMsg{CharacterID: id} }
		}
		return m, nil

	case "r":
		m.loading = true
		m.characters = nil
		m.selectedIndex = 0
		m.offset = 0
		m.err = nil
		return m, tea.Batch(m.fetchCharactersCmd, m.spinner.Tick)
	}

	return m, nil
}

// View implements tea.Model.
func (m VoreskyModel) View() tea.View {
	var content strings.Builder

	content.WriteString(theme.StyleHeaderSubtle.Render("My Characters"))
	content.WriteString("\n")

	availableHeight := max(1, m.height-1)

	if m.loading && len(m.characters) == 0 {
		content.WriteString(lipgloss.Place(
			m.width,
			availableHeight,
			lipgloss.Center,
			lipgloss.Center,
			theme.StyleMuted.Render(m.spinner.View()+" Loading characters..."),
		))
		v := tea.NewView(content.String())
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if m.err != nil {
		content.WriteString(lipgloss.Place(
			m.width,
			availableHeight,
			lipgloss.Center,
			lipgloss.Center,
			theme.StyleError.Render(fmt.Sprintf("Error: %v\n\nPress 'r' to retry", m.err)),
		))
		v := tea.NewView(content.String())
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if len(m.characters) == 0 {
		content.WriteString(lipgloss.Place(
			m.width,
			availableHeight,
			lipgloss.Center,
			lipgloss.Center,
			theme.StyleMuted.Italic(true).Render("No characters found"),
		))
		v := tea.NewView(content.String())
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	var rendered string
	linesUsed := 0
	for i := m.offset; i < len(m.characters); i++ {
		char := m.renderCharacter(i, i == m.selectedIndex)
		rendered += char
		linesUsed += strings.Count(char, "\n")
		if linesUsed >= availableHeight {
			break
		}
	}
	content.WriteString(rendered)

	v := tea.NewView(content.String())
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m VoreskyModel) renderCharacter(index int, selected bool) string {
	var b strings.Builder
	c := m.characters[index]

	isMain := c.ID == m.mainCharacterID
	mainIndicator := ""
	if isMain {
		mainIndicator = " " + mainCharacterStyle.Render("★")
	}

	var nameStr string
	if selected {
		nameStr = selectedNameStyle.Render(c.Name)
	} else {
		nameStr = nameStyle.Render(c.Name)
	}

	var statusStr string
	switch c.Status {
	case "active":
		statusStr = statusActiveStyle.Render("● active")
	default:
		statusStr = statusInactiveStyle.Render("○ " + c.Status)
	}

	_, _ = fmt.Fprintf(&b, "%s%s  %s\n", nameStr, mainIndicator, statusStr)

	if c.FeaturedUniverse != "" {
		_, _ = fmt.Fprintf(&b, "  %s\n", universeStyle.Render(c.FeaturedUniverse))
	}

	if c.Description != "" {
		desc := truncateText(c.Description, 60)
		_, _ = fmt.Fprintf(&b, "  %s\n", descriptionStyle.Render(desc))
	}

	return shared.RenderItemWithBorder(b.String(), selected, m.width)
}

func (m *VoreskyModel) ensureSelectedVisible() {
	headerHeight := 1
	m.offset = shared.EnsureSelectedVisible(len(m.characters), m.selectedIndex, m.offset, m.height-headerHeight, func(index int) string {
		return m.renderCharacter(index, false)
	})
}

func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	truncated := string(runes[:maxLen])
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}
	return truncated + "..."
}

package vtab

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	voresky "github.com/dotBeeps/noms/internal/api/voresky"
	"github.com/dotBeeps/noms/internal/ui/images"
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

func nameStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true)
}
func selectedNameStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorAccent).Bold(true)
}
func mainCharacterStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(theme.ColorAccent) }
func statusActiveStyle() lipgloss.Style  { return lipgloss.NewStyle().Foreground(theme.ColorSuccess) }
func statusInactiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorMuted)
}
func descriptionStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(theme.ColorMuted) }

// VoreskyModel is the BubbleTea model for the Voresky character list tab.
type VoreskyModel struct {
	client          *voresky.VoreskyClient
	characters      []voresky.Character
	mainCharacterID string
	loading         bool
	err             error
	width           int
	height          int
	imageCache      images.ImageRenderer
	lazyRenderer    *images.LazyRenderer
	spinner         spinner.Model
	keys            KeyMap
	viewport        shared.ItemViewport
}

func NewVoreskyModel(client *voresky.VoreskyClient, width, height int, imageCache images.ImageRenderer) VoreskyModel {
	sp := shared.NewNetworkSpinner()
	headerHeight := 1
	return VoreskyModel{
		client:       client,
		imageCache:   imageCache,
		lazyRenderer: &images.LazyRenderer{Inner: imageCache},
		width:        width,
		height:       height,
		loading:      true,
		spinner:      sp,
		keys:         DefaultKeyMap,
		viewport:     shared.NewItemViewport(width, max(1, height-headerHeight)),
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := m.client.GetMyCharacters(ctx)
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
	case images.ImageFetchedMsg:
		m.rebuildViewport()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetSize(msg.Width, max(1, msg.Height-1))
		m.rebuildViewport()
		return m, nil

	case CharactersLoadedMsg:
		m.loading = false
		m.characters = msg.Characters
		m.mainCharacterID = msg.MainCharacterID
		m.rebuildViewport()
		var cmds []tea.Cmd
		if m.imageCache != nil && m.imageCache.Enabled() {
			for _, char := range m.characters {
				if char.Avatar != "" {
					cmds = append(cmds, m.imageCache.FetchAvatar(char.Avatar))
				}
			}
		}
		return m, tea.Batch(cmds...)

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

	case shared.ScrollTickMsg:
		if m.viewport.UpdateSpring() {
			return m, m.viewport.SpringCmd()
		}
		return m, nil

	case tea.MouseWheelMsg:
		if len(m.characters) == 0 {
			return m, nil
		}
		moved := false
		switch msg.Button {
		case tea.MouseWheelDown:
			moved = m.viewport.MoveDownN(3)
		case tea.MouseWheelUp:
			moved = m.viewport.MoveUpN(3)
		}
		if moved {
			m.rebuildViewport()
		}
		return m, nil
	}

	return m, nil
}

func (m VoreskyModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	km := m.keys
	switch {
	case key.Matches(msg, km.Down):
		if m.viewport.MoveDown() {
			prev := m.viewport.YOffset()
			m.rebuildViewport()
			m.viewport.AnimateFrom(prev)
		}
		return m, m.viewport.SpringCmd()

	case key.Matches(msg, km.Up):
		if m.viewport.MoveUp() {
			prev := m.viewport.YOffset()
			m.rebuildViewport()
			m.viewport.AnimateFrom(prev)
		}
		return m, m.viewport.SpringCmd()

	case key.Matches(msg, km.Open):
		idx := m.viewport.SelectedIndex()
		if idx < len(m.characters) {
			id := m.characters[idx].ID
			return m, func() tea.Msg { return NavigateToCharacterMsg{CharacterID: id} }
		}
		return m, nil

	case msg.String() == "r":
		m.loading = true
		m.characters = nil
		m.err = nil
		m.viewport.Reset()
		return m, tea.Batch(m.fetchCharactersCmd, m.spinner.Tick)
	}

	return m, nil
}

// Keys returns the vtab key map for help rendering.
func (m VoreskyModel) Keys() KeyMap {
	return m.keys
}

// View implements tea.Model.
func (m VoreskyModel) View() tea.View {
	var content strings.Builder

	content.WriteString(theme.StyleHeaderSubtle().Render("My Characters"))
	content.WriteString("\n")

	availableHeight := max(1, m.height-1)

	if m.loading && len(m.characters) == 0 {
		content.WriteString(lipgloss.Place(
			m.width,
			availableHeight,
			lipgloss.Center,
			lipgloss.Center,
			theme.StyleMuted().Render(m.spinner.View()+" Loading characters..."),
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
			theme.StyleError().Render(fmt.Sprintf("Error: %v\n\nPress 'r' to retry", m.err)),
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
			theme.StyleMuted().Italic(true).Render("No characters found"),
		))
		v := tea.NewView(content.String())
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	content.WriteString(m.viewport.View())

	v := tea.NewView(content.String())
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m VoreskyModel) renderCharacter(index int, selected bool, renderer images.ImageRenderer) string {
	var b strings.Builder
	c := m.characters[index]

	isMain := c.ID == m.mainCharacterID
	mainIndicator := ""
	if isMain {
		mainIndicator = " " + mainCharacterStyle().Render("★")
	}

	var nameStr string
	if selected {
		nameStr = selectedNameStyle().Render(c.Name)
	} else {
		nameStr = nameStyle().Render(c.Name)
	}

	var statusStr string
	switch c.Status {
	case "active":
		statusStr = statusActiveStyle().Render("● active")
	default:
		statusStr = statusInactiveStyle().Render("○ " + c.Status)
	}

	var avatarBlock string
	if renderer != nil && renderer.Enabled() && c.Avatar != "" {
		if renderer.IsCached(c.Avatar) {
			avatarBlock = renderer.RenderImage(c.Avatar, shared.AvatarCols, shared.AvatarRows)
		} else {
			avatarBlock = shared.RenderPlaceholder(shared.AvatarCols, shared.AvatarRows)
		}
	}

	// RenderItemWithBorder overhead: border(1) + padding(1) = 2
	contentWidth := m.width - 2
	if avatarBlock != "" {
		contentWidth = max(10, m.width-2-shared.AvatarCols-1)
	}

	_, _ = fmt.Fprintf(&b, "%s%s  %s\n", nameStr, mainIndicator, statusStr)

	if c.FeaturedUniverse != "" {
		_, _ = fmt.Fprintf(&b, "  %s\n", shared.UniverseStyle().Render(c.FeaturedUniverse))
	}

	if c.Description != "" {
		desc := ansi.Truncate(c.Description, contentWidth-2, "…")
		_, _ = fmt.Fprintf(&b, "  %s\n", descriptionStyle().Render(desc))
	}

	textContent := lipgloss.NewStyle().Width(contentWidth).Render(b.String())

	if avatarBlock != "" {
		joined := shared.JoinWithGutter(avatarBlock, textContent, " ", shared.AvatarCols)
		return shared.RenderItemWithBorder(joined, selected, m.width)
	}
	return shared.RenderItemWithBorder(textContent, selected, m.width)
}

func (m *VoreskyModel) rebuildViewport() {
	m.viewport.SetItems(len(m.characters), func(index int, selected bool) string {
		m.lazyRenderer.NearVisible = m.viewport.IsNearVisible(index, m.viewport.Height())
		return m.renderCharacter(index, selected, m.lazyRenderer)
	})
}

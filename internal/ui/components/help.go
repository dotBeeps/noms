package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dotBeeps/noms/internal/ui/theme"
)

// Style factory functions — constructed on call so they always reflect the active theme.

func helpOverlayStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorSurface).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorPrimary)
}
func helpKeyStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorAccent).Bold(true)
}
func helpDescStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorMuted)
}
func helpSectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorMuted).Italic(true)
}

type KeyBinding struct {
	Key         string
	Description string
}

type HelpContext int

const (
	HelpContextLogin HelpContext = iota
	HelpContextMain
	HelpContextFeed
	HelpContextThread
	HelpContextProfile
	HelpContextNotifications
	HelpContextSearch
	HelpContextCompose
	HelpContextVoresky
	HelpContextVoreskyNotifications
)

var loginKeyBindings = []KeyBinding{
	{Key: "Enter", Description: "Submit handle"},
	{Key: "Tab", Description: "Switch focus"},
	{Key: "Ctrl+C", Description: "Quit"},
}

var globalKeyBindings = []KeyBinding{
	{Key: "1-6", Description: "Switch tabs"},
	{Key: "j/k", Description: "Navigate up/down"},
	{Key: "[ / ]", Description: "Previous/next theme"},
	{Key: "Ctrl+T", Description: "Theme picker"},
	{Key: "v", Description: "Voresky setup"},
	{Key: "?", Description: "Toggle help"},
	{Key: "q", Description: "Quit"},
	{Key: "Ctrl+C", Description: "Force quit"},
}

var feedKeyBindings = []KeyBinding{
	{Key: "Enter", Description: "Open thread"},
	{Key: "l", Description: "Like/unlike post"},
	{Key: "t", Description: "Repost/un-repost"},
	{Key: "r", Description: "Reply to post"},
	{Key: "c", Description: "Compose new post"},
	{Key: "d d", Description: "Delete your post"},
}

var threadKeyBindings = []KeyBinding{
	{Key: "Enter", Description: "Open thread"},
	{Key: "l", Description: "Like/unlike post"},
	{Key: "t", Description: "Repost/un-repost"},
	{Key: "r", Description: "Reply to post"},
	{Key: "p", Description: "View profile"},
	{Key: "d d", Description: "Delete your post"},
	{Key: "Esc", Description: "Back"},
}

var profileKeyBindings = []KeyBinding{
	{Key: "Enter", Description: "Open thread"},
	{Key: "f", Description: "Follow/unfollow"},
	{Key: "d d", Description: "Delete your post"},
	{Key: "r", Description: "Refresh"},
	{Key: "Esc", Description: "Back"},
}

var notificationsKeyBindings = []KeyBinding{
	{Key: "Enter", Description: "Open notification"},
	{Key: "r", Description: "Refresh & mark read"},
}

var searchKeyBindings = []KeyBinding{
	{Key: "/", Description: "Focus search input"},
	{Key: "Tab", Description: "Toggle posts/people"},
	{Key: "Enter", Description: "Select result"},
}

var composeKeyBindings = []KeyBinding{
	{Key: "Ctrl+Enter", Description: "Submit post"},
	{Key: "Esc", Description: "Cancel"},
}

type HelpModel struct {
	Visible bool
	Width   int
	Height  int
	Context HelpContext
}

func NewHelpModel() HelpModel {
	return HelpModel{
		Context: HelpContextLogin,
	}
}

func (m HelpModel) Init() tea.Cmd {
	return nil
}

func (m HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}
	return m, nil
}

func (m HelpModel) View() tea.View {
	if !m.Visible {
		return tea.NewView("")
	}

	var viewBindings []KeyBinding
	var globals []KeyBinding
	var title string

	switch m.Context {
	case HelpContextLogin:
		viewBindings = loginKeyBindings
		title = "Login Help"
	case HelpContextMain, HelpContextFeed:
		viewBindings = feedKeyBindings
		globals = globalKeyBindings
		title = "Feed"
	case HelpContextThread:
		viewBindings = threadKeyBindings
		globals = globalKeyBindings
		title = "Thread"
	case HelpContextProfile:
		viewBindings = profileKeyBindings
		globals = globalKeyBindings
		title = "Profile"
	case HelpContextNotifications:
		viewBindings = notificationsKeyBindings
		globals = globalKeyBindings
		title = "Notifications"
	case HelpContextSearch:
		viewBindings = searchKeyBindings
		globals = globalKeyBindings
		title = "Search"
	case HelpContextCompose:
		viewBindings = composeKeyBindings
		title = "Compose"
	case HelpContextVoresky:
		viewBindings = []KeyBinding{
			{Key: "j/k", Description: "Navigate characters"},
			{Key: "enter", Description: "View character"},
		}
		globals = globalKeyBindings
		title = "Voresky"
	case HelpContextVoreskyNotifications:
		viewBindings = []KeyBinding{
			{Key: "j/k", Description: "Navigate notifications"},
			{Key: "r", Description: "Mark as read"},
		}
		globals = globalKeyBindings
		title = "Voresky Notifications"
	default:
		viewBindings = globalKeyBindings
		title = "Keyboard Shortcuts"
	}

	allBindings := make([]KeyBinding, 0, len(viewBindings)+len(globals))
	allBindings = append(allBindings, viewBindings...)
	allBindings = append(allBindings, globals...)

	maxKeyLen := 0
	for _, kb := range allBindings {
		if len(kb.Key) > maxKeyLen {
			maxKeyLen = len(kb.Key)
		}
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorPrimary).Render(title))
	lines = append(lines, "")

	for _, kb := range viewBindings {
		key := helpKeyStyle().Render(fmt.Sprintf("%-*s", maxKeyLen, kb.Key))
		desc := helpDescStyle().Render(kb.Description)
		lines = append(lines, fmt.Sprintf("  %s  %s", key, desc))
	}

	if len(globals) > 0 {
		lines = append(lines, "")
		lines = append(lines, helpSectionStyle().Render("  ── Global ──"))
		lines = append(lines, "")
		for _, kb := range globals {
			key := helpKeyStyle().Render(fmt.Sprintf("%-*s", maxKeyLen, kb.Key))
			desc := helpDescStyle().Render(kb.Description)
			lines = append(lines, fmt.Sprintf("  %s  %s", key, desc))
		}
	}

	content := strings.Join(lines, "\n")
	rendered := helpOverlayStyle().Render(content)

	return tea.NewView(rendered)
}

func (m *HelpModel) Toggle() {
	m.Visible = !m.Visible
}

func (m *HelpModel) SetContext(ctx HelpContext) {
	m.Context = ctx
}

func (m *HelpModel) Show() {
	m.Visible = true
}

func (m *HelpModel) Hide() {
	m.Visible = false
}

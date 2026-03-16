package components

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dotBeeps/noms/internal/ui/theme"
)

func helpOverlayStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorSurface).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorPrimary)
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

// keyMapAdapter implements help.KeyMap from two slices of KeyBinding.
type keyMapAdapter struct {
	view   []key.Binding
	global []key.Binding
}

func (k keyMapAdapter) ShortHelp() []key.Binding { return k.view }
func (k keyMapAdapter) FullHelp() [][]key.Binding {
	if len(k.global) == 0 {
		return [][]key.Binding{k.view}
	}
	return [][]key.Binding{k.view, k.global}
}

func toKeyBindings(kbs []KeyBinding) []key.Binding {
	out := make([]key.Binding, len(kbs))
	for i, kb := range kbs {
		out[i] = key.NewBinding(key.WithKeys(kb.Key), key.WithHelp(kb.Key, kb.Description))
	}
	return out
}

type HelpModel struct {
	Visible bool
	Width   int
	Height  int
	Context HelpContext
	helper  help.Model
}

func NewHelpModel() HelpModel {
	h := help.New()
	h.ShowAll = true
	return HelpModel{
		Context: HelpContextLogin,
		helper:  h,
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

	km := keyMapAdapter{
		view:   toKeyBindings(viewBindings),
		global: toKeyBindings(globals),
	}

	heading := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorPrimary).Render(title)
	body := m.helper.View(km)
	rendered := helpOverlayStyle().Render(heading + "\n\n" + body)

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

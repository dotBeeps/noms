package vnotifications

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

// VNotificationsLoadedMsg is emitted when the notification list is fetched successfully.
type VNotificationsLoadedMsg struct {
	Notifications []voresky.Notification
	Cursor        string
}

// VNotificationsErrorMsg is emitted when the notification list fetch fails.
type VNotificationsErrorMsg struct {
	Err error
}

// VNotifUnreadCountMsg is emitted when the unread count is fetched.
type VNotifUnreadCountMsg struct {
	Count int
}

// NavigateToNotificationMsg is emitted when the user selects a notification.
type NavigateToNotificationMsg struct {
	Notification voresky.Notification
}

var (
	unreadDotStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent)

	sourceNameStyle = lipgloss.NewStyle().
			Foreground(theme.ColorPrimary).
			Bold(true)

	targetNameStyle = lipgloss.NewStyle().
			Foreground(theme.ColorSecondary).
			Bold(true)

	universeStyle = lipgloss.NewStyle().
			Foreground(theme.ColorSecondary)

	timeStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted)

	pokeStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent)

	stalkStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted)

	housingStyle = lipgloss.NewStyle().
			Foreground(theme.ColorSuccess)

	interactionStyle = lipgloss.NewStyle().
				Foreground(theme.ColorError)

	collarStyle = lipgloss.NewStyle().
			Foreground(theme.ColorPrimary)
)

// VNotificationsModel is the BubbleTea model for the Voresky notifications tab.
type VNotificationsModel struct {
	client        *voresky.VoreskyClient
	notifications []voresky.Notification
	selectedIndex int
	cursor        string
	loading       bool
	unreadCount   int
	err           error
	width         int
	height        int
	offset        int
	spinner       spinner.Model
}

func NewVNotificationsModel(client *voresky.VoreskyClient, width, height int) VNotificationsModel {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.ColorAccent)),
	)
	return VNotificationsModel{
		client:        client,
		width:         width,
		height:        height,
		notifications: make([]voresky.Notification, 0),
		loading:       true,
		spinner:       sp,
	}
}

// Init implements tea.Model.
func (m VNotificationsModel) Init() tea.Cmd {
	return tea.Batch(
		m.fetchNotificationsCmd,
		m.fetchUnreadCountCmd,
		m.spinner.Tick,
	)
}

func (m VNotificationsModel) fetchNotificationsCmd() tea.Msg {
	if m.client == nil {
		return VNotificationsErrorMsg{Err: fmt.Errorf("client not initialized")}
	}
	notifications, cursor, err := m.client.GetNotifications(context.Background(), 50, m.cursor)
	if err != nil {
		return VNotificationsErrorMsg{Err: err}
	}
	return VNotificationsLoadedMsg{
		Notifications: notifications,
		Cursor:        cursor,
	}
}

func (m VNotificationsModel) fetchUnreadCountCmd() tea.Msg {
	if m.client == nil {
		return VNotifUnreadCountMsg{Count: 0}
	}
	count, err := m.client.GetUnreadNotificationCount(context.Background())
	if err != nil {
		return VNotifUnreadCountMsg{Count: 0}
	}
	return VNotifUnreadCountMsg{Count: count}
}

func (m VNotificationsModel) markSelectedReadCmd() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil || m.selectedIndex >= len(m.notifications) {
			return nil
		}
		n := m.notifications[m.selectedIndex]
		if n.IsRead {
			return nil
		}
		if err := m.client.MarkNotificationsRead(context.Background(), []string{n.ID}); err != nil {
			return VNotificationsErrorMsg{Err: err}
		}
		return VNotifUnreadCountMsg{Count: max(0, m.unreadCount-1)}
	}
}

// Update implements tea.Model.
func (m VNotificationsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case VNotificationsLoadedMsg:
		m.loading = false
		m.cursor = msg.Cursor
		if m.notifications == nil {
			m.notifications = msg.Notifications
		} else {
			m.notifications = append(m.notifications, msg.Notifications...)
		}
		return m, nil

	case VNotificationsErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case VNotifUnreadCountMsg:
		m.unreadCount = msg.Count
		if m.selectedIndex < len(m.notifications) {
			m.notifications[m.selectedIndex].IsRead = true
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.MouseWheelMsg:
		mouse := msg.Mouse()
		switch mouse.Button {
		case tea.MouseWheelDown:
			moved := false
			for range 3 {
				if m.selectedIndex < len(m.notifications)-1 {
					m.selectedIndex++
					moved = true
				}
			}
			if moved {
				m.ensureSelectedVisible()
			}
			if m.selectedIndex >= len(m.notifications)-3 && m.cursor != "" && !m.loading {
				m.loading = true
				return m, tea.Batch(m.fetchNotificationsCmd, m.spinner.Tick)
			}
		case tea.MouseWheelUp:
			moved := false
			for range 3 {
				if m.selectedIndex > 0 {
					m.selectedIndex--
					moved = true
				}
			}
			if moved {
				m.ensureSelectedVisible()
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}
func (m *VNotificationsModel) ensureSelectedVisible() {
	headerHeight := 2
	m.offset = shared.EnsureSelectedVisible(len(m.notifications), m.selectedIndex, m.offset, m.height-headerHeight, func(index int) string {
		return m.renderNotification(index, false)
	})
}

func (m VNotificationsModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.selectedIndex < len(m.notifications)-1 {
			m.selectedIndex++
			m.ensureSelectedVisible()
			if m.selectedIndex >= len(m.notifications)-3 && m.cursor != "" && !m.loading {
				m.loading = true
				return m, tea.Batch(m.fetchNotificationsCmd, m.spinner.Tick)
			}
		}
		return m, nil

	case "k", "up":
		if m.selectedIndex > 0 {
			m.selectedIndex--
			m.ensureSelectedVisible()
		}
		return m, nil

	case "r":
		m.loading = true
		m.cursor = ""
		m.notifications = nil
		m.selectedIndex = 0
		m.offset = 0
		m.ensureSelectedVisible()
		m.err = nil
		return m, tea.Batch(m.fetchNotificationsCmd, m.fetchUnreadCountCmd, m.spinner.Tick)

	case "enter":
		if m.selectedIndex < len(m.notifications) {
			n := m.notifications[m.selectedIndex]
			return m, func() tea.Msg { return NavigateToNotificationMsg{Notification: n} }
		}
		return m, nil

	case "m":
		return m, m.markSelectedReadCmd()
	}

	return m, nil
}

// View implements tea.Model.
func (m VNotificationsModel) View() tea.View {
	var content strings.Builder

	header := "Voresky Notifications"
	if m.unreadCount > 0 {
		header = fmt.Sprintf("Voresky Notifications (%d unread)", m.unreadCount)
	}
	content.WriteString(theme.StyleHeaderSubtle.Render(header))
	content.WriteString("\n\n")
	headerHeight := strings.Count(content.String(), "\n")
	availableHeight := max(1, m.height-headerHeight)

	if m.loading && len(m.notifications) == 0 {
		content.WriteString(lipgloss.Place(
			m.width,
			availableHeight,
			lipgloss.Center,
			lipgloss.Center,
			theme.StyleMuted.Render(m.spinner.View()+" Loading notifications..."),
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

	if len(m.notifications) == 0 {
		content.WriteString(lipgloss.Place(
			m.width,
			availableHeight,
			lipgloss.Center,
			lipgloss.Center,
			theme.StyleMuted.Italic(true).Render("No Voresky notifications"),
		))
		v := tea.NewView(content.String())
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	var rendered string
	linesUsed := 0
	for i := m.offset; i < len(m.notifications); i++ {
		notif := m.renderNotification(i, i == m.selectedIndex)
		rendered += notif
		linesUsed += strings.Count(notif, "\n")
		if linesUsed >= availableHeight {
			break
		}
	}
	content.WriteString(rendered)

	if m.loading {
		content.WriteString(theme.StyleMuted.Render(m.spinner.View() + " Loading more..."))
	}

	v := tea.NewView(content.String())
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m VNotificationsModel) renderNotification(index int, selected bool) string {
	n := m.notifications[index]
	var b strings.Builder

	unreadDot := ""
	if !n.IsRead {
		unreadDot = unreadDotStyle.Render("●") + " "
	}

	icon, label, style := notifIcon(n.Type)
	iconStr := style.Render(icon)

	sourceName := ""
	if n.SourceCharacter != nil {
		sourceName = n.SourceCharacter.Name
	}
	targetName := ""
	if n.TargetCharacter != nil {
		targetName = n.TargetCharacter.Name
	}

	line := fmt.Sprintf("%s%s %s", unreadDot, iconStr, label)
	if sourceName != "" {
		line += "  " + sourceNameStyle.Render(sourceName)
	}
	if targetName != "" {
		line += " → " + targetNameStyle.Render(targetName)
	}
	b.WriteString(line + "\n")

	if n.Universe != "" {
		_, _ = fmt.Fprintf(&b, "  %s\n", universeStyle.Render(n.Universe))
	}

	ts := n.CreatedAt.Format("Jan 2, 2006 15:04")
	_, _ = fmt.Fprintf(&b, "  %s", timeStyle.Render(ts))

	result := b.String()
	return shared.RenderItemWithBorder(result, selected, m.width)
}

func notifIcon(t voresky.NotificationType) (icon, label string, style lipgloss.Style) {
	s := string(t)
	switch {
	case t == voresky.NotifPoke:
		return "👉", "poke", pokeStyle
	case t == voresky.NotifStalk || t == voresky.NotifStalkTargetAvailable:
		return "👀", "stalk", stalkStyle
	case strings.HasPrefix(s, "HOUSING_"):
		return "🏠", humanReadable(t), housingStyle
	case strings.HasPrefix(s, "INTERACTION_"):
		return "⚔️", humanReadable(t), interactionStyle
	case strings.HasPrefix(s, "COLLAR_"):
		return "🔗", humanReadable(t), collarStyle
	default:
		return "•", humanReadable(t), theme.StyleMuted
	}
}

func humanReadable(t voresky.NotificationType) string {
	s := string(t)
	for _, prefix := range []string{"HOUSING_", "INTERACTION_", "COLLAR_"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimPrefix(s, prefix)
			break
		}
	}
	s = strings.ReplaceAll(s, "_", " ")
	return strings.ToLower(s)
}

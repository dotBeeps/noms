package notifications

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"

	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/feed"
	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// Notification type constants
const (
	ReasonLike    = "like"
	ReasonRepost  = "repost"
	ReasonFollow  = "follow"
	ReasonMention = "mention"
	ReasonReply   = "reply"
	ReasonQuote   = "quote"
)

// Message types for Bubble Tea
type NotificationsLoadedMsg struct {
	Notifications []*bsky.NotificationListNotifications_Notification
	Cursor        string
}

type NotificationsErrorMsg struct {
	Err error
}

type UnreadCountMsg struct {
	Count int
}

type NavigateToPostMsg struct {
	URI string
}

type NavigateToProfileMsg struct {
	DID string
}

// Styles for notification rendering
var (
	unreadDotStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent)

	likeStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent)

	repostStyle = lipgloss.NewStyle().
			Foreground(theme.ColorSuccess)

	followStyle = lipgloss.NewStyle().
			Foreground(theme.ColorPrimary)

	mentionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("51")) // cyan

	replyStyle = lipgloss.NewStyle().
			Foreground(theme.ColorHighlight)

	quoteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")) // purple

	authorStyle = lipgloss.NewStyle().
			Foreground(theme.ColorPrimary).
			Bold(true)

	contentPreviewStyle = lipgloss.NewStyle().
				Foreground(theme.ColorMuted)

	timeStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted)
)

type notifGroup struct {
	reason  string
	subject string
	authors []string
	count   int
	allRead bool
	notif   *bsky.NotificationListNotifications_Notification
	preview string
}

type NotificationsModel struct {
	notifications []*bsky.NotificationListNotifications_Notification
	selected      int
	cursor        string
	loading       bool
	unreadCount   int
	width         int
	height        int
	offset        int
	client        bluesky.BlueskyClient
	err           error
	spinner       spinner.Model
	groups        []notifGroup
}

// NewNotificationsModel creates a new notifications model
func NewNotificationsModel(client bluesky.BlueskyClient, width, height int) NotificationsModel {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.ColorAccent)),
	)
	return NotificationsModel{
		client:        client,
		width:         width,
		height:        height,
		notifications: make([]*bsky.NotificationListNotifications_Notification, 0),
		loading:       true,
		spinner:       sp,
	}
}

// Init implements tea.Model
func (m NotificationsModel) Init() tea.Cmd {
	return tea.Batch(
		m.fetchNotificationsCmd,
		m.fetchUnreadCountCmd,
		m.spinner.Tick,
	)
}

func (m NotificationsModel) fetchNotificationsCmd() tea.Msg {
	if m.client == nil {
		return NotificationsErrorMsg{Err: fmt.Errorf("client not initialized")}
	}

	notifications, cursor, err := m.client.ListNotifications(context.Background(), m.cursor, 50)
	if err != nil {
		return NotificationsErrorMsg{Err: err}
	}

	return NotificationsLoadedMsg{
		Notifications: notifications,
		Cursor:        cursor,
	}
}

func (m NotificationsModel) fetchUnreadCountCmd() tea.Msg {
	if m.client == nil {
		return UnreadCountMsg{Count: 0}
	}

	count, err := m.client.GetUnreadCount(context.Background())
	if err != nil {
		return UnreadCountMsg{Count: 0}
	}

	return UnreadCountMsg{Count: count}
}

func (m NotificationsModel) markAsReadCmd() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return nil
		}

		err := m.client.MarkNotificationsRead(context.Background(), time.Now())
		if err != nil {
			return NotificationsErrorMsg{Err: err}
		}

		return UnreadCountMsg{Count: 0}
	}
}

// Update implements tea.Model
func (m NotificationsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case NotificationsLoadedMsg:
		m.loading = false
		m.cursor = msg.Cursor
		if m.notifications == nil {
			m.notifications = msg.Notifications
		} else {
			m.notifications = append(m.notifications, msg.Notifications...)
		}
		m.buildGroups()
		return m, nil

	case NotificationsErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case UnreadCountMsg:
		m.unreadCount = msg.Count
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
			for range 3 {
				if m.selected < len(m.groups)-1 {
					m.selected++
				}
			}
			m.ensureSelectedVisible()
			if m.selected >= len(m.groups)-3 && m.cursor != "" && !m.loading {
				m.loading = true
				return m, tea.Batch(m.fetchNotificationsCmd, m.spinner.Tick)
			}
		case tea.MouseWheelUp:
			for range 3 {
				if m.selected > 0 {
					m.selected--
				}
			}
			m.ensureSelectedVisible()
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m NotificationsModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.selected < len(m.groups)-1 {
			m.selected++
			m.ensureSelectedVisible()
			if m.selected >= len(m.groups)-3 && m.cursor != "" && !m.loading {
				m.loading = true
				return m, tea.Batch(m.fetchNotificationsCmd, m.spinner.Tick)
			}
		}
		return m, nil

	case "k", "up":
		if m.selected > 0 {
			m.selected--
			m.ensureSelectedVisible()
		}
		return m, nil

	case "enter":
		return m.handleNavigation()

	case "r":
		m.loading = true
		m.cursor = ""
		m.notifications = nil
		m.groups = nil
		m.selected = 0
		m.offset = 0
		return m, tea.Batch(
			m.fetchNotificationsCmd,
			m.markAsReadCmd(),
			m.spinner.Tick,
		)
	}

	return m, nil
}

func (m NotificationsModel) handleNavigation() (tea.Model, tea.Cmd) {
	if len(m.groups) == 0 || m.selected >= len(m.groups) {
		return m, nil
	}

	g := m.groups[m.selected]

	if g.reason == ReasonFollow && g.notif != nil && g.notif.Author != nil {
		return m, func() tea.Msg {
			return NavigateToProfileMsg{DID: g.notif.Author.Did}
		}
	}

	if g.subject != "" {
		return m, func() tea.Msg {
			return NavigateToPostMsg{URI: g.subject}
		}
	}

	if g.notif != nil && g.notif.Uri != "" {
		uri := g.notif.Uri
		return m, func() tea.Msg {
			return NavigateToPostMsg{URI: uri}
		}
	}

	return m, nil
}

func (m NotificationsModel) View() tea.View {
	var content strings.Builder

	header := "Notifications"
	if m.unreadCount > 0 {
		header = fmt.Sprintf("Notifications (%d unread)", m.unreadCount)
	}
	content.WriteString(theme.StyleHeaderSubtle.Render(header))
	content.WriteString("\n\n")
	headerHeight := 2

	mouseView := func(s string) tea.View {
		v := tea.NewView(s)
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if m.loading && len(m.notifications) == 0 {
		content.WriteString(lipgloss.Place(m.width, m.height-headerHeight, lipgloss.Center, lipgloss.Center, theme.StyleMuted.Render(m.spinner.View()+" Loading notifications...")))
		return mouseView(content.String())
	}

	if m.err != nil {
		content.WriteString(lipgloss.Place(m.width, m.height-headerHeight, lipgloss.Center, lipgloss.Center, theme.StyleError.Render(fmt.Sprintf("Error: %v", m.err))))
		return mouseView(content.String())
	}

	if len(m.groups) == 0 {
		emptyText := lipgloss.NewStyle().Foreground(theme.ColorMuted).Italic(true).Render("No notifications yet")
		content.WriteString(lipgloss.Place(m.width, m.height-headerHeight, lipgloss.Center, lipgloss.Center, emptyText))
		return mouseView(content.String())
	}

	availableHeight := max(0, m.height-headerHeight)
	var rendered string
	linesUsed := 0
	for i := m.offset; i < len(m.groups); i++ {
		group := m.renderGroup(i, i == m.selected)
		rendered += group
		linesUsed += strings.Count(group, "\n")
		if linesUsed >= availableHeight {
			break
		}
	}
	content.WriteString(rendered)

	if m.loading {
		content.WriteString("\n")
		content.WriteString(theme.StyleMuted.Render(m.spinner.View() + " Loading more..."))
	}

	return mouseView(content.String())
}

func (m *NotificationsModel) ensureSelectedVisible() {
	headerHeight := 2 // header line + blank line
	m.offset = shared.EnsureSelectedVisible(len(m.groups), m.selected, m.offset, m.height-headerHeight, func(index int) string {
		return m.renderGroup(index, false)
	})
}

func (m *NotificationsModel) buildGroups() {
	m.groups = nil
	seen := make(map[string]int)

	for _, notif := range m.notifications {
		if (notif.Reason == ReasonLike || notif.Reason == ReasonRepost) && notif.ReasonSubject != nil && *notif.ReasonSubject != "" {
			key := notif.Reason + ":" + *notif.ReasonSubject
			if idx, ok := seen[key]; ok {
				g := &m.groups[idx]
				g.count++
				if notif.Author != nil {
					name := notif.Author.Handle
					if notif.Author.DisplayName != nil && *notif.Author.DisplayName != "" {
						name = *notif.Author.DisplayName
					}
					g.authors = append(g.authors, name)
				}
				if !notif.IsRead {
					g.allRead = false
				}
				continue
			}
		}

		authorName := ""
		if notif.Author != nil {
			authorName = notif.Author.Handle
			if notif.Author.DisplayName != nil && *notif.Author.DisplayName != "" {
				authorName = *notif.Author.DisplayName
			}
		}

		subject := ""
		if notif.ReasonSubject != nil {
			subject = *notif.ReasonSubject
		}

		g := notifGroup{
			reason:  notif.Reason,
			subject: subject,
			authors: []string{authorName},
			count:   1,
			allRead: notif.IsRead,
			notif:   notif,
			preview: getContentPreview(notif),
		}

		if (notif.Reason == ReasonLike || notif.Reason == ReasonRepost) && subject != "" {
			seen[notif.Reason+":"+subject] = len(m.groups)
		}

		m.groups = append(m.groups, g)
	}
}

func (m NotificationsModel) renderGroup(index int, selected bool) string {
	var b strings.Builder
	g := m.groups[index]

	unreadDot := ""
	if !g.allRead {
		unreadDot = unreadDotStyle.Render("●") + " "
	}

	icon, action, style := getNotificationStyle(g.reason)

	authorDisplay := ""
	if len(g.authors) > 0 {
		authorDisplay = g.authors[0]
	}
	if g.count > 1 {
		authorDisplay = fmt.Sprintf("%s and %d others", authorDisplay, g.count-1)
	}

	authorStyled := authorStyle.Render(authorDisplay)
	actionLine := fmt.Sprintf("%s%s %s", unreadDot, style.Render(icon), action)
	b.WriteString(fmt.Sprintf("%s %s\n", authorStyled, actionLine))

	if g.preview != "" {
		b.WriteString(fmt.Sprintf("  %s\n", contentPreviewStyle.Render(g.preview)))
	}

	var timeStr string
	if g.notif != nil {
		if t, err := time.Parse(time.RFC3339, g.notif.IndexedAt); err == nil {
			timeStr = feed.FormatRelativeTime(t)
		} else {
			timeStr = g.notif.IndexedAt
		}
	}
	b.WriteString(fmt.Sprintf("  %s", timeStyle.Render(timeStr)))

	return shared.RenderItemWithBorder(b.String(), selected, m.width)
}

func getNotificationStyle(reason string) (icon, action string, style lipgloss.Style) {
	switch reason {
	case ReasonLike:
		return "♡", "liked your post", likeStyle
	case ReasonRepost:
		return "⟲", "reposted your post", repostStyle
	case ReasonFollow:
		return "+", "followed you", followStyle
	case ReasonMention:
		return "@", "mentioned you", mentionStyle
	case ReasonReply:
		return "💬", "replied to your post", replyStyle
	case ReasonQuote:
		return "❝", "quoted your post", quoteStyle
	default:
		return "•", reason, theme.StyleMuted
	}
}

func getContentPreview(notif *bsky.NotificationListNotifications_Notification) string {
	// Try to extract text content from the record
	if notif.Record == nil {
		return ""
	}

	// Type assert to different record types
	switch record := notif.Record.Val.(type) {
	case *bsky.FeedPost:
		return truncateText(record.Text, 50)
	}

	return ""
}

func truncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	// Find a good break point
	truncated := string(runes[:maxLen])
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}
	return truncated + "..."
}

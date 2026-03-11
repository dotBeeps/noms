package notifications

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"

	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/feed"
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
	notificationStyle = lipgloss.NewStyle().
				Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(theme.ColorHighlight).
			Bold(true).
			Padding(0, 1)

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

	loadingStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Padding(1, 2)

	emptyStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Padding(1, 2).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(theme.ColorError).
			Padding(1, 2)
)

// NotificationsModel is the Bubble Tea model for the notifications screen
type NotificationsModel struct {
	notifications []*bsky.NotificationListNotifications_Notification
	selected      int
	cursor        string
	loading       bool
	unreadCount   int
	width         int
	height        int
	client        bluesky.BlueskyClient
	err           error
}

// NewNotificationsModel creates a new notifications model
func NewNotificationsModel(client bluesky.BlueskyClient, width, height int) NotificationsModel {
	return NotificationsModel{
		client:        client,
		width:         width,
		height:        height,
		notifications: make([]*bsky.NotificationListNotifications_Notification, 0),
	}
}

// Init implements tea.Model
func (m NotificationsModel) Init() tea.Cmd {
	return tea.Batch(
		m.fetchNotificationsCmd,
		m.fetchUnreadCountCmd,
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
		return m, nil

	case NotificationsErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case UnreadCountMsg:
		m.unreadCount = msg.Count
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m NotificationsModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.selected < len(m.notifications)-1 {
			m.selected++
			// Load more when near bottom
			if m.selected >= len(m.notifications)-3 && m.cursor != "" && !m.loading {
				m.loading = true
				return m, m.fetchNotificationsCmd
			}
		}
		return m, nil

	case "k", "up":
		if m.selected > 0 {
			m.selected--
		}
		return m, nil

	case "enter":
		return m.handleNavigation()

	case "r":
		// Refresh notifications and mark as read
		m.loading = true
		m.cursor = ""
		m.notifications = nil
		m.selected = 0
		return m, tea.Batch(
			m.fetchNotificationsCmd,
			m.markAsReadCmd(),
		)
	}

	return m, nil
}

func (m NotificationsModel) handleNavigation() (tea.Model, tea.Cmd) {
	if len(m.notifications) == 0 || m.selected >= len(m.notifications) {
		return m, nil
	}

	notif := m.notifications[m.selected]

	// For follow notifications, navigate to profile
	if notif.Reason == ReasonFollow && notif.Author != nil {
		return m, func() tea.Msg {
			return NavigateToProfileMsg{DID: notif.Author.Did}
		}
	}

	// For other notification types, navigate to the referenced post
	var uri string
	if notif.ReasonSubject != nil && *notif.ReasonSubject != "" {
		uri = *notif.ReasonSubject
	} else if notif.Uri != "" {
		uri = notif.Uri
	}

	if uri != "" {
		return m, func() tea.Msg {
			return NavigateToPostMsg{URI: uri}
		}
	}

	return m, nil
}

// View implements tea.Model
func (m NotificationsModel) View() tea.View {
	var content strings.Builder

	// Header with unread count
	header := "Notifications"
	if m.unreadCount > 0 {
		header = fmt.Sprintf("Notifications (%d unread)", m.unreadCount)
	}
	content.WriteString(theme.StyleHeader.Render(header))
	content.WriteString("\n\n")

	// Handle loading state
	if m.loading && len(m.notifications) == 0 {
		content.WriteString(loadingStyle.Render("Loading notifications..."))
		return tea.NewView(content.String())
	}

	// Handle error state
	if m.err != nil {
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		return tea.NewView(content.String())
	}

	// Handle empty state
	if len(m.notifications) == 0 {
		content.WriteString(emptyStyle.Render("No notifications yet"))
		return tea.NewView(content.String())
	}

	// Render notifications
	for i, notif := range m.notifications {
		content.WriteString(m.renderNotification(notif, i == m.selected))
		content.WriteString("\n")
	}

	// Show loading indicator at bottom if loading more
	if m.loading {
		content.WriteString(loadingStyle.Render("Loading more..."))
	}

	return tea.NewView(content.String())
}

func (m NotificationsModel) renderNotification(notif *bsky.NotificationListNotifications_Notification, selected bool) string {
	var b strings.Builder

	// Selection indicator
	indicator := "  "
	if selected {
		indicator = "▶ "
	}

	// Unread indicator
	unreadDot := ""
	if !notif.IsRead {
		unreadDot = unreadDotStyle.Render("●") + " "
	}

	// Icon and action based on notification type
	icon, action, style := getNotificationStyle(notif.Reason)

	// Author display name
	if notif.Author == nil {
		return notificationStyle.Render(indicator + style.Render(icon) + " " + action)
	}
	authorName := notif.Author.Handle
	if notif.Author.DisplayName != nil && *notif.Author.DisplayName != "" {
		authorName = *notif.Author.DisplayName
	}

	// First line: icon + author + action
	authorStyled := authorStyle.Render(authorName)
	actionLine := fmt.Sprintf("%s%s %s", unreadDot, style.Render(icon), action)
	b.WriteString(fmt.Sprintf("%s%s %s\n", indicator, authorStyled, actionLine))

	// Second line: content preview (if applicable)
	preview := getContentPreview(notif)
	if preview != "" {
		b.WriteString(fmt.Sprintf("%s    %s\n", indicator, contentPreviewStyle.Render(preview)))
	}

	// Third line: timestamp
	var timeStr string
	if t, err := time.Parse(time.RFC3339, notif.IndexedAt); err == nil {
		timeStr = feed.FormatRelativeTime(t)
	} else {
		timeStr = notif.IndexedAt
	}
	b.WriteString(fmt.Sprintf("%s    %s", indicator, timeStyle.Render(timeStr)))

	// Apply selection style
	result := b.String()
	if selected {
		result = selectedStyle.Render(result)
	} else {
		result = notificationStyle.Render(result)
	}

	return result
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

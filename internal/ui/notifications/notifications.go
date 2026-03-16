package notifications

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

	bsky "github.com/bluesky-social/indigo/api/bsky"

	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/feed"
	"github.com/dotBeeps/noms/internal/ui/images"
	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// Notification type constants
const (
	ReasonLike              = "like"
	ReasonRepost            = "repost"
	ReasonFollow            = "follow"
	ReasonMention           = "mention"
	ReasonReply             = "reply"
	ReasonQuote             = "quote"
	ReasonLikeViaRepost     = "like-via-repost"
	ReasonRepostViaRepost   = "repost-via-repost"
	ReasonStarterpackJoined = "starterpack-joined"
	ReasonVerified          = "verified"
	ReasonUnverified        = "unverified"
	ReasonSubscribedPost    = "subscribed-post"
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

// Style factory functions — constructed on call so they always reflect the active theme.

func likeStyle() lipgloss.Style    { return lipgloss.NewStyle().Foreground(theme.ColorAccent) }
func repostStyle() lipgloss.Style  { return lipgloss.NewStyle().Foreground(theme.ColorSuccess) }
func followStyle() lipgloss.Style  { return lipgloss.NewStyle().Foreground(theme.ColorPrimary) }
func mentionStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(theme.ColorMention) }
func replyStyle() lipgloss.Style   { return lipgloss.NewStyle().Foreground(theme.ColorHighlight) }
func quoteStyle() lipgloss.Style   { return lipgloss.NewStyle().Foreground(theme.ColorTag) }
func authorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true)
}
func contentPreviewStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(theme.ColorMuted) }

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
	cursor        string
	loading       bool
	unreadCount   int
	width         int
	height        int
	client        bluesky.BlueskyClient
	imageCache    images.ImageRenderer
	err           error
	spinner       spinner.Model
	groups        []notifGroup
	keys          KeyMap
	viewport      shared.ItemViewport
}

// NewNotificationsModel creates a new notifications model
func NewNotificationsModel(client bluesky.BlueskyClient, width, height int, imageCache images.ImageRenderer) NotificationsModel {
	sp := shared.NewSpinner()
	headerHeight := 2
	return NotificationsModel{
		client:        client,
		imageCache:    imageCache,
		width:         width,
		height:        height,
		notifications: make([]*bsky.NotificationListNotifications_Notification, 0),
		loading:       true,
		spinner:       sp,
		keys:          DefaultKeyMap,
		viewport:      shared.NewItemViewport(width, max(1, height-headerHeight)),
	}
}

// Init implements tea.Model
func (m NotificationsModel) Init() tea.Cmd {
	return tea.Batch(
		m.fetchNotificationsCmd(),
		m.fetchUnreadCountCmd(),
		m.spinner.Tick,
	)
}

func (m NotificationsModel) fetchNotificationsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return NotificationsErrorMsg{Err: fmt.Errorf("client not initialized")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		notifications, cursor, err := m.client.ListNotifications(ctx, m.cursor, 50)
		if err != nil {
			return NotificationsErrorMsg{Err: err}
		}
		return NotificationsLoadedMsg{
			Notifications: notifications,
			Cursor:        cursor,
		}
	}
}

func (m NotificationsModel) fetchUnreadCountCmd() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return UnreadCountMsg{Count: 0}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		count, err := m.client.GetUnreadCount(ctx)
		if err != nil {
			return UnreadCountMsg{Count: 0}
		}
		return UnreadCountMsg{Count: count}
	}
}

func (m NotificationsModel) markAsReadCmd() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := m.client.MarkNotificationsRead(ctx, time.Now())
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
	case images.ImageFetchedMsg:
		m.rebuildViewport()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetSize(msg.Width, max(1, msg.Height-2))
		m.rebuildViewport()
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
		m.rebuildViewport()
		if m.imageCache != nil && m.imageCache.Enabled() {
			for _, notif := range msg.Notifications {
				if notif.Author != nil && notif.Author.Avatar != nil {
					cmds = append(cmds, m.imageCache.FetchAvatar(*notif.Author.Avatar))
				}
			}
		}
		return m, tea.Batch(cmds...)

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
			if m.viewport.MoveDownN(3) {
				m.rebuildViewport()
			}
			if m.viewport.NearBottom(shared.PaginationThreshold) && m.cursor != "" && !m.loading {
				m.loading = true
				return m, tea.Batch(m.fetchNotificationsCmd(), m.spinner.Tick)
			}
		case tea.MouseWheelUp:
			if m.viewport.MoveUpN(3) {
				m.rebuildViewport()
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m NotificationsModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	km := m.keys
	switch {
	case key.Matches(msg, km.Down):
		if m.viewport.MoveDown() {
			m.rebuildViewport()
			if m.viewport.NearBottom(shared.PaginationThreshold) && m.cursor != "" && !m.loading {
				m.loading = true
				return m, tea.Batch(m.fetchNotificationsCmd(), m.spinner.Tick)
			}
		}
		return m, nil

	case key.Matches(msg, km.Up):
		if m.viewport.MoveUp() {
			m.rebuildViewport()
		}
		return m, nil

	case key.Matches(msg, km.Open):
		return m.handleNavigation()

	case key.Matches(msg, km.Refresh):
		m.loading = true
		m.cursor = ""
		m.notifications = nil
		m.groups = nil
		m.viewport.Reset()
		if m.imageCache != nil {
			m.imageCache.InvalidateTransmissions()
		}
		return m, tea.Batch(
			m.fetchNotificationsCmd(),
			m.markAsReadCmd(),
			m.spinner.Tick,
		)
	}

	return m, nil
}

// Keys returns the key map for this model.
func (m NotificationsModel) Keys() KeyMap { return m.keys }

func (m NotificationsModel) handleNavigation() (tea.Model, tea.Cmd) {
	idx := m.viewport.SelectedIndex()
	if len(m.groups) == 0 || idx >= len(m.groups) {
		return m, nil
	}

	g := m.groups[idx]

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
	content.WriteString(theme.StyleHeaderSubtle().Render(header))
	content.WriteString("\n\n")
	headerHeight := 2

	mouseView := func(s string) tea.View {
		v := tea.NewView(s)
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if m.loading && len(m.notifications) == 0 {
		content.WriteString(lipgloss.Place(m.width, m.height-headerHeight, lipgloss.Center, lipgloss.Center, theme.StyleMuted().Render(m.spinner.View()+" Loading notifications...")))
		return mouseView(content.String())
	}

	if m.err != nil {
		content.WriteString(lipgloss.Place(m.width, m.height-headerHeight, lipgloss.Center, lipgloss.Center, theme.StyleError().Render(fmt.Sprintf("Error: %v", m.err))))
		return mouseView(content.String())
	}

	if len(m.groups) == 0 {
		emptyText := lipgloss.NewStyle().Foreground(theme.ColorMuted).Italic(true).Render("No notifications yet")
		content.WriteString(lipgloss.Place(m.width, m.height-headerHeight, lipgloss.Center, lipgloss.Center, emptyText))
		return mouseView(content.String())
	}

	content.WriteString(m.viewport.View())

	if m.loading {
		content.WriteString("\n")
		content.WriteString(theme.StyleMuted().Render(m.spinner.View() + " Loading more..."))
	}

	return mouseView(content.String())
}

func (m *NotificationsModel) rebuildViewport() {
	lazy := &images.LazyRenderer{Inner: m.imageCache}
	m.viewport.SetItems(len(m.groups), func(index int, selected bool) string {
		lazy.NearVisible = m.viewport.IsNearVisible(index, m.viewport.Height())
		return m.renderGroup(index, selected, lazy)
	})
}

func (m *NotificationsModel) buildGroups() {
	m.groups = nil
	seen := make(map[string]int)

	for _, notif := range m.notifications {
		isGroupable := notif.Reason == ReasonLike || notif.Reason == ReasonRepost ||
			notif.Reason == ReasonLikeViaRepost || notif.Reason == ReasonRepostViaRepost
		if isGroupable && notif.ReasonSubject != nil && *notif.ReasonSubject != "" {
			key := notif.Reason + ":" + *notif.ReasonSubject
			if idx, ok := seen[key]; ok {
				g := &m.groups[idx]
				g.count++
				if notif.Author != nil {
					name := notif.Author.Handle
					if notif.Author.DisplayName != nil && *notif.Author.DisplayName != "" {
						name = ansi.Strip(*notif.Author.DisplayName)
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
				authorName = ansi.Strip(*notif.Author.DisplayName)
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

		if isGroupable && subject != "" {
			seen[notif.Reason+":"+subject] = len(m.groups)
		}

		m.groups = append(m.groups, g)
	}
}

func (m NotificationsModel) renderGroup(index int, selected bool, renderer images.ImageRenderer) string {
	var b strings.Builder
	g := m.groups[index]

	unreadDot := ""
	if !g.allRead {
		unreadDot = theme.NotifUnreadDotStyle().Render("●") + " "
	}

	icon, action, style := getNotificationStyle(g.reason)

	authorDisplay := ""
	if len(g.authors) > 0 {
		authorDisplay = g.authors[0]
	}
	if g.count > 1 {
		authorDisplay = fmt.Sprintf("%s and %d others", authorDisplay, g.count-1)
	}

	authorStyled := authorStyle().Render(authorDisplay)
	actionLine := fmt.Sprintf("%s%s %s", unreadDot, style.Render(icon), action)
	_, _ = fmt.Fprintf(&b, "%s %s\n", authorStyled, actionLine)

	if g.preview != "" {
		_, _ = fmt.Fprintf(&b, "  %s\n", contentPreviewStyle().Render(g.preview))
	}

	var timeStr string
	if g.notif != nil {
		if t, err := time.Parse(time.RFC3339, g.notif.IndexedAt); err == nil {
			timeStr = feed.FormatRelativeTime(t)
		} else {
			timeStr = g.notif.IndexedAt
		}
	}
	_, _ = fmt.Fprintf(&b, "  %s", theme.StyleMuted().Render(timeStr))

	contentStr := b.String()

	var avatarBlock string
	if renderer != nil && renderer.Enabled() && g.notif != nil && g.notif.Author != nil && g.notif.Author.Avatar != nil {
		avatarURL := *g.notif.Author.Avatar
		if renderer.IsCached(avatarURL) {
			avatarBlock = renderer.RenderImage(avatarURL, shared.AvatarCols, shared.AvatarRows)
		} else {
			avatarBlock = shared.RenderPlaceholder(shared.AvatarCols, shared.AvatarRows)
		}
	}

	var finalContent string
	if avatarBlock != "" {
		avatarContentWidth := max(10, m.width-2-shared.AvatarCols-1)
		wrappedContent := lipgloss.NewStyle().Width(avatarContentWidth).Render(contentStr)
		finalContent = shared.JoinWithGutter(avatarBlock, wrappedContent, " ", shared.AvatarCols)
	} else {
		finalContent = contentStr
	}

	return shared.RenderItemWithBorder(finalContent, selected, m.width)
}

func getNotificationStyle(reason string) (icon, action string, style lipgloss.Style) {
	switch reason {
	case ReasonLike:
		return "♡", "liked your post", likeStyle()
	case ReasonRepost:
		return "⟲", "reposted your post", repostStyle()
	case ReasonFollow:
		return "+", "followed you", followStyle()
	case ReasonMention:
		return "@", "mentioned you", mentionStyle()
	case ReasonReply:
		return "💬", "replied to your post", replyStyle()
	case ReasonQuote:
		return "❝", "quoted your post", quoteStyle()
	case ReasonLikeViaRepost:
		return "♡", "liked your repost", likeStyle()
	case ReasonRepostViaRepost:
		return "⟲", "reposted your repost", repostStyle()
	case ReasonStarterpackJoined:
		return "+", "joined your starter pack", followStyle()
	case ReasonVerified:
		return "✓", "verified you", followStyle()
	case ReasonUnverified:
		return "✗", "removed your verification", theme.StyleMuted()
	case ReasonSubscribedPost:
		return "🔔", "subscribed post was updated", mentionStyle()
	default:
		return "•", reason, theme.StyleMuted()
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
		return shared.TruncateStr(ansi.Strip(record.Text), 50)
	}

	return ""
}

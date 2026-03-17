package vnotifications

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	voresky "github.com/dotBeeps/noms/internal/api/voresky"
	"github.com/dotBeeps/noms/internal/ui/images"
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

func pokeStyle() lipgloss.Style        { return lipgloss.NewStyle().Foreground(theme.ColorAccent) }
func stalkStyle() lipgloss.Style       { return lipgloss.NewStyle().Foreground(theme.ColorMuted) }
func housingStyle() lipgloss.Style     { return lipgloss.NewStyle().Foreground(theme.ColorSuccess) }
func interactionStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(theme.ColorError) }
func collarStyle() lipgloss.Style      { return lipgloss.NewStyle().Foreground(theme.ColorPrimary) }

// VNotificationsModel is the BubbleTea model for the Voresky notifications tab.
type VNotificationsModel struct {
	client        *voresky.VoreskyClient
	notifications []voresky.Notification
	cursor        string
	loading       bool
	unreadCount   int
	err           error
	width         int
	height        int
	imageCache    images.ImageRenderer
	spinner       spinner.Model
	keys          KeyMap
	viewport      shared.ItemViewport
}

func NewVNotificationsModel(client *voresky.VoreskyClient, width, height int, imageCache images.ImageRenderer) VNotificationsModel {
	sp := shared.NewSpinner()
	headerHeight := 2
	return VNotificationsModel{
		client:        client,
		imageCache:    imageCache,
		width:         width,
		height:        height,
		notifications: make([]voresky.Notification, 0),
		loading:       true,
		spinner:       sp,
		keys:          DefaultKeyMap,
		viewport:      shared.NewItemViewport(width, max(1, height-headerHeight)),
	}
}

// Init implements tea.Model.
func (m VNotificationsModel) Init() tea.Cmd {
	return tea.Batch(
		m.fetchNotificationsCmd(),
		m.fetchUnreadCountCmd(),
		m.spinner.Tick,
	)
}

func (m VNotificationsModel) fetchNotificationsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return VNotificationsErrorMsg{Err: fmt.Errorf("client not initialized")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		notifications, cursor, err := m.client.GetNotifications(ctx, 50, m.cursor)
		if err != nil {
			return VNotificationsErrorMsg{Err: err}
		}
		return VNotificationsLoadedMsg{
			Notifications: notifications,
			Cursor:        cursor,
		}
	}
}

func (m VNotificationsModel) fetchUnreadCountCmd() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return VNotifUnreadCountMsg{Count: 0}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		count, err := m.client.GetUnreadNotificationCount(ctx)
		if err != nil {
			return VNotifUnreadCountMsg{Count: 0}
		}
		return VNotifUnreadCountMsg{Count: count}
	}
}

func (m VNotificationsModel) markSelectedReadCmd() tea.Cmd {
	return func() tea.Msg {
		idx := m.viewport.SelectedIndex()
		if m.client == nil || idx >= len(m.notifications) {
			return nil
		}
		n := m.notifications[idx]
		if n.IsRead {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := m.client.MarkNotificationsRead(ctx, []string{n.ID}); err != nil {
			return VNotificationsErrorMsg{Err: err}
		}
		return VNotifUnreadCountMsg{Count: max(0, m.unreadCount-1)}
	}
}

// Update implements tea.Model.
func (m VNotificationsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case VNotificationsLoadedMsg:
		m.loading = false
		m.cursor = msg.Cursor
		if m.notifications == nil {
			m.notifications = msg.Notifications
		} else {
			m.notifications = append(m.notifications, msg.Notifications...)
		}

		var cmds []tea.Cmd
		if m.imageCache != nil && m.imageCache.Enabled() {
			for _, notif := range msg.Notifications {
				if notif.SourceCharacter != nil && notif.SourceCharacter.Avatar.URL != "" {
					cmds = append(cmds, m.imageCache.FetchAvatar(notif.SourceCharacter.Avatar.URL))
				}
				if notif.TargetCharacter != nil && notif.TargetCharacter.Avatar.URL != "" {
					cmds = append(cmds, m.imageCache.FetchAvatar(notif.TargetCharacter.Avatar.URL))
				}
			}
		}

		m.rebuildViewport()
		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil

	case VNotificationsErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case VNotifUnreadCountMsg:
		m.unreadCount = msg.Count
		idx := m.viewport.SelectedIndex()
		if idx < len(m.notifications) {
			m.notifications[idx].IsRead = true
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case shared.ScrollTickMsg:
		if m.viewport.UpdateSpring() {
			return m, m.viewport.SpringCmd()
		}
		return m, nil

	case tea.MouseWheelMsg:
		mouse := msg.Mouse()
		switch mouse.Button {
		case tea.MouseWheelDown:
			if m.viewport.MoveDownN(3) {
				m.rebuildViewport()
			}
			if m.viewport.NearBottom(3) && m.cursor != "" && !m.loading {
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

	return m, nil
}
func (m *VNotificationsModel) rebuildViewport() {
	lazy := &images.LazyRenderer{Inner: m.imageCache}
	m.viewport.SetItems(len(m.notifications), func(index int, selected bool) string {
		lazy.NearVisible = m.viewport.IsNearVisible(index, m.viewport.Height())
		return m.renderNotification(index, selected, lazy)
	})
}

func (m VNotificationsModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	km := m.keys
	switch {
	case key.Matches(msg, km.Down):
		if m.viewport.MoveDown() {
			prev := m.viewport.YOffset()
			m.rebuildViewport()
			m.viewport.AnimateFrom(prev)
			if m.viewport.NearBottom(3) && m.cursor != "" && !m.loading {
				m.loading = true
				return m, tea.Batch(m.fetchNotificationsCmd(), m.spinner.Tick, m.viewport.SpringCmd())
			}
		}
		return m, m.viewport.SpringCmd()

	case key.Matches(msg, km.Up):
		if m.viewport.MoveUp() {
			prev := m.viewport.YOffset()
			m.rebuildViewport()
			m.viewport.AnimateFrom(prev)
		}
		return m, m.viewport.SpringCmd()

	case key.Matches(msg, km.MarkRead):
		m.loading = true
		m.cursor = ""
		m.notifications = nil
		m.err = nil
		m.viewport.Reset()
		return m, tea.Batch(m.fetchNotificationsCmd(), m.fetchUnreadCountCmd(), m.spinner.Tick)

	case msg.String() == "enter":
		idx := m.viewport.SelectedIndex()
		if idx < len(m.notifications) {
			n := m.notifications[idx]
			return m, func() tea.Msg { return NavigateToNotificationMsg{Notification: n} }
		}
		return m, nil

	case msg.String() == "m":
		return m, m.markSelectedReadCmd()
	}

	return m, nil
}

// Keys returns the vnotifications key map for help rendering.
func (m VNotificationsModel) Keys() KeyMap {
	return m.keys
}

// View implements tea.Model.
func (m VNotificationsModel) View() tea.View {
	var content strings.Builder

	header := "Voresky Notifications"
	if m.unreadCount > 0 {
		header = fmt.Sprintf("Voresky Notifications (%d unread)", m.unreadCount)
	}
	content.WriteString(theme.StyleHeaderSubtle().Render(header))
	content.WriteString("\n\n")
	headerHeight := strings.Count(content.String(), "\n")
	availableHeight := max(1, m.height-headerHeight)

	if m.loading && len(m.notifications) == 0 {
		content.WriteString(lipgloss.Place(
			m.width,
			availableHeight,
			lipgloss.Center,
			lipgloss.Center,
			theme.StyleMuted().Render(m.spinner.View()+" Loading notifications..."),
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

	if len(m.notifications) == 0 {
		content.WriteString(lipgloss.Place(
			m.width,
			availableHeight,
			lipgloss.Center,
			lipgloss.Center,
			theme.StyleMuted().Italic(true).Render("No Voresky notifications"),
		))
		v := tea.NewView(content.String())
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	content.WriteString(m.viewport.View())

	if m.loading {
		content.WriteString(theme.StyleMuted().Render(m.spinner.View() + " Loading more..."))
	}

	v := tea.NewView(content.String())
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m VNotificationsModel) renderNotification(index int, selected bool, renderer images.ImageRenderer) string {
	n := m.notifications[index]
	var b strings.Builder

	unreadDot := ""
	if !n.IsRead {
		unreadDot = theme.NotifUnreadDotStyle().Render("●") + " "
	}

	line := unreadDot + notificationStyle(n.Type).Render(formatNotification(&n))
	b.WriteString(line + "\n")

	if n.Universe != "" {
		_, _ = fmt.Fprintf(&b, "  %s\n", shared.UniverseStyle().Render(n.Universe))
	}

	ts := n.CreatedAt.Format("Jan 2, 2006 15:04")
	_, _ = fmt.Fprintf(&b, "  %s", theme.StyleMuted().Render(ts))

	contentStr := b.String()

	var avatarBlock string
	var gutterWidth int
	if renderer != nil && renderer.Enabled() {
		sourceAv, targetAv := "", ""

		if n.SourceCharacter != nil && n.SourceCharacter.Avatar.URL != "" {
			url := n.SourceCharacter.Avatar.URL
			if renderer.IsCached(url) {
				sourceAv = renderer.RenderImage(url, shared.AvatarCols, shared.AvatarRows)
			} else {
				sourceAv = shared.RenderPlaceholder(shared.AvatarCols, shared.AvatarRows)
			}
		}

		if n.TargetCharacter != nil && n.TargetCharacter.Avatar.URL != "" {
			url := n.TargetCharacter.Avatar.URL
			if renderer.IsCached(url) {
				targetAv = renderer.RenderImage(url, shared.AvatarCols, shared.AvatarRows)
			} else {
				targetAv = shared.RenderPlaceholder(shared.AvatarCols, shared.AvatarRows)
			}
		}

		hasBoth := sourceAv != "" && targetAv != ""
		if hasBoth && m.width >= shared.AvatarCols*2+1+15 {
			avatarBlock = shared.JoinHorizontalRaw(sourceAv, targetAv, " ")
			gutterWidth = shared.AvatarCols*2 + 1
		} else if sourceAv != "" {
			avatarBlock = sourceAv
			gutterWidth = shared.AvatarCols
		} else if targetAv != "" {
			avatarBlock = targetAv
			gutterWidth = shared.AvatarCols
		}
	}

	if avatarBlock != "" {
		contentWidth := max(10, m.width-2-gutterWidth-1)
		contentStr = lipgloss.NewStyle().Width(contentWidth).Render(contentStr)
		contentStr = shared.JoinWithGutter(avatarBlock, contentStr, " ", gutterWidth)
	}

	return shared.RenderItemWithBorder(contentStr, selected, m.width)
}

func notificationStyle(t voresky.NotificationType) lipgloss.Style {
	s := string(t)
	switch {
	case t == voresky.NotifPoke:
		return pokeStyle()
	case t == voresky.NotifStalk || t == voresky.NotifStalkTargetAvailable:
		return stalkStyle()
	case strings.HasPrefix(s, "housing_"):
		return housingStyle()
	case strings.HasPrefix(s, "interaction_"):
		return interactionStyle()
	case strings.HasPrefix(s, "collar_"):
		return collarStyle()
	default:
		return theme.StyleMuted()
	}
}

func formatNotification(notif *voresky.Notification) string {
	if notif == nil {
		return "Notification"
	}

	sourceName := "Someone"
	targetName := "someone"
	if notif.SourceCharacter != nil && notif.SourceCharacter.Name != "" {
		sourceName = notif.SourceCharacter.Name
	}
	if notif.TargetCharacter != nil && notif.TargetCharacter.Name != "" {
		targetName = notif.TargetCharacter.Name
	}

	payload, _ := voresky.ParsePayload(notif.Type, notif.Payload)

	switch notif.Type {
	case voresky.NotifPoke:
		return fmt.Sprintf("%s poked %s", sourceName, targetName)
	case voresky.NotifStalk:
		return fmt.Sprintf("%s is stalking %s", sourceName, targetName)
	case voresky.NotifStalkTargetAvailable:
		return fmt.Sprintf("%s is now available", targetName)

	case voresky.NotifHousingInvite:
		if p, ok := payload.(*voresky.HousingPayload); ok && p != nil {
			owner := sourceName
			member := targetName
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			if p.MemberCharacterName != "" {
				member = p.MemberCharacterName
			}
			return fmt.Sprintf("%s invited %s to their home", owner, member)
		}
		return fmt.Sprintf("%s invited %s to their home", sourceName, targetName)
	case voresky.NotifHousingRequest:
		if p, ok := payload.(*voresky.HousingPayload); ok && p != nil {
			member := sourceName
			owner := targetName
			if p.MemberCharacterName != "" {
				member = p.MemberCharacterName
			}
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			return fmt.Sprintf("%s requested to join %s's home", member, owner)
		}
		return fmt.Sprintf("%s requested to join %s's home", sourceName, targetName)
	case voresky.NotifHousingJoin:
		if p, ok := payload.(*voresky.HousingPayload); ok && p != nil {
			member := sourceName
			owner := targetName
			if p.MemberCharacterName != "" {
				member = p.MemberCharacterName
			}
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			return fmt.Sprintf("%s joined %s's home", member, owner)
		}
		return fmt.Sprintf("%s joined %s's home", sourceName, targetName)
	case voresky.NotifHousingLeave:
		if p, ok := payload.(*voresky.HousingPayload); ok && p != nil {
			member := sourceName
			owner := targetName
			if p.MemberCharacterName != "" {
				member = p.MemberCharacterName
			}
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			return fmt.Sprintf("%s left %s's home", member, owner)
		}
		return fmt.Sprintf("%s left %s's home", sourceName, targetName)
	case voresky.NotifHousingKick:
		if p, ok := payload.(*voresky.HousingPayload); ok && p != nil {
			owner := sourceName
			member := targetName
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			if p.MemberCharacterName != "" {
				member = p.MemberCharacterName
			}
			return fmt.Sprintf("%s kicked %s from their home", owner, member)
		}
		return fmt.Sprintf("%s kicked %s from their home", sourceName, targetName)
	case voresky.NotifHousingInviteAccepted, voresky.NotifHousingRequestAccepted:
		if p, ok := payload.(*voresky.HousingPayload); ok && p != nil {
			member := sourceName
			owner := targetName
			if p.MemberCharacterName != "" {
				member = p.MemberCharacterName
			}
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			return fmt.Sprintf("%s accepted %s's housing request", member, owner)
		}
		return fmt.Sprintf("%s accepted %s's housing request", sourceName, targetName)
	case voresky.NotifHousingInviteRejected, voresky.NotifHousingRequestRejected:
		if p, ok := payload.(*voresky.HousingPayload); ok && p != nil {
			member := sourceName
			owner := targetName
			if p.MemberCharacterName != "" {
				member = p.MemberCharacterName
			}
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			return fmt.Sprintf("%s rejected %s's housing request", member, owner)
		}
		return fmt.Sprintf("%s rejected %s's housing request", sourceName, targetName)

	case voresky.NotifInteractionProposal:
		if p, ok := payload.(*voresky.InteractionProposalPayload); ok && p != nil {
			pred, prey, path := p.PredatorCharacterName, p.PreyCharacterName, p.PathName
			if pred == "" {
				pred = sourceName
			}
			if prey == "" {
				prey = targetName
			}
			if path == "" {
				path = "their path"
			}
			if p.InitiatedBy == "prey" {
				return fmt.Sprintf("%s offered to be caught by %s via %s", prey, pred, path)
			}
			return fmt.Sprintf("%s proposed catching %s via %s", pred, prey, path)
		}
		return fmt.Sprintf("%s proposed an interaction with %s", sourceName, targetName)
	case voresky.NotifInteractionAccepted:
		if p, ok := payload.(*voresky.InteractionBasePayload); ok && p != nil {
			path := p.PathName
			if path == "" {
				return fmt.Sprintf("%s accepted an interaction with %s", sourceName, targetName)
			}
			return fmt.Sprintf("%s accepted the %s interaction with %s", sourceName, path, targetName)
		}
		return fmt.Sprintf("%s accepted an interaction with %s", sourceName, targetName)
	case voresky.NotifInteractionRejected:
		if p, ok := payload.(*voresky.InteractionBasePayload); ok && p != nil {
			path := p.PathName
			if path == "" {
				return fmt.Sprintf("%s rejected an interaction with %s", sourceName, targetName)
			}
			return fmt.Sprintf("%s rejected the %s interaction with %s", sourceName, path, targetName)
		}
		return fmt.Sprintf("%s rejected an interaction with %s", sourceName, targetName)
	case voresky.NotifInteractionCounter:
		if p, ok := payload.(*voresky.InteractionCounterPayload); ok && p != nil {
			pred := p.PredatorCharacterName
			prey := p.PreyCharacterName
			if pred == "" {
				pred = sourceName
			}
			if prey == "" {
				prey = targetName
			}
			if p.PathName != "" {
				return fmt.Sprintf("%s countered the proposal with %s on %s", pred, prey, p.PathName)
			}
			return fmt.Sprintf("%s countered a proposal with %s", pred, prey)
		}
		return fmt.Sprintf("%s countered a proposal with %s", sourceName, targetName)
	case voresky.NotifInteractionVipCaught:
		if p, ok := payload.(*voresky.InteractionBasePayload); ok && p != nil {
			pred := p.PredatorCharacterName
			prey := p.PreyCharacterName
			path := p.PathName
			if pred == "" {
				pred = sourceName
			}
			if prey == "" {
				prey = targetName
			}
			if path != "" {
				return fmt.Sprintf("%s caught %s via %s!", pred, prey, path)
			}
			return fmt.Sprintf("%s caught %s!", pred, prey)
		}
		return fmt.Sprintf("%s caught %s!", sourceName, targetName)
	case voresky.NotifInteractionNodeChanged:
		if p, ok := payload.(*voresky.InteractionNodePayload); ok && p != nil && p.NewNodeVerbPast != "" {
			return fmt.Sprintf("Interaction progressed: %s was %s", targetName, p.NewNodeVerbPast)
		}
		return "Interaction progressed"
	case voresky.NotifInteractionEscaped:
		if p, ok := payload.(*voresky.InteractionBasePayload); ok && p != nil {
			pred := p.PredatorCharacterName
			prey := p.PreyCharacterName
			if pred == "" {
				pred = sourceName
			}
			if prey == "" {
				prey = targetName
			}
			return fmt.Sprintf("%s escaped from %s!", prey, pred)
		}
		return fmt.Sprintf("%s escaped from %s!", targetName, sourceName)
	case voresky.NotifInteractionReleased:
		if p, ok := payload.(*voresky.InteractionBasePayload); ok && p != nil {
			pred := p.PredatorCharacterName
			prey := p.PreyCharacterName
			if pred == "" {
				pred = sourceName
			}
			if prey == "" {
				prey = targetName
			}
			return fmt.Sprintf("%s released %s", pred, prey)
		}
		return fmt.Sprintf("%s released %s", sourceName, targetName)
	case voresky.NotifInteractionRetreated:
		if p, ok := payload.(*voresky.InteractionRetreatPayload); ok && p != nil {
			prey := p.PreyCharacterName
			if prey == "" {
				prey = targetName
			}
			if p.RetreatedToNode != "" {
				return fmt.Sprintf("%s retreated to %s", prey, p.RetreatedToNode)
			}
			return fmt.Sprintf("%s retreated", prey)
		}
		return fmt.Sprintf("%s retreated", targetName)
	case voresky.NotifInteractionRespawning:
		if p, ok := payload.(*voresky.InteractionRespawnPayload); ok && p != nil {
			prey := p.PreyCharacterName
			if prey == "" {
				prey = targetName
			}
			return fmt.Sprintf("%s is respawning...", prey)
		}
		return fmt.Sprintf("%s is respawning...", targetName)
	case voresky.NotifInteractionCompleted:
		if p, ok := payload.(*voresky.InteractionCompletedPayload); ok && p != nil {
			if p.VerbPast != "" {
				prey := p.PreyCharacterName
				pred := p.PredatorCharacterName
				if prey == "" {
					prey = targetName
				}
				if pred == "" {
					pred = sourceName
				}
				return fmt.Sprintf("%s was %s by %s", prey, p.VerbPast, pred)
			}
			if p.PathName != "" {
				return fmt.Sprintf("Interaction completed: %s × %s on %s", sourceName, targetName, p.PathName)
			}
			return fmt.Sprintf("Interaction completed: %s × %s", sourceName, targetName)
		}
		return fmt.Sprintf("Interaction completed: %s × %s", sourceName, targetName)
	case voresky.NotifInteractionSafeword:
		if p, ok := payload.(*voresky.InteractionSafewordPayload); ok && p != nil && p.PathName != "" {
			return fmt.Sprintf("A safe word was used in %s", p.PathName)
		}
		return "A safe word was used"

	case voresky.NotifCollarOffer:
		if p, ok := payload.(*voresky.CollarPayload); ok && p != nil {
			owner := sourceName
			pet := targetName
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			if p.PetCharacterName != "" {
				pet = p.PetCharacterName
			}
			return fmt.Sprintf("%s offered a collar to %s", owner, pet)
		}
		return fmt.Sprintf("%s offered a collar to %s", sourceName, targetName)
	case voresky.NotifCollarRequest:
		if p, ok := payload.(*voresky.CollarPayload); ok && p != nil {
			owner := sourceName
			pet := targetName
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			if p.PetCharacterName != "" {
				pet = p.PetCharacterName
			}
			return fmt.Sprintf("%s requested to collar %s", owner, pet)
		}
		return fmt.Sprintf("%s requested to collar %s", sourceName, targetName)
	case voresky.NotifCollarAccepted:
		if p, ok := payload.(*voresky.CollarPayload); ok && p != nil {
			pet := sourceName
			owner := targetName
			if p.PetCharacterName != "" {
				pet = p.PetCharacterName
			}
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			return fmt.Sprintf("%s accepted %s's collar", pet, owner)
		}
		return fmt.Sprintf("%s accepted %s's collar", sourceName, targetName)
	case voresky.NotifCollarRejected:
		if p, ok := payload.(*voresky.CollarPayload); ok && p != nil {
			pet := sourceName
			owner := targetName
			if p.PetCharacterName != "" {
				pet = p.PetCharacterName
			}
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			return fmt.Sprintf("%s rejected %s's collar", pet, owner)
		}
		return fmt.Sprintf("%s rejected %s's collar", sourceName, targetName)
	case voresky.NotifCollarBroken:
		if p, ok := payload.(*voresky.CollarPayload); ok && p != nil {
			owner := sourceName
			pet := targetName
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			if p.PetCharacterName != "" {
				pet = p.PetCharacterName
			}
			return fmt.Sprintf("%s broke the collar with %s", owner, pet)
		}
		return fmt.Sprintf("%s broke the collar with %s", sourceName, targetName)
	case voresky.NotifCollarLockRequest:
		if p, ok := payload.(*voresky.CollarPayload); ok && p != nil {
			owner := sourceName
			pet := targetName
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			if p.PetCharacterName != "" {
				pet = p.PetCharacterName
			}
			return fmt.Sprintf("%s requested to lock %s's collar", owner, pet)
		}
		return fmt.Sprintf("%s requested to lock %s's collar", sourceName, targetName)
	case voresky.NotifCollarLocked:
		if p, ok := payload.(*voresky.CollarPayload); ok && p != nil {
			owner := sourceName
			pet := targetName
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			if p.PetCharacterName != "" {
				pet = p.PetCharacterName
			}
			return fmt.Sprintf("%s locked %s's collar", owner, pet)
		}
		return fmt.Sprintf("%s locked %s's collar", sourceName, targetName)
	case voresky.NotifCollarUnlocked:
		if p, ok := payload.(*voresky.CollarPayload); ok && p != nil {
			owner := sourceName
			pet := targetName
			if p.OwnerCharacterName != "" {
				owner = p.OwnerCharacterName
			}
			if p.PetCharacterName != "" {
				pet = p.PetCharacterName
			}
			return fmt.Sprintf("%s unlocked %s's collar", owner, pet)
		}
		return fmt.Sprintf("%s unlocked %s's collar", sourceName, targetName)
	default:
		return fmt.Sprintf("%s → %s", sourceName, targetName)
	}
}

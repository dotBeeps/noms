package profile

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/feed"
	"github.com/dotBeeps/noms/internal/ui/images"
	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// ProfileModel represents the profile screen UI state.
type ProfileModel struct {
	client       bluesky.BlueskyClient
	viewDID      string
	ownDID       string
	isOwnProfile bool

	profile    *bsky.ActorDefs_ProfileViewDetailed
	authorFeed []*bsky.FeedDefs_FeedViewPost
	cursor     string

	loading         bool
	loadingFeed     bool
	err             error
	width           int
	height          int
	spinner         spinner.Model
	confirmDelete   int // -1 = none
	imageCache      *images.Cache
	avatarOverrides map[string]string
	keys            KeyMap
	viewport        shared.ItemViewport
}

// NewProfileModel creates a new profile model.
func NewProfileModel(client bluesky.BlueskyClient, viewDID, ownDID string, width, height int, cache *images.Cache) ProfileModel {
	sp := shared.NewSpinner()
	return ProfileModel{
		client:        client,
		viewDID:       viewDID,
		ownDID:        ownDID,
		isOwnProfile:  viewDID == ownDID,
		width:         width,
		height:        height,
		loading:       true,
		loadingFeed:   true,
		spinner:       sp,
		confirmDelete: -1,
		imageCache:    cache,
		keys:          DefaultKeyMap,
		viewport:      shared.NewItemViewport(width, height),
	}
}

func (m *ProfileModel) SetAvatarOverrides(overrides map[string]string) {
	m.avatarOverrides = overrides
}

// Messages for profile operations.

// ProfileLoadedMsg is sent when profile data is loaded.
type ProfileLoadedMsg struct {
	Profile *bsky.ActorDefs_ProfileViewDetailed
}

// AuthorFeedLoadedMsg is sent when author feed is loaded.
type AuthorFeedLoadedMsg struct {
	Posts  []*bsky.FeedDefs_FeedViewPost
	Cursor string
}

// ProfileErrorMsg is sent when an error occurs.
type ProfileErrorMsg struct {
	Err error
}

// FollowToggledMsg is sent when follow state changes.
type FollowToggledMsg struct {
	Following bool
}

// ViewThreadMsg is sent to navigate to a thread view.
type ViewThreadMsg struct {
	URI string
}

// BackMsg is sent to navigate back to the parent view.
type BackMsg struct{}

// Init initializes the profile model by fetching profile and feed in parallel.
func (m ProfileModel) Init() tea.Cmd {
	return tea.Batch(
		m.fetchProfile(),
		m.fetchAuthorFeed(""),
		m.spinner.Tick,
	)
}

func (m ProfileModel) fetchProfile() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		p, err := m.client.GetProfile(ctx, m.viewDID)
		if err != nil {
			return ProfileErrorMsg{Err: err}
		}
		return ProfileLoadedMsg{Profile: p}
	}
}

func (m ProfileModel) fetchAuthorFeed(cursor string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		posts, nextCursor, err := m.client.GetAuthorFeed(ctx, m.viewDID, cursor, 30)
		if err != nil {
			return ProfileErrorMsg{Err: err}
		}
		return AuthorFeedLoadedMsg{Posts: posts, Cursor: nextCursor}
	}
}

func (m ProfileModel) toggleFollow() tea.Cmd {
	return func() tea.Msg {
		if m.isOwnProfile || m.profile == nil || m.profile.Viewer == nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Check if currently following
		isFollowing := m.profile.Viewer.Following != nil && *m.profile.Viewer.Following != ""

		if isFollowing {
			// Unfollow: use the follow record URI
			followURI := *m.profile.Viewer.Following
			err := m.client.UnfollowActor(ctx, followURI)
			if err != nil {
				return ProfileErrorMsg{Err: err}
			}
			return FollowToggledMsg{Following: false}
		}

		// Follow
		err := m.client.FollowActor(ctx, m.viewDID)
		if err != nil {
			return ProfileErrorMsg{Err: err}
		}
		return FollowToggledMsg{Following: true}
	}
}

// Update handles messages and updates the model state.
func (m ProfileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetSize(msg.Width, msg.Height)
		m.rebuildViewport()

	case ProfileLoadedMsg:
		m.profile = msg.Profile
		m.loading = false
		m.rebuildViewport() // header height changed
		if m.profile.Avatar != nil && *m.profile.Avatar != "" {
			if cmd := images.FetchAvatar(m.imageCache, *m.profile.Avatar); cmd != nil {
				return m, cmd
			}
		}

	case AuthorFeedLoadedMsg:
		m.loadingFeed = false
		if m.cursor == "" {
			m.authorFeed = msg.Posts
		} else {
			m.authorFeed = append(m.authorFeed, msg.Posts...)
		}
		m.cursor = msg.Cursor
		var fetchCmds []tea.Cmd
		for _, p := range msg.Posts {
			for _, url := range feed.ExtractImageURLs(p) {
				if cmd := images.Fetch(m.imageCache, url); cmd != nil {
					fetchCmds = append(fetchCmds, cmd)
				}
			}
			avatarURL := feed.ExtractAvatarURL(p)
			if avatarURL != "" {
				if cmd := images.FetchAvatar(m.imageCache, avatarURL); cmd != nil {
					fetchCmds = append(fetchCmds, cmd)
				}
			}
		}
		m.rebuildViewport()
		if len(fetchCmds) > 0 {
			return m, tea.Batch(fetchCmds...)
		}

	case images.ImageFetchedMsg:
		m.rebuildViewport()
		return m, nil

	case ProfileErrorMsg:
		m.err = msg.Err
		m.loading = false
		m.loadingFeed = false

	case FollowToggledMsg:
		if m.profile != nil && m.profile.Viewer != nil {
			if msg.Following {
				// Set a dummy URI to indicate following state
				dummy := "at://following"
				m.profile.Viewer.Following = &dummy
			} else {
				m.profile.Viewer.Following = nil
			}
		}

	case spinner.TickMsg:
		if m.loading || m.loadingFeed {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case feed.LikeResultMsg:
		if post := feed.FindPostByURI(m.authorFeed, msg.PostURI); post != nil {
			if msg.Err != nil {
				feed.RollbackLike(post.Post)
			} else {
				if post.Post.Viewer == nil {
					post.Post.Viewer = &bsky.FeedDefs_ViewerState{}
				}
				post.Post.Viewer.Like = &msg.LikeURI
			}
		}
		return m, nil

	case feed.UnlikeResultMsg:
		if post := feed.FindPostByURI(m.authorFeed, msg.PostURI); post != nil {
			if msg.Err != nil {
				feed.OptimisticLike(post.Post)
			}
		}
		return m, nil

	case feed.RepostResultMsg:
		if post := feed.FindPostByURI(m.authorFeed, msg.PostURI); post != nil {
			if msg.Err != nil {
				feed.RollbackRepost(post.Post)
			} else {
				if post.Post.Viewer == nil {
					post.Post.Viewer = &bsky.FeedDefs_ViewerState{}
				}
				post.Post.Viewer.Repost = &msg.RepostURI
			}
		}
		return m, nil

	case feed.UnRepostResultMsg:
		if post := feed.FindPostByURI(m.authorFeed, msg.PostURI); post != nil {
			if msg.Err != nil {
				feed.OptimisticRepost(post.Post)
			}
		}
		return m, nil

	case feed.DeletePostResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		for i, p := range m.authorFeed {
			if p.Post != nil && p.Post.Uri == msg.URI {
				m.authorFeed = append(m.authorFeed[:i], m.authorFeed[i+1:]...)
				idx := m.viewport.SelectedIndex()
				if idx >= len(m.authorFeed) && idx > 0 {
					m.viewport.SetSelectedIndex(len(m.authorFeed) - 1)
				}
				break
			}
		}
		m.confirmDelete = -1
		m.rebuildViewport()
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
			if m.viewport.NearBottom(shared.PaginationThreshold) && m.cursor != "" && !m.loadingFeed {
				m.loadingFeed = true
				return m, tea.Batch(m.fetchAuthorFeed(m.cursor), m.spinner.Tick)
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

func (m ProfileModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	km := m.keys

	if !key.Matches(msg, km.Delete) {
		m.confirmDelete = -1
	}

	idx := m.viewport.SelectedIndex()
	switch {
	case key.Matches(msg, km.Down):
		if m.viewport.MoveDown() {
			prev := m.viewport.YOffset()
			m.rebuildViewport()
			m.viewport.AnimateFrom(prev)
		}
		if m.viewport.NearBottom(shared.PaginationThreshold) && m.cursor != "" && !m.loadingFeed {
			m.loadingFeed = true
			return m, tea.Batch(m.fetchAuthorFeed(m.cursor), m.spinner.Tick, m.viewport.SpringCmd())
		}
		return m, m.viewport.SpringCmd()

	case key.Matches(msg, km.Up):
		if m.viewport.MoveUp() {
			prev := m.viewport.YOffset()
			m.rebuildViewport()
			m.viewport.AnimateFrom(prev)
		}
		return m, m.viewport.SpringCmd()

	case key.Matches(msg, km.Follow):
		if !m.isOwnProfile && m.profile != nil {
			return m, m.toggleFollow()
		}

	case key.Matches(msg, km.Refresh):
		m.loading = true
		m.loadingFeed = true
		m.cursor = ""
		m.authorFeed = nil
		m.viewport.Reset()
		return m, tea.Batch(m.fetchProfile(), m.fetchAuthorFeed(""), m.spinner.Tick)

	case key.Matches(msg, km.Open):
		if idx < len(m.authorFeed) {
			post := m.authorFeed[idx]
			if post.Post != nil {
				return m, func() tea.Msg {
					return ViewThreadMsg{URI: post.Post.Uri}
				}
			}
		}

	case key.Matches(msg, km.Delete):
		if idx < len(m.authorFeed) {
			post := m.authorFeed[idx]
			if post.Post != nil && post.Post.Author != nil {
				res := shared.CheckConfirmDelete(m.confirmDelete, idx, post.Post.Author.Did, m.ownDID, post.Post.Uri)
				m.confirmDelete = res.ConfirmDelete
				if res.Confirmed {
					uri := res.URI
					return m, func() tea.Msg { return feed.DeletePostMsg{URI: uri} }
				}
				if res.URI == "" && res.ConfirmDelete == idx {
					return m, nil
				}
			}
		}

	case key.Matches(msg, km.Back):
		return m, func() tea.Msg {
			return BackMsg{}
		}
	}

	return m, nil
}

// Keys returns the current key map.
func (m ProfileModel) Keys() KeyMap { return m.keys }

// View renders the profile screen.
func (m ProfileModel) View() tea.View {
	mouseView := func(s string) tea.View {
		v := tea.NewView(s)
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if m.err != nil {
		errorText := lipgloss.NewStyle().
			Foreground(theme.ColorError).
			Render(fmt.Sprintf("Error: %v\n\nPress 'r' to retry or 'esc' to go back", m.err))
		return mouseView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, errorText))
	}

	if m.loading && m.profile == nil {
		loadingText := lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Render(m.spinner.View() + " Loading profile...")
		return mouseView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, loadingText))
	}

	var b strings.Builder
	headerHeight := 0

	// Render header
	if m.profile != nil {
		before := b.Len()
		m.renderHeader(&b)
		headerHeight += strings.Count(b.String()[before:], "\n")
	}

	// Render separator
	b.WriteString(lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Render(strings.Repeat("─", min(m.width, 60))))
	b.WriteString("\n")
	headerHeight++
	availableHeight := max(1, m.height-headerHeight)

	// Render feed
	if len(m.authorFeed) == 0 && !m.loadingFeed {
		emptyText := lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Render("No posts yet")
		b.WriteString(lipgloss.Place(m.width, availableHeight, lipgloss.Center, lipgloss.Center, emptyText))
	} else {
		b.WriteString(m.viewport.View())
	}

	if m.loadingFeed {
		loadingStyle := lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Padding(0, 2)
		b.WriteString(loadingStyle.Render(m.spinner.View() + " Loading more posts..."))
	}

	if m.confirmDelete >= 0 {
		confirmStyle := lipgloss.NewStyle().
			Foreground(theme.ColorWarning).
			Bold(true)
		b.WriteString("\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
			confirmStyle.Render("Press d to confirm delete, any other key to cancel")))
	}

	return mouseView(b.String())
}

func (m ProfileModel) renderHeader(b *strings.Builder) {
	displayName := m.profile.Handle
	if m.profile.DisplayName != nil && *m.profile.DisplayName != "" {
		displayName = *m.profile.DisplayName
	}

	nameStyle := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true)
	handleStyle := lipgloss.NewStyle().Foreground(theme.ColorMuted)

	var followers, following, posts int64
	if m.profile.FollowersCount != nil {
		followers = *m.profile.FollowersCount
	}
	if m.profile.FollowsCount != nil {
		following = *m.profile.FollowsCount
	}
	if m.profile.PostsCount != nil {
		posts = *m.profile.PostsCount
	}
	stats := fmt.Sprintf("%s followers · %s following · %s posts",
		formatCount(followers), formatCount(following), formatCount(posts))

	avatarStr := ""
	if m.profile.Avatar != nil && *m.profile.Avatar != "" && m.imageCache != nil && m.imageCache.Enabled() {
		avatarURL := *m.profile.Avatar
		if m.imageCache.IsCached(avatarURL) {
			avatarStr = strings.TrimRight(m.imageCache.RenderImage(avatarURL, shared.AvatarCols, shared.AvatarRows), "\n ")
			if avatarStr == "" {
				avatarStr = shared.RenderPlaceholder(shared.AvatarCols, shared.AvatarRows)
			}
		} else {
			avatarStr = shared.RenderPlaceholder(shared.AvatarCols, shared.AvatarRows)
		}
	}

	if avatarStr != "" {
		isFollowing := m.profile.Viewer != nil && m.profile.Viewer.Following != nil && *m.profile.Viewer.Following != ""
		var textBlock strings.Builder
		textBlock.WriteString(nameStyle.Render(displayName) + "\n")
		textBlock.WriteString(handleStyle.Render("@"+m.profile.Handle) + "\n")
		textBlock.WriteString(handleStyle.Render(stats) + "\n")
		if !m.isOwnProfile {
			if isFollowing {
				textBlock.WriteString(lipgloss.NewStyle().Foreground(theme.ColorSuccess).Bold(true).Render("[Following ✓]") + "\n")
			} else {
				textBlock.WriteString(lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true).Render("[Follow]") + "\n")
			}
		}
		b.WriteString(shared.JoinWithGutter(avatarStr, textBlock.String(), " ", shared.AvatarCols))
		b.WriteString("\n")
	} else {
		paddedName := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true).Padding(0, 2)
		paddedMuted := lipgloss.NewStyle().Foreground(theme.ColorMuted).Padding(0, 2)
		b.WriteString(paddedName.Render(displayName) + "\n")
		b.WriteString(paddedMuted.Render("@"+m.profile.Handle) + "\n")
		b.WriteString(paddedMuted.Render(stats) + "\n")
		if !m.isOwnProfile {
			m.renderFollowButton(b)
		}
		b.WriteString("\n")
	}

	// Bio is always full-width below the header block
	if m.profile.Description != nil && *m.profile.Description != "" {
		m.renderBio(b, *m.profile.Description)
	}
}

func (m ProfileModel) renderBio(b *strings.Builder, bio string) {
	// Extract facets from profile description if available
	var facets []*bsky.RichtextFacet
	// Note: ActorDefs_ProfileViewDetailed may have facets in the future
	// For now, we just render plain text with potential auto-detected facets

	// Try to parse facets from the bio text
	segments := bluesky.ParseFacets(bio, facets)

	bioStyle := lipgloss.NewStyle().Padding(0, 2)
	linkStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Underline(true)
	mentionStyle := lipgloss.NewStyle().
		Foreground(theme.ColorPrimary)
	tagStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent)

	var bioBuilder strings.Builder
	for _, seg := range segments {
		switch seg.Type {
		case bluesky.SegmentLink:
			bioBuilder.WriteString(linkStyle.Render(seg.Text))
		case bluesky.SegmentMention:
			bioBuilder.WriteString(mentionStyle.Render(seg.Text))
		case bluesky.SegmentTag:
			bioBuilder.WriteString(tagStyle.Render(seg.Text))
		default:
			bioBuilder.WriteString(seg.Text)
		}
	}

	b.WriteString(bioStyle.Render(bioBuilder.String()))
	b.WriteString("\n")
}

func (m ProfileModel) renderFollowButton(b *strings.Builder) {
	isFollowing := m.profile.Viewer != nil &&
		m.profile.Viewer.Following != nil &&
		*m.profile.Viewer.Following != ""

	buttonStyle := lipgloss.NewStyle().
		Padding(0, 2)

	var button string
	if isFollowing {
		followingStyle := lipgloss.NewStyle().
			Foreground(theme.ColorSuccess).
			Bold(true)
		button = followingStyle.Render("[Following ✓]")
	} else {
		followStyle := lipgloss.NewStyle().
			Foreground(theme.ColorPrimary).
			Bold(true)
		button = followStyle.Render("[Follow]")
	}

	b.WriteString(buttonStyle.Render(button))
	b.WriteString("\n")
}

func (m *ProfileModel) rebuildViewport() {
	// Adjust viewport height for the header
	headerHeight := 0
	if m.profile != nil {
		var b strings.Builder
		m.renderHeader(&b)
		headerHeight = strings.Count(b.String(), "\n") + 1
	}
	m.viewport.SetSize(m.width, max(1, m.height-headerHeight))
	lazy := &images.LazyRenderer{Inner: m.imageCache}
	m.viewport.SetItems(len(m.authorFeed), func(index int, selected bool) string {
		lazy.NearVisible = m.viewport.IsNearVisible(index, m.viewport.Height())
		return feed.RenderPost(m.authorFeed[index], m.width, selected, lazy, m.avatarOverrides)
	})
}

// formatCount formats large numbers with K/M suffixes.
func formatCount(count int64) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000.0)
	}
	if count >= 1000 {
		// Format with one decimal place, remove trailing .0
		result := fmt.Sprintf("%.1fK", float64(count)/1000.0)
		// Handle cases like 1.0K -> 1K
		if strings.HasSuffix(result, ".0K") {
			return fmt.Sprintf("%dK", count/1000)
		}
		return result
	}
	return fmt.Sprintf("%d", count)
}

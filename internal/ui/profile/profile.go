package profile

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
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

	selectedIndex int
	loading       bool
	loadingFeed   bool
	err           error
	width         int
	height        int
}

// NewProfileModel creates a new profile model.
func NewProfileModel(client bluesky.BlueskyClient, viewDID, ownDID string, width, height int) ProfileModel {
	return ProfileModel{
		client:        client,
		viewDID:       viewDID,
		ownDID:        ownDID,
		isOwnProfile:  viewDID == ownDID,
		width:         width,
		height:        height,
		loading:       true,
		selectedIndex: 0,
	}
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
	)
}

func (m ProfileModel) fetchProfile() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		p, err := m.client.GetProfile(ctx, m.viewDID)
		if err != nil {
			return ProfileErrorMsg{Err: err}
		}
		return ProfileLoadedMsg{Profile: p}
	}
}

func (m ProfileModel) fetchAuthorFeed(cursor string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
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

		ctx := context.Background()

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

	case ProfileLoadedMsg:
		m.profile = msg.Profile
		m.loading = false

	case AuthorFeedLoadedMsg:
		m.loadingFeed = false
		if m.cursor == "" {
			m.authorFeed = msg.Posts
		} else {
			m.authorFeed = append(m.authorFeed, msg.Posts...)
		}
		m.cursor = msg.Cursor

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

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m ProfileModel) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "j", "down":
		if m.selectedIndex < len(m.authorFeed)-1 {
			m.selectedIndex++
		} else if m.cursor != "" && !m.loadingFeed {
			// Fetch more posts
			m.loadingFeed = true
			return m, m.fetchAuthorFeed(m.cursor)
		}

	case "k", "up":
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
		// Note: selectedIndex 0 is the first post; header is always visible above

	case "f":
		if !m.isOwnProfile && m.profile != nil {
			return m, m.toggleFollow()
		}

	case "r":
		m.loading = true
		m.loadingFeed = true
		m.cursor = ""
		m.selectedIndex = 0
		m.authorFeed = nil
		return m, tea.Batch(m.fetchProfile(), m.fetchAuthorFeed(""))

	case "enter":
		if m.selectedIndex >= 0 && m.selectedIndex < len(m.authorFeed) {
			post := m.authorFeed[m.selectedIndex]
			if post.Post != nil {
				return m, func() tea.Msg {
					return ViewThreadMsg{URI: post.Post.Uri}
				}
			}
		}

	case "esc", "backspace":
		return m, func() tea.Msg {
			return BackMsg{}
		}
	}

	return m, nil
}

// View renders the profile screen.
func (m ProfileModel) View() tea.View {
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(theme.ColorError).
			Padding(1, 2)
		return tea.NewView(errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress 'r' to retry or 'esc' to go back", m.err)))
	}

	if m.loading && m.profile == nil {
		loadingStyle := lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Padding(1, 2)
		return tea.NewView(loadingStyle.Render("Loading profile..."))
	}

	var b strings.Builder

	// Render header
	if m.profile != nil {
		m.renderHeader(&b)
	}

	// Render separator
	b.WriteString(lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Render(strings.Repeat("─", min(m.width, 60))))
	b.WriteString("\n")

	// Render feed
	if len(m.authorFeed) == 0 && !m.loadingFeed {
		mutedStyle := lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Padding(1, 2)
		b.WriteString(mutedStyle.Render("No posts yet"))
	} else {
		m.renderFeed(&b)
	}

	if m.loadingFeed {
		loadingStyle := lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Padding(0, 2)
		b.WriteString(loadingStyle.Render("Loading more posts..."))
	}

	return tea.NewView(b.String())
}

func (m ProfileModel) renderHeader(b *strings.Builder) {
	// Display name (bold, primary color) - large
	displayName := m.profile.Handle
	if m.profile.DisplayName != nil && *m.profile.DisplayName != "" {
		displayName = *m.profile.DisplayName
	}

	nameStyle := lipgloss.NewStyle().
		Foreground(theme.ColorPrimary).
		Bold(true).
		Padding(0, 2)

	b.WriteString(nameStyle.Render(displayName))
	b.WriteString("\n")

	// Handle (muted)
	handleStyle := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Padding(0, 2)
	b.WriteString(handleStyle.Render("@" + m.profile.Handle))
	b.WriteString("\n")

	// Bio with rich text facet rendering
	if m.profile.Description != nil && *m.profile.Description != "" {
		m.renderBio(b, *m.profile.Description)
	}

	// Stats line
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
	b.WriteString(handleStyle.Render(stats))
	b.WriteString("\n")

	// Follow button (only if not own profile)
	if !m.isOwnProfile {
		m.renderFollowButton(b)
	}

	b.WriteString("\n")
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

func (m ProfileModel) renderFeed(b *strings.Builder) {
	postStyle := lipgloss.NewStyle().Padding(0, 2)
	selectedStyle := lipgloss.NewStyle().
		Foreground(theme.ColorAccent).
		Bold(true).
		Padding(0, 2)
	authorStyle := lipgloss.NewStyle().
		Foreground(theme.ColorPrimary).
		Bold(true)
	mutedStyle := lipgloss.NewStyle().
		Foreground(theme.ColorMuted)

	for i, post := range m.authorFeed {
		if post.Post == nil {
			continue
		}

		// Cursor and selection
		cursorStr := "  "
		style := postStyle
		if i == m.selectedIndex {
			cursorStr = "▶ "
			style = selectedStyle
		}

		// Author line
		if post.Post.Author == nil {
			continue
		}
		authorName := post.Post.Author.Handle
		if post.Post.Author.DisplayName != nil {
			authorName = *post.Post.Author.DisplayName
		}

		// Post text
		var text string
		if record, ok := post.Post.Record.Val.(*bsky.FeedPost); ok {
			text = record.Text
		}

		// Engagement counts
		var likes, reposts, replies int64
		if post.Post.LikeCount != nil {
			likes = *post.Post.LikeCount
		}
		if post.Post.RepostCount != nil {
			reposts = *post.Post.RepostCount
		}
		if post.Post.ReplyCount != nil {
			replies = *post.Post.ReplyCount
		}

		// Render post
		line := fmt.Sprintf("%s%s: %s",
			cursorStr,
			authorStyle.Render(authorName),
			text)

		b.WriteString(style.Render(line))
		b.WriteString("\n")

		// Engagement line
		engagement := fmt.Sprintf("    %s",
			mutedStyle.Render(fmt.Sprintf("♥ %d  ↻ %d  ⤶ %d", likes, reposts, replies)))
		b.WriteString(style.Render(engagement))
		b.WriteString("\n")
	}
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

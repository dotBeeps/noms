package feed

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/images"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

type ViewThreadMsg struct{ URI string }
type LikePostMsg struct{ URI, CID string }
type RepostMsg struct{ URI, CID string }
type ComposeMsg struct{}
type FeedLoadedMsg struct {
	Posts  []*bsky.FeedDefs_FeedViewPost
	Cursor string
}
type FeedErrorMsg struct{ Err error }
type FeedRefreshMsg struct{}

type FeedModel struct {
	client          bluesky.BlueskyClient
	ownDID          string
	avatarOverrides map[string]string
	posts           []*bsky.FeedDefs_FeedViewPost
	selectedIndex   int
	cursor          string
	loading         bool
	width, height   int
	err             error
	offset          int
	spinner         spinner.Model
	confirmDelete   int // index of post pending delete confirmation, -1 = none
	imageCache      *images.Cache
}

func NewFeedModel(client bluesky.BlueskyClient, ownDID string, width, height int, cache *images.Cache) FeedModel {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.ColorAccent)),
	)
	return FeedModel{
		client:        client,
		ownDID:        ownDID,
		width:         width,
		height:        height,
		loading:       true,
		spinner:       sp,
		confirmDelete: -1,
		imageCache:    cache,
	}
}

func (m FeedModel) Init() tea.Cmd {
	return tea.Batch(m.fetchTimeline(""), m.spinner.Tick)
}

func (m FeedModel) fetchTimeline(cursor string) tea.Cmd {
	return func() tea.Msg {
		posts, nextCursor, err := m.client.GetTimeline(context.Background(), cursor, 20)
		if err != nil {
			return FeedErrorMsg{Err: err}
		}
		return FeedLoadedMsg{Posts: posts, Cursor: nextCursor}
	}
}

func (m FeedModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case FeedLoadedMsg:
		m.loading = false
		if m.cursor == "" || m.posts == nil {
			m.posts = msg.Posts
		} else {
			m.posts = append(m.posts, msg.Posts...)
		}
		m.cursor = msg.Cursor
		var fetchCmds []tea.Cmd
		for _, p := range msg.Posts {
			for _, url := range ExtractImageURLs(p) {
				if cmd := images.Fetch(m.imageCache, url); cmd != nil {
					fetchCmds = append(fetchCmds, cmd)
				}
			}
			avatarURL := ExtractAvatarURL(p)
			if p != nil && p.Post != nil && p.Post.Author != nil {
				if override, ok := m.avatarOverrides[p.Post.Author.Did]; ok && override != "" {
					avatarURL = override
				}
			}
			if avatarURL != "" {
				if cmd := images.FetchAvatar(m.imageCache, avatarURL); cmd != nil {
					fetchCmds = append(fetchCmds, cmd)
				}
			}
		}
		if len(fetchCmds) > 0 {
			return m, tea.Batch(fetchCmds...)
		}
		return m, nil

	case FeedErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case images.ImageFetchedMsg:
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case FeedRefreshMsg:
		m.loading = true
		m.posts = nil
		m.cursor = ""
		m.selectedIndex = 0
		m.offset = 0
		m.err = nil
		m.confirmDelete = -1
		return m, tea.Batch(m.fetchTimeline(""), m.spinner.Tick)

	case LikePostMsg:
		if post := FindPostByURI(m.posts, msg.URI); post != nil {
			if post.Post.Viewer == nil || post.Post.Viewer.Like == nil || *post.Post.Viewer.Like == "" {
				OptimisticLike(post.Post)
				return m, PerformLike(m.client, msg.URI, msg.CID)
			}
			likeURI := *post.Post.Viewer.Like
			OptimisticUnlike(post.Post)
			return m, PerformUnlike(m.client, msg.URI, likeURI)
		}
		return m, nil

	case RepostMsg:
		if post := FindPostByURI(m.posts, msg.URI); post != nil {
			if post.Post.Viewer == nil || post.Post.Viewer.Repost == nil || *post.Post.Viewer.Repost == "" {
				OptimisticRepost(post.Post)
				return m, PerformRepost(m.client, msg.URI, msg.CID)
			}
			repostURI := *post.Post.Viewer.Repost
			OptimisticUnRepost(post.Post)
			return m, PerformUnRepost(m.client, msg.URI, repostURI)
		}
		return m, nil

	case LikeResultMsg:
		if post := FindPostByURI(m.posts, msg.PostURI); post != nil {
			if msg.Err != nil {
				RollbackLike(post.Post)
			} else {
				if post.Post.Viewer == nil {
					post.Post.Viewer = &bsky.FeedDefs_ViewerState{}
				}
				post.Post.Viewer.Like = &msg.LikeURI
			}
		}
		return m, nil

	case UnlikeResultMsg:
		if post := FindPostByURI(m.posts, msg.PostURI); post != nil {
			if msg.Err != nil {
				OptimisticLike(post.Post)
			}
		}
		return m, nil

	case RepostResultMsg:
		if post := FindPostByURI(m.posts, msg.PostURI); post != nil {
			if msg.Err != nil {
				RollbackRepost(post.Post)
			} else {
				if post.Post.Viewer == nil {
					post.Post.Viewer = &bsky.FeedDefs_ViewerState{}
				}
				post.Post.Viewer.Repost = &msg.RepostURI
			}
		}
		return m, nil

	case UnRepostResultMsg:
		if post := FindPostByURI(m.posts, msg.PostURI); post != nil {
			if msg.Err != nil {
				OptimisticRepost(post.Post)
			}
		}
		return m, nil

	case DeletePostResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		for i, p := range m.posts {
			if p.Post != nil && p.Post.Uri == msg.URI {
				m.posts = append(m.posts[:i], m.posts[i+1:]...)
				if m.selectedIndex >= len(m.posts) && m.selectedIndex > 0 {
					m.selectedIndex--
				}
				if m.offset > 0 && m.offset >= len(m.posts) {
					m.offset = max(0, len(m.posts)-1)
				}
				break
			}
		}
		m.confirmDelete = -1
		return m, nil

	case tea.MouseWheelMsg:
		mouse := msg.Mouse()
		switch mouse.Button {
		case tea.MouseWheelDown:
			for range 3 {
				if m.selectedIndex < len(m.posts)-1 {
					m.selectedIndex++
				}
			}
			m.offset = ensureSelectedVisible(m.posts, m.selectedIndex, m.offset, m.width, m.height, m.imageCache, m.avatarOverrides)
			if m.selectedIndex >= len(m.posts)-3 && !m.loading && m.cursor != "" {
				m.loading = true
				return m, tea.Batch(m.fetchTimeline(m.cursor), m.spinner.Tick)
			}
		case tea.MouseWheelUp:
			for range 3 {
				if m.selectedIndex > 0 {
					m.selectedIndex--
				}
			}
			m.offset = ensureSelectedVisible(m.posts, m.selectedIndex, m.offset, m.width, m.height, m.imageCache, m.avatarOverrides)
		}
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() != "d" {
			m.confirmDelete = -1
		}

		if m.err != nil && msg.String() == "r" {
			return m, func() tea.Msg { return FeedRefreshMsg{} }
		}

		switch msg.String() {
		case "j", "down":
			if m.selectedIndex < len(m.posts)-1 {
				m.selectedIndex++
				m.offset = ensureSelectedVisible(m.posts, m.selectedIndex, m.offset, m.width, m.height, m.imageCache, m.avatarOverrides)
				if m.selectedIndex >= len(m.posts)-3 && !m.loading && m.cursor != "" {
					m.loading = true
					return m, tea.Batch(m.fetchTimeline(m.cursor), m.spinner.Tick)
				}
			}
		case "k", "up":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				m.offset = ensureSelectedVisible(m.posts, m.selectedIndex, m.offset, m.width, m.height, m.imageCache, m.avatarOverrides)
			}
		case "r":
			if m.selectedIndex < len(m.posts) {
				post := m.posts[m.selectedIndex].Post
				if post != nil {
					return m, func() tea.Msg { return ComposeReplyMsg{URI: post.Uri, CID: post.Cid} }
				}
			}
			return m, func() tea.Msg { return FeedRefreshMsg{} }
		case "enter":
			if m.selectedIndex < len(m.posts) {
				post := m.posts[m.selectedIndex].Post
				if post != nil {
					return m, func() tea.Msg { return ViewThreadMsg{URI: post.Uri} }
				}
			}
		case "l":
			if m.selectedIndex < len(m.posts) {
				post := m.posts[m.selectedIndex].Post
				if post != nil {
					return m, func() tea.Msg { return LikePostMsg{URI: post.Uri, CID: post.Cid} }
				}
			}
		case "t":
			if m.selectedIndex < len(m.posts) {
				post := m.posts[m.selectedIndex].Post
				if post != nil {
					return m, func() tea.Msg { return RepostMsg{URI: post.Uri, CID: post.Cid} }
				}
			}
		case "c":
			return m, func() tea.Msg { return ComposeMsg{} }
		case "d":
			if m.selectedIndex < len(m.posts) {
				post := m.posts[m.selectedIndex]
				if post.Post != nil && post.Post.Author != nil && post.Post.Author.Did == m.ownDID {
					if m.confirmDelete == m.selectedIndex {
						// Second press — confirm delete
						uri := post.Post.Uri
						m.confirmDelete = -1
						return m, func() tea.Msg { return DeletePostMsg{URI: uri} }
					}
					// First press — enter confirmation
					m.confirmDelete = m.selectedIndex
					return m, nil
				}
			}
		}
	}
	return m, nil
}

func ensureSelectedVisible(posts []*bsky.FeedDefs_FeedViewPost, selectedIndex, offset, width, height int, cache *images.Cache, avatarOverrides map[string]string) int {
	if len(posts) == 0 {
		return 0
	}
	if selectedIndex < offset {
		return selectedIndex
	}

	totalHeight := 0
	heights := make([]int, 0, selectedIndex-offset+1)
	for i := offset; i <= selectedIndex && i < len(posts); i++ {
		h := strings.Count(RenderPost(posts[i], width, false, cache, avatarOverrides), "\n")
		heights = append(heights, h)
		totalHeight += h
	}

	const margin = 2
	for totalHeight+margin > height && offset < selectedIndex {
		totalHeight -= heights[0]
		heights = heights[1:]
		offset++
	}

	return offset
}

func (m FeedModel) View() tea.View {
	var content string

	if m.err != nil {
		s := theme.StyleError.Render("Error: "+m.err.Error()) + "\n\nPress 'r' to retry"
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s)
	} else if len(m.posts) == 0 {
		if m.loading {
			content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.spinner.View()+" Loading feed...")
		} else {
			content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "No posts yet")
		}
	} else {
		var rendered string
		linesUsed := 0
		for i := m.offset; i < len(m.posts); i++ {
			post := RenderPost(m.posts[i], m.width, i == m.selectedIndex, m.imageCache, m.avatarOverrides)
			rendered += post
			linesUsed += strings.Count(post, "\n")
			if linesUsed >= m.height {
				break
			}
		}
		if m.loading {
			rendered += "\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center, m.spinner.View()+" Loading more...")
		}
		if m.confirmDelete >= 0 {
			confirmStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("203")).
				Bold(true)
			rendered += "\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
				confirmStyle.Render("Press d to confirm delete, any other key to cancel"))
		}
		content = rendered
	}

	v := tea.NewView(content)
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *FeedModel) SetAvatarOverrides(overrides map[string]string) {
	m.avatarOverrides = overrides
}

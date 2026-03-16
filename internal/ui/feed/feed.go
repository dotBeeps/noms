package feed

import (
	"context"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/images"
	"github.com/dotBeeps/noms/internal/ui/shared"
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
	cursor          string
	loading         bool
	width, height   int
	err             error
	spinner         spinner.Model
	confirmDelete   int // index of post pending delete confirmation, -1 = none
	imageCache      *images.Cache
	keys            KeyMap
	viewport        shared.ItemViewport
}

func NewFeedModel(client bluesky.BlueskyClient, ownDID string, width, height int, cache *images.Cache) FeedModel {
	sp := shared.NewSpinner()
	return FeedModel{
		client:        client,
		ownDID:        ownDID,
		width:         width,
		height:        height,
		loading:       true,
		spinner:       sp,
		confirmDelete: -1,
		imageCache:    cache,
		keys:          DefaultKeyMap,
		viewport:      shared.NewItemViewport(width, height),
	}
}

func (m FeedModel) Init() tea.Cmd {
	return tea.Batch(m.fetchTimeline(""), m.spinner.Tick)
}

func (m FeedModel) fetchTimeline(cursor string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		posts, nextCursor, err := m.client.GetTimeline(ctx, cursor, 20)
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
		m.viewport.SetSize(msg.Width, msg.Height)
		m.rebuildViewport()
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
		m.rebuildViewport()
		if len(fetchCmds) > 0 {
			return m, tea.Batch(append(fetchCmds, m.spinner.Tick)...)
		}
		return m, nil

	case FeedErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case images.ImageFetchedMsg:
		m.rebuildViewport()
		if m.imageCache != nil && m.imageCache.PendingCount() > 0 {
			return m, m.spinner.Tick
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading || (m.imageCache != nil && m.imageCache.PendingCount() > 0) {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case FeedRefreshMsg:
		m.loading = true
		m.posts = nil
		m.cursor = ""
		m.err = nil
		m.confirmDelete = -1
		m.viewport.Reset()
		if m.imageCache != nil {
			m.imageCache.InvalidateTransmissions()
			m.imageCache.ClearFailedFetches()
		}
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
				idx := m.viewport.SelectedIndex()
				if idx >= len(m.posts) && idx > 0 {
					m.viewport.SetSelectedIndex(len(m.posts) - 1)
				}
				break
			}
		}
		m.confirmDelete = -1
		m.rebuildViewport()
		return m, nil

	case tea.MouseWheelMsg:
		mouse := msg.Mouse()
		switch mouse.Button {
		case tea.MouseWheelDown:
			if m.viewport.MoveDownN(3) {
				m.rebuildViewport()
			}
			if m.viewport.NearBottom(shared.PaginationThreshold) && !m.loading && m.cursor != "" {
				m.loading = true
				return m, tea.Batch(m.fetchTimeline(m.cursor), m.spinner.Tick)
			}
		case tea.MouseWheelUp:
			if m.viewport.MoveUpN(3) {
				m.rebuildViewport()
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		km := m.keys
		if !key.Matches(msg, km.Delete) {
			m.confirmDelete = -1
		}

		if m.err != nil && key.Matches(msg, km.Reply) {
			return m, func() tea.Msg { return FeedRefreshMsg{} }
		}

		idx := m.viewport.SelectedIndex()
		switch {
		case key.Matches(msg, km.Down):
			if m.viewport.MoveDown() {
				m.rebuildViewport()
			}
			if m.viewport.NearBottom(shared.PaginationThreshold) && !m.loading && m.cursor != "" {
				m.loading = true
				return m, tea.Batch(m.fetchTimeline(m.cursor), m.spinner.Tick)
			}
		case key.Matches(msg, km.Up):
			if m.viewport.MoveUp() {
				m.rebuildViewport()
			}
		case key.Matches(msg, km.Reply):
			if idx < len(m.posts) {
				post := m.posts[idx].Post
				if post != nil {
					return m, func() tea.Msg { return ComposeReplyMsg{URI: post.Uri, CID: post.Cid} }
				}
			}
			return m, func() tea.Msg { return FeedRefreshMsg{} }
		case key.Matches(msg, km.Open):
			if idx < len(m.posts) {
				post := m.posts[idx].Post
				if post != nil {
					return m, func() tea.Msg { return ViewThreadMsg{URI: post.Uri} }
				}
			}
		case key.Matches(msg, km.Like):
			if idx < len(m.posts) {
				post := m.posts[idx].Post
				if post != nil {
					return m, func() tea.Msg { return LikePostMsg{URI: post.Uri, CID: post.Cid} }
				}
			}
		case key.Matches(msg, km.Repost):
			if idx < len(m.posts) {
				post := m.posts[idx].Post
				if post != nil {
					return m, func() tea.Msg { return RepostMsg{URI: post.Uri, CID: post.Cid} }
				}
			}
		case key.Matches(msg, km.Compose):
			return m, func() tea.Msg { return ComposeMsg{} }
		case key.Matches(msg, km.Delete):
			if idx < len(m.posts) {
				post := m.posts[idx]
				if post.Post != nil && post.Post.Author != nil {
					res := shared.CheckConfirmDelete(m.confirmDelete, idx, post.Post.Author.Did, m.ownDID, post.Post.Uri)
					m.confirmDelete = res.ConfirmDelete
					if res.Confirmed {
						uri := res.URI
						return m, func() tea.Msg { return DeletePostMsg{URI: uri} }
					}
					if res.URI == "" && res.ConfirmDelete == idx {
						return m, nil
					}
				}
			}
		}
	}
	return m, nil
}

func (m *FeedModel) rebuildViewport() {
	lazy := &images.LazyRenderer{Inner: m.imageCache}
	m.viewport.SetItems(len(m.posts), func(index int, selected bool) string {
		lazy.NearVisible = m.viewport.IsNearVisible(index, m.viewport.Height())
		return RenderPost(m.posts[index], m.width, selected, lazy, m.avatarOverrides)
	})
}

func (m FeedModel) View() tea.View {
	var content string

	if m.err != nil {
		s := theme.StyleError().Render("Error: "+m.err.Error()) + "\n\nPress 'r' to retry"
		content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s)
	} else if len(m.posts) == 0 {
		if m.loading {
			content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.spinner.View()+" Loading feed...")
		} else {
			content = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "No posts yet")
		}
	} else {
		rendered := m.viewport.View()
		if m.loading {
			rendered += "\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center, m.spinner.View()+" Loading more...")
		} else if m.imageCache != nil && m.imageCache.PendingCount() > 0 {
			rendered += "\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center, m.spinner.View()+" Loading images...")
		}
		if m.confirmDelete >= 0 {
			confirmStyle := lipgloss.NewStyle().
				Foreground(theme.ColorWarning).
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

// Keys returns the feed key map for help rendering.
func (m FeedModel) Keys() KeyMap {
	return m.keys
}

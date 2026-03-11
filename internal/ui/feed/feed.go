package feed

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
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
	client        bluesky.BlueskyClient
	posts         []*bsky.FeedDefs_FeedViewPost
	selectedIndex int
	cursor        string
	loading       bool
	width, height int
	err           error
	offset        int
}

func NewFeedModel(client bluesky.BlueskyClient, width, height int) FeedModel {
	return FeedModel{
		client:  client,
		width:   width,
		height:  height,
		loading: true,
	}
}

func (m FeedModel) Init() tea.Cmd {
	return m.fetchTimeline("")
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
		return m, nil

	case FeedErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case FeedRefreshMsg:
		m.loading = true
		m.posts = nil
		m.cursor = ""
		m.selectedIndex = 0
		m.offset = 0
		m.err = nil
		return m, m.fetchTimeline("")

	case LikePostMsg:
		if post := findPostByURI(m.posts, msg.URI); post != nil {
			if post.Post.Viewer == nil || post.Post.Viewer.Like == nil || *post.Post.Viewer.Like == "" {
				optimisticLike(post)
				uri, cid := msg.URI, msg.CID
				return m, performLike(m.client, uri, cid)
			}
			likeURI := *post.Post.Viewer.Like
			optimisticUnlike(post)
			return m, performUnlike(m.client, msg.URI, likeURI)
		}
		return m, nil

	case RepostMsg:
		if post := findPostByURI(m.posts, msg.URI); post != nil {
			if post.Post.Viewer == nil || post.Post.Viewer.Repost == nil || *post.Post.Viewer.Repost == "" {
				optimisticRepost(post)
				uri, cid := msg.URI, msg.CID
				return m, performRepost(m.client, uri, cid)
			}
			repostURI := *post.Post.Viewer.Repost
			optimisticUnRepost(post)
			return m, performUnRepost(m.client, msg.URI, repostURI)
		}
		return m, nil

	case LikeResultMsg:
		if post := findPostByURI(m.posts, msg.PostURI); post != nil {
			if msg.Err != nil {
				rollbackLike(post)
			} else {
				if post.Post.Viewer == nil {
					post.Post.Viewer = &bsky.FeedDefs_ViewerState{}
				}
				post.Post.Viewer.Like = &msg.LikeURI
			}
		}
		return m, nil

	case UnlikeResultMsg:
		if post := findPostByURI(m.posts, msg.PostURI); post != nil {
			if msg.Err != nil {
				optimisticLike(post)
			}
		}
		return m, nil

	case RepostResultMsg:
		if post := findPostByURI(m.posts, msg.PostURI); post != nil {
			if msg.Err != nil {
				rollbackRepost(post)
			} else {
				if post.Post.Viewer == nil {
					post.Post.Viewer = &bsky.FeedDefs_ViewerState{}
				}
				post.Post.Viewer.Repost = &msg.RepostURI
			}
		}
		return m, nil

	case UnRepostResultMsg:
		if post := findPostByURI(m.posts, msg.PostURI); post != nil {
			if msg.Err != nil {
				optimisticRepost(post)
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.err != nil && msg.String() == "r" {
			return m, func() tea.Msg { return FeedRefreshMsg{} }
		}

		switch msg.String() {
		case "j", "down":
			if m.selectedIndex < len(m.posts)-1 {
				m.selectedIndex++
				if m.selectedIndex > m.offset+m.visibleCount()-1 {
					m.offset++
				}
				if m.selectedIndex >= len(m.posts)-3 && !m.loading && m.cursor != "" {
					m.loading = true
					return m, m.fetchTimeline(m.cursor)
				}
			}
		case "k", "up":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				if m.selectedIndex < m.offset {
					m.offset = m.selectedIndex
				}
			}
		case "r":
			if m.selectedIndex < len(m.posts) {
				post := m.posts[m.selectedIndex].Post
				return m, func() tea.Msg { return ComposeReplyMsg{URI: post.Uri, CID: post.Cid} }
			}
			return m, func() tea.Msg { return FeedRefreshMsg{} }
		case "enter":
			if m.selectedIndex < len(m.posts) {
				return m, func() tea.Msg { return ViewThreadMsg{URI: m.posts[m.selectedIndex].Post.Uri} }
			}
		case "l":
			if m.selectedIndex < len(m.posts) {
				post := m.posts[m.selectedIndex].Post
				return m, func() tea.Msg { return LikePostMsg{URI: post.Uri, CID: post.Cid} }
			}
		case "t":
			if m.selectedIndex < len(m.posts) {
				post := m.posts[m.selectedIndex].Post
				return m, func() tea.Msg { return RepostMsg{URI: post.Uri, CID: post.Cid} }
			}
		case "c":
			return m, func() tea.Msg { return ComposeMsg{} }
		}
	}
	return m, nil
}

func (m FeedModel) visibleCount() int {
	// rough estimate of posts that fit in height, each post might be ~6 lines
	return m.height / 6
}

func (m FeedModel) View() tea.View {
	if m.err != nil {
		s := theme.StyleError.Render("Error: "+m.err.Error()) + "\n\nPress 'r' to retry"
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s))
	}

	if len(m.posts) == 0 {
		if m.loading {
			return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "Loading..."))
		}
		return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "No posts yet"))
	}

	var rendered string
	for i := m.offset; i < len(m.posts) && i < m.offset+m.visibleCount()+1; i++ {
		rendered += RenderPost(m.posts[i], m.width, i == m.selectedIndex)
	}

	if m.loading {
		rendered += "\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center, "Loading more...")
	}

	return tea.NewView(rendered)
}

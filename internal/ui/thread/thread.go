package thread

import (
	"context"
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

type ThreadLoadedMsg struct {
	Thread *bsky.FeedGetPostThread_Output
}
type ThreadErrorMsg struct{ Err error }
type BackMsg struct{}

// New messages not in feed package
type ComposeReplyMsg struct{ URI, CID string }
type ViewProfileMsg struct{ DID string }

type ThreadPost struct {
	Post     *bsky.FeedDefs_PostView
	IsTarget bool
	Depth    int
	IsParent bool
	NotFound bool
	Blocked  bool
}

type ThreadModel struct {
	threadPosts     []ThreadPost
	loading         bool
	width, height   int
	client          bluesky.BlueskyClient
	ownDID          string
	targetURI       string
	err             error
	spinner         spinner.Model
	confirmDelete   int // -1 = none
	imageCache      *images.Cache
	avatarOverrides map[string]string
	keys            KeyMap
	viewport        shared.ItemViewport
}

func NewThreadModel(client bluesky.BlueskyClient, uri, ownDID string, width, height int, cache *images.Cache) ThreadModel {
	sp := shared.NewSpinner()
	return ThreadModel{
		client:        client,
		targetURI:     uri,
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

func (m *ThreadModel) SetAvatarOverrides(overrides map[string]string) {
	m.avatarOverrides = overrides
}

func (m ThreadModel) Keys() KeyMap { return m.keys }

func (m ThreadModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		thread, err := m.client.GetPostThread(ctx, m.targetURI, 10)
		if err != nil {
			return ThreadErrorMsg{Err: err}
		}
		return ThreadLoadedMsg{Thread: thread}
	})
}

func (m ThreadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetSize(msg.Width, msg.Height)
		m.rebuildViewport()
		return m, nil

	case ThreadLoadedMsg:
		m.loading = false
		if msg.Thread != nil && msg.Thread.Thread != nil {
			m.threadPosts = flattenThread(msg.Thread.Thread)
			for i, p := range m.threadPosts {
				if p.IsTarget {
					m.viewport.SetSelectedIndex(i)
					break
				}
			}
		}
		m.rebuildViewport()
		var fetchCmds []tea.Cmd
		for _, tp := range m.threadPosts {
			if tp.Post == nil {
				continue
			}
			fvp := &bsky.FeedDefs_FeedViewPost{Post: tp.Post}

			if tp.Post.Embed != nil {
				for _, url := range feed.ExtractImageURLs(fvp) {
					if cmd := images.Fetch(m.imageCache, url); cmd != nil {
						fetchCmds = append(fetchCmds, cmd)
					}
				}
			}

			avatarURL := feed.ExtractAvatarURL(fvp)
			if tp.Post.Author != nil {
				if override, ok := m.avatarOverrides[tp.Post.Author.Did]; ok && override != "" {
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

	case images.ImageFetchedMsg:
		m.rebuildViewport()
		return m, nil

	case ThreadErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case feed.LikePostMsg:
		for _, tp := range m.threadPosts {
			if tp.Post != nil && tp.Post.Uri == msg.URI {
				if tp.Post.Viewer == nil || tp.Post.Viewer.Like == nil || *tp.Post.Viewer.Like == "" {
					feed.OptimisticLike(tp.Post)
					return m, feed.PerformLike(m.client, msg.URI, msg.CID)
				}
				likeURI := *tp.Post.Viewer.Like
				feed.OptimisticUnlike(tp.Post)
				return m, feed.PerformUnlike(m.client, msg.URI, likeURI)
			}
		}
		return m, nil

	case feed.RepostMsg:
		for _, tp := range m.threadPosts {
			if tp.Post != nil && tp.Post.Uri == msg.URI {
				if tp.Post.Viewer == nil || tp.Post.Viewer.Repost == nil || *tp.Post.Viewer.Repost == "" {
					feed.OptimisticRepost(tp.Post)
					return m, feed.PerformRepost(m.client, msg.URI, msg.CID)
				}
				repostURI := *tp.Post.Viewer.Repost
				feed.OptimisticUnRepost(tp.Post)
				return m, feed.PerformUnRepost(m.client, msg.URI, repostURI)
			}
		}
		return m, nil

	case feed.LikeResultMsg:
		for _, tp := range m.threadPosts {
			if tp.Post != nil && tp.Post.Uri == msg.PostURI {
				if msg.Err != nil {
					feed.RollbackLike(tp.Post)
				} else {
					if tp.Post.Viewer == nil {
						tp.Post.Viewer = &bsky.FeedDefs_ViewerState{}
					}
					tp.Post.Viewer.Like = &msg.LikeURI
				}
				break
			}
		}
		return m, nil

	case feed.UnlikeResultMsg:
		for _, tp := range m.threadPosts {
			if tp.Post != nil && tp.Post.Uri == msg.PostURI {
				if msg.Err != nil {
					feed.OptimisticLike(tp.Post)
				}
				break
			}
		}
		return m, nil

	case feed.RepostResultMsg:
		for _, tp := range m.threadPosts {
			if tp.Post != nil && tp.Post.Uri == msg.PostURI {
				if msg.Err != nil {
					feed.RollbackRepost(tp.Post)
				} else {
					if tp.Post.Viewer == nil {
						tp.Post.Viewer = &bsky.FeedDefs_ViewerState{}
					}
					tp.Post.Viewer.Repost = &msg.RepostURI
				}
				break
			}
		}
		return m, nil

	case feed.UnRepostResultMsg:
		for _, tp := range m.threadPosts {
			if tp.Post != nil && tp.Post.Uri == msg.PostURI {
				if msg.Err != nil {
					feed.OptimisticRepost(tp.Post)
				}
				break
			}
		}
		return m, nil

	case feed.DeletePostResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		for i, p := range m.threadPosts {
			if p.Post != nil && p.Post.Uri == msg.URI {
				m.threadPosts = append(m.threadPosts[:i], m.threadPosts[i+1:]...)
				idx := m.viewport.SelectedIndex()
				if idx >= len(m.threadPosts) && idx > 0 {
					m.viewport.SetSelectedIndex(len(m.threadPosts) - 1)
				}
				break
			}
		}
		m.confirmDelete = -1
		m.rebuildViewport()
		return m, nil

	case tea.KeyPressMsg:
		km := m.keys
		if m.err != nil {
			if key.Matches(msg, km.Back) {
				return m, func() tea.Msg { return BackMsg{} }
			}
			return m, nil
		}

		if !key.Matches(msg, km.Delete) {
			m.confirmDelete = -1
		}

		idx := m.viewport.SelectedIndex()
		switch {
		case key.Matches(msg, km.Back):
			return m, func() tea.Msg { return BackMsg{} }
		case key.Matches(msg, km.Down):
			if m.viewport.MoveDown() {
				m.rebuildViewport()
			}
		case key.Matches(msg, km.Up):
			if m.viewport.MoveUp() {
				m.rebuildViewport()
			}
		case key.Matches(msg, km.Open):
			if idx < len(m.threadPosts) {
				p := m.threadPosts[idx]
				if p.Post != nil && p.Depth > 0 {
					return m, func() tea.Msg { return feed.ViewThreadMsg{URI: p.Post.Uri} }
				}
			}
		case key.Matches(msg, km.Like):
			if idx < len(m.threadPosts) {
				p := m.threadPosts[idx]
				if p.Post != nil {
					return m, func() tea.Msg { return feed.LikePostMsg{URI: p.Post.Uri, CID: p.Post.Cid} }
				}
			}
		case key.Matches(msg, km.Repost):
			if idx < len(m.threadPosts) {
				p := m.threadPosts[idx]
				if p.Post != nil {
					return m, func() tea.Msg { return feed.RepostMsg{URI: p.Post.Uri, CID: p.Post.Cid} }
				}
			}
		case key.Matches(msg, km.Reply):
			if idx < len(m.threadPosts) {
				p := m.threadPosts[idx]
				if p.Post != nil {
					return m, func() tea.Msg { return ComposeReplyMsg{URI: p.Post.Uri, CID: p.Post.Cid} }
				}
			}
		case key.Matches(msg, km.Profile):
			if idx < len(m.threadPosts) {
				p := m.threadPosts[idx]
				if p.Post != nil && p.Post.Author != nil {
					return m, func() tea.Msg { return ViewProfileMsg{DID: p.Post.Author.Did} }
				}
			}
		case key.Matches(msg, km.Delete):
			if idx < len(m.threadPosts) {
				p := m.threadPosts[idx]
				if p.Post != nil && p.Post.Author != nil {
					res := shared.CheckConfirmDelete(m.confirmDelete, idx, p.Post.Author.Did, m.ownDID, p.Post.Uri)
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
		}

	case tea.MouseWheelMsg:
		mouse := msg.Mouse()
		switch mouse.Button {
		case tea.MouseWheelDown:
			if m.viewport.MoveDownN(3) {
				m.rebuildViewport()
			}
		case tea.MouseWheelUp:
			if m.viewport.MoveUpN(3) {
				m.rebuildViewport()
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *ThreadModel) rebuildViewport() {
	lazy := &images.LazyRenderer{Inner: m.imageCache}
	m.viewport.SetItems(len(m.threadPosts), func(index int, selected bool) string {
		lazy.NearVisible = m.viewport.IsNearVisible(index, m.viewport.Height())
		return m.renderThreadPost(index, selected, lazy)
	})
}

func (m ThreadModel) renderThreadPost(index int, selected bool, renderer images.ImageRenderer) string {
	tp := m.threadPosts[index]

	if tp.NotFound {
		content := "[Deleted post]"
		if selected {
			return theme.StyleSelected().Render("▶ "+content) + "\n\n"
		}
		return theme.StyleMuted().Render("  "+content) + "\n\n"
	}

	if tp.Blocked {
		content := "[Blocked post]"
		if selected {
			return theme.StyleSelected().Render("▶ "+content) + "\n\n"
		}
		return theme.StyleMuted().Render("  "+content) + "\n\n"
	}

	if tp.Post == nil {
		return ""
	}

	fvp := &bsky.FeedDefs_FeedViewPost{Post: tp.Post}

	postWidth := m.width
	indent := ""

	if tp.IsParent {
		indent = "│ "
		postWidth -= 2
	} else if tp.Depth > 0 {
		spaces := strings.Repeat("  ", tp.Depth)
		indent = spaces
		postWidth -= len(spaces)
	}

	if postWidth < 20 {
		postWidth = 20
	}

	if tp.IsTarget {
		contentWidth := max(10, m.width-2) // border(1) + padding(1)
		rawContent := feed.RenderPostContent(fvp, contentWidth, renderer, m.avatarOverrides)

		separator := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Render(strings.Repeat("═", m.width))
		bordered := shared.RenderItemWithBorder(rawContent, true, m.width)
		return separator + "\n" + bordered + separator + "\n"
	}

	// Non-target: get raw content, apply indent, then single border via RenderItemWithBorder.
	contentWidth := max(10, postWidth-2) // border(1) + padding(1)
	rawContent := feed.RenderPostContent(fvp, contentWidth, renderer, m.avatarOverrides)

	if indent != "" {
		indentedLines := strings.Split(rawContent, "\n")
		for j, line := range indentedLines {
			if line == "" {
				continue
			}
			if tp.IsParent {
				indentedLines[j] = theme.StyleMuted().Render(indent) + line
			} else {
				indentedLines[j] = indent + line
			}
		}
		rawContent = strings.Join(indentedLines, "\n")
	}

	return shared.RenderItemWithBorder(rawContent, selected, postWidth)
}

func (m ThreadModel) View() tea.View {
	mouseView := func(s string) tea.View {
		v := tea.NewView(s)
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if m.err != nil {
		s := theme.StyleError().Render("Error: "+m.err.Error()) + "\n\nPress 'esc' to go back"
		return mouseView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s))
	}

	if m.loading {
		return mouseView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.spinner.View()+" Loading thread..."))
	}

	if len(m.threadPosts) == 0 {
		return mouseView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "No posts found"))
	}

	content := m.viewport.View()

	if m.confirmDelete >= 0 {
		confirmStyle := lipgloss.NewStyle().
			Foreground(theme.ColorWarning).
			Bold(true)
		content += "\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
			confirmStyle.Render("Press d to confirm delete, any other key to cancel"))
	}

	return mouseView(content)
}

func flattenThread(root *bsky.FeedGetPostThread_Output_Thread) []ThreadPost {
	var result []ThreadPost

	if root.FeedDefs_ThreadViewPost != nil {
		tvp := root.FeedDefs_ThreadViewPost

		var parents []ThreadPost
		currParent := tvp.Parent
		for currParent != nil {
			if currParent.FeedDefs_ThreadViewPost != nil {
				parents = append(parents, ThreadPost{
					Post:     currParent.FeedDefs_ThreadViewPost.Post,
					IsParent: true,
				})
				currParent = currParent.FeedDefs_ThreadViewPost.Parent
			} else if currParent.FeedDefs_NotFoundPost != nil {
				parents = append(parents, ThreadPost{NotFound: true, IsParent: true})
				break
			} else if currParent.FeedDefs_BlockedPost != nil {
				parents = append(parents, ThreadPost{Blocked: true, IsParent: true})
				break
			} else {
				break
			}
		}

		// Reverse parents
		for i, j := 0, len(parents)-1; i < j; i, j = i+1, j-1 {
			parents[i], parents[j] = parents[j], parents[i]
		}

		result = append(result, parents...)

		result = append(result, ThreadPost{
			Post:     tvp.Post,
			IsTarget: true,
		})

		result = append(result, flattenReplies(tvp.Replies, 1)...)

	} else if root.FeedDefs_NotFoundPost != nil {
		result = append(result, ThreadPost{NotFound: true})
	} else if root.FeedDefs_BlockedPost != nil {
		result = append(result, ThreadPost{Blocked: true})
	}

	return result
}

func flattenReplies(replies []*bsky.FeedDefs_ThreadViewPost_Replies_Elem, depth int) []ThreadPost {
	var result []ThreadPost
	for _, r := range replies {
		if r.FeedDefs_ThreadViewPost != nil {
			result = append(result, ThreadPost{
				Post:  r.FeedDefs_ThreadViewPost.Post,
				Depth: depth,
			})
			if len(r.FeedDefs_ThreadViewPost.Replies) > 0 {
				result = append(result, flattenReplies(r.FeedDefs_ThreadViewPost.Replies, depth+1)...)
			}
		} else if r.FeedDefs_NotFoundPost != nil {
			result = append(result, ThreadPost{NotFound: true, Depth: depth})
		} else if r.FeedDefs_BlockedPost != nil {
			result = append(result, ThreadPost{Blocked: true, Depth: depth})
		}
	}
	return result
}

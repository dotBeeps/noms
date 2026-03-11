package thread

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/feed"
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
	threadPosts   []ThreadPost
	selectedIndex int
	loading       bool
	width, height int
	client        bluesky.BlueskyClient
	ownDID        string
	targetURI     string
	err           error
	offset        int
	spinner       spinner.Model
	confirmDelete int // -1 = none
}

func NewThreadModel(client bluesky.BlueskyClient, uri, ownDID string, width, height int) ThreadModel {
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(theme.ColorAccent)),
	)
	return ThreadModel{
		client:        client,
		targetURI:     uri,
		ownDID:        ownDID,
		width:         width,
		height:        height,
		loading:       true,
		spinner:       sp,
		confirmDelete: -1,
	}
}

func (m ThreadModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		thread, err := m.client.GetPostThread(context.Background(), m.targetURI, 10)
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
		return m, nil

	case ThreadLoadedMsg:
		m.loading = false
		if msg.Thread != nil && msg.Thread.Thread != nil {
			m.threadPosts = flattenThread(msg.Thread.Thread)
			// Find target to select it initially
			for i, p := range m.threadPosts {
				if p.IsTarget {
					m.selectedIndex = i
					m.offset = max(0, i-2) // Try to show some context above
					break
				}
			}
		}
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

	case feed.DeletePostResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		for i, p := range m.threadPosts {
			if p.Post != nil && p.Post.Uri == msg.URI {
				m.threadPosts = append(m.threadPosts[:i], m.threadPosts[i+1:]...)
				if m.selectedIndex >= len(m.threadPosts) && m.selectedIndex > 0 {
					m.selectedIndex--
				}
				break
			}
		}
		m.confirmDelete = -1
		return m, nil

	case tea.KeyPressMsg:
		if m.err != nil {
			if msg.String() == "esc" || msg.String() == "backspace" {
				return m, func() tea.Msg { return BackMsg{} }
			}
			return m, nil
		}

		switch msg.String() {
		case "esc", "backspace":
			return m, func() tea.Msg { return BackMsg{} }
		case "j", "down":
			if m.selectedIndex < len(m.threadPosts)-1 {
				m.selectedIndex++
				if m.selectedIndex > m.offset+m.visibleCount()-1 {
					m.offset++
				}
			}
		case "k", "up":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				if m.selectedIndex < m.offset {
					m.offset = m.selectedIndex
				}
			}
		case "enter":
			if m.selectedIndex >= 0 && m.selectedIndex < len(m.threadPosts) {
				p := m.threadPosts[m.selectedIndex]
				if p.Post != nil && p.Depth > 0 {
					return m, func() tea.Msg { return feed.ViewThreadMsg{URI: p.Post.Uri} }
				}
			}
		case "l":
			if m.selectedIndex >= 0 && m.selectedIndex < len(m.threadPosts) {
				p := m.threadPosts[m.selectedIndex]
				if p.Post != nil {
					return m, func() tea.Msg { return feed.LikePostMsg{URI: p.Post.Uri, CID: p.Post.Cid} }
				}
			}
		case "t":
			if m.selectedIndex >= 0 && m.selectedIndex < len(m.threadPosts) {
				p := m.threadPosts[m.selectedIndex]
				if p.Post != nil {
					return m, func() tea.Msg { return feed.RepostMsg{URI: p.Post.Uri, CID: p.Post.Cid} }
				}
			}
		case "r":
			if m.selectedIndex >= 0 && m.selectedIndex < len(m.threadPosts) {
				p := m.threadPosts[m.selectedIndex]
				if p.Post != nil {
					return m, func() tea.Msg { return ComposeReplyMsg{URI: p.Post.Uri, CID: p.Post.Cid} }
				}
			}
		case "p":
			if m.selectedIndex >= 0 && m.selectedIndex < len(m.threadPosts) {
				p := m.threadPosts[m.selectedIndex]
				if p.Post != nil && p.Post.Author != nil {
					return m, func() tea.Msg { return ViewProfileMsg{DID: p.Post.Author.Did} }
				}
			}
		case "d":
			if m.selectedIndex >= 0 && m.selectedIndex < len(m.threadPosts) {
				p := m.threadPosts[m.selectedIndex]
				if p.Post != nil && p.Post.Author != nil && p.Post.Author.Did == m.ownDID {
					if m.confirmDelete == m.selectedIndex {
						uri := p.Post.Uri
						m.confirmDelete = -1
						return m, func() tea.Msg { return feed.DeletePostMsg{URI: uri} }
					}
					m.confirmDelete = m.selectedIndex
					return m, nil
				}
			}
		}

		if msg.String() != "d" {
			m.confirmDelete = -1
		}

	case tea.MouseWheelMsg:
		mouse := msg.Mouse()
		switch mouse.Button {
		case tea.MouseWheelDown:
			for range 3 {
				if m.selectedIndex < len(m.threadPosts)-1 {
					m.selectedIndex++
				}
			}
			if m.selectedIndex > m.offset+m.visibleCount()-1 {
				m.offset = m.selectedIndex - m.visibleCount() + 1
			}
		case tea.MouseWheelUp:
			for range 3 {
				if m.selectedIndex > 0 {
					m.selectedIndex--
				}
			}
			if m.selectedIndex < m.offset {
				m.offset = m.selectedIndex
			}
		}
		return m, nil
	}
	return m, nil
}

func (m ThreadModel) visibleCount() int {
	return max(1, m.height/6)
}

func (m ThreadModel) View() tea.View {
	mouseView := func(s string) tea.View {
		v := tea.NewView(s)
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	if m.err != nil {
		s := theme.StyleError.Render("Error: "+m.err.Error()) + "\n\nPress 'esc' to go back"
		return mouseView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s))
	}

	if m.loading {
		return mouseView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.spinner.View()+" Loading thread..."))
	}

	if len(m.threadPosts) == 0 {
		return mouseView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "No posts found"))
	}

	var rendered strings.Builder
	for i := m.offset; i < len(m.threadPosts) && i < m.offset+m.visibleCount()+1; i++ {
		tp := m.threadPosts[i]
		isSelected := (i == m.selectedIndex)

		if tp.NotFound {
			content := "[Deleted post]"
			if isSelected {
				rendered.WriteString(theme.StyleSelected.Render("▶ " + content))
			} else {
				rendered.WriteString(theme.StyleMuted.Render("  " + content))
			}
			rendered.WriteString("\n\n")
			continue
		}

		if tp.Blocked {
			content := "[Blocked post]"
			if isSelected {
				rendered.WriteString(theme.StyleSelected.Render("▶ " + content))
			} else {
				rendered.WriteString(theme.StyleMuted.Render("  " + content))
			}
			rendered.WriteString("\n\n")
			continue
		}

		if tp.Post == nil {
			continue
		}

		fvp := &bsky.FeedDefs_FeedViewPost{Post: tp.Post}

		// Apply indentation/styling based on depth/parent
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

		postStr := feed.RenderPost(fvp, postWidth, isSelected)

		// Apply indent line by line
		lines := strings.Split(postStr, "\n")
		for j, line := range lines {
			if tp.IsParent {
				lines[j] = theme.StyleMuted.Render(indent) + line
			} else {
				lines[j] = indent + line
			}
		}

		finalStr := strings.Join(lines, "\n")

		// Highlight target post
		if tp.IsTarget {
			// Apply a border or some styling to the whole block if desired,
			// but we will just wrap it with an accent border.
			style := lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(theme.ColorPrimary).
				Padding(0, 1).
				Width(m.width - 4)
			finalStr = style.Render(finalStr)
		}

		rendered.WriteString(finalStr)
		rendered.WriteString("\n")
	}

	return mouseView(rendered.String())
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

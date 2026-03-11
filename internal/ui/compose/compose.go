package compose

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/feed"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// ComposeMode represents the type of compose action
type ComposeMode int

const (
	ModeNewPost ComposeMode = iota
	ModeReply
	ModeQuote

	maxPostChars = 300
)

// PostCreatedMsg is emitted when a post is successfully created
type PostCreatedMsg struct {
	URI string
	CID string
}

// CancelComposeMsg is emitted when the user cancels composing
type CancelComposeMsg struct{}

// ComposeErrorMsg represents an internal error during compose
type ComposeErrorMsg struct {
	Err error
}

// ParentPostLoadedMsg delivers a fetched parent post to the compose model.
type ParentPostLoadedMsg struct {
	Post *bsky.FeedDefs_PostView
}

// postSuccessMsg is an internal message for successful post creation
type postSuccessMsg struct {
	uri string
	cid string
}

// ComposeModel is the Bubble Tea model for the compose screen
type ComposeModel struct {
	textarea   textarea.Model
	mode       ComposeMode
	parentPost *bsky.FeedDefs_PostView
	client     bluesky.BlueskyClient
	width      int
	height     int
	loading    bool
	err        error
}

// NewComposeModel creates a new compose model
func NewComposeModel(client bluesky.BlueskyClient, mode ComposeMode, parentPost *bsky.FeedDefs_PostView, width, height int) ComposeModel {
	ta := textarea.New()
	ta.SetWidth(max(1, width-4))
	ta.SetHeight(6)
	ta.Placeholder = "What's on your mind?"
	ta.Focus()

	return ComposeModel{
		textarea:   ta,
		mode:       mode,
		parentPost: parentPost,
		client:     client,
		width:      width,
		height:     height,
	}
}

// Init initializes the compose model
func (m ComposeModel) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
	)
}

// Update handles messages for the compose model
func (m ComposeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(max(1, m.width-4))
		return m, nil

	case postSuccessMsg:
		return m, func() tea.Msg { return PostCreatedMsg{URI: msg.uri, CID: msg.cid} }

	case ParentPostLoadedMsg:
		m.parentPost = msg.Post
		return m, nil

	case ComposeErrorMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case tea.KeyPressMsg:
		// Handle special keys before passing to textarea
		switch msg.String() {
		case "ctrl+enter":
			if m.loading {
				return m, nil
			}
			m.loading = true
			return m, m.submitPost()

		case "esc":
			if m.err != nil {
				// Clear error on esc
				m.err = nil
				return m, nil
			}
			return m, func() tea.Msg { return CancelComposeMsg{} }
		}
	}

	// Update textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// submitPost creates and submits the post
func (m ComposeModel) submitPost() tea.Cmd {
	return func() tea.Msg {
		text := m.textarea.Value()
		if len(strings.TrimSpace(text)) == 0 {
			return ComposeErrorMsg{Err: fmt.Errorf("post cannot be empty")}
		}

		// Detect facets (mentions, links)
		facets := bluesky.DetectFacets(text)

		var reply *bsky.FeedPost_ReplyRef
		var embed *bsky.FeedPost_Embed

		if m.mode == ModeReply && m.parentPost != nil {
			reply = &bsky.FeedPost_ReplyRef{
				Root: &comatproto.RepoStrongRef{
					Cid: m.parentPost.Cid,
					Uri: m.parentPost.Uri,
				},
				Parent: &comatproto.RepoStrongRef{
					Cid: m.parentPost.Cid,
					Uri: m.parentPost.Uri,
				},
			}
		}

		if m.mode == ModeQuote && m.parentPost != nil {
			embed = &bsky.FeedPost_Embed{
				EmbedRecord: &bsky.EmbedRecord{
					LexiconTypeID: "app.bsky.embed.record",
					Record: &comatproto.RepoStrongRef{
						Uri: m.parentPost.Uri,
						Cid: m.parentPost.Cid,
					},
				},
			}
		}

		uri, cid, err := m.client.CreatePost(context.Background(), text, facets, reply, embed)
		if err != nil {
			return ComposeErrorMsg{Err: err}
		}

		return postSuccessMsg{uri: uri, cid: cid}
	}
}

// View renders the compose screen
func (m ComposeModel) View() tea.View {
	var b strings.Builder

	// Modal overlay style
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorPrimary).
		Padding(1, 2).
		Width(m.width - 4)

	// Header based on mode
	var header string
	switch m.mode {
	case ModeNewPost:
		header = theme.StyleHeader.Render("New Post")
	case ModeReply:
		header = theme.StyleHeader.Render("Reply")
	case ModeQuote:
		header = theme.StyleHeader.Render("Quote Post")
	}
	b.WriteString(header)
	b.WriteString("\n\n")

	// For reply mode, show parent post above textarea
	if m.mode == ModeReply && m.parentPost != nil {
		parentFeedPost := &bsky.FeedDefs_FeedViewPost{
			Post: m.parentPost,
		}
		parentRendered := feed.RenderPost(parentFeedPost, m.width-8, false)
		b.WriteString(theme.StyleMuted.Render("Replying to:"))
		b.WriteString("\n")
		b.WriteString(parentRendered)
		b.WriteString("\n")
	}

	// Textarea
	b.WriteString(m.textarea.View())
	b.WriteString("\n\n")

	// Character counter
	text := m.textarea.Value()
	charCount := len([]rune(text))
	counterStyle := theme.StyleMuted
	if charCount > maxPostChars {
		counterStyle = theme.StyleError
	}
	counter := counterStyle.Render(fmt.Sprintf("%d/%d", charCount, maxPostChars))
	b.WriteString(counter)

	// Key hints
	hints := theme.StyleMuted.Render("  Ctrl+Enter: Submit  |  Esc: Cancel")
	b.WriteString(hints)
	b.WriteString("\n")

	// For quote mode, show quoted post below textarea
	if m.mode == ModeQuote && m.parentPost != nil {
		b.WriteString("\n")
		b.WriteString(theme.StyleMuted.Render("Quoting:"))
		b.WriteString("\n")
		quotedFeedPost := &bsky.FeedDefs_FeedViewPost{
			Post: m.parentPost,
		}
		quotedRendered := feed.RenderPost(quotedFeedPost, m.width-8, false)
		b.WriteString(quotedRendered)
	}

	// Error display
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(theme.StyleError.Render("Error: " + m.err.Error()))
		b.WriteString("\n")
		b.WriteString(theme.StyleMuted.Render("Press Esc to clear error"))
	}

	// Loading indicator
	if m.loading {
		b.WriteString("\n")
		b.WriteString(theme.StyleMuted.Render("Posting..."))
	}

	content := style.Render(b.String())
	return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content))
}

// SetText sets the text content of the textarea
func (m *ComposeModel) SetText(text string) {
	m.textarea.SetValue(text)
}

// Text returns the current text content
func (m ComposeModel) Text() string {
	return m.textarea.Value()
}

// CharCount returns the current character count (Unicode-aware)
func (m ComposeModel) CharCount() int {
	return len([]rune(m.textarea.Value()))
}

// Mode returns the current compose mode
func (m ComposeModel) Mode() ComposeMode {
	return m.mode
}

// SetLoading sets the loading state
func (m *ComposeModel) SetLoading(loading bool) {
	m.loading = loading
}

// Error returns the current error
func (m ComposeModel) Error() error {
	return m.err
}

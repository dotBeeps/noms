package compose

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/feed"
	"github.com/dotBeeps/noms/internal/ui/images"
	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
	"github.com/rivo/uniseg"
)

// ComposeMode represents the type of compose action
type ComposeMode int

const (
	ModeNewPost ComposeMode = iota
	ModeReply
	ModeQuote

	maxPostChars    = 300
	warnCharsThresh = 270
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
	textarea        textarea.Model
	spinner         spinner.Model
	progress        progress.Model
	mode            ComposeMode
	parentPost      *bsky.FeedDefs_PostView
	client          bluesky.BlueskyClient
	imageCache      images.ImageRenderer
	width           int
	height          int
	loading         bool
	err             error
	avatarOverrides map[string]string
}

// NewComposeModel creates a new compose model
func NewComposeModel(client bluesky.BlueskyClient, mode ComposeMode, parentPost *bsky.FeedDefs_PostView, width, height int, imageCache images.ImageRenderer) ComposeModel {
	ta := textarea.New()
	ta.SetWidth(max(1, width-10))
	ta.SetHeight(6)
	ta.Placeholder = "What's on your mind?"
	ta.Focus()

	prog := progress.New(
		progress.WithColors(theme.ColorPrimary),
		progress.WithoutPercentage(),
	)
	prog.SetWidth(max(1, width-10))

	return ComposeModel{
		textarea:   ta,
		spinner:    shared.NewSpinner(),
		progress:   prog,
		mode:       mode,
		parentPost: parentPost,
		client:     client,
		imageCache: imageCache,
		width:      width,
		height:     height,
	}
}

func (m *ComposeModel) SetAvatarOverrides(overrides map[string]string) {
	m.avatarOverrides = overrides
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
		m.textarea.SetWidth(max(1, m.width-10))
		m.progress.SetWidth(max(1, m.width-10))
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

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyPressMsg:
		km := DefaultKeyMap
		switch {
		case key.Matches(msg, km.Submit):
			if m.loading {
				return m, nil
			}
			m.loading = true
			return m, tea.Batch(m.submitPost(), m.spinner.Tick)

		case key.Matches(msg, km.Cancel):
			if m.err != nil {
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

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		uri, cid, err := m.client.CreatePost(ctx, text, facets, reply, embed)
		if err != nil {
			return ComposeErrorMsg{Err: err}
		}

		return postSuccessMsg{uri: uri, cid: cid}
	}
}

// View renders the compose screen
func (m ComposeModel) View() tea.View {
	// Modal overlay style
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorPrimary).
		Padding(1, 2).
		Width(m.width - 6)

	// Header based on mode
	var header string
	switch m.mode {
	case ModeNewPost:
		header = theme.StyleHeader().Render("New Post")
	case ModeReply:
		header = theme.StyleHeader().Render("Reply")
	case ModeQuote:
		header = theme.StyleHeader().Render("Quote Post")
	}

	var sections []string
	sections = append(sections, header, "") // blank line after header

	// For reply mode, show parent post above textarea
	if m.mode == ModeReply && m.parentPost != nil {
		parentFeedPost := &bsky.FeedDefs_FeedViewPost{
			Post: m.parentPost,
		}
		parentRendered := feed.RenderPost(parentFeedPost, m.width-8, false, m.imageCache, m.avatarOverrides)
		sections = append(sections,
			theme.StyleMuted().Render("Replying to:"),
			parentRendered,
		)
	}

	// Textarea
	sections = append(sections, m.textarea.View(), "") // blank line after textarea

	// Character counter + key hints
	text := m.textarea.Value()
	charCount := uniseg.GraphemeClusterCount(text)
	counterStyle := theme.StyleMuted()
	progressColor := theme.ColorPrimary
	switch {
	case charCount > maxPostChars:
		counterStyle = theme.StyleError()
		progressColor = theme.ColorError
	case charCount >= warnCharsThresh:
		counterStyle = theme.StyleWarning()
		progressColor = theme.ColorWarning
	}
	counter := counterStyle.Render(fmt.Sprintf("%d/%d", charCount, maxPostChars))
	hints := theme.StyleMuted().Render("  Ctrl+Enter: Submit  |  Esc: Cancel")
	sections = append(sections, counter+hints)

	// Progress bar
	pct := min(float64(charCount)/float64(maxPostChars), 1.0)
	m.progress.FullColor = progressColor
	sections = append(sections, m.progress.ViewAs(pct))

	// For quote mode, show quoted post below textarea
	if m.mode == ModeQuote && m.parentPost != nil {
		quotedFeedPost := &bsky.FeedDefs_FeedViewPost{
			Post: m.parentPost,
		}
		quotedRendered := feed.RenderPost(quotedFeedPost, m.width-8, false, m.imageCache, m.avatarOverrides)
		sections = append(sections,
			"",
			theme.StyleMuted().Render("Quoting:"),
			quotedRendered,
		)
	}

	// Error display
	if m.err != nil {
		sections = append(sections,
			"",
			theme.StyleError().Render("Error: "+m.err.Error()),
			theme.StyleMuted().Render("Press Esc to clear error"),
		)
	}

	// Loading indicator
	if m.loading {
		sections = append(sections,
			"",
			theme.StyleMuted().Render(m.spinner.View()+" Posting..."),
		)
	}

	content := style.Render(lipgloss.JoinVertical(lipgloss.Left, sections...))
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
	return uniseg.GraphemeClusterCount(m.textarea.Value())
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

package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/feed"
	"github.com/dotBeeps/noms/internal/ui/images"
	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

type SearchMode int

const (
	ModePosts SearchMode = iota
	ModePeople
)

type debounceMsg struct {
	ts time.Time
}

type SearchResultsMsg struct {
	Posts  []*bsky.FeedDefs_PostView
	Actors []*bsky.ActorDefs_ProfileView
	Cursor string
	Mode   SearchMode
	Append bool
}

type SearchErrorMsg struct {
	Err error
}

type SearchNextPageMsg struct{}

// Reuse feed.ViewThreadMsg for posts navigation
// type feed.ViewThreadMsg struct{ URI string }

type ViewProfileMsg struct {
	DID string
}

type SearchModel struct {
	client          bluesky.BlueskyClient
	input           textinput.Model
	mode            SearchMode
	postResults     []*bsky.FeedDefs_PostView
	actorResults    []*bsky.ActorDefs_ProfileView
	cursor          string
	loading         bool
	lastKeystroke   time.Time
	width           int
	height          int
	query           string
	err             error
	spinner         spinner.Model
	imageCache      *images.Cache
	avatarOverrides map[string]string
	keys            KeyMap
	viewport        shared.ItemViewport
}

func NewSearchModel(client bluesky.BlueskyClient, width, height int, cache *images.Cache) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.Focus()
	ti.Prompt = "🔍 "

	sp := shared.NewSpinner()

	contentHeight := max(1, height-5)
	return SearchModel{
		client:       client,
		input:        ti,
		mode:         ModePosts,
		postResults:  []*bsky.FeedDefs_PostView{},
		actorResults: []*bsky.ActorDefs_ProfileView{},
		loading:      false,
		width:        width,
		height:       height,
		spinner:      sp,
		imageCache:   cache,
		keys:         DefaultKeyMap,
		viewport:     shared.NewItemViewport(width, contentHeight),
	}
}

func (m *SearchModel) SetAvatarOverrides(overrides map[string]string) {
	m.avatarOverrides = overrides
}

func (m SearchModel) Keys() KeyMap { return m.keys }

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(m.width - 4)
		m.viewport.SetSize(m.width, max(1, m.height-5))
		m.rebuildViewport()

	case tea.KeyPressMsg:
		km := m.keys
		switch {
		case msg.String() == "esc":
			if m.input.Focused() {
				m.input.Blur()
			} else {
				m.input.SetValue("")
				m.query = ""
				m.postResults = nil
				m.actorResults = nil
				m.cursor = ""
				m.viewport.Reset()
				m.err = nil
			}
			return m, nil

		case key.Matches(msg, km.Focus):
			if !m.input.Focused() {
				m.input.Focus()
				return m, textinput.Blink
			}

		case key.Matches(msg, km.Toggle):
			if m.mode == ModePosts {
				m.mode = ModePeople
			} else {
				m.mode = ModePosts
			}
			m.viewport.Reset()
			m.cursor = ""
			m.postResults = nil
			m.actorResults = nil
			if m.query != "" {
				m.loading = true
				return m, tea.Batch(m.performSearch(m.query, "", m.mode, false), m.spinner.Tick)
			}
			return m, nil

		case key.Matches(msg, km.Open):
			if m.input.Focused() {
				m.input.Blur()
				return m, nil
			}
			idx := m.viewport.SelectedIndex()
			if m.mode == ModePosts && len(m.postResults) > 0 && idx < len(m.postResults) {
				post := m.postResults[idx]
				return m, func() tea.Msg { return feed.ViewThreadMsg{URI: post.Uri} }
			} else if m.mode == ModePeople && len(m.actorResults) > 0 && idx < len(m.actorResults) {
				actor := m.actorResults[idx]
				return m, func() tea.Msg { return ViewProfileMsg{DID: actor.Did} }
			}

		case key.Matches(msg, km.Up):
			if !m.input.Focused() {
				if m.viewport.MoveUp() {
					m.rebuildViewport()
				}
				return m, nil
			}

		case key.Matches(msg, km.Down):
			if !m.input.Focused() {
				if m.viewport.MoveDown() {
					m.rebuildViewport()
				}
				if m.viewport.NearBottom(shared.PaginationThreshold) && m.cursor != "" && !m.loading {
					m.loading = true
					return m, tea.Batch(m.performSearch(m.query, m.cursor, m.mode, true), m.spinner.Tick)
				}
				return m, nil
			}
		}

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.MouseWheelMsg:
		if !m.input.Focused() {
			mouse := msg.Mouse()
			switch mouse.Button {
			case tea.MouseWheelDown:
				if m.viewport.MoveDownN(3) {
					m.rebuildViewport()
				}
				if m.viewport.NearBottom(shared.PaginationThreshold) && m.cursor != "" && !m.loading {
					m.loading = true
					return m, tea.Batch(m.performSearch(m.query, m.cursor, m.mode, true), m.spinner.Tick)
				}
			case tea.MouseWheelUp:
				if m.viewport.MoveUpN(3) {
					m.rebuildViewport()
				}
			}
			return m, nil
		}

	case debounceMsg:
		if msg.ts.Equal(m.lastKeystroke) {
			q := m.input.Value()
			if q != m.query {
				m.query = q
				m.cursor = ""
				m.viewport.Reset()
				m.postResults = nil
				m.actorResults = nil
				if q != "" {
					m.loading = true
					return m, tea.Batch(m.performSearch(q, "", m.mode, false), m.spinner.Tick)
				}
			}
		}

	case SearchResultsMsg:
		m.loading = false
		if msg.Mode != m.mode {
			return m, nil // Stale response
		}
		var fetchCmds []tea.Cmd
		if msg.Append {
			if m.mode == ModePosts {
				m.postResults = append(m.postResults, msg.Posts...)
			} else {
				m.actorResults = append(m.actorResults, msg.Actors...)
			}
		} else {
			if m.mode == ModePosts {
				m.postResults = msg.Posts
			} else {
				m.actorResults = msg.Actors
			}
		}
		m.cursor = msg.Cursor
		m.err = nil
		if msg.Mode == ModePosts {
			for _, pv := range msg.Posts {
				fvp := &bsky.FeedDefs_FeedViewPost{Post: pv}
				for _, url := range feed.ExtractImageURLs(fvp) {
					if cmd := images.Fetch(m.imageCache, url); cmd != nil {
						fetchCmds = append(fetchCmds, cmd)
					}
				}
				avatarURL := feed.ExtractAvatarURL(fvp)
				if avatarURL != "" {
					if cmd := images.FetchAvatar(m.imageCache, avatarURL); cmd != nil {
						fetchCmds = append(fetchCmds, cmd)
					}
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

	case SearchErrorMsg:
		m.loading = false
		m.err = msg.Err
	}

	// Handle input updates
	if m.input.Focused() {
		prevVal := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)

		if m.input.Value() != prevVal {
			m.lastKeystroke = time.Now()
			cmds = append(cmds, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
				return debounceMsg{ts: m.lastKeystroke}
			}))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m SearchModel) performSearch(query, cursor string, mode SearchMode, append bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if mode == ModePosts {
			posts, nextCursor, err := m.client.SearchPosts(ctx, query, cursor, 25)
			if err != nil {
				return SearchErrorMsg{Err: err}
			}
			return SearchResultsMsg{Posts: posts, Cursor: nextCursor, Mode: ModePosts, Append: append}
		} else {
			actors, nextCursor, err := m.client.SearchActors(ctx, query, cursor, 25)
			if err != nil {
				return SearchErrorMsg{Err: err}
			}
			return SearchResultsMsg{Actors: actors, Cursor: nextCursor, Mode: ModePeople, Append: append}
		}
	}
}

func (m *SearchModel) rebuildViewport() {
	if m.mode == ModePosts {
		lazy := &images.LazyRenderer{Inner: m.imageCache}
		m.viewport.SetItems(len(m.postResults), func(index int, selected bool) string {
			if index < 0 || index >= len(m.postResults) {
				return ""
			}
			lazy.NearVisible = m.viewport.IsNearVisible(index, m.viewport.Height())
			fvp := &bsky.FeedDefs_FeedViewPost{Post: m.postResults[index]}
			return feed.RenderPost(fvp, m.width, selected, lazy, m.avatarOverrides)
		})
	} else {
		m.viewport.SetItems(len(m.actorResults), func(index int, selected bool) string {
			return m.renderPerson(index, selected)
		})
	}
}

func (m SearchModel) renderPerson(index int, selected bool) string {
	if index < 0 || index >= len(m.actorResults) {
		return ""
	}
	actor := m.actorResults[index]

	displayName := actor.Handle
	if actor.DisplayName != nil && *actor.DisplayName != "" {
		displayName = *actor.DisplayName
	}

	bio := ""
	if actor.Description != nil {
		bio = strings.ReplaceAll(*actor.Description, "\n", " ")
		if len(bio) > 80 {
			bio = bio[:77] + "..."
		}
	}

	nameStr := theme.StyleHeader().Render(displayName)
	handleStr := theme.StyleMuted().Render("@" + actor.Handle)
	bioStr := theme.StyleMuted().Render(bio)

	line := fmt.Sprintf("%s %s — %s", handleStr, nameStr, bioStr)
	return shared.RenderItemWithBorder(line, selected, m.width)
}

func (m SearchModel) View() tea.View {
	var b strings.Builder

	// Top bar: Input
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	// Mode tabs
	postsTab := "[Posts]"
	peopleTab := "[People]"
	if m.mode == ModePosts {
		postsTab = theme.StyleTabActive().Render(postsTab)
		peopleTab = theme.StyleTabInactive().Render(peopleTab)
	} else {
		postsTab = theme.StyleTabInactive().Render(postsTab)
		peopleTab = theme.StyleTabActive().Render(peopleTab)
	}
	_, _ = fmt.Fprintf(&b, "%s  %s (Press Tab to toggle)\n\n", postsTab, peopleTab)
	availableHeight := max(1, m.height-5)

	// Content area
	if m.err != nil {
		b.WriteString(lipgloss.Place(m.width, availableHeight, lipgloss.Center, lipgloss.Center, theme.StyleError().Render(fmt.Sprintf("Error: %v", m.err))))
	} else if m.query == "" {
		b.WriteString(lipgloss.Place(m.width, availableHeight, lipgloss.Center, lipgloss.Center, theme.StyleMuted().Render("Type to search...")))
	} else if m.loading && len(m.postResults) == 0 && len(m.actorResults) == 0 {
		b.WriteString(lipgloss.Place(m.width, availableHeight, lipgloss.Center, lipgloss.Center, m.spinner.View()+" Searching..."))
	} else if m.mode == ModePosts && len(m.postResults) == 0 {
		b.WriteString(lipgloss.Place(m.width, availableHeight, lipgloss.Center, lipgloss.Center, theme.StyleMuted().Render(fmt.Sprintf("No results for '%s'", m.query))))
	} else if m.mode == ModePeople && len(m.actorResults) == 0 {
		b.WriteString(lipgloss.Place(m.width, availableHeight, lipgloss.Center, lipgloss.Center, theme.StyleMuted().Render(fmt.Sprintf("No results for '%s'", m.query))))
	} else {
		b.WriteString(m.viewport.View())
	}

	b.WriteString("\n")

	// Status bar
	status := ""
	if m.loading {
		status = m.spinner.View() + " Searching..."
	} else if m.mode == ModePosts {
		status = fmt.Sprintf("%d results", len(m.postResults))
	} else {
		status = fmt.Sprintf("%d results", len(m.actorResults))
	}
	b.WriteString(theme.StyleMuted().Render(status))

	v := tea.NewView(b.String())
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

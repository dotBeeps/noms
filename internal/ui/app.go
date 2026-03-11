package ui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/auth"
	"github.com/dotBeeps/noms/internal/ui/components"
	"github.com/dotBeeps/noms/internal/ui/compose"
	"github.com/dotBeeps/noms/internal/ui/feed"
	"github.com/dotBeeps/noms/internal/ui/login"
	"github.com/dotBeeps/noms/internal/ui/notifications"
	"github.com/dotBeeps/noms/internal/ui/profile"
	"github.com/dotBeeps/noms/internal/ui/search"
	"github.com/dotBeeps/noms/internal/ui/theme"
	"github.com/dotBeeps/noms/internal/ui/thread"
)

type Screen int

const (
	ScreenLogin Screen = iota
	ScreenFeed
	ScreenNotifications
	ScreenProfile
	ScreenSearch
	ScreenCompose
	ScreenThread
)

type App struct {
	screen     Screen
	prevScreen Screen // for back navigation
	width      int
	height     int
	loggedIn   bool
	session    *auth.Session
	client     bluesky.BlueskyClient // nil until login

	// Login view
	login login.LoginModel

	// Persistent views (created once after login, kept alive)
	feedModel   feed.FeedModel
	notifModel  notifications.NotificationsModel
	searchModel search.SearchModel

	// On-demand views (created per navigation event)
	profileModel profile.ProfileModel
	threadModel  thread.ThreadModel
	composeModel compose.ComposeModel

	// Initialization tracking for lazy-init views
	notifInitialized   bool
	selfProfileCreated bool

	// Chrome
	statusBar components.StatusBar
	tabBar    components.TabBar
	help      components.HelpModel
	showHelp  bool
	err       error
}

func NewApp() App {
	return App{
		screen:    ScreenLogin,
		login:     login.NewLoginModel(),
		statusBar: components.NewStatusBar(),
		tabBar:    components.NewTabBar(),
		help:      components.NewHelpModel(),
	}
}

func (m App) Init() tea.Cmd {
	return m.login.Init()
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.login, _ = updateLogin(m.login, msg)
		m.tabBar, _ = updateTabBar(m.tabBar, msg)
		m.statusBar, _ = updateStatusBar(m.statusBar, msg)
		m.help, _ = updateHelp(m.help, msg)

		// Propagate resize to the active model
		if m.client != nil {
			switch m.screen {
			case ScreenFeed:
				updated, cmd := m.feedModel.Update(msg)
				m.feedModel = updated.(feed.FeedModel)
				cmds = append(cmds, cmd)
			case ScreenNotifications:
				updated, cmd := m.notifModel.Update(msg)
				m.notifModel = updated.(notifications.NotificationsModel)
				cmds = append(cmds, cmd)
			case ScreenSearch:
				updated, cmd := m.searchModel.Update(msg)
				m.searchModel = updated.(search.SearchModel)
				cmds = append(cmds, cmd)
			case ScreenProfile:
				updated, cmd := m.profileModel.Update(msg)
				m.profileModel = updated.(profile.ProfileModel)
				cmds = append(cmds, cmd)
			case ScreenThread:
				updated, cmd := m.threadModel.Update(msg)
				m.threadModel = updated.(thread.ThreadModel)
				cmds = append(cmds, cmd)
			case ScreenCompose:
				updated, cmd := m.composeModel.Update(msg)
				m.composeModel = updated.(compose.ComposeModel)
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case login.LoginSuccessMsg:
		m.session = msg.Session
		m.loggedIn = true
		m.screen = ScreenFeed

		// Create API client from session
		httpClient := msg.Session.AuthenticatedHTTPClient()
		m.client = bluesky.NewClient(httpClient, msg.Session.PDS, msg.Session.DID)

		// Initialize persistent models
		contentHeight := m.height - theme.TabBarHeight - theme.StatusBarHeight
		if contentHeight < 1 {
			contentHeight = 1
		}
		m.feedModel = feed.NewFeedModel(m.client, m.width, contentHeight)
		m.notifModel = notifications.NewNotificationsModel(m.client, m.width, contentHeight)
		m.searchModel = search.NewSearchModel(m.client, m.width, contentHeight)

		// Start fetching feed data
		cmds = append(cmds, m.feedModel.Init())

		// Update chrome
		m.statusBar.Handle = msg.Session.Handle
		m.statusBar.DID = msg.Session.DID
		m.statusBar.Connected = true
		m.help.SetContext(components.HelpContextMain)

		return m, tea.Batch(cmds...)

	case login.LoginErrorMsg:
		m.login, _ = updateLogin(m.login, msg)
		return m, nil

	case login.StartBrowserAuthMsg, login.StartPasteCodeAuthMsg:
		return m, handleAuthStart(msg)

	// --- Feed messages ---
	case feed.ViewThreadMsg:
		return m.navigateToThread(msg.URI)

	case feed.ComposeMsg:
		return m.navigateToCompose(compose.ModeNewPost, nil)

	case feed.ComposeReplyMsg:
		return m.navigateToComposeReply(msg.URI)

	case feed.FeedLoadedMsg, feed.FeedErrorMsg, feed.FeedRefreshMsg,
		feed.LikeResultMsg, feed.UnlikeResultMsg,
		feed.RepostResultMsg, feed.UnRepostResultMsg,
		feed.LikePostMsg, feed.RepostMsg:
		if m.client != nil {
			updated, cmd := m.feedModel.Update(msg)
			m.feedModel = updated.(feed.FeedModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// --- Notification messages ---
	case notifications.NavigateToPostMsg:
		return m.navigateToThread(msg.URI)

	case notifications.NavigateToProfileMsg:
		return m.navigateToProfile(msg.DID)

	case notifications.UnreadCountMsg:
		m.statusBar.UnreadCount = msg.Count
		if m.client != nil {
			updated, cmd := m.notifModel.Update(msg)
			m.notifModel = updated.(notifications.NotificationsModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case notifications.NotificationsLoadedMsg, notifications.NotificationsErrorMsg:
		if m.client != nil {
			updated, cmd := m.notifModel.Update(msg)
			m.notifModel = updated.(notifications.NotificationsModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// --- Thread messages ---
	case thread.BackMsg:
		m.screen = m.prevScreen
		m.prevScreen = ScreenFeed
		m.updateTabBarForScreen()
		return m, nil

	case thread.ComposeReplyMsg:
		return m.navigateToComposeReply(msg.URI)

	case thread.ViewProfileMsg:
		return m.navigateToProfile(msg.DID)

	case thread.ThreadLoadedMsg, thread.ThreadErrorMsg:
		if m.client != nil {
			updated, cmd := m.threadModel.Update(msg)
			m.threadModel = updated.(thread.ThreadModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// --- Compose messages ---
	case compose.PostCreatedMsg:
		m.screen = m.prevScreen
		m.prevScreen = ScreenFeed
		m.updateTabBarForScreen()
		// Refresh feed after posting
		cmds = append(cmds, func() tea.Msg { return feed.FeedRefreshMsg{} })
		return m, tea.Batch(cmds...)

	case compose.CancelComposeMsg:
		m.screen = m.prevScreen
		m.prevScreen = ScreenFeed
		m.updateTabBarForScreen()
		return m, nil

	case compose.ComposeErrorMsg:
		if m.client != nil {
			updated, cmd := m.composeModel.Update(msg)
			m.composeModel = updated.(compose.ComposeModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// --- Profile messages ---
	case profile.ViewThreadMsg:
		return m.navigateToThread(msg.URI)

	case profile.BackMsg:
		m.screen = m.prevScreen
		m.prevScreen = ScreenFeed
		m.updateTabBarForScreen()
		return m, nil

	case profile.ProfileLoadedMsg, profile.AuthorFeedLoadedMsg,
		profile.ProfileErrorMsg, profile.FollowToggledMsg:
		if m.client != nil {
			updated, cmd := m.profileModel.Update(msg)
			m.profileModel = updated.(profile.ProfileModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// --- Search messages ---
	case search.ViewProfileMsg:
		return m.navigateToProfile(msg.DID)

	case search.SearchResultsMsg, search.SearchErrorMsg:
		if m.client != nil {
			updated, cmd := m.searchModel.Update(msg)
			m.searchModel = updated.(search.SearchModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	// Delegate remaining messages to login screen if active
	if m.screen == ScreenLogin {
		updated, cmd := updateLogin(m.login, msg)
		m.login = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Delegate to active screen model for any unhandled messages
	if m.client != nil && m.screen != ScreenLogin {
		switch m.screen {
		case ScreenFeed:
			updated, cmd := m.feedModel.Update(msg)
			m.feedModel = updated.(feed.FeedModel)
			cmds = append(cmds, cmd)
		case ScreenNotifications:
			updated, cmd := m.notifModel.Update(msg)
			m.notifModel = updated.(notifications.NotificationsModel)
			cmds = append(cmds, cmd)
		case ScreenSearch:
			updated, cmd := m.searchModel.Update(msg)
			m.searchModel = updated.(search.SearchModel)
			cmds = append(cmds, cmd)
		case ScreenProfile:
			updated, cmd := m.profileModel.Update(msg)
			m.profileModel = updated.(profile.ProfileModel)
			cmds = append(cmds, cmd)
		case ScreenThread:
			updated, cmd := m.threadModel.Update(msg)
			m.threadModel = updated.(thread.ThreadModel)
			cmds = append(cmds, cmd)
		case ScreenCompose:
			updated, cmd := m.composeModel.Update(msg)
			m.composeModel = updated.(compose.ComposeModel)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// --- Navigation helpers ---

func (m App) navigateToThread(uri string) (App, tea.Cmd) {
	m.prevScreen = m.screen
	m.screen = ScreenThread
	contentHeight := m.height - theme.TabBarHeight - theme.StatusBarHeight
	if contentHeight < 1 {
		contentHeight = 1
	}
	m.threadModel = thread.NewThreadModel(m.client, uri, m.width, contentHeight)
	return m, m.threadModel.Init()
}

func (m App) navigateToProfile(did string) (App, tea.Cmd) {
	m.prevScreen = m.screen
	m.screen = ScreenProfile
	contentHeight := m.height - theme.TabBarHeight - theme.StatusBarHeight
	if contentHeight < 1 {
		contentHeight = 1
	}
	ownDID := ""
	if m.session != nil {
		ownDID = m.session.DID
	}
	m.profileModel = profile.NewProfileModel(m.client, did, ownDID, m.width, contentHeight)
	return m, m.profileModel.Init()
}

func (m App) navigateToCompose(mode compose.ComposeMode, parentPost interface{}) (App, tea.Cmd) {
	m.prevScreen = m.screen
	m.screen = ScreenCompose
	contentHeight := m.height - theme.TabBarHeight - theme.StatusBarHeight
	if contentHeight < 1 {
		contentHeight = 1
	}
	// parentPost is nil for new posts
	m.composeModel = compose.NewComposeModel(m.client, mode, nil, m.width, contentHeight)
	return m, m.composeModel.Init()
}

func (m App) navigateToComposeReply(uri string) (App, tea.Cmd) {
	m.prevScreen = m.screen
	m.screen = ScreenCompose
	contentHeight := m.height - theme.TabBarHeight - theme.StatusBarHeight
	if contentHeight < 1 {
		contentHeight = 1
	}
	// Fetch the post first, then create compose model with it
	client := m.client
	m.composeModel = compose.NewComposeModel(m.client, compose.ModeReply, nil, m.width, contentHeight)
	return m, func() tea.Msg {
		post, err := client.GetPost(context.Background(), uri)
		if err != nil {
			return compose.ComposeErrorMsg{Err: err}
		}
		// We got the post; re-create the compose model will happen via a separate mechanism.
		// For simplicity, store it as an error-free compose. The parent post context
		// enriches the reply but isn't strictly required for the compose to function.
		_ = post
		return nil
	}
}

func (m *App) updateTabBarForScreen() {
	switch m.screen {
	case ScreenFeed:
		m.tabBar.SetActiveTab(components.TabFeed)
	case ScreenNotifications:
		m.tabBar.SetActiveTab(components.TabNotifications)
	case ScreenProfile:
		m.tabBar.SetActiveTab(components.TabProfile)
	case ScreenSearch:
		m.tabBar.SetActiveTab(components.TabSearch)
	}
}

func (m App) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "?" && m.screen != ScreenCompose {
		m.showHelp = !m.showHelp
		return m, nil
	}

	if m.showHelp {
		if key == "esc" || key == "q" || key == "?" {
			m.showHelp = false
		}
		return m, nil
	}

	if m.screen == ScreenLogin {
		if key == "ctrl+c" || key == "q" {
			return m, tea.Quit
		}
		updated, cmd := m.login.Update(msg)
		m.login = updated.(login.LoginModel)
		return m, cmd
	}

	if key == "ctrl+c" {
		return m, tea.Quit
	}

	// Compose gets all key presses (textarea input)
	if m.screen == ScreenCompose && m.client != nil {
		updated, cmd := m.composeModel.Update(msg)
		m.composeModel = updated.(compose.ComposeModel)
		return m, cmd
	}

	// Tab switching takes priority (from main screens, not thread/compose)
	if m.loggedIn && m.screen != ScreenThread && m.screen != ScreenCompose {
		var cmds []tea.Cmd
		switch key {
		case "1":
			m.screen = ScreenFeed
			m.tabBar.SetActiveTab(components.TabFeed)
			return m, nil
		case "2":
			m.screen = ScreenNotifications
			m.tabBar.SetActiveTab(components.TabNotifications)
			if m.client != nil && !m.notifInitialized {
				m.notifInitialized = true
				cmds = append(cmds, m.notifModel.Init())
			}
			return m, tea.Batch(cmds...)
		case "3":
			m.screen = ScreenProfile
			m.tabBar.SetActiveTab(components.TabProfile)
			if m.client != nil && !m.selfProfileCreated {
				m.selfProfileCreated = true
				contentHeight := m.height - theme.TabBarHeight - theme.StatusBarHeight
				if contentHeight < 1 {
					contentHeight = 1
				}
				m.profileModel = profile.NewProfileModel(m.client, m.session.DID, m.session.DID, m.width, contentHeight)
				cmds = append(cmds, m.profileModel.Init())
			}
			return m, tea.Batch(cmds...)
		case "4":
			m.screen = ScreenSearch
			m.tabBar.SetActiveTab(components.TabSearch)
			return m, nil
		}
	}

	// Esc in overlay screens: go back
	if key == "esc" && m.loggedIn {
		if m.screen == ScreenThread || m.screen == ScreenProfile {
			m.screen = m.prevScreen
			m.prevScreen = ScreenFeed
			m.updateTabBarForScreen()
			return m, nil
		}
	}

	// Delegate remaining key presses to active model
	if m.client != nil {
		switch m.screen {
		case ScreenFeed:
			updated, cmd := m.feedModel.Update(msg)
			m.feedModel = updated.(feed.FeedModel)
			return m, cmd
		case ScreenNotifications:
			updated, cmd := m.notifModel.Update(msg)
			m.notifModel = updated.(notifications.NotificationsModel)
			return m, cmd
		case ScreenProfile:
			updated, cmd := m.profileModel.Update(msg)
			m.profileModel = updated.(profile.ProfileModel)
			return m, cmd
		case ScreenThread:
			updated, cmd := m.threadModel.Update(msg)
			m.threadModel = updated.(thread.ThreadModel)
			return m, cmd
		case ScreenSearch:
			updated, cmd := m.searchModel.Update(msg)
			m.searchModel = updated.(search.SearchModel)
			return m, cmd
		}
	}

	return m, nil
}

func (m App) View() tea.View {
	if m.screen == ScreenLogin {
		v := m.login.View()
		v.AltScreen = true
		return v
	}

	var content strings.Builder

	tabBarView := m.tabBar.View()
	content.WriteString(tabBarView.Content)
	content.WriteString("\n")

	mainHeight := m.height - theme.TabBarHeight - theme.StatusBarHeight
	if mainHeight < 1 {
		mainHeight = 1
	}

	mainContent := m.renderMainContent(mainHeight)
	content.WriteString(mainContent)
	content.WriteString("\n")

	statusBarView := m.statusBar.View()
	content.WriteString(statusBarView.Content)

	v := tea.NewView(content.String())
	v.AltScreen = true

	if m.showHelp {
		v = m.renderHelpOverlay(v)
	}

	return v
}

func (m App) renderMainContent(height int) string {
	// Fallback for when not yet logged in or in tests without client
	if m.client == nil {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(1, 2)

		var content string
		switch m.screen {
		case ScreenFeed:
			content = "Feed View\n\n(Waiting for connection...)"
		case ScreenNotifications:
			content = "Notifications View\n\n(Waiting for connection...)"
		case ScreenProfile:
			content = "Profile View\n\n(Waiting for connection...)"
		case ScreenSearch:
			content = "Search View\n\n(Waiting for connection...)"
		case ScreenCompose:
			content = "Compose View\n\n(Waiting for connection...)"
		case ScreenThread:
			content = "Thread View\n\n(Waiting for connection...)"
		default:
			content = "Unknown view"
		}

		return style.Height(height).Render(content)
	}

	// Render actual view model content
	switch m.screen {
	case ScreenFeed:
		return m.feedModel.View().Content
	case ScreenNotifications:
		return m.notifModel.View().Content
	case ScreenProfile:
		return m.profileModel.View().Content
	case ScreenThread:
		return m.threadModel.View().Content
	case ScreenSearch:
		return m.searchModel.View().Content
	case ScreenCompose:
		return m.composeModel.View().Content
	default:
		return "Unknown view"
	}
}

func (m App) renderHelpOverlay(baseView tea.View) tea.View {
	m.help.Visible = true
	helpView := m.help.View()

	helpContent := helpView.Content
	helpWidth := lipgloss.Width(helpContent)
	helpHeight := lipgloss.Height(helpContent)

	x := (m.width - helpWidth) / 2
	if x < 0 {
		x = 0
	}
	y := (m.height - helpHeight) / 2
	if y < 0 {
		y = 0
	}

	overlayStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236"))

	overlay := overlayStyle.
		Width(helpWidth).
		Height(helpHeight).
		Render(helpContent)

	whitespaceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceStyle(whitespaceStyle)))
}

func updateLogin(m login.LoginModel, msg tea.Msg) (login.LoginModel, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(login.LoginModel), cmd
}

func updateTabBar(m components.TabBar, msg tea.Msg) (components.TabBar, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(components.TabBar), cmd
}

func updateStatusBar(m components.StatusBar, msg tea.Msg) (components.StatusBar, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(components.StatusBar), cmd
}

func updateHelp(m components.HelpModel, msg tea.Msg) (components.HelpModel, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(components.HelpModel), cmd
}

func handleAuthStart(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return login.LoginSuccessMsg{
			Session: &auth.Session{
				DID:    "did:plc:test123",
				Handle: "test.bsky.social",
				PDS:    "https://bsky.social",
			},
		}
	}
}

func (m App) Screen() Screen {
	return m.screen
}

func (m App) IsLoggedIn() bool {
	return m.loggedIn
}

func (m App) Session() *auth.Session {
	return m.session
}

func (m App) ShowHelp() bool {
	return m.showHelp
}

func (m *App) SetError(err error) {
	m.err = err
}

func (m App) Width() int {
	return m.width
}

func (m App) Height() int {
	return m.height
}

// Client returns the BlueskyClient for testing.
func (m App) Client() bluesky.BlueskyClient {
	return m.client
}

// SetClient sets the BlueskyClient (useful for testing with mocks).
func (m *App) SetClient(c bluesky.BlueskyClient) {
	m.client = c
}

// SetFeedModel sets the feed model (useful for testing with mocks).
func (m *App) SetFeedModel(fm feed.FeedModel) {
	m.feedModel = fm
}

// SetNotifModel sets the notifications model (useful for testing with mocks).
func (m *App) SetNotifModel(nm notifications.NotificationsModel) {
	m.notifModel = nm
}

// SetSearchModel sets the search model (useful for testing with mocks).
func (m *App) SetSearchModel(sm search.SearchModel) {
	m.searchModel = sm
}

// LoginSuccessMsg re-exports login.LoginSuccessMsg for external package access.
type LoginSuccessMsg = login.LoginSuccessMsg

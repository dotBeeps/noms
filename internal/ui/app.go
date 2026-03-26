package ui

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/api/voresky"
	"github.com/dotBeeps/noms/internal/auth"
	"github.com/dotBeeps/noms/internal/config"
	"github.com/dotBeeps/noms/internal/ui/components"
	"github.com/dotBeeps/noms/internal/ui/compose"
	"github.com/dotBeeps/noms/internal/ui/enrichment"
	"github.com/dotBeeps/noms/internal/ui/feed"
	"github.com/dotBeeps/noms/internal/ui/images"
	"github.com/dotBeeps/noms/internal/ui/login"
	"github.com/dotBeeps/noms/internal/ui/notifications"
	"github.com/dotBeeps/noms/internal/ui/profile"
	"github.com/dotBeeps/noms/internal/ui/search"
	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
	"github.com/dotBeeps/noms/internal/ui/thread"
	"github.com/dotBeeps/noms/internal/ui/vnotifications"
	"github.com/dotBeeps/noms/internal/ui/vsetup"
	"github.com/dotBeeps/noms/internal/ui/vtab"
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
	ScreenVoreskySetup
	ScreenVoresky
	ScreenVoreskyNotifications
)

const defaultVoreskyURL = "https://voresky.app"

func helpStyles() help.Styles {
	return help.Styles{
		ShortKey:       lipgloss.NewStyle().Foreground(theme.ColorAccent).Bold(true),
		ShortDesc:      lipgloss.NewStyle().Foreground(theme.ColorMuted),
		ShortSeparator: lipgloss.NewStyle().Foreground(theme.ColorMuted),
		FullKey:        lipgloss.NewStyle().Foreground(theme.ColorAccent).Bold(true),
		FullDesc:       lipgloss.NewStyle().Foreground(theme.ColorMuted),
		FullSeparator:  lipgloss.NewStyle().Foreground(theme.ColorMuted),
	}
}
func overlayStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(theme.ColorText).
		Background(theme.ColorSurface).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorPrimary)
}
func overlayWhitespaceStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorSurface)
}
func themePickerTitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(theme.ColorPrimary)
}
func themePickerSelectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorOnPrimary).Background(theme.ColorPrimary).Bold(true)
}
func themePickerRowStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorText)
}
func themePickerHintStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorMuted)
}
func waitingStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.ColorText).Padding(1, 2)
}

type App struct {
	screen     Screen
	prevScreen Screen // for back navigation
	width      int
	height     int
	loggedIn   bool
	session    *auth.Session
	client     bluesky.BlueskyClient // nil until login

	// Session persistence
	tokenStore config.TokenStore
	cfg        *config.Config

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
	vsetupModel  vsetup.Model

	// Voresky
	voreskyAuth         *voresky.VoreskyAuth
	voreskyClient       *voresky.VoreskyClient
	voreskyTabModel     vtab.VoreskyModel
	vnotifModel         vnotifications.VNotificationsModel
	mainCharacter       *voresky.Character
	mainCharacterAvatar string
	enrichManager       *enrichment.Manager

	// Initialization tracking for lazy-init views
	notifInitialized   bool
	selfProfileCreated bool
	voreskyTabInit     bool
	vnotifInit         bool

	// Chrome
	statusBar components.StatusBar
	tabBar    components.TabBar
	help      help.Model
	showHelp  bool
	toast     shared.ToastModel

	// Screen transition animation
	transitionSnapshot string
	transitionDir      int // +1 push (slide up), -1 pop (slide down)
	transitionStart    time.Time
	transitionActive   bool

	showThemePicker  bool
	themePickerIndex int
	err              error

	ownDID     string
	imageCache *images.Cache
}

func NewApp() App {
	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{}
	}
	theme.Apply(cfg.Theme.Name)

	return App{
		screen:     ScreenLogin,
		login:      login.NewLoginModel(),
		statusBar:  components.NewStatusBar(),
		tabBar:     components.NewTabBar(),
		help:       help.New(),
		imageCache: images.New(),
		tokenStore: config.NewTokenStore(),
		cfg:        cfg,
	}
}

func (m App) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.RequestBackgroundColor}
	if m.cfg != nil && m.cfg.DefaultAccount != "" {
		cmds = append(cmds, m.tryRestoreSession)
	} else {
		cmds = append(cmds, m.login.Init())
	}
	return tea.Batch(cmds...)
}

type sessionRestoreFailedMsg struct{ err error }
type transitionTickMsg struct{}

// browserAuthPreparedMsg carries intermediate state between browser auth phases.
// Phase 1 (setup + resolve) produces this so phase 2 can open the browser.
type browserAuthPreparedMsg struct {
	manager  *auth.OAuthManager
	ctx      context.Context
	cancel   context.CancelFunc
	handle   string
	prepared *auth.PreparedAuth
}

// browserAuthWaitingMsg signals that the browser has opened and we're waiting for the user.
type browserAuthWaitingMsg struct {
	manager  *auth.OAuthManager
	ctx      context.Context
	cancel   context.CancelFunc
	handle   string
	prepared *auth.PreparedAuth
}

// browserAuthCallbackMsg carries the auth code received from the browser callback.
type browserAuthCallbackMsg struct {
	manager  *auth.OAuthManager
	ctx      context.Context
	cancel   context.CancelFunc
	handle   string
	prepared *auth.PreparedAuth
	code     string
}

func transitionTick() tea.Cmd {
	return tea.Tick(time.Second/60, func(time.Time) tea.Msg { return transitionTickMsg{} })
}

func (m App) tryRestoreSession() tea.Msg {
	dpopKeyPath := filepath.Join(config.DataDir(), "dpop.key")
	session, err := auth.RestoreSession(m.tokenStore, m.cfg.DefaultAccount, dpopKeyPath)
	if err != nil {
		return sessionRestoreFailedMsg{err: err}
	}
	return login.LoginSuccessMsg{Session: session}
}

func (m App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Always route to toast model (handles its own tick type)
	{
		updated, cmd := m.toast.Update(msg)
		m.toast = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Always route to tab bar (handles its own tick type for underline slide)
	{
		var cmd tea.Cmd
		m.tabBar, cmd = updateTabBar(m.tabBar, msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Always route to status bar (handles status clear ticks)
	{
		var cmd tea.Cmd
		m.statusBar, cmd = updateStatusBar(m.statusBar, msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Screen transition animation
	if m.transitionActive {
		if _, ok := msg.(transitionTickMsg); ok {
			progress := shared.AnimProgress(m.transitionStart, 150*time.Millisecond)
			if progress >= 1 {
				m.transitionActive = false
				m.transitionSnapshot = ""
			} else {
				cmds = append(cmds, transitionTick())
			}
			return m, tea.Batch(cmds...)
		}
	}

	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		theme.SetDarkMode(msg.IsDark())
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.login, _ = updateLogin(m.login, msg)
		// tabBar is updated above (always-routed)
		m.statusBar, _ = updateStatusBar(m.statusBar, msg)
		m.help.SetWidth(msg.Width - 8)

		contentMsg := tea.WindowSizeMsg{Width: msg.Width, Height: m.contentHeight()}

		if m.screen == ScreenVoreskySetup {
			updated, cmd := m.vsetupModel.Update(contentMsg)
			m.vsetupModel = updated.(vsetup.Model)
			cmds = append(cmds, cmd)
		}

		if m.client != nil {
			switch m.screen {
			case ScreenFeed:
				updated, cmd := m.feedModel.Update(contentMsg)
				m.feedModel = updated.(feed.FeedModel)
				cmds = append(cmds, cmd)
			case ScreenNotifications:
				updated, cmd := m.notifModel.Update(contentMsg)
				m.notifModel = updated.(notifications.NotificationsModel)
				cmds = append(cmds, cmd)
			case ScreenSearch:
				updated, cmd := m.searchModel.Update(contentMsg)
				m.searchModel = updated.(search.SearchModel)
				cmds = append(cmds, cmd)
			case ScreenProfile:
				updated, cmd := m.profileModel.Update(contentMsg)
				m.profileModel = updated.(profile.ProfileModel)
				cmds = append(cmds, cmd)
			case ScreenThread:
				updated, cmd := m.threadModel.Update(contentMsg)
				m.threadModel = updated.(thread.ThreadModel)
				cmds = append(cmds, cmd)
			case ScreenCompose:
				updated, cmd := m.composeModel.Update(contentMsg)
				m.composeModel = updated.(compose.ComposeModel)
				cmds = append(cmds, cmd)
			case ScreenVoresky:
				updated, cmd := m.voreskyTabModel.Update(contentMsg)
				m.voreskyTabModel = updated.(vtab.VoreskyModel)
				cmds = append(cmds, cmd)
			case ScreenVoreskyNotifications:
				updated, cmd := m.vnotifModel.Update(contentMsg)
				m.vnotifModel = updated.(vnotifications.VNotificationsModel)
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case login.LoginSuccessMsg:
		m.session = msg.Session
		if m.tokenStore != nil {
			_ = auth.SaveSession(m.tokenStore, msg.Session)
			if msg.Session.TokenManager != nil {
				store := m.tokenStore
				sess := msg.Session
				msg.Session.TokenManager.OnTokenRefresh = func(_ *auth.TokenSet) {
					_ = auth.SaveSession(store, sess)
				}
			}
			if m.cfg != nil {
				m.cfg.DefaultAccount = msg.Session.DID
				_ = config.Save(m.cfg)
			}
		}
		httpClient := msg.Session.AuthenticatedHTTPClient()
		client := bluesky.NewClient(httpClient, msg.Session.PDS, msg.Session.DID)
		return m.finishLogin(client, msg.Session.DID, msg.Session.Handle)

	case login.AppPasswordLoginSuccessMsg:
		client := bluesky.NewClientFromAPI(msg.Client, msg.DID)
		return m.finishLogin(client, msg.DID, msg.Handle)

	case sessionRestoreFailedMsg:
		if msg.err != nil {
			m.login.SetError(fmt.Errorf("saved session expired, please log in again"))
		}
		return m, m.login.Init()

	case login.LoginErrorMsg:
		m.login, _ = updateLogin(m.login, msg)
		return m, nil

	case voreskySessionLoadedMsg:
		cmds = append(cmds, m.initVoresky(msg.auth, ScreenFeed))
		return m, tea.Batch(cmds...)

	case voreskySessionNotFoundMsg:
		return m, nil

	case vsetup.CookieSubmitMsg:
		return m, m.validateVoreskyCookie(msg.Cookie)

	case voreskyAuthSuccessMsg:
		cmds = append(cmds, m.initVoresky(msg.auth, m.prevScreen))
		return m, tea.Batch(cmds...)

	case mainCharacterLoadedMsg:
		m.mainCharacter = msg.character
		m.mainCharacterAvatar = ""
		if msg.character != nil {
			m.mainCharacterAvatar = msg.character.Avatar
		}
		overrides := m.buildAvatarOverrides()
		m.pushAvatarOverrides(overrides)
		if m.mainCharacterAvatar != "" {
			if cmd := images.FetchAvatar(m.imageCache, m.mainCharacterAvatar); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case mainCharacterErrorMsg:
		m.mainCharacter = nil
		m.mainCharacterAvatar = ""
		return m, nil

	case voreskyAuthErrorMsg:
		updated, cmd := m.vsetupModel.Update(vsetup.AuthErrorMsg{Err: msg.err})
		m.vsetupModel = updated.(vsetup.Model)
		return m, cmd

	case vsetup.SkipMsg:
		m.screen = m.prevScreen
		return m, m.updateTabBarForScreen()

	case login.StartBrowserAuthMsg:
		return m, browserAuthPrepare(msg.Handle)

	case browserAuthPreparedMsg:
		m.login, _ = updateLogin(m.login, login.AuthStepMsg{Method: login.AuthMethodBrowser, Step: 1})
		return m, browserAuthOpen(msg)

	case browserAuthWaitingMsg:
		m.login, _ = updateLogin(m.login, login.AuthStepMsg{Method: login.AuthMethodBrowser, Step: 2})
		return m, browserAuthWait(msg)

	case browserAuthCallbackMsg:
		m.login, _ = updateLogin(m.login, login.AuthStepMsg{Method: login.AuthMethodBrowser, Step: 3})
		return m, browserAuthExchange(msg)

	case login.StartAppPasswordAuthMsg:
		return m, handleAppPasswordAuth(msg.Handle, msg.Password)

	// --- Feed messages ---
	case feed.ViewThreadMsg:
		return m.navigateToThread(msg.URI)

	case feed.ComposeMsg:
		return m.navigateToCompose(compose.ModeNewPost, nil)

	case feed.ComposeReplyMsg:
		return m.navigateToComposeReply(msg.URI)

	case feed.DeletePostMsg:
		if m.client != nil {
			client := m.client
			uri := msg.URI
			return m, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				err := client.DeletePost(ctx, uri)
				return feed.DeletePostResultMsg{URI: uri, Err: err}
			}
		}
		return m, nil

	case feed.DeletePostResultMsg:
		if m.client != nil {
			updated, cmd := m.feedModel.Update(msg)
			m.feedModel = updated.(feed.FeedModel)
			cmds = append(cmds, cmd)
			{
				updated, cmd := m.threadModel.Update(msg)
				m.threadModel = updated.(thread.ThreadModel)
				cmds = append(cmds, cmd)
			}
			if m.selfProfileCreated {
				updated, cmd := m.profileModel.Update(msg)
				m.profileModel = updated.(profile.ProfileModel)
				cmds = append(cmds, cmd)
			}
		}
		if msg.Err != nil {
			cmds = append(cmds, func() tea.Msg { return shared.ToastMsg{Text: "Delete failed", IsError: true} })
		} else {
			cmds = append(cmds, func() tea.Msg { return shared.ToastMsg{Text: "Post deleted"} })
		}
		return m, tea.Batch(cmds...)

	case feed.LikePostMsg, feed.RepostMsg:
		switch m.screen {
		case ScreenFeed:
			updated, cmd := m.feedModel.Update(msg)
			m.feedModel = updated.(feed.FeedModel)
			cmds = append(cmds, cmd)
		case ScreenThread:
			updated, cmd := m.threadModel.Update(msg)
			m.threadModel = updated.(thread.ThreadModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case feed.LikeResultMsg, feed.UnlikeResultMsg,
		feed.RepostResultMsg, feed.UnRepostResultMsg:
		if m.client != nil {
			updated, cmd := m.feedModel.Update(msg)
			m.feedModel = updated.(feed.FeedModel)
			cmds = append(cmds, cmd)
			{
				updated, cmd := m.threadModel.Update(msg)
				m.threadModel = updated.(thread.ThreadModel)
				cmds = append(cmds, cmd)
			}
			if m.selfProfileCreated {
				updated, cmd := m.profileModel.Update(msg)
				m.profileModel = updated.(profile.ProfileModel)
				cmds = append(cmds, cmd)
			}
		}
		// Toast on errors
		switch v := msg.(type) {
		case feed.LikeResultMsg:
			if v.Err != nil {
				cmds = append(cmds, func() tea.Msg { return shared.ToastMsg{Text: "Like failed", IsError: true} })
			}
		case feed.RepostResultMsg:
			if v.Err != nil {
				cmds = append(cmds, func() tea.Msg { return shared.ToastMsg{Text: "Repost failed", IsError: true} })
			}
		}
		return m, tea.Batch(cmds...)

	case feed.FeedLoadedMsg:
		if m.client != nil {
			updated, cmd := m.feedModel.Update(msg)
			m.feedModel = updated.(feed.FeedModel)
			cmds = append(cmds, cmd)
			if cmd := m.enrichDIDsFromFeedPosts(msg.Posts); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		// Successful load → restore connected status
		m.statusBar, _ = updateStatusBar(m.statusBar, components.StatusUpdateMsg{Status: components.StatusNormal})
		return m, tea.Batch(cmds...)

	case feed.FeedErrorMsg:
		if m.client != nil {
			updated, cmd := m.feedModel.Update(msg)
			m.feedModel = updated.(feed.FeedModel)
			cmds = append(cmds, cmd)
		}
		// Surface the error type in the status bar
		statusMsg := components.StatusUpdateMsg{Status: components.StatusOffline}
		if bluesky.IsRateLimited(msg.Err) {
			statusMsg.Status = components.StatusRateLimited
		}
		var statusCmd tea.Cmd
		m.statusBar, statusCmd = updateStatusBar(m.statusBar, statusMsg)
		if statusCmd != nil {
			cmds = append(cmds, statusCmd)
		}
		return m, tea.Batch(cmds...)

	case feed.FeedRefreshMsg:
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
		prev := m.statusBar.UnreadCount
		m.statusBar.UnreadCount = msg.Count
		if msg.Count > prev {
			updated, cmd := m.statusBar.Update(components.StatusBarBounceMsg{})
			m.statusBar = updated.(components.StatusBar)
			cmds = append(cmds, cmd)
		}
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
		m.startTransition(-1) // pop (slide down)
		m.screen = m.prevScreen
		m.prevScreen = ScreenFeed
		return m, tea.Batch(m.updateTabBarForScreen(), transitionTick())

	case thread.ComposeReplyMsg:
		return m.navigateToComposeReply(msg.URI)

	case thread.ViewProfileMsg:
		return m.navigateToProfile(msg.DID)

	case thread.ThreadLoadedMsg:
		if m.client != nil {
			updated, cmd := m.threadModel.Update(msg)
			m.threadModel = updated.(thread.ThreadModel)
			cmds = append(cmds, cmd)
			if msg.Thread != nil && msg.Thread.Thread != nil {
				if cmd := m.enrichDIDsFromThread(msg.Thread.Thread); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		return m, tea.Batch(cmds...)

	case thread.ThreadErrorMsg:
		if m.client != nil {
			updated, cmd := m.threadModel.Update(msg)
			m.threadModel = updated.(thread.ThreadModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// --- Compose messages ---
	case compose.PostCreatedMsg:
		m.startTransition(-1) // pop (slide down)
		m.screen = m.prevScreen
		m.prevScreen = ScreenFeed
		cmds = append(cmds, m.updateTabBarForScreen())
		cmds = append(cmds, func() tea.Msg { return feed.FeedRefreshMsg{} })
		cmds = append(cmds, func() tea.Msg { return shared.ToastMsg{Text: "Post sent!"} })
		cmds = append(cmds, transitionTick())
		return m, tea.Batch(cmds...)

	case compose.CancelComposeMsg:
		m.startTransition(-1) // pop (slide down)
		m.screen = m.prevScreen
		m.prevScreen = ScreenFeed
		return m, tea.Batch(m.updateTabBarForScreen(), transitionTick())

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
		return m, m.updateTabBarForScreen()

	case profile.AuthorFeedLoadedMsg:
		if m.client != nil {
			updated, cmd := m.profileModel.Update(msg)
			m.profileModel = updated.(profile.ProfileModel)
			cmds = append(cmds, cmd)
			if cmd := m.enrichDIDsFromFeedPosts(msg.Posts); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case profile.ProfileLoadedMsg, profile.ProfileErrorMsg, profile.FollowToggledMsg:
		if m.client != nil {
			updated, cmd := m.profileModel.Update(msg)
			m.profileModel = updated.(profile.ProfileModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// --- Voresky tab messages ---
	case vtab.CharactersLoadedMsg, vtab.CharactersErrorMsg:
		if m.voreskyClient != nil {
			updated, cmd := m.voreskyTabModel.Update(msg)
			m.voreskyTabModel = updated.(vtab.VoreskyModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// --- Voresky notification messages ---
	case vnotifications.VNotificationsLoadedMsg, vnotifications.VNotificationsErrorMsg,
		vnotifications.VNotifUnreadCountMsg:
		if m.voreskyClient != nil {
			updated, cmd := m.vnotifModel.Update(msg)
			m.vnotifModel = updated.(vnotifications.VNotificationsModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// --- Search messages ---
	case search.ViewProfileMsg:
		return m.navigateToProfile(msg.DID)

	case search.SearchResultsMsg:
		if m.client != nil {
			updated, cmd := m.searchModel.Update(msg)
			m.searchModel = updated.(search.SearchModel)
			cmds = append(cmds, cmd)
			if cmd := m.enrichDIDsFromPostViews(msg.Posts); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case search.SearchErrorMsg:
		if m.client != nil {
			updated, cmd := m.searchModel.Update(msg)
			m.searchModel = updated.(search.SearchModel)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	// --- Enrichment messages ---
	case enrichResultMsg:
		if m.enrichManager != nil && msg.overrides != nil {
			m.enrichManager.Store(msg.overrides)
			m.enrichManager.ResolveSnapshots()
			pending := m.enrichManager.PendingSnapshots()
			for _, ps := range pending {
				cmds = append(cmds, m.fetchSnapshot(ps.BlobHash))
			}
			overrides := m.buildAvatarOverrides()
			m.pushAvatarOverrides(overrides)
			cmds = append(cmds, m.prefetchAvatars(overrides)...)
		}
		return m, tea.Batch(cmds...)

	case enrichErrorMsg:
		// Silent failure — users see Bluesky avatars as fallback
		return m, nil

	case snapshotResultMsg:
		if m.enrichManager != nil && msg.blob != nil {
			m.enrichManager.StoreSnapshot(msg.hash, msg.blob)
			m.enrichManager.ResolveSnapshots()
			overrides := m.buildAvatarOverrides()
			m.pushAvatarOverrides(overrides)
			cmds = append(cmds, m.prefetchAvatars(overrides)...)
		}
		return m, tea.Batch(cmds...)

	case snapshotErrorMsg:
		// Silent failure — caught users show their base avatar or Bluesky avatar
		return m, nil

	case autoRefreshMsg:
		if m.client != nil {
			client := m.client
			cmds = append(cmds, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				count, err := client.GetUnreadCount(ctx)
				if err != nil {
					return nil
				}
				return notifications.UnreadCountMsg{Count: count}
			})
			cmds = append(cmds, scheduleAutoRefresh())
		}
		return m, tea.Batch(cmds...)
	}

	// Broadcast ImageFetchedMsg to all models that show images. Persistent models
	// are always updated so off-screen tabs stay fresh. Non-persistent models only
	// need it when they're the active screen. Return early to prevent the generic
	// active-screen delegation below from double-updating persistent models.
	if _, ok := msg.(images.ImageFetchedMsg); ok && m.client != nil {
		updated, cmd := m.feedModel.Update(msg)
		m.feedModel = updated.(feed.FeedModel)
		cmds = append(cmds, cmd)
		updated2, cmd2 := m.notifModel.Update(msg)
		m.notifModel = updated2.(notifications.NotificationsModel)
		cmds = append(cmds, cmd2)
		if m.voreskyClient != nil {
			updated3, cmd3 := m.voreskyTabModel.Update(msg)
			m.voreskyTabModel = updated3.(vtab.VoreskyModel)
			cmds = append(cmds, cmd3)
			updated4, cmd4 := m.vnotifModel.Update(msg)
			m.vnotifModel = updated4.(vnotifications.VNotificationsModel)
			cmds = append(cmds, cmd4)
		}
		// Also update the active non-persistent screen when it shows images.
		switch m.screen {
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
		case ScreenVoreskySetup:
			updated, cmd := m.vsetupModel.Update(msg)
			m.vsetupModel = updated.(vsetup.Model)
			cmds = append(cmds, cmd)
		case ScreenVoresky:
			updated, cmd := m.voreskyTabModel.Update(msg)
			m.voreskyTabModel = updated.(vtab.VoreskyModel)
			cmds = append(cmds, cmd)
		case ScreenVoreskyNotifications:
			updated, cmd := m.vnotifModel.Update(msg)
			m.vnotifModel = updated.(vnotifications.VNotificationsModel)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// --- Navigation helpers ---

func (m App) navigateToThread(uri string) (App, tea.Cmd) {
	m.startTransition(1) // push (slide up)
	m.prevScreen = m.screen
	m.screen = ScreenThread
	contentHeight := m.contentHeight()
	ownDID := ""
	if m.session != nil {
		ownDID = m.session.DID
	}
	m.threadModel = thread.NewThreadModel(m.client, uri, ownDID, m.width, contentHeight, m.imageCache)
	m.threadModel.SetAvatarOverrides(m.buildAvatarOverrides())

	return m, tea.Batch(m.threadModel.Init(), transitionTick())
}

func (m App) navigateToProfile(did string) (App, tea.Cmd) {
	m.prevScreen = m.screen
	m.screen = ScreenProfile
	contentHeight := m.contentHeight()
	ownDID := ""
	if m.session != nil {
		ownDID = m.session.DID
	}
	m.profileModel = profile.NewProfileModel(m.client, did, ownDID, m.width, contentHeight, m.imageCache)
	m.profileModel.SetAvatarOverrides(m.buildAvatarOverrides())

	return m, m.profileModel.Init()
}

func (m App) navigateToCompose(mode compose.ComposeMode, parentPost interface{}) (App, tea.Cmd) {
	m.startTransition(1) // push (slide up)
	m.prevScreen = m.screen
	m.screen = ScreenCompose
	contentHeight := m.contentHeight()
	m.composeModel = compose.NewComposeModel(m.client, mode, nil, m.width, contentHeight, m.imageCache)
	m.composeModel.SetAvatarOverrides(m.buildAvatarOverrides())

	return m, tea.Batch(m.composeModel.Init(), transitionTick())
}

func (m App) navigateToComposeReply(uri string) (App, tea.Cmd) {
	m.startTransition(1) // push (slide up)
	m.prevScreen = m.screen
	m.screen = ScreenCompose
	contentHeight := m.contentHeight()
	client := m.client
	m.composeModel = compose.NewComposeModel(m.client, compose.ModeReply, nil, m.width, contentHeight, m.imageCache)
	m.composeModel.SetAvatarOverrides(m.buildAvatarOverrides())

	return m, tea.Batch(transitionTick(), func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		post, err := client.GetPost(ctx, uri)
		if err != nil {
			return compose.ComposeErrorMsg{Err: fmt.Errorf("loading parent post: %w", err)}
		}
		return compose.ParentPostLoadedMsg{Post: post}
	})
}

func (m *App) updateTabBarForScreen() tea.Cmd {
	switch m.screen {
	case ScreenFeed:
		return m.tabBar.SetActiveTab(components.TabFeed)
	case ScreenNotifications:
		return m.tabBar.SetActiveTab(components.TabNotifications)
	case ScreenProfile:
		return m.tabBar.SetActiveTab(components.TabProfile)
	case ScreenSearch:
		return m.tabBar.SetActiveTab(components.TabSearch)
	case ScreenVoresky:
		return m.tabBar.SetActiveTab(components.TabVoresky)
	case ScreenVoreskyNotifications:
		return m.tabBar.SetActiveTab(components.TabVoreskyNotifications)
	}
	return nil
}

func (m App) availableTabs() []components.Tab {
	tabs := []components.Tab{
		components.TabFeed, components.TabNotifications,
		components.TabProfile, components.TabSearch,
	}
	if m.voreskyClient != nil {
		tabs = append(tabs, components.TabVoresky, components.TabVoreskyNotifications)
	}
	return tabs
}

func (m *App) switchToTab(tab components.Tab) tea.Cmd {
	var cmds []tea.Cmd
	sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.contentHeight()}
	switch tab {
	case components.TabFeed:
		m.screen = ScreenFeed
		updated, cmd := m.feedModel.Update(sizeMsg)
		m.feedModel = updated.(feed.FeedModel)
		cmds = append(cmds, cmd)
	case components.TabNotifications:
		m.screen = ScreenNotifications
		updated, cmd := m.notifModel.Update(sizeMsg)
		m.notifModel = updated.(notifications.NotificationsModel)
		cmds = append(cmds, cmd)
		if m.client != nil && !m.notifInitialized {
			m.notifInitialized = true
			cmds = append(cmds, m.notifModel.Init())
		}
	case components.TabProfile:
		m.screen = ScreenProfile
		if m.client != nil && !m.selfProfileCreated {
			m.selfProfileCreated = true
			contentHeight := m.contentHeight()
			m.profileModel = profile.NewProfileModel(m.client, m.session.DID, m.session.DID, m.width, contentHeight, m.imageCache)
			m.profileModel.SetAvatarOverrides(m.buildAvatarOverrides())
			cmds = append(cmds, m.profileModel.Init())
		}
	case components.TabSearch:
		m.screen = ScreenSearch
		updated, cmd := m.searchModel.Update(sizeMsg)
		m.searchModel = updated.(search.SearchModel)
		cmds = append(cmds, cmd)
	case components.TabVoresky:
		if m.voreskyClient == nil {
			return nil
		}
		m.screen = ScreenVoresky
		updated, cmd := m.voreskyTabModel.Update(sizeMsg)
		m.voreskyTabModel = updated.(vtab.VoreskyModel)
		cmds = append(cmds, cmd)
		if !m.voreskyTabInit {
			m.voreskyTabInit = true
			cmds = append(cmds, m.voreskyTabModel.Init())
		}
	case components.TabVoreskyNotifications:
		if m.voreskyClient == nil {
			return nil
		}
		m.screen = ScreenVoreskyNotifications
		updated, cmd := m.vnotifModel.Update(sizeMsg)
		m.vnotifModel = updated.(vnotifications.VNotificationsModel)
		cmds = append(cmds, cmd)
		if !m.vnotifInit {
			m.vnotifInit = true
			cmds = append(cmds, m.vnotifModel.Init())
		}
	}
	cmds = append(cmds, m.tabBar.SetActiveTab(tab))
	return tea.Batch(cmds...)
}

func (m *App) cycleTab(forward bool) tea.Cmd {
	tabs := m.availableTabs()
	idx := 0
	for i, t := range tabs {
		if t == m.tabBar.ActiveTab {
			idx = i
			break
		}
	}
	if forward {
		idx = (idx + 1) % len(tabs)
	} else {
		idx = (idx - 1 + len(tabs)) % len(tabs)
	}
	return m.switchToTab(tabs[idx])
}

func (m App) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	gk := globalKeys
	k := msg.String()

	if key.Matches(msg, gk.Help) && m.screen != ScreenCompose && !m.showThemePicker {
		m.showHelp = !m.showHelp
		return m, nil
	}

	if m.showHelp {
		k := msg.String()
		if k == "esc" || k == "q" || k == "?" {
			m.showHelp = false
		}
		return m, nil
	}

	if m.showThemePicker {
		themes := theme.AvailableThemes()
		if len(themes) == 0 {
			m.showThemePicker = false
			return m, nil
		}

		switch k {
		case "ctrl+c":
			m.imageCache.Close()
			return m, tea.Quit
		case "esc", "q", "ctrl+t":
			m.showThemePicker = false
			return m, nil
		case "j", "down":
			if m.themePickerIndex < len(themes)-1 {
				m.themePickerIndex++
			}
			return m, nil
		case "k", "up":
			if m.themePickerIndex > 0 {
				m.themePickerIndex--
			}
			return m, nil
		case "g":
			m.themePickerIndex = 0
			return m, nil
		case "G":
			m.themePickerIndex = len(themes) - 1
			return m, nil
		case "enter":
			if m.themePickerIndex < 0 || m.themePickerIndex >= len(themes) {
				m.themePickerIndex = 0
			}
			m.applyTheme(themes[m.themePickerIndex])
			m.showThemePicker = false
			return m, nil
		case "[":
			m.cycleTheme(-1)
			themes = theme.AvailableThemes()
			m.themePickerIndex = m.themeIndex(themes)
			return m, nil
		case "]":
			m.cycleTheme(1)
			themes = theme.AvailableThemes()
			m.themePickerIndex = m.themeIndex(themes)
			return m, nil
		}

		return m, nil
	}

	if m.screen == ScreenLogin {
		if k == "ctrl+c" || k == "q" {
			m.imageCache.Close()
			return m, tea.Quit
		}
		updated, cmd := m.login.Update(msg)
		m.login = updated.(login.LoginModel)
		return m, cmd
	}

	if m.screen == ScreenVoreskySetup {
		if k == "ctrl+c" {
			m.imageCache.Close()
			return m, tea.Quit
		}
		updated, cmd := m.vsetupModel.Update(msg)
		m.vsetupModel = updated.(vsetup.Model)
		return m, cmd
	}

	if k == "ctrl+c" || k == "q" {
		m.imageCache.Close()
		return m, tea.Quit
	}

	if m.screen == ScreenCompose && m.client != nil {
		updated, cmd := m.composeModel.Update(msg)
		m.composeModel = updated.(compose.ComposeModel)
		return m, cmd
	}

	if m.loggedIn && m.screen != ScreenLogin && m.screen != ScreenVoreskySetup && m.screen != ScreenCompose {
		switch k {
		case "[":
			m.cycleTheme(-1)
			return m, nil
		case "]":
			m.cycleTheme(1)
			return m, nil
		case "ctrl+t":
			themes := theme.AvailableThemes()
			if len(themes) == 0 {
				return m, nil
			}
			m.showThemePicker = true
			m.showHelp = false
			m.themePickerIndex = m.themeIndex(themes)
			return m, nil
		}
	}

	// Tab switching takes priority (from main screens, not thread/compose)
	if m.loggedIn && m.screen != ScreenThread && m.screen != ScreenCompose {
		switch k {
		case "tab":
			return m, m.cycleTab(true)
		case "shift+tab":
			return m, m.cycleTab(false)
		case "v":
			m.prevScreen = m.screen
			m.screen = ScreenVoreskySetup
			m.vsetupModel = vsetup.New()
			return m, m.vsetupModel.Init()
		}
	}

	// Esc in overlay screens: go back
	if k == "esc" && m.loggedIn {
		if m.screen == ScreenThread || m.screen == ScreenProfile {
			m.startTransition(-1) // pop (slide down)
			m.screen = m.prevScreen
			m.prevScreen = ScreenFeed
			return m, tea.Batch(m.updateTabBarForScreen(), transitionTick())
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
		case ScreenVoresky:
			updated, cmd := m.voreskyTabModel.Update(msg)
			m.voreskyTabModel = updated.(vtab.VoreskyModel)
			return m, cmd
		case ScreenVoreskyNotifications:
			updated, cmd := m.vnotifModel.Update(msg)
			m.vnotifModel = updated.(vnotifications.VNotificationsModel)
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

	toastLine := m.toast.View(m.width)
	toastHeight := 0
	if toastLine != "" {
		toastHeight = 1
	}

	mainHeight := m.height - theme.TabBarHeight - theme.StatusBarHeight - toastHeight
	if mainHeight < 1 {
		mainHeight = 1
	}

	mainContent := m.renderMainContent(mainHeight)

	// Apply screen transition if active
	if m.transitionActive {
		progress := shared.EaseOutQuad(shared.AnimProgress(m.transitionStart, 150*time.Millisecond))
		mainContent = m.compositeTransition(mainContent, mainHeight, progress)
	}

	content.WriteString(mainContent)
	content.WriteString("\n")

	if toastLine != "" {
		content.WriteString(toastLine)
		content.WriteString("\n")
	}

	statusBarView := m.statusBar.View()
	content.WriteString(statusBarView.Content)

	v := tea.NewView(content.String())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	if m.showHelp {
		v = m.renderHelpOverlay(v)
	}

	if m.showThemePicker {
		v = m.renderThemePickerOverlay(v)
	}

	return v
}

func (m App) renderMainContent(height int) string {
	if m.screen == ScreenVoreskySetup {
		return m.vsetupModel.View().Content
	}

	if m.client == nil {
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
		case ScreenVoresky:
			content = "Voresky View\n\n(Waiting for connection...)"
		case ScreenVoreskyNotifications:
			content = "Voresky Notifications View\n\n(Waiting for connection...)"
		case ScreenCompose:
			content = "Compose View\n\n(Waiting for connection...)"
		case ScreenThread:
			content = "Thread View\n\n(Waiting for connection...)"
		default:
			content = "Unknown view"
		}

		return waitingStyle().Height(height).Render(content)
	}

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
	case ScreenVoresky:
		return m.voreskyTabModel.View().Content
	case ScreenVoreskyNotifications:
		return m.vnotifModel.View().Content
	case ScreenCompose:
		return m.composeModel.View().Content
	default:
		return "Unknown view"
	}
}

func (m App) activeKeyMap() help.KeyMap {
	switch m.screen {
	case ScreenFeed:
		return m.feedModel.Keys()
	case ScreenNotifications:
		return m.notifModel.Keys()
	case ScreenProfile:
		return m.profileModel.Keys()
	case ScreenThread:
		return m.threadModel.Keys()
	case ScreenSearch:
		return m.searchModel.Keys()
	case ScreenCompose:
		return compose.DefaultKeyMap
	case ScreenVoresky:
		return m.voreskyTabModel.Keys()
	case ScreenVoreskyNotifications:
		return m.vnotifModel.Keys()
	case ScreenLogin:
		return login.DefaultKeyMap
	default:
		return globalKeys
	}
}

func (m App) renderHelpOverlay(baseView tea.View) tea.View {
	h := m.help
	h.ShowAll = true
	h.Styles = helpStyles()
	h.SetWidth(m.width - 8)

	helpContent := h.View(m.activeKeyMap())

	overlay := overlayStyle().Render(helpContent)

	return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceStyle(overlayWhitespaceStyle())))
}

func (m App) renderThemePickerOverlay(baseView tea.View) tea.View {
	themes := theme.AvailableThemes()
	if len(themes) == 0 {
		return baseView
	}

	selectedIndex := m.themePickerIndex
	if selectedIndex < 0 || selectedIndex >= len(themes) {
		selectedIndex = 0
	}

	active := theme.ActiveTheme()

	var lines []string
	lines = append(lines, themePickerTitleStyle().Render("Theme Picker"))
	lines = append(lines, "")

	for i, name := range themes {
		line := "  " + name
		if name == active {
			line += " (current)"
		}

		if i == selectedIndex {
			line = "› " + name
			if name == active {
				line += " (current)"
			}
			lines = append(lines, themePickerSelectedStyle().Render(line))
			continue
		}

		lines = append(lines, themePickerRowStyle().Render(line))
	}

	lines = append(lines, "")
	lines = append(lines, themePickerHintStyle().Render("j/k or ↑/↓: move  •  Enter: apply  •  [ ]: cycle  •  Esc: close"))

	content := strings.Join(lines, "\n")

	overlay := overlayStyle().Render(content)

	return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceStyle(overlayWhitespaceStyle())))
}

func (m *App) applyTheme(name string) {
	resolved := theme.Apply(name)
	if m.cfg != nil {
		m.cfg.Theme.Name = resolved
		_ = config.Save(m.cfg)
	}
}

func (m *App) cycleTheme(delta int) {
	themes := theme.AvailableThemes()
	if len(themes) == 0 {
		return
	}

	idx := m.themeIndex(themes)
	idx = (idx + delta + len(themes)) % len(themes)
	m.applyTheme(themes[idx])
	m.themePickerIndex = idx
}

func (m *App) themeIndex(themes []string) int {
	if len(themes) == 0 {
		return 0
	}

	active := theme.ActiveTheme()
	for i, name := range themes {
		if name == active {
			return i
		}
	}

	return 0
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

func oauthSetup() (string, *auth.DPoPSigner, error) {
	if err := config.EnsureDirs(); err != nil {
		return "", nil, fmt.Errorf("creating data directories: %w", err)
	}

	dpopKeyPath := filepath.Join(config.DataDir(), "dpop.key")
	dpop, err := auth.NewDPoPSigner(dpopKeyPath)
	if err != nil {
		return "", nil, fmt.Errorf("initializing DPoP signer: %w", err)
	}

	clientID := "http://localhost/?redirect_uri=" +
		url.QueryEscape("http://127.0.0.1/callback") +
		"&scope=" + url.QueryEscape("atproto transition:generic")

	return clientID, dpop, nil
}

// browserAuthPrepare sets up OAuth and resolves identity + auth server metadata.
// Returns a browserAuthPreparedMsg carrying the prepared state.
func browserAuthPrepare(handle string) tea.Cmd {
	return func() tea.Msg {
		clientID, dpop, err := oauthSetup()
		if err != nil {
			return login.LoginErrorMsg{Err: err}
		}

		flow := auth.NewLoopbackFlow()
		oauthCfg := auth.OAuthConfig{
			ClientID: clientID,
			Scopes:   []string{"atproto", "transition:generic"},
		}

		manager := auth.NewOAuthManager(oauthCfg, flow, dpop)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

		prepared, err := manager.Prepare(ctx, handle)
		if err != nil {
			cancel()
			return login.LoginErrorMsg{Err: fmt.Errorf("authentication failed: %w", err)}
		}

		return browserAuthPreparedMsg{
			manager:  manager,
			ctx:      ctx,
			cancel:   cancel,
			handle:   handle,
			prepared: prepared,
		}
	}
}

// browserAuthOpen opens the browser with the authorization URL.
func browserAuthOpen(msg browserAuthPreparedMsg) tea.Cmd {
	return func() tea.Msg {
		if err := msg.manager.OpenBrowser(msg.ctx, msg.prepared); err != nil {
			msg.cancel()
			return login.LoginErrorMsg{Err: fmt.Errorf("authentication failed: %w", err)}
		}

		return browserAuthWaitingMsg(msg)
	}
}

// browserAuthWait waits for the OAuth callback from the browser.
func browserAuthWait(msg browserAuthWaitingMsg) tea.Cmd {
	return func() tea.Msg {
		code, err := msg.manager.WaitForCallback(msg.ctx, msg.prepared)
		if err != nil {
			msg.cancel()
			return login.LoginErrorMsg{Err: fmt.Errorf("authentication failed: %w", err)}
		}

		return browserAuthCallbackMsg{
			manager:  msg.manager,
			ctx:      msg.ctx,
			cancel:   msg.cancel,
			handle:   msg.handle,
			prepared: msg.prepared,
			code:     code,
		}
	}
}

// browserAuthExchange exchanges the authorization code for tokens.
func browserAuthExchange(msg browserAuthCallbackMsg) tea.Cmd {
	return func() tea.Msg {
		defer msg.cancel()

		session, err := msg.manager.ExchangeToken(msg.ctx, msg.handle, msg.code, msg.prepared)
		if err != nil {
			return login.LoginErrorMsg{Err: fmt.Errorf("authentication failed: %w", err)}
		}

		return login.LoginSuccessMsg{Session: session}
	}
}

func handleAppPasswordAuth(handle, password string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		dir := identity.DefaultDirectory()
		username, err := syntax.ParseAtIdentifier(handle)
		if err != nil {
			return login.LoginErrorMsg{Err: fmt.Errorf("invalid handle %q: %w", handle, err)}
		}

		apiClient, err := atclient.LoginWithPassword(ctx, dir, username, password, "", nil)
		if err != nil {
			return login.LoginErrorMsg{Err: fmt.Errorf("app password login failed: %w", err)}
		}

		if apiClient.AccountDID == nil {
			return login.LoginErrorMsg{Err: fmt.Errorf("app password login succeeded but server returned no DID")}
		}
		did := apiClient.AccountDID.String()

		return login.AppPasswordLoginSuccessMsg{
			Client: apiClient,
			DID:    did,
			Handle: handle,
			PDS:    apiClient.Host,
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

// VoreskySkipMsg re-exports vsetup.SkipMsg for external package access (tests).
type VoreskySkipMsg = vsetup.SkipMsg

type autoRefreshMsg struct{}

func scheduleAutoRefresh() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(30 * time.Second)
		return autoRefreshMsg{}
	}
}

type voreskySessionLoadedMsg struct{ auth *voresky.VoreskyAuth }
type voreskySessionNotFoundMsg struct{}
type voreskyAuthSuccessMsg struct{ auth *voresky.VoreskyAuth }
type voreskyAuthErrorMsg struct{ err error }
type mainCharacterLoadedMsg struct{ character *voresky.Character }
type mainCharacterErrorMsg struct{ err error }

func (m App) tryLoadVoreskySession() tea.Msg {
	va := voresky.NewVoreskyAuth(defaultVoreskyURL, m.tokenStore)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := va.LoadStoredSession(ctx); err != nil {
		return voreskySessionNotFoundMsg{}
	}
	if va.GetCookie() == "" {
		return voreskySessionNotFoundMsg{}
	}

	return voreskySessionLoadedMsg{auth: va}
}

func (m App) validateVoreskyCookie(cookie string) tea.Cmd {
	return func() tea.Msg {
		normalized := vsetup.NormalizeCookie(cookie)
		va := voresky.NewVoreskyAuth(defaultVoreskyURL, m.tokenStore)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := va.AuthenticateWithCookie(ctx, normalized); err != nil {
			return voreskyAuthErrorMsg{err: err}
		}

		return voreskyAuthSuccessMsg{auth: va}
	}
}

func (m *App) initVoresky(va *voresky.VoreskyAuth, returnScreen Screen) tea.Cmd {
	m.voreskyAuth = va
	m.voreskyClient = voresky.NewVoreskyClient(defaultVoreskyURL, va)
	m.enrichManager = enrichment.New()
	m.tabBar.VoreskyActive = true
	contentHeight := m.contentHeight()
	m.voreskyTabModel = vtab.NewVoreskyModel(m.voreskyClient, m.width, contentHeight, m.imageCache)
	m.vnotifModel = vnotifications.NewVNotificationsModel(m.voreskyClient, m.width, contentHeight, m.imageCache)
	m.voreskyTabInit = false
	m.vnotifInit = false
	m.screen = returnScreen
	return tea.Batch(m.updateTabBarForScreen(), m.fetchMainCharacter())
}

func (m *App) fetchMainCharacter() tea.Cmd {
	voreskyAuth := m.voreskyAuth
	voreskyClient := m.voreskyClient
	return func() tea.Msg {
		if voreskyAuth == nil || voreskyClient == nil {
			return mainCharacterLoadedMsg{character: nil}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		session, err := voreskyAuth.ValidateSession(ctx)
		if err != nil {
			return mainCharacterErrorMsg{err: err}
		}
		if session.MainCharacterID == "" {
			return mainCharacterLoadedMsg{character: nil}
		}

		char, err := voreskyClient.GetCharacter(ctx, session.MainCharacterID)
		if err != nil {
			return mainCharacterErrorMsg{err: err}
		}
		return mainCharacterLoadedMsg{character: &char.Character}
	}
}

type enrichResultMsg struct {
	overrides map[string]*voresky.CaughtState
}

type enrichErrorMsg struct {
	err error
}

type snapshotResultMsg struct {
	hash string
	blob *voresky.SnapshotBlob
}

type snapshotErrorMsg struct {
	err error
}

func (m *App) buildAvatarOverrides() map[string]string {
	if m.enrichManager != nil {
		return m.enrichManager.BuildAvatarOverrides(m.ownDID, m.mainCharacterAvatar)
	}
	if m.mainCharacterAvatar != "" && m.ownDID != "" {
		return map[string]string{m.ownDID: m.mainCharacterAvatar}
	}
	return nil
}

func (m *App) pushAvatarOverrides(overrides map[string]string) {
	m.feedModel.SetAvatarOverrides(overrides)
	m.searchModel.SetAvatarOverrides(overrides)
	if m.screen == ScreenThread {
		m.threadModel.SetAvatarOverrides(overrides)
	}
	if m.selfProfileCreated {
		m.profileModel.SetAvatarOverrides(overrides)
	}
	if m.screen == ScreenCompose {
		m.composeModel.SetAvatarOverrides(overrides)
	}
}

func (m *App) prefetchAvatars(overrides map[string]string) []tea.Cmd {
	var cmds []tea.Cmd
	for _, url := range overrides {
		if cmd := images.FetchAvatar(m.imageCache, url); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return cmds
}

func (m *App) enrichDIDsFromFeedPosts(posts []*bsky.FeedDefs_FeedViewPost) tea.Cmd {
	if m.enrichManager == nil || m.voreskyClient == nil || len(posts) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var dids []string
	for _, post := range posts {
		if post == nil || post.Post == nil || post.Post.Author == nil {
			continue
		}
		did := post.Post.Author.Did
		if did != "" && !seen[did] {
			seen[did] = true
			dids = append(dids, did)
		}
	}
	return m.enrichDIDs(dids)
}

func (m *App) enrichDIDsFromPostViews(posts []*bsky.FeedDefs_PostView) tea.Cmd {
	if m.enrichManager == nil || m.voreskyClient == nil || len(posts) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var dids []string
	for _, post := range posts {
		if post == nil || post.Author == nil {
			continue
		}
		did := post.Author.Did
		if did != "" && !seen[did] {
			seen[did] = true
			dids = append(dids, did)
		}
	}
	return m.enrichDIDs(dids)
}

func (m *App) enrichDIDsFromThread(thread *bsky.FeedGetPostThread_Output_Thread) tea.Cmd {
	if m.enrichManager == nil || m.voreskyClient == nil || thread == nil {
		return nil
	}
	seen := make(map[string]bool)
	var dids []string

	var walk func(tvp *bsky.FeedDefs_ThreadViewPost)
	walk = func(tvp *bsky.FeedDefs_ThreadViewPost) {
		if tvp == nil {
			return
		}
		if tvp.Post != nil && tvp.Post.Author != nil {
			did := tvp.Post.Author.Did
			if did != "" && !seen[did] {
				seen[did] = true
				dids = append(dids, did)
			}
		}
		if tvp.Parent != nil && tvp.Parent.FeedDefs_ThreadViewPost != nil {
			walk(tvp.Parent.FeedDefs_ThreadViewPost)
		}
		for _, r := range tvp.Replies {
			if r != nil && r.FeedDefs_ThreadViewPost != nil {
				walk(r.FeedDefs_ThreadViewPost)
			}
		}
	}

	if thread.FeedDefs_ThreadViewPost != nil {
		walk(thread.FeedDefs_ThreadViewPost)
	}

	return m.enrichDIDs(dids)
}

// enrichDIDs captures copies of mutable state before spawning the goroutine
// to call the Voresky enrich API. Batches into chunks of MaxEnrichDIDs.
func (m *App) enrichDIDs(dids []string) tea.Cmd {
	if len(dids) == 0 {
		return nil
	}
	unknown := m.enrichManager.NeedEnrichment(dids)
	if len(unknown) == 0 {
		return nil
	}

	knownStates := m.enrichManager.KnownStates()
	client := m.voreskyClient

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		allOverrides := make(map[string]*voresky.CaughtState)
		for i := 0; i < len(unknown); i += voresky.MaxEnrichDIDs {
			end := i + voresky.MaxEnrichDIDs
			if end > len(unknown) {
				end = len(unknown)
			}
			chunk := unknown[i:end]
			resp, err := client.Enrich(ctx, chunk, knownStates)
			if err != nil {
				return enrichErrorMsg{err: err}
			}
			for k, v := range resp.CaughtOverrides {
				allOverrides[k] = v
			}
		}
		return enrichResultMsg{overrides: allOverrides}
	}
}

func (m *App) fetchSnapshot(hash string) tea.Cmd {
	client := m.voreskyClient
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		blob, err := client.GetSnapshot(ctx, hash)
		if err != nil {
			return snapshotErrorMsg{err: err}
		}
		return snapshotResultMsg{hash: hash, blob: blob}
	}
}

// startTransition captures a snapshot of the current main content and begins
// a slide transition. dir: +1 = push (new slides up from bottom), -1 = pop (old slides down).
func (m *App) startTransition(dir int) { //nolint:govet
	mainHeight := m.height - theme.TabBarHeight - theme.StatusBarHeight
	if mainHeight < 1 {
		mainHeight = 1
	}
	m.transitionSnapshot = m.renderMainContent(mainHeight)
	m.transitionDir = dir
	m.transitionStart = time.Now()
	m.transitionActive = true
}

// compositeTransition blends old and new content for a vertical slide transition.
// For push (+1): new content slides up from the bottom (old scrolls up).
// For pop (-1): old content slides down from the top (new is revealed underneath).
func (m App) compositeTransition(newContent string, height int, progress float64) string {
	old := shared.PadToHeight(m.transitionSnapshot, height)
	new := shared.PadToHeight(newContent, height)

	// splitLine = how many lines of "new" content are visible
	splitLine := int(progress * float64(height))
	if splitLine > height {
		splitLine = height
	}

	if m.transitionDir > 0 {
		// Push: new slides up from bottom. Top shows old (scrolling up), bottom shows new.
		topPart := shared.SliceContent(old, splitLine, height)
		bottomPart := shared.SliceContent(new, height-splitLine, height)
		return topPart + "\n" + bottomPart
	}
	// Pop: old slides down. Top shows new (being revealed), bottom shows old (sliding away).
	topPart := shared.SliceContent(new, 0, splitLine)
	bottomPart := shared.SliceContent(old, 0, height-splitLine)
	return topPart + "\n" + bottomPart
}

func (m App) contentHeight() int {
	h := m.height - theme.TabBarHeight - theme.StatusBarHeight
	if h < 1 {
		return 1
	}
	return h
}

// finishLogin wires up client models and navigates to the post-login screen.
// It is shared between OAuth and app-password login paths.
func (m App) finishLogin(client bluesky.BlueskyClient, did, handle string) (App, tea.Cmd) {
	m.loggedIn = true
	m.client = client

	contentHeight := m.contentHeight()
	m.feedModel = feed.NewFeedModel(m.client, did, m.width, contentHeight, m.imageCache)
	m.notifModel = notifications.NewNotificationsModel(m.client, m.width, contentHeight, m.imageCache)
	m.searchModel = search.NewSearchModel(m.client, m.width, contentHeight, m.imageCache)

	m.statusBar.Handle = handle
	m.statusBar.DID = did
	m.ownDID = did

	m.prevScreen = ScreenFeed
	m.screen = ScreenVoreskySetup
	m.vsetupModel = vsetup.New()

	return m, tea.Batch(
		m.feedModel.Init(),
		m.vsetupModel.Init(),
		m.tryLoadVoreskySession,
		scheduleAutoRefresh(),
	)
}

package ui

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/dotBeeps/noms/internal/auth"
	"github.com/dotBeeps/noms/internal/ui/login"
	"github.com/dotBeeps/noms/internal/ui/theme"
	"github.com/dotBeeps/noms/internal/ui/vsetup"
)

var appTestAnsiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsiApp(s string) string { return appTestAnsiRe.ReplaceAllString(s, "") }

func TestAppInitShowsLogin(t *testing.T) {
	t.Parallel()
	app := NewApp()

	if app.Screen() != ScreenLogin {
		t.Errorf("Expected initial screen to be ScreenLogin, got %v", app.Screen())
	}

	if app.IsLoggedIn() {
		t.Errorf("Expected app to not be logged in initially")
	}
}

func TestAppLoginSuccess(t *testing.T) {
	t.Parallel()
	app := NewApp()

	session := &auth.Session{
		DID:    "did:plc:test123",
		Handle: "test.bsky.social",
		PDS:    "https://bsky.social",
	}

	updated, _ := app.Update(login.LoginSuccessMsg{Session: session})
	app = updated.(App)

	if app.Screen() != ScreenVoreskySetup {
		t.Errorf("Expected screen to switch to ScreenVoreskySetup after login, got %v", app.Screen())
	}

	updated, _ = app.Update(vsetup.SkipMsg{})
	app = updated.(App)

	if app.Screen() != ScreenFeed {
		t.Errorf("Expected screen to switch to ScreenFeed after voresky skip, got %v", app.Screen())
	}

	if !app.IsLoggedIn() {
		t.Errorf("Expected app to be logged in after LoginSuccessMsg")
	}

	if app.Session().Handle != "test.bsky.social" {
		t.Errorf("Expected session handle to be 'test.bsky.social', got %s", app.Session().Handle)
	}
}

func TestScreenSwitchWithNumbers(t *testing.T) {
	t.Parallel()
	app := NewApp()
	app.loggedIn = true
	app.screen = ScreenFeed

	updated, _ := app.Update(tea.KeyPressMsg{})
	app = updated.(App)

	if app.Screen() != ScreenFeed {
		t.Errorf("Expected screen to remain Feed for non-number key, got %v", app.Screen())
	}
}

func TestKeyBindingQuit(t *testing.T) {
	t.Parallel()
	app := NewApp()
	app.screen = ScreenLogin

	updated, cmd := app.Update(tea.KeyPressMsg{})
	app = updated.(App)

	if cmd != nil {
		t.Errorf("Expected nil cmd for non-quit key, got %v", cmd)
	}
}

func TestKeyBindingHelp(t *testing.T) {
	t.Parallel()
	app := NewApp()
	app.loggedIn = true
	app.screen = ScreenFeed

	if app.ShowHelp() {
		t.Errorf("Expected help to be hidden initially")
	}

	updated, _ := app.Update(tea.KeyPressMsg{})
	app = updated.(App)

	if app.ShowHelp() {
		t.Errorf("Expected help to remain hidden for non-? key")
	}
}

func TestWindowResize(t *testing.T) {
	t.Parallel()
	app := NewApp()

	updated, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	app = updated.(App)

	if app.Width() != 100 {
		t.Errorf("Expected width to be 100, got %d", app.Width())
	}

	if app.Height() != 40 {
		t.Errorf("Expected height to be 40, got %d", app.Height())
	}
}

func TestAppViewLogin(t *testing.T) {
	t.Parallel()
	app := NewApp()
	app.screen = ScreenLogin

	v := app.View()
	content := v.Content

	if !strings.Contains(content, "Handle:") {
		t.Errorf("Expected login view to contain 'Handle:', got %q", content)
	}
}

func TestAppViewFeed(t *testing.T) {
	t.Parallel()
	app := NewApp()
	app.screen = ScreenFeed
	app.loggedIn = true
	app.width = 80
	app.height = 24

	v := app.View()
	content := v.Content

	if !strings.Contains(content, "Feed") {
		t.Errorf("Expected feed view to contain 'Feed', got %q", content)
	}
}

func TestTabBarIntegration(t *testing.T) {
	t.Parallel()
	app := NewApp()
	app.loggedIn = true
	app.screen = ScreenFeed
	app.width = 80
	app.height = 24

	v := app.View()
	content := v.Content

	if !strings.Contains(stripAnsiApp(content), "[1]") {
		t.Errorf("Expected view to contain tab bar with [1], got %q", content)
	}
}

func TestStatusBarIntegration(t *testing.T) {
	t.Parallel()
	app := NewApp()
	app.loggedIn = true
	app.screen = ScreenFeed
	app.width = 80
	app.height = 24
	app.statusBar.Handle = "test.bsky.social"
	app.statusBar.DID = "did:plc:test123"
	app.statusBar.Connected = true

	v := app.View()
	content := v.Content

	if !strings.Contains(content, "test.bsky.social") {
		t.Errorf("Expected view to contain status bar with handle, got %q", content)
	}
}

func TestHelpKeyMapChangesWithScreen(t *testing.T) {
	t.Parallel()
	app := NewApp()

	// On login screen, activeKeyMap returns login keys
	if app.screen != ScreenLogin {
		t.Errorf("Expected screen to be Login initially, got %v", app.screen)
	}

	session := &auth.Session{
		DID:    "did:plc:test123",
		Handle: "test.bsky.social",
		PDS:    "https://bsky.social",
	}

	updated, _ := app.Update(login.LoginSuccessMsg{Session: session})
	app = updated.(App)

	updated, _ = app.Update(vsetup.SkipMsg{})
	app = updated.(App)

	if app.screen != ScreenFeed {
		t.Errorf("Expected screen to be Feed after voresky skip, got %v", app.screen)
	}
}

func TestAltScreen(t *testing.T) {
	t.Parallel()
	app := NewApp()
	app.screen = ScreenLogin

	v := app.View()

	if !v.AltScreen {
		t.Errorf("Expected AltScreen to be true for login view")
	}

	app.screen = ScreenFeed
	app.loggedIn = true
	app.width = 80
	app.height = 24

	v = app.View()

	if !v.AltScreen {
		t.Errorf("Expected AltScreen to be true for main view")
	}
}

func TestThemeCycleKeybindings(t *testing.T) {
	theme.Apply("default")
	t.Cleanup(func() {
		theme.Apply("default")
	})

	app := NewApp()
	app.cfg = nil
	app.loggedIn = true
	app.screen = ScreenFeed

	original := theme.ActiveTheme()

	updated, _ := app.Update(tea.KeyPressMsg{Text: "]"})
	app = updated.(App)

	afterNext := theme.ActiveTheme()
	if afterNext == original {
		t.Fatalf("expected ] to change theme, stayed on %q", afterNext)
	}

	updated, _ = app.Update(tea.KeyPressMsg{Text: "["})
	app = updated.(App)

	if theme.ActiveTheme() != original {
		t.Fatalf("expected [ to return to original theme %q, got %q", original, theme.ActiveTheme())
	}
}

func TestThemePickerHotkeyAndSelection(t *testing.T) {
	theme.Apply("default")
	t.Cleanup(func() {
		theme.Apply("default")
	})

	app := NewApp()
	app.cfg = nil
	app.loggedIn = true
	app.screen = ScreenFeed
	app.width = 80
	app.height = 24

	updated, _ := app.Update(tea.KeyPressMsg{Text: "ctrl+t"})
	app = updated.(App)

	if !app.showThemePicker {
		t.Fatalf("expected ctrl+t to open theme picker")
	}

	v := app.View()
	if !strings.Contains(v.Content, "Theme Picker") {
		t.Fatalf("expected theme picker overlay to be visible, got %q", v.Content)
	}

	updated, _ = app.Update(tea.KeyPressMsg{Text: "j"})
	app = updated.(App)
	updated, _ = app.Update(tea.KeyPressMsg{Text: "enter"})
	app = updated.(App)

	if app.showThemePicker {
		t.Fatalf("expected theme picker to close after Enter")
	}

	if theme.ActiveTheme() == "default" {
		t.Fatalf("expected selection in picker to apply a non-default theme")
	}
}

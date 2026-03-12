package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/dotBeeps/noms/internal/auth"
	"github.com/dotBeeps/noms/internal/ui/components"
	"github.com/dotBeeps/noms/internal/ui/login"
	"github.com/dotBeeps/noms/internal/ui/vsetup"
)

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

	if !strings.Contains(content, "[1]") {
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

func TestHelpContextChange(t *testing.T) {
	t.Parallel()
	app := NewApp()

	if app.help.Context != components.HelpContextLogin {
		t.Errorf("Expected help context to be Login initially, got %v", app.help.Context)
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

	if app.help.Context != components.HelpContextFeed {
		t.Errorf("Expected help context to be Feed after voresky skip, got %v", app.help.Context)
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

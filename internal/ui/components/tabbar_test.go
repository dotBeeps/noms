package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestTabBarInit(t *testing.T) {
	t.Parallel()
	tb := NewTabBar()
	if cmd := tb.Init(); cmd != nil {
		t.Errorf("Expected nil cmd, got %v", cmd)
	}
}

func TestTabNavigationWithNumbers(t *testing.T) {
	t.Parallel()
	tb := NewTabBar()
	tb.Width = 80

	if tb.ActiveTab != TabFeed {
		t.Errorf("Expected initial tab to be Feed, got %v", tb.ActiveTab)
	}

	tb.SetActiveTab(TabNotifications)
	if tb.ActiveTab != TabNotifications {
		t.Errorf("Expected tab to be Notifications, got %v", tb.ActiveTab)
	}

	tb.SetActiveTab(TabProfile)
	if tb.ActiveTab != TabProfile {
		t.Errorf("Expected tab to be Profile, got %v", tb.ActiveTab)
	}

	tb.SetActiveTab(TabSearch)
	if tb.ActiveTab != TabSearch {
		t.Errorf("Expected tab to be Search, got %v", tb.ActiveTab)
	}

	tb.SetActiveTab(TabFeed)
	if tb.ActiveTab != TabFeed {
		t.Errorf("Expected tab to be Feed, got %v", tb.ActiveTab)
	}
}

func TestTabHighlight(t *testing.T) {
	t.Parallel()
	tb := NewTabBar()
	tb.Width = 80
	tb.ActiveTab = TabFeed

	v := tb.View()
	content := v.Content

	if !strings.Contains(content, "Feed") {
		t.Errorf("Expected tab bar to contain 'Feed', got %q", content)
	}

	tb.ActiveTab = TabNotifications
	v = tb.View()
	content = v.Content

	if !strings.Contains(content, "Notifications") {
		t.Errorf("Expected tab bar to contain 'Notifications', got %q", content)
	}
}

func TestTabBarWidthUpdate(t *testing.T) {
	t.Parallel()
	tb := NewTabBar()
	tb.Width = 60

	updated, _ := tb.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	tb = updated.(TabBar)

	if tb.Width != 100 {
		t.Errorf("Expected width to be 100, got %d", tb.Width)
	}
}

func TestTabBarInvalidTab(t *testing.T) {
	t.Parallel()
	tb := NewTabBar()
	tb.ActiveTab = TabFeed

	tb.SetActiveTab(Tab(-1))
	if tb.ActiveTab != TabFeed {
		t.Errorf("Expected tab to remain Feed after invalid set, got %v", tb.ActiveTab)
	}

	tb.SetActiveTab(TabCount)
	if tb.ActiveTab != TabFeed {
		t.Errorf("Expected tab to remain Feed after invalid set, got %v", tb.ActiveTab)
	}
}

func TestTabBarActiveTabName(t *testing.T) {
	t.Parallel()
	tb := NewTabBar()
	tb.ActiveTab = TabProfile

	if name := tb.ActiveTabName(); name != "Profile" {
		t.Errorf("Expected active tab name to be 'Profile', got %q", name)
	}
}

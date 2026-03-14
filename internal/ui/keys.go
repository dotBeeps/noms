package ui

import "charm.land/bubbles/v2/key"

// GlobalKeyMap defines key bindings shared across all screens.
type GlobalKeyMap struct {
	Help       key.Binding
	Quit       key.Binding
	ForceQuit  key.Binding
	Tab1       key.Binding
	Tab2       key.Binding
	Tab3       key.Binding
	Tab4       key.Binding
	Tab5       key.Binding
	Tab6       key.Binding
	PrevTheme  key.Binding
	NextTheme  key.Binding
	ThemePick  key.Binding
	VoreskySet key.Binding
}

// ShortHelp satisfies help.KeyMap.
func (k GlobalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp satisfies help.KeyMap.
func (k GlobalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3, k.Tab4},
		{k.PrevTheme, k.NextTheme, k.ThemePick},
		{k.Help, k.Quit, k.ForceQuit},
	}
}

var globalKeys = GlobalKeyMap{
	Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
	Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	ForceQuit:  key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "force quit")),
	Tab1:       key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "feed")),
	Tab2:       key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "notifications")),
	Tab3:       key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "profile")),
	Tab4:       key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "search")),
	Tab5:       key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "voresky")),
	Tab6:       key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "v-notifs")),
	PrevTheme:  key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev theme")),
	NextTheme:  key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next theme")),
	ThemePick:  key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "theme picker")),
	VoreskySet: key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "voresky setup")),
}

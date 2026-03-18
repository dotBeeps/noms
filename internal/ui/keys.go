package ui

import "charm.land/bubbles/v2/key"

// GlobalKeyMap defines key bindings shared across all screens.
type GlobalKeyMap struct {
	Help       key.Binding
	Quit       key.Binding
	ForceQuit  key.Binding
	TabNext    key.Binding
	TabPrev    key.Binding
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
		{k.TabNext, k.TabPrev},
		{k.PrevTheme, k.NextTheme, k.ThemePick},
		{k.Help, k.Quit, k.ForceQuit},
	}
}

var globalKeys = GlobalKeyMap{
	Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
	Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	ForceQuit:  key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "force quit")),
	TabNext:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	TabPrev:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
	PrevTheme:  key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev theme")),
	NextTheme:  key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next theme")),
	ThemePick:  key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "theme picker")),
	VoreskySet: key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "voresky setup")),
}

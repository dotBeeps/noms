package vnotifications

import "charm.land/bubbles/v2/key"

// KeyMap defines key bindings for the Voresky notifications screen.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	MarkRead key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.MarkRead}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.MarkRead},
	}
}

var DefaultKeyMap = KeyMap{
	Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "down")),
	MarkRead: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "mark as read")),
}

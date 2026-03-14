package notifications

import "charm.land/bubbles/v2/key"

// KeyMap defines key bindings for the notifications screen.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Refresh key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Open, k.Refresh}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Open, k.Refresh},
	}
}

var DefaultKeyMap = KeyMap{
	Up:      key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "down")),
	Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open notification")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh & mark read")),
}

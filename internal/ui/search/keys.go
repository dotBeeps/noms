package search

import "charm.land/bubbles/v2/key"

// KeyMap defines key bindings for the search screen.
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Open   key.Binding
	Focus  key.Binding
	Toggle key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Focus, k.Toggle, k.Open}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Open},
		{k.Focus, k.Toggle},
	}
}

var DefaultKeyMap = KeyMap{
	Up:     key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "up")),
	Down:   key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "down")),
	Open:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select result")),
	Focus:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "focus search")),
	Toggle: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle posts/people")),
}

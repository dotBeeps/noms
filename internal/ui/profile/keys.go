package profile

import "charm.land/bubbles/v2/key"

// KeyMap defines key bindings for the profile screen.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Follow  key.Binding
	Delete  key.Binding
	Refresh key.Binding
	Back    key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Open, k.Follow, k.Back}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Open, k.Back},
		{k.Follow, k.Delete, k.Refresh},
	}
}

var DefaultKeyMap = KeyMap{
	Up:      key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "down")),
	Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open thread")),
	Follow:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "follow/unfollow")),
	Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d d", "delete your post")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Back:    key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
}

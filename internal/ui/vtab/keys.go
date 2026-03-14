package vtab

import "charm.land/bubbles/v2/key"

// KeyMap defines key bindings for the Voresky tab screen.
type KeyMap struct {
	Up   key.Binding
	Down key.Binding
	Open key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Open}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Open},
	}
}

var DefaultKeyMap = KeyMap{
	Up:   key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "up")),
	Down: key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "down")),
	Open: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "view character")),
}

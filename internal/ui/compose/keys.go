package compose

import "charm.land/bubbles/v2/key"

// KeyMap defines key bindings for the compose screen.
type KeyMap struct {
	Submit key.Binding
	Cancel key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Submit, k.Cancel}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Submit, k.Cancel},
	}
}

var DefaultKeyMap = KeyMap{
	Submit: key.NewBinding(key.WithKeys("ctrl+enter"), key.WithHelp("ctrl+enter", "submit post")),
	Cancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

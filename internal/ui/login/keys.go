package login

import "charm.land/bubbles/v2/key"

// KeyMap defines key bindings for the login screen.
type KeyMap struct {
	Submit key.Binding
	Tab    key.Binding
	Back   key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Submit, k.Tab}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Submit, k.Tab, k.Back},
	}
}

var DefaultKeyMap = KeyMap{
	Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
	Tab:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch focus")),
	Back:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
}

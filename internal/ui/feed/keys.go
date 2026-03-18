package feed

import "charm.land/bubbles/v2/key"

// KeyMap defines key bindings for the feed screen.
type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Open       key.Binding
	Like       key.Binding
	Repost     key.Binding
	Reply      key.Binding
	Compose    key.Binding
	Delete     key.Binding
	ViewImages key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Open, k.Like}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Open},
		{k.Like, k.Repost, k.Reply},
		{k.Compose, k.Delete, k.ViewImages},
	}
}

var DefaultKeyMap = KeyMap{
	Up:         key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "up")),
	Down:       key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "down")),
	Open:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open thread")),
	Like:       key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "like/unlike")),
	Repost:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "repost/un-repost")),
	Reply:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reply")),
	Compose:    key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "compose")),
	Delete:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d d", "delete your post")),
	ViewImages: key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "view images")),
}

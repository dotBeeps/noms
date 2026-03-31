package testutil

import tea "charm.land/bubbletea/v2"

// ExecBatch executes a tea.Cmd and recursively unwraps any tea.BatchMsg,
// returning all leaf messages as a flat slice.
func ExecBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			msgs = append(msgs, ExecBatch(c)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}

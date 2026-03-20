package components

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
)

// FormKeyMap returns a customized huh KeyMap with:
//   - Esc added to Quit
//   - ↑/↓ added to Prev/Next for all field types (Confirm, Input, Text)
func FormKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"))

	// Confirm: add up/down for field navigation
	km.Confirm.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑/shift+tab", "back"))
	km.Confirm.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))

	// Input: add up/down for field navigation
	km.Input.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑/shift+tab", "back"))
	km.Input.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))

	// Text: add up/down for field navigation
	km.Text.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑/shift+tab", "back"))
	km.Text.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))

	return km
}

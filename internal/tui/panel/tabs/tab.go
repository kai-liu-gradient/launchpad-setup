package tabs

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/tui/components"
)

// RunFormAsEdit takes a huh.Form, applies the standard keymap and alt-screen,
// runs it, and returns any error.
func RunFormAsEdit(form *huh.Form) error {
	form.WithKeyMap(components.FormKeyMap()).
		WithProgramOptions(tea.WithAltScreen())
	return form.Run()
}

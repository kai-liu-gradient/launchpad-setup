package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type MenuItem struct {
	Name string
}

type SideMenu struct {
	Items  []MenuItem
	Active int
}

func (m *SideMenu) Next() {
	m.Active = (m.Active + 1) % len(m.Items)
}

func (m *SideMenu) Prev() {
	m.Active = (m.Active - 1 + len(m.Items)) % len(m.Items)
}

func (m *SideMenu) Render() string {
	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(PrimaryColor).
		Padding(0, 1).
		Width(16)
	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Padding(0, 1).
		Width(16)

	var lines []string
	for i, item := range m.Items {
		prefix := "  "
		if i == m.Active {
			prefix = "▸ "
			lines = append(lines, activeStyle.Render(fmt.Sprintf("%s%s", prefix, item.Name)))
		} else {
			lines = append(lines, inactiveStyle.Render(fmt.Sprintf("%s%s", prefix, item.Name)))
		}
	}
	return strings.Join(lines, "\n")
}

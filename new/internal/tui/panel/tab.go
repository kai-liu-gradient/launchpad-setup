package panel

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gradient8/launchpad/internal/tui/components"
)

type Tab struct {
	Name string
	// Content will be rendered by each tab's View function
}

type TabBar struct {
	Tabs   []Tab
	Active int
}

func (tb *TabBar) Next() {
	tb.Active = (tb.Active + 1) % len(tb.Tabs)
}

func (tb *TabBar) Prev() {
	tb.Active = (tb.Active - 1 + len(tb.Tabs)) % len(tb.Tabs)
}

func (tb *TabBar) Render() string {
	var tabs []string
	activeStyle := lipgloss.NewStyle().
		Background(components.PrimaryColor).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)
	inactiveStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#888888")).
		Padding(0, 1)

	for i, tab := range tb.Tabs {
		if i == tb.Active {
			tabs = append(tabs, activeStyle.Render(tab.Name))
		} else {
			tabs = append(tabs, inactiveStyle.Render(tab.Name))
		}
	}
	return strings.Join(tabs, " ")
}

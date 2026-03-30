package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/tui/components"
	"github.com/gradient8/launchpad/internal/tui/panel/tabs"
)

// Mode selects which install wizard to present to the user.
type Mode int

const (
	Express Mode = iota
	Custom
)

// ExitReason indicates why the wizard panel exited.
type ExitReason int

const (
	ExitQuit    ExitReason = iota // user pressed q/esc/ctrl+c
	ExitDeploy                    // user pressed d to deploy
	ExitEdit                      // user pressed enter to edit a tab (internal)
)

// EditableTab matches the configure panel's tab interface.
type EditableTab interface {
	View() string
	Form() *huh.Form
	Apply(cfg *config.Config)
}

// Model is the bubbletea model for the install wizard.
type Model struct {
	mode       Mode
	form       *huh.Form // only used in Express mode
	cfg        *config.Config
	menu       components.SideMenu
	tabList    []EditableTab
	result     *config.Config
	done       bool
	aborted    bool
	exitReason ExitReason
	message    string
}

// New creates a wizard Model for the given mode.
func New(mode Mode) Model {
	if mode == Express {
		return Model{
			mode: mode,
			form: NewExpressForm(),
		}
	}
	return newCustomPanel(config.DefaultConfig(""))
}

// NewWithConfig creates a Custom wizard pre-filled with an existing config.
func NewWithConfig(cfg *config.Config) Model {
	return newCustomPanel(cfg)
}

func newCustomPanel(cfg *config.Config) Model {
	items := []components.MenuItem{
		{Name: "Basic"},
		{Name: "SSL"},
		{Name: "Database"},
		{Name: "Kubernetes"},
		{Name: "SMTP"},
		{Name: "SSO"},
		{Name: "Storage"},
		{Name: "Telegram"},
		{Name: "AI"},
		{Name: "Stripe"},
		{Name: "Performance"},
		{Name: "Experimental"},
	}

	tabList := []EditableTab{
		tabs.NewBasicTab(cfg),
		tabs.NewSSLTab(cfg),
		tabs.NewDatabaseTab(cfg),
		tabs.NewKubernetesTab(cfg),
		tabs.NewSMTPTab(cfg),
		tabs.NewSSOTab(cfg),
		tabs.NewStorageTab(cfg),
		tabs.NewTelegramTab(cfg),
		tabs.NewAITab(cfg),
		tabs.NewStripeTab(cfg),
		tabs.NewPerformanceTab(cfg),
		tabs.NewExperimentalTab(cfg),
	}

	return Model{
		mode:    Custom,
		cfg:     cfg,
		menu:    components.SideMenu{Items: items, Active: 0},
		tabList: tabList,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.mode == Express {
		return m.form.Init()
	}
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.mode == Express {
		return m.updateExpress(msg)
	}
	return m.updateCustom(msg)
}

func (m Model) updateExpress(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "ctrl+c" {
			m.aborted = true
			return m, tea.Quit
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		if !expressConfirm {
			// Switch to Custom panel, carry over the domain
			cfg := config.DefaultConfig(expressDomain)
			cfg.ProjectDomain = expressProjectDomain
			applyExpressAdmin(cfg)
			panel := newCustomPanel(cfg)
			return panel, nil
		}
		m.done = true
		m.result = config.DefaultConfig(expressDomain)
		m.result.ProjectDomain = expressProjectDomain
		applyExpressAdmin(m.result)
		return m, tea.Quit
	}

	return m, cmd
}

func (m Model) updateCustom(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.menu.Prev()
			m.message = ""
		case "down", "j":
			m.menu.Next()
			m.message = ""
		case "q", "ctrl+c":
			m.aborted = true
			return m, tea.Quit
		case "enter":
			m.exitReason = ExitEdit
			return m, tea.Quit
		case "d":
			// Apply all tabs to config, then deploy
			for _, tab := range m.tabList {
				tab.Apply(m.cfg)
			}
			// Ensure subdomain and admin email are set
			m.cfg.Subdomain = "launchpad"
			m.cfg.AdminEmail = "admin@" + m.cfg.Domain
			if m.cfg.Domain == "" {
				m.message = "Domain is required — edit Basic first"
				return m, nil
			}
			m.done = true
			m.result = m.cfg
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	header := components.TitleStyle.Render("AniLaunchpad Setup")

	if m.mode == Express {
		subtitle := components.SubtitleStyle.Render("Express Setup — quick deploy with defaults")
		return fmt.Sprintf("%s\n%s\n\n%s", header, subtitle, m.form.View())
	}

	subtitle := components.SubtitleStyle.Render("Custom Setup — configure each section")

	menuStr := m.menu.Render()
	content := m.tabList[m.menu.Active].View()

	if m.message != "" {
		content += "\n\n  " + components.SuccessStyle.Render(m.message)
	}

	menuBox := lipgloss.NewStyle().
		Width(18).
		MarginRight(2).
		Render(menuStr)

	contentLines := strings.Split(content, "\n")
	contentStr := strings.Join(contentLines, "\n")
	contentBox := lipgloss.NewStyle().Render(contentStr)

	body := lipgloss.JoinHorizontal(lipgloss.Top, menuBox, contentBox)

	hint := "↑/↓ navigate · enter edit · d deploy · q quit"
	statusBar := components.MutedStyle.Render(hint)

	return fmt.Sprintf("\n%s\n%s\n\n%s\n\n%s\n", header, subtitle, body, statusBar)
}

// Result returns the completed Config.
func (m Model) Result() *config.Config {
	return m.result
}

// Done reports whether the wizard completed successfully.
func (m Model) Done() bool {
	return m.done
}

// Aborted reports whether the user cancelled.
func (m Model) Aborted() bool {
	return m.aborted
}

// ExitReason returns why the panel exited (for edit loop).
func (m Model) ExitReason() ExitReason {
	return m.exitReason
}

// ActiveTab returns the currently selected tab index.
func (m Model) ActiveTab() int {
	return m.menu.Active
}

// Tabs returns the tab list for external edit loop access.
func (m Model) Tabs() []EditableTab {
	return m.tabList
}

// applyExpressAdmin applies express admin email/password overrides to the config.
func applyExpressAdmin(cfg *config.Config) {
	if strings.TrimSpace(expressAdminEmail) != "" {
		cfg.AdminEmail = strings.TrimSpace(expressAdminEmail)
	}
}


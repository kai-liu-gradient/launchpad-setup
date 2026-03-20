package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type NoticesTab struct {
	ProjectListEnabled     bool
	ProjectListSeverity    string
	ProjectListDismissible bool
	ProjectListTitle       string
	ProjectListMessage     string
	ProjectListLinkURL     string
	ProjectListLinkText    string

	CodingAgentEnabled     bool
	CodingAgentSeverity    string
	CodingAgentDismissible bool
	CodingAgentTitle       string
	CodingAgentMessage     string
	CodingAgentLinkURL     string
	CodingAgentLinkText    string

	BillingEnabled     bool
	BillingSeverity    string
	BillingDismissible bool
	BillingTitle       string
	BillingMessage     string
	BillingLinkURL     string
	BillingLinkText    string

	CreditDrawerEnabled     bool
	CreditDrawerSeverity    string
	CreditDrawerDismissible bool
	CreditDrawerTitle       string
	CreditDrawerMessage     string
	CreditDrawerLinkURL     string
	CreditDrawerLinkText    string
}

func NewNoticesTab(s *settingsmod.Settings) *NoticesTab {
	return &NoticesTab{
		ProjectListEnabled:     s.Notices.ProjectList.Enabled,
		ProjectListSeverity:    s.Notices.ProjectList.Severity,
		ProjectListDismissible: s.Notices.ProjectList.Dismissible,
		ProjectListTitle:       s.Notices.ProjectList.Title,
		ProjectListMessage:     s.Notices.ProjectList.Message,
		ProjectListLinkURL:     s.Notices.ProjectList.LinkURL,
		ProjectListLinkText:    s.Notices.ProjectList.LinkText,

		CodingAgentEnabled:     s.Notices.CodingAgent.Enabled,
		CodingAgentSeverity:    s.Notices.CodingAgent.Severity,
		CodingAgentDismissible: s.Notices.CodingAgent.Dismissible,
		CodingAgentTitle:       s.Notices.CodingAgent.Title,
		CodingAgentMessage:     s.Notices.CodingAgent.Message,
		CodingAgentLinkURL:     s.Notices.CodingAgent.LinkURL,
		CodingAgentLinkText:    s.Notices.CodingAgent.LinkText,

		BillingEnabled:     s.Notices.Billing.Enabled,
		BillingSeverity:    s.Notices.Billing.Severity,
		BillingDismissible: s.Notices.Billing.Dismissible,
		BillingTitle:       s.Notices.Billing.Title,
		BillingMessage:     s.Notices.Billing.Message,
		BillingLinkURL:     s.Notices.Billing.LinkURL,
		BillingLinkText:    s.Notices.Billing.LinkText,

		CreditDrawerEnabled:     s.Notices.CreditDrawer.Enabled,
		CreditDrawerSeverity:    s.Notices.CreditDrawer.Severity,
		CreditDrawerDismissible: s.Notices.CreditDrawer.Dismissible,
		CreditDrawerTitle:       s.Notices.CreditDrawer.Title,
		CreditDrawerMessage:     s.Notices.CreditDrawer.Message,
		CreditDrawerLinkURL:     s.Notices.CreditDrawer.LinkURL,
		CreditDrawerLinkText:    s.Notices.CreditDrawer.LinkText,
	}
}

var severityOptions = []huh.Option[string]{
	huh.NewOption("Warning", "warning"),
	huh.NewOption("Info", "info"),
	huh.NewOption("Error", "error"),
}

func (t *NoticesTab) Form() *huh.Form {
	return huh.NewForm(
		// Project List
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Project List Notice — Enabled").
				Value(&t.ProjectListEnabled),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Project List — Severity").
				Options(severityOptions...).
				Value(&t.ProjectListSeverity),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Project List — Dismissible").
				Value(&t.ProjectListDismissible),
			huh.NewInput().
				Title("Project List — Title").
				Value(&t.ProjectListTitle),
			huh.NewInput().
				Title("Project List — Message").
				Value(&t.ProjectListMessage),
			huh.NewInput().
				Title("Project List — Link URL").
				Value(&t.ProjectListLinkURL),
			huh.NewInput().
				Title("Project List — Link Text").
				Value(&t.ProjectListLinkText),
		).WithHideFunc(func() bool { return !t.ProjectListEnabled }),

		// Coding Agent
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Coding Agent Notice — Enabled").
				Value(&t.CodingAgentEnabled),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Coding Agent — Severity").
				Options(severityOptions...).
				Value(&t.CodingAgentSeverity),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Coding Agent — Dismissible").
				Value(&t.CodingAgentDismissible),
			huh.NewInput().
				Title("Coding Agent — Title").
				Value(&t.CodingAgentTitle),
			huh.NewInput().
				Title("Coding Agent — Message").
				Value(&t.CodingAgentMessage),
			huh.NewInput().
				Title("Coding Agent — Link URL").
				Value(&t.CodingAgentLinkURL),
			huh.NewInput().
				Title("Coding Agent — Link Text").
				Value(&t.CodingAgentLinkText),
		).WithHideFunc(func() bool { return !t.CodingAgentEnabled }),

		// Billing
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Billing Notice — Enabled").
				Value(&t.BillingEnabled),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Billing — Severity").
				Options(severityOptions...).
				Value(&t.BillingSeverity),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Billing — Dismissible").
				Value(&t.BillingDismissible),
			huh.NewInput().
				Title("Billing — Title").
				Value(&t.BillingTitle),
			huh.NewInput().
				Title("Billing — Message").
				Value(&t.BillingMessage),
			huh.NewInput().
				Title("Billing — Link URL").
				Value(&t.BillingLinkURL),
			huh.NewInput().
				Title("Billing — Link Text").
				Value(&t.BillingLinkText),
		).WithHideFunc(func() bool { return !t.BillingEnabled }),

		// Credit Drawer
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Credit Drawer Notice — Enabled").
				Value(&t.CreditDrawerEnabled),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Credit Drawer — Severity").
				Options(severityOptions...).
				Value(&t.CreditDrawerSeverity),
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Credit Drawer — Dismissible").
				Value(&t.CreditDrawerDismissible),
			huh.NewInput().
				Title("Credit Drawer — Title").
				Value(&t.CreditDrawerTitle),
			huh.NewInput().
				Title("Credit Drawer — Message").
				Value(&t.CreditDrawerMessage),
			huh.NewInput().
				Title("Credit Drawer — Link URL").
				Value(&t.CreditDrawerLinkURL),
			huh.NewInput().
				Title("Credit Drawer — Link Text").
				Value(&t.CreditDrawerLinkText),
		).WithHideFunc(func() bool { return !t.CreditDrawerEnabled }),
	)
}

func noticeStatus(enabled bool, title string) string {
	if !enabled {
		return "off"
	}
	if title == "" {
		return "on"
	}
	return fmt.Sprintf("on — %s", title)
}

func (t *NoticesTab) View() string {
	return fmt.Sprintf(
		"  Project List:  %s\n  Coding Agent:  %s\n  Billing:       %s\n  Credit Drawer: %s",
		noticeStatus(t.ProjectListEnabled, t.ProjectListTitle),
		noticeStatus(t.CodingAgentEnabled, t.CodingAgentTitle),
		noticeStatus(t.BillingEnabled, t.BillingTitle),
		noticeStatus(t.CreditDrawerEnabled, t.CreditDrawerTitle),
	)
}

func (t *NoticesTab) Apply(s *settingsmod.Settings) {
	s.Notices.ProjectList = settingsmod.Notice{
		Enabled:     t.ProjectListEnabled,
		Severity:    t.ProjectListSeverity,
		Dismissible: t.ProjectListDismissible,
		Title:       t.ProjectListTitle,
		Message:     t.ProjectListMessage,
		LinkURL:     t.ProjectListLinkURL,
		LinkText:    t.ProjectListLinkText,
	}
	s.Notices.CodingAgent = settingsmod.Notice{
		Enabled:     t.CodingAgentEnabled,
		Severity:    t.CodingAgentSeverity,
		Dismissible: t.CodingAgentDismissible,
		Title:       t.CodingAgentTitle,
		Message:     t.CodingAgentMessage,
		LinkURL:     t.CodingAgentLinkURL,
		LinkText:    t.CodingAgentLinkText,
	}
	s.Notices.Billing = settingsmod.Notice{
		Enabled:     t.BillingEnabled,
		Severity:    t.BillingSeverity,
		Dismissible: t.BillingDismissible,
		Title:       t.BillingTitle,
		Message:     t.BillingMessage,
		LinkURL:     t.BillingLinkURL,
		LinkText:    t.BillingLinkText,
	}
	s.Notices.CreditDrawer = settingsmod.Notice{
		Enabled:     t.CreditDrawerEnabled,
		Severity:    t.CreditDrawerSeverity,
		Dismissible: t.CreditDrawerDismissible,
		Title:       t.CreditDrawerTitle,
		Message:     t.CreditDrawerMessage,
		LinkURL:     t.CreditDrawerLinkURL,
		LinkText:    t.CreditDrawerLinkText,
	}
}

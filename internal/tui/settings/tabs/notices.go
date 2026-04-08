package tabs

import (
	"fmt"

	settingsmod "github.com/gradient8/launchpad/internal/settings"
	"github.com/gradient8/launchpad/internal/tui/components"
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

func (t *NoticesTab) Edit() error {
	severityOpts := []string{"warning", "info", "error"}

	nodes := []components.TreeNode{
		{
			Label: "Project List",
			OnToggle: func() {
				t.ProjectListEnabled = !t.ProjectListEnabled
			},
			Status: func() string {
				if t.ProjectListEnabled {
					return "enabled"
				}
				return "off"
			},
			Expanded: t.ProjectListEnabled,
			Fields: []components.TreeField{
				{Label: "Severity", Type: components.FieldSelect, SelectValue: &t.ProjectListSeverity, SelectOpts: severityOpts},
				{Label: "Dismissible", Type: components.FieldToggle, BoolValue: &t.ProjectListDismissible},
				{Label: "Title", Type: components.FieldText, TextValue: &t.ProjectListTitle},
				{Label: "Message", Type: components.FieldText, TextValue: &t.ProjectListMessage},
				{Label: "Link URL", Type: components.FieldText, TextValue: &t.ProjectListLinkURL},
				{Label: "Link Text", Type: components.FieldText, TextValue: &t.ProjectListLinkText},
			},
		},
		{
			Label: "Coding Agent",
			OnToggle: func() {
				t.CodingAgentEnabled = !t.CodingAgentEnabled
			},
			Status: func() string {
				if t.CodingAgentEnabled {
					return "enabled"
				}
				return "off"
			},
			Expanded: t.CodingAgentEnabled,
			Fields: []components.TreeField{
				{Label: "Severity", Type: components.FieldSelect, SelectValue: &t.CodingAgentSeverity, SelectOpts: severityOpts},
				{Label: "Dismissible", Type: components.FieldToggle, BoolValue: &t.CodingAgentDismissible},
				{Label: "Title", Type: components.FieldText, TextValue: &t.CodingAgentTitle},
				{Label: "Message", Type: components.FieldText, TextValue: &t.CodingAgentMessage},
				{Label: "Link URL", Type: components.FieldText, TextValue: &t.CodingAgentLinkURL},
				{Label: "Link Text", Type: components.FieldText, TextValue: &t.CodingAgentLinkText},
			},
		},
		{
			Label: "Billing",
			OnToggle: func() {
				t.BillingEnabled = !t.BillingEnabled
			},
			Status: func() string {
				if t.BillingEnabled {
					return "enabled"
				}
				return "off"
			},
			Expanded: t.BillingEnabled,
			Fields: []components.TreeField{
				{Label: "Severity", Type: components.FieldSelect, SelectValue: &t.BillingSeverity, SelectOpts: severityOpts},
				{Label: "Dismissible", Type: components.FieldToggle, BoolValue: &t.BillingDismissible},
				{Label: "Title", Type: components.FieldText, TextValue: &t.BillingTitle},
				{Label: "Message", Type: components.FieldText, TextValue: &t.BillingMessage},
				{Label: "Link URL", Type: components.FieldText, TextValue: &t.BillingLinkURL},
				{Label: "Link Text", Type: components.FieldText, TextValue: &t.BillingLinkText},
			},
		},
		{
			Label: "Credit Drawer",
			OnToggle: func() {
				t.CreditDrawerEnabled = !t.CreditDrawerEnabled
			},
			Status: func() string {
				if t.CreditDrawerEnabled {
					return "enabled"
				}
				return "off"
			},
			Expanded: t.CreditDrawerEnabled,
			Fields: []components.TreeField{
				{Label: "Severity", Type: components.FieldSelect, SelectValue: &t.CreditDrawerSeverity, SelectOpts: severityOpts},
				{Label: "Dismissible", Type: components.FieldToggle, BoolValue: &t.CreditDrawerDismissible},
				{Label: "Title", Type: components.FieldText, TextValue: &t.CreditDrawerTitle},
				{Label: "Message", Type: components.FieldText, TextValue: &t.CreditDrawerMessage},
				{Label: "Link URL", Type: components.FieldText, TextValue: &t.CreditDrawerLinkURL},
				{Label: "Link Text", Type: components.FieldText, TextValue: &t.CreditDrawerLinkText},
			},
		},
	}

	return components.NewTreeForm("Notices", nodes).Run()
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

package tabs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	settingsmod "github.com/gradient8/launchpad/internal/settings"
)

type OpsReportTab struct {
	Enabled             bool
	AdminEmails         string
	SendTime            string
	BalanceForecastDays string
	TopResourceCount    string
}

func NewOpsReportTab(s *settingsmod.Settings) *OpsReportTab {
	balanceDays := ""
	if s.OpsReport.BalanceForecastDays != 0 {
		balanceDays = strconv.Itoa(s.OpsReport.BalanceForecastDays)
	}
	topCount := ""
	if s.OpsReport.TopResourceCount != 0 {
		topCount = strconv.Itoa(s.OpsReport.TopResourceCount)
	}
	return &OpsReportTab{
		Enabled:             s.OpsReport.Enabled,
		AdminEmails:         strings.Join(s.OpsReport.AdminEmails, ","),
		SendTime:            s.OpsReport.SendTime,
		BalanceForecastDays: balanceDays,
		TopResourceCount:    topCount,
	}
}

func (t *OpsReportTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().WithButtonAlignment(lipgloss.Left).
				Title("Enable Ops Report").
				Description("Send periodic operations reports to admins").
				Value(&t.Enabled),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Admin Emails").
				Description("Comma-separated list of admin email addresses").
				Value(&t.AdminEmails),
			huh.NewInput().
				Title("Send Time").
				Description("Time to send the report (e.g. 08:00)").
				Value(&t.SendTime),
			huh.NewInput().
				Title("Balance Forecast Days").
				Description("Number of days to forecast balance (integer)").
				Value(&t.BalanceForecastDays),
			huh.NewInput().
				Title("Top Resource Count").
				Description("Number of top resources to include (integer)").
				Value(&t.TopResourceCount),
		).WithHideFunc(func() bool { return !t.Enabled }),
	)
}

func (t *OpsReportTab) Edit() error { return RunFormAsEdit(t.Form()) }

func (t *OpsReportTab) View() string {
	if !t.Enabled {
		return "  Status: disabled"
	}
	return fmt.Sprintf(
		"  Status:               enabled\n  Admin Emails:         %s\n  Send Time:            %s\n  Balance Forecast Days: %s\n  Top Resource Count:   %s",
		truncate(displayValue(t.AdminEmails), 40),
		displayValue(t.SendTime),
		displayValue(t.BalanceForecastDays),
		displayValue(t.TopResourceCount),
	)
}

func (t *OpsReportTab) Apply(s *settingsmod.Settings) {
	s.OpsReport.Enabled = t.Enabled
	s.OpsReport.SendTime = t.SendTime

	if t.AdminEmails == "" {
		s.OpsReport.AdminEmails = nil
	} else {
		parts := strings.Split(t.AdminEmails, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		s.OpsReport.AdminEmails = result
	}

	if n, err := strconv.Atoi(t.BalanceForecastDays); err == nil {
		s.OpsReport.BalanceForecastDays = n
	} else {
		s.OpsReport.BalanceForecastDays = 0
	}

	if n, err := strconv.Atoi(t.TopResourceCount); err == nil {
		s.OpsReport.TopResourceCount = n
	} else {
		s.OpsReport.TopResourceCount = 0
	}
}

package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type DatabaseTab struct {
	Mode         string
	DBMain       string
	DBMonitoring string
	DBEvents     string
	DBBilling    string
	DBStats      string
	DBGateway    string
}

func NewDatabaseTab(cfg *config.Config) *DatabaseTab {
	t := &DatabaseTab{
		Mode: cfg.Database.Mode,
	}
	if cfg.Database.URLs != nil {
		t.DBMain = cfg.Database.URLs["main"]
		t.DBMonitoring = cfg.Database.URLs["monitoring"]
		t.DBEvents = cfg.Database.URLs["events"]
		t.DBBilling = cfg.Database.URLs["billing"]
		t.DBStats = cfg.Database.URLs["stats"]
		t.DBGateway = cfg.Database.URLs["gateway"]
	}
	return t
}

func (t *DatabaseTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Database Mode").
				Options(
					huh.NewOption("Built-in (managed)", "builtin"),
					huh.NewOption("External", "external"),
				).
				Value(&t.Mode),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Main DB URL").
				Placeholder("postgres://user:pass@host:5432/main").
				Value(&t.DBMain),
			huh.NewInput().
				Title("Monitoring DB URL").
				Placeholder("postgres://user:pass@host:5432/monitoring").
				Value(&t.DBMonitoring),
			huh.NewInput().
				Title("Events DB URL").
				Placeholder("postgres://user:pass@host:5432/events").
				Value(&t.DBEvents),
			huh.NewInput().
				Title("Billing DB URL").
				Placeholder("postgres://user:pass@host:5432/billing").
				Value(&t.DBBilling),
			huh.NewInput().
				Title("Stats DB URL").
				Placeholder("postgres://user:pass@host:5432/stats").
				Value(&t.DBStats),
			huh.NewInput().
				Title("Gateway DB URL").
				Placeholder("postgres://user:pass@host:5432/gateway").
				Value(&t.DBGateway),
		).WithHideFunc(func() bool { return t.Mode != "external" }),
	)
}

func (t *DatabaseTab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Database Mode").
				Options(
					huh.NewOption("Built-in (managed)", "builtin"),
					huh.NewOption("External", "external"),
				).
				Value(&t.Mode),
		).Title("Database"),
		huh.NewGroup(
			huh.NewInput().
				Title("Main DB URL").
				Placeholder("postgres://user:pass@host:5432/main").
				Value(&t.DBMain),
			huh.NewInput().
				Title("Monitoring DB URL").
				Placeholder("postgres://user:pass@host:5432/monitoring").
				Value(&t.DBMonitoring),
			huh.NewInput().
				Title("Events DB URL").
				Placeholder("postgres://user:pass@host:5432/events").
				Value(&t.DBEvents),
			huh.NewInput().
				Title("Billing DB URL").
				Placeholder("postgres://user:pass@host:5432/billing").
				Value(&t.DBBilling),
			huh.NewInput().
				Title("Stats DB URL").
				Placeholder("postgres://user:pass@host:5432/stats").
				Value(&t.DBStats),
			huh.NewInput().
				Title("Gateway DB URL").
				Placeholder("postgres://user:pass@host:5432/gateway").
				Value(&t.DBGateway),
		).Title("Database — External URLs").WithHideFunc(func() bool { return t.Mode != "external" }),
	}
}

func (t *DatabaseTab) View() string {
	s := fmt.Sprintf("  Mode: %s", t.Mode)
	if t.Mode == "external" {
		s += fmt.Sprintf(
			"\n  Main:       %s\n  Monitoring: %s\n  Events:     %s\n  Billing:    %s\n  Stats:      %s\n  Gateway:    %s",
			displayValue(t.DBMain), displayValue(t.DBMonitoring),
			displayValue(t.DBEvents), displayValue(t.DBBilling),
			displayValue(t.DBStats), displayValue(t.DBGateway))
	}
	return s
}

func (t *DatabaseTab) Apply(cfg *config.Config) {
	cfg.Database.Mode = t.Mode
	if t.Mode == "external" {
		if cfg.Database.URLs == nil {
			cfg.Database.URLs = make(map[string]string)
		}
		cfg.Database.URLs["main"] = t.DBMain
		cfg.Database.URLs["monitoring"] = t.DBMonitoring
		cfg.Database.URLs["events"] = t.DBEvents
		cfg.Database.URLs["billing"] = t.DBBilling
		cfg.Database.URLs["stats"] = t.DBStats
		cfg.Database.URLs["gateway"] = t.DBGateway
	}
}

func (t *DatabaseTab) Edit() error {
	return RunFormAsEdit(t.Form())
}

package tabs

import (
	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type DatabaseTab struct {
	Mode       string
	ExternalDB string
}

func NewDatabaseTab(cfg *config.Config) *DatabaseTab {
	externalDB := ""
	if cfg.Database.URLs != nil {
		externalDB = cfg.Database.URLs["default"]
	}
	return &DatabaseTab{
		Mode:       cfg.Database.Mode,
		ExternalDB: externalDB,
	}
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
			huh.NewInput().
				Title("External DB URL").
				Description("postgres://user:pass@host:5432/db (for external mode)").
				Value(&t.ExternalDB),
		),
	)
}

func (t *DatabaseTab) Apply(cfg *config.Config) {
	cfg.Database.Mode = t.Mode
	if t.Mode == "external" && t.ExternalDB != "" {
		if cfg.Database.URLs == nil {
			cfg.Database.URLs = make(map[string]string)
		}
		cfg.Database.URLs["default"] = t.ExternalDB
	}
}

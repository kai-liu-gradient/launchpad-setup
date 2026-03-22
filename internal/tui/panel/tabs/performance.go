package tabs

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type PerformanceTab struct {
	APIReplicas    string
	RouterReplicas string
	DBConnLimit    string
}

func NewPerformanceTab(cfg *config.Config) *PerformanceTab {
	apiReplicas := "1"
	if cfg.Performance.APIReplicas > 0 {
		apiReplicas = strconv.Itoa(cfg.Performance.APIReplicas)
	}
	routerReplicas := "1"
	if cfg.Performance.RouterReplicas > 0 {
		routerReplicas = strconv.Itoa(cfg.Performance.RouterReplicas)
	}
	connLimit := "100"
	if cfg.Performance.DBConnLimit > 0 {
		connLimit = strconv.Itoa(cfg.Performance.DBConnLimit)
	}
	return &PerformanceTab{
		APIReplicas:    apiReplicas,
		RouterReplicas: routerReplicas,
		DBConnLimit:    connLimit,
	}
}

func (t *PerformanceTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("API Replicas").
				Placeholder("1").
				Value(&t.APIReplicas),
			huh.NewInput().
				Title("Router Replicas").
				Placeholder("1").
				Value(&t.RouterReplicas),
			huh.NewInput().
				Title("DB Connection Limit").
				Placeholder("100").
				Value(&t.DBConnLimit),
		),
	)
}

func (t *PerformanceTab) View() string {
	return fmt.Sprintf("  API Replicas:    %s\n  Router Replicas: %s\n  DB Conn Limit:   %s",
		displayValue(t.APIReplicas), displayValue(t.RouterReplicas), displayValue(t.DBConnLimit))
}

func (t *PerformanceTab) Apply(cfg *config.Config) {
	if r, err := strconv.Atoi(t.APIReplicas); err == nil && r > 0 {
		cfg.Performance.APIReplicas = r
	}
	if r, err := strconv.Atoi(t.RouterReplicas); err == nil && r > 0 {
		cfg.Performance.RouterReplicas = r
	}
	if c, err := strconv.Atoi(t.DBConnLimit); err == nil && c > 0 {
		cfg.Performance.DBConnLimit = c
	}
}

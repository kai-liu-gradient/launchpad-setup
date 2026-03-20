package tabs

import (
	"fmt"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/tui/components"
)

type OverviewTab struct {
	cfg *config.Config
}

func NewOverviewTab(cfg *config.Config) *OverviewTab {
	return &OverviewTab{cfg: cfg}
}

func (t *OverviewTab) View() string {
	cfg := t.cfg

	smtpStatus := "disabled"
	if cfg.SMTP.Host != "" {
		smtpStatus = "enabled"
	}

	ssoStatus := "disabled"
	if cfg.SSO.Enabled {
		ssoStatus = "enabled"
	}

	return fmt.Sprintf(`
  %s

  Domain:       %s
  Subdomain:    %s
  Admin Email:  %s

  SSL Mode:     %s
  DB Mode:      %s
  K8s Mode:     %s
  Storage Mode: %s
  SMTP:         %s
  SSO:          %s
`,
		components.TitleStyle.Render("Deployment Overview"),
		cfg.Domain,
		cfg.Subdomain,
		cfg.AdminEmail,
		cfg.SSL.Mode,
		cfg.Database.Mode,
		cfg.Kubernetes.Mode,
		cfg.Storage.Mode,
		smtpStatus,
		ssoStatus,
	)
}

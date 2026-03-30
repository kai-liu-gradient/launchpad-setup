package engine

import (
	"fmt"
	"strings"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
)

// DeployResult holds information to display after deployment completes.
type DeployResult struct {
	DashboardURL string
	GiteaURL     string
	AdminURL     string
	AdminEmail   string
	AdminPass    string
	K8sMode      string
	Registered   bool
	HostIP       string
	Domain       string
	Domains      []string
}

// BuildResult creates a DeployResult from config and secrets.
func BuildResult(cfg *config.Config, sec *secrets.Secrets, registered bool) *DeployResult {
	launchpadDomain := cfg.Subdomain + "." + cfg.Domain
	giteaDomain := cfg.Subdomain + "-gitea." + cfg.Domain

	return &DeployResult{
		DashboardURL: "https://" + launchpadDomain,
		GiteaURL:     "https://" + giteaDomain,
		AdminURL:     "https://" + launchpadDomain + "/admin",
		AdminEmail:   cfg.AdminEmail,
		AdminPass:    sec.AdminPassword,
		K8sMode:      cfg.Kubernetes.Mode,
		Registered:   registered,
		HostIP:       detectHostIP(),
		Domain:       cfg.Domain,
		Domains: []string{
			launchpadDomain,
			giteaDomain,
			"*." + cfg.Domain,
		},
	}
}

// FormatResult returns a formatted string showing deployment results.
func (r *DeployResult) FormatResult() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("═══════════════════════════════════════════════════\n")
	b.WriteString("  ✓ Deployment Complete\n")
	b.WriteString("═══════════════════════════════════════════════════\n")
	b.WriteString("\n")

	b.WriteString("  URLs:\n")
	b.WriteString(fmt.Sprintf("    Dashboard:  %s\n", r.DashboardURL))
	b.WriteString(fmt.Sprintf("    Gitea:      %s\n", r.GiteaURL))
	b.WriteString(fmt.Sprintf("    Admin:      %s\n", r.AdminURL))
	b.WriteString("\n")

	b.WriteString("  ⚠️  Admin Credentials (SAVE THESE!):\n")
	b.WriteString(fmt.Sprintf("    Email:      %s\n", r.AdminEmail))
	b.WriteString(fmt.Sprintf("    Password:   %s\n", r.AdminPass))
	b.WriteString("\n")
	b.WriteString("    ⮕  This password is auto-generated and will NOT be shown again.\n")
	b.WriteString("    ⮕  To retrieve later: cat .secrets.yaml | grep admin_password\n")
	b.WriteString("\n")

	b.WriteString("  Kubernetes:\n")
	b.WriteString(fmt.Sprintf("    Mode:       %s\n", r.K8sMode))
	if r.Registered {
		b.WriteString("    Status:     ✓ Registered\n")
	} else {
		b.WriteString("    Status:     ⚠ Not registered\n")
	}
	b.WriteString("\n")

	b.WriteString("  DNS Setup:\n")
	b.WriteString(fmt.Sprintf("    Point these domains to %s:\n", r.HostIP))
	for i, d := range r.Domains {
		prefix := "├─"
		if i == len(r.Domains)-1 {
			prefix = "└─"
		}
		b.WriteString(fmt.Sprintf("    %s %s\n", prefix, d))
	}
	b.WriteString("\n")
	b.WriteString("═══════════════════════════════════════════════════\n")

	return b.String()
}

package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type SSLTab struct {
	Mode        string
	DNSProvider string
	DNSAPIToken string
	CertPath    string
	KeyPath     string
}

func NewSSLTab(cfg *config.Config) *SSLTab {
	return &SSLTab{
		Mode:        cfg.SSL.Mode,
		DNSProvider: cfg.SSL.DNSProvider,
		DNSAPIToken: cfg.SSL.DNSAPIToken,
		CertPath:    cfg.SSL.CertPath,
		KeyPath:     cfg.SSL.KeyPath,
	}
}

func (t *SSLTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("SSL Mode").
				Options(
					huh.NewOption("Self-signed", "selfsigned"),
					huh.NewOption("Let's Encrypt", "letsencrypt"),
					huh.NewOption("Custom Certificate", "custom"),
				).
				Value(&t.Mode),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("DNS Provider").
				Description("e.g. cloudflare").
				Value(&t.DNSProvider),
			huh.NewInput().
				Title("DNS API Token").
				EchoMode(huh.EchoModePassword).
				Value(&t.DNSAPIToken),
		).WithHideFunc(func() bool { return t.Mode != "letsencrypt" }),
		huh.NewGroup(
			huh.NewInput().
				Title("Certificate Path").
				Description("Path to fullchain.pem").
				Value(&t.CertPath),
			huh.NewInput().
				Title("Key Path").
				Description("Path to privkey.pem").
				Value(&t.KeyPath),
		).WithHideFunc(func() bool { return t.Mode != "custom" }),
	)
}

func (t *SSLTab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("SSL Mode").
				Options(
					huh.NewOption("Self-signed", "selfsigned"),
					huh.NewOption("Let's Encrypt", "letsencrypt"),
					huh.NewOption("Custom Certificate", "custom"),
				).
				Value(&t.Mode),
		).Title("SSL"),
		huh.NewGroup(
			huh.NewInput().
				Title("DNS Provider").
				Description("e.g. cloudflare").
				Value(&t.DNSProvider),
			huh.NewInput().
				Title("DNS API Token").
				EchoMode(huh.EchoModePassword).
				Value(&t.DNSAPIToken),
		).Title("SSL — Let's Encrypt").WithHideFunc(func() bool { return t.Mode != "letsencrypt" }),
		huh.NewGroup(
			huh.NewInput().
				Title("Certificate Path").
				Description("Path to fullchain.pem").
				Value(&t.CertPath),
			huh.NewInput().
				Title("Key Path").
				Description("Path to privkey.pem").
				Value(&t.KeyPath),
		).Title("SSL — Custom Certificate").WithHideFunc(func() bool { return t.Mode != "custom" }),
	}
}

func (t *SSLTab) View() string {
	s := fmt.Sprintf("  Mode: %s", t.Mode)
	switch t.Mode {
	case "letsencrypt":
		s += fmt.Sprintf("\n  DNS Provider:  %s\n  DNS API Token: %s",
			displayValue(t.DNSProvider), maskValue(t.DNSAPIToken))
	case "custom":
		s += fmt.Sprintf("\n  Cert Path: %s\n  Key Path:  %s",
			displayValue(t.CertPath), displayValue(t.KeyPath))
	}
	return s
}

func (t *SSLTab) Apply(cfg *config.Config) {
	cfg.SSL.Mode = t.Mode
	cfg.SSL.DNSProvider = t.DNSProvider
	cfg.SSL.DNSAPIToken = t.DNSAPIToken
	cfg.SSL.CertPath = t.CertPath
	cfg.SSL.KeyPath = t.KeyPath
}

func (t *SSLTab) Edit() error {
	return RunFormAsEdit(t.Form())
}

package tabs

import (
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
			huh.NewInput().Title("DNS Provider").Value(&t.DNSProvider),
			huh.NewInput().Title("DNS API Token").Value(&t.DNSAPIToken),
			huh.NewInput().Title("Certificate Path").Value(&t.CertPath),
			huh.NewInput().Title("Key Path").Value(&t.KeyPath),
		),
	)
}

func (t *SSLTab) Apply(cfg *config.Config) {
	cfg.SSL.Mode = t.Mode
	cfg.SSL.DNSProvider = t.DNSProvider
	cfg.SSL.DNSAPIToken = t.DNSAPIToken
	cfg.SSL.CertPath = t.CertPath
	cfg.SSL.KeyPath = t.KeyPath
}

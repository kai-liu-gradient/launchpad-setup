package tabs

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type SMTPTab struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func NewSMTPTab(cfg *config.Config) *SMTPTab {
	port := ""
	if cfg.SMTP.Port != 0 {
		port = strconv.Itoa(cfg.SMTP.Port)
	}
	return &SMTPTab{
		Host:     cfg.SMTP.Host,
		Port:     port,
		User:     cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	}
}

func (t *SMTPTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("SMTP Host").Value(&t.Host),
			huh.NewInput().Title("SMTP Port").Value(&t.Port),
			huh.NewInput().Title("SMTP User").Value(&t.User),
			huh.NewInput().
				Title("SMTP Password").
				EchoMode(huh.EchoModePassword).
				Value(&t.Password),
			huh.NewInput().Title("From Address").Value(&t.From),
		),
	)
}

func (t *SMTPTab) Groups() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewInput().Title("SMTP Host").Value(&t.Host),
			huh.NewInput().Title("SMTP Port").Value(&t.Port),
			huh.NewInput().Title("SMTP User").Value(&t.User),
			huh.NewInput().
				Title("SMTP Password").
				EchoMode(huh.EchoModePassword).
				Value(&t.Password),
			huh.NewInput().Title("From Address").Value(&t.From),
		).Title("SMTP"),
	}
}

func (t *SMTPTab) View() string {
	return fmt.Sprintf(
		"  Host:     %s\n  Port:     %s\n  User:     %s\n  Password: %s\n  From:     %s",
		displayValue(t.Host), displayValue(t.Port), displayValue(t.User), maskValue(t.Password), displayValue(t.From))
}

func (t *SMTPTab) Apply(cfg *config.Config) {
	cfg.SMTP.Host = t.Host
	if p, err := strconv.Atoi(t.Port); err == nil {
		cfg.SMTP.Port = p
	}
	cfg.SMTP.User = t.User
	cfg.SMTP.Password = t.Password
	cfg.SMTP.From = t.From
}

func (t *SMTPTab) Edit() error {
	return RunFormAsEdit(t.Form())
}

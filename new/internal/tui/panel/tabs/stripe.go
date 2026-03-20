package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type StripeTab struct {
	SecretKey      string
	WebhookSecret  string
	PublishableKey string
}

func NewStripeTab(cfg *config.Config) *StripeTab {
	return &StripeTab{
		SecretKey:      cfg.Stripe.SecretKey,
		WebhookSecret:  cfg.Stripe.WebhookSecret,
		PublishableKey: cfg.Stripe.PublishableKey,
	}
}

func (t *StripeTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Stripe Secret Key").
				EchoMode(huh.EchoModePassword).
				Value(&t.SecretKey),
			huh.NewInput().
				Title("Stripe Webhook Secret").
				EchoMode(huh.EchoModePassword).
				Value(&t.WebhookSecret),
			huh.NewInput().
				Title("Stripe Publishable Key").
				Value(&t.PublishableKey),
		),
	)
}

func (t *StripeTab) View() string {
	return fmt.Sprintf("  Secret Key:      %s\n  Webhook Secret:  %s\n  Publishable Key: %s",
		maskValue(t.SecretKey), maskValue(t.WebhookSecret), displayValue(t.PublishableKey))
}

func (t *StripeTab) Apply(cfg *config.Config) {
	cfg.Stripe.SecretKey = t.SecretKey
	cfg.Stripe.WebhookSecret = t.WebhookSecret
	cfg.Stripe.PublishableKey = t.PublishableKey
}

package wizard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// Express form field values — readable after form completion.
var (
	expressDomain  string
	expressEmail   string
	expressConfirm bool
)

// NewExpressForm creates a huh.Form for the express setup path.
// It asks only for domain, email, and a final confirmation.
func NewExpressForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Domain name").
				Placeholder("example.com").
				Value(&expressDomain).
				Validate(domainValidator),

			huh.NewInput().
				Title("Admin email").
				Placeholder("admin@example.com").
				Value(&expressEmail).
				Validate(emailValidator),

			huh.NewConfirm().
				Title("Deploy with defaults?").
				Description("Self-signed SSL, built-in K3s + PostgreSQL + Redis").
				Value(&expressConfirm),
		),
	)
}

// domainValidator ensures the value is non-empty and resembles a valid domain.
func domainValidator(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("domain name is required")
	}
	if !strings.Contains(s, ".") {
		return fmt.Errorf("domain must contain at least one dot (e.g. example.com)")
	}
	if strings.HasPrefix(s, ".") || strings.HasSuffix(s, ".") {
		return fmt.Errorf("domain must not start or end with a dot")
	}
	return nil
}

// emailValidator ensures the value is non-empty and contains an @.
func emailValidator(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("admin email is required")
	}
	if !strings.Contains(s, "@") {
		return fmt.Errorf("email must contain @")
	}
	return nil
}

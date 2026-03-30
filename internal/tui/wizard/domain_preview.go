package wizard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// DomainPreviewResult holds the user's confirmed domain settings.
type DomainPreviewResult struct {
	Subdomain      string
	GiteaSubdomain string
	Confirmed      bool
	GoBack         bool
}

// RunDomainPreview shows the generated domains and lets the user confirm or edit.
// It returns the (possibly modified) subdomain settings.
func RunDomainPreview(domain, subdomain, projectDomain, adminEmail string) (*DomainPreviewResult, error) {
	giteaSub := subdomain + "-gitea"
	wildcardDomain := projectDomain
	if wildcardDomain == "" {
		wildcardDomain = domain
	}

	for {
		// Show preview and ask user to confirm, edit subdomains, or go back
		var choice string
		displayEmail := adminEmail
		if displayEmail == "" {
			displayEmail = "admin@" + domain
		}
		previewDesc := fmt.Sprintf(
			"  Launchpad:       %s.%s\n  Gitea:           %s.%s\n  Project Domain:  \\*.%s\n  Admin:           %s",
			subdomain, domain,
			giteaSub, domain,
			wildcardDomain,
			displayEmail,
		)

		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("Domain Preview").
					Description(previewDesc),
				huh.NewSelect[string]().
					Title("").
					Options(
						huh.NewOption("Confirm", "confirm"),
						huh.NewOption("Edit subdomains", "edit"),
						huh.NewOption("Back (change domain)", "back"),
					).
					Value(&choice),
			),
		).WithProgramOptions(tea.WithAltScreen())

		if err := confirmForm.Run(); err != nil {
			return nil, fmt.Errorf("domain preview: %w", err)
		}

		switch choice {
		case "confirm":
			result := &DomainPreviewResult{
				Subdomain:      subdomain,
				GiteaSubdomain: giteaSub,
				Confirmed:      true,
			}
			// Only set GiteaSubdomain if it differs from the default pattern
			if giteaSub == subdomain+"-gitea" {
				result.GiteaSubdomain = ""
			}
			return result, nil

		case "back":
			return &DomainPreviewResult{GoBack: true}, nil

		case "edit":
			editForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Launchpad subdomain").
						Value(&subdomain).
						Validate(subdomainValidator),
					huh.NewInput().
						Title("Gitea subdomain").
						Value(&giteaSub).
						Validate(subdomainValidator),
				),
			).WithProgramOptions(tea.WithAltScreen())

			if err := editForm.Run(); err != nil {
				return nil, fmt.Errorf("domain edit: %w", err)
			}
			// Loop back to preview with updated values.
		}
	}
}

// subdomainValidator checks that a subdomain label is non-empty and valid.
func subdomainValidator(s string) error {
	if s == "" {
		return fmt.Errorf("subdomain is required")
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
			return fmt.Errorf("subdomain may only contain letters, digits, and hyphens")
		}
	}
	if s[0] == '-' || s[len(s)-1] == '-' {
		return fmt.Errorf("subdomain must not start or end with a hyphen")
	}
	return nil
}

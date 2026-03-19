package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validate checks cfg for structural correctness. It uses struct-tag
// validation (required, fqdn, email) plus custom business rules.
func Validate(cfg *Config) error {
	v := validator.New()

	// Struct-tag validation (domain, email, etc.)
	if err := v.Struct(cfg); err != nil {
		return formatValidationErrors(err)
	}

	// --- custom business rules ---

	// SSL mode
	switch cfg.SSL.Mode {
	case "selfsigned", "letsencrypt", "custom":
		// ok
	default:
		return fmt.Errorf("ssl.mode %q is not valid; must be one of: selfsigned, letsencrypt, custom", cfg.SSL.Mode)
	}

	// Letsencrypt requires dns_provider
	if cfg.SSL.Mode == "letsencrypt" && cfg.SSL.DNSProvider == "" {
		return fmt.Errorf("ssl.dns_provider is required when ssl.mode is letsencrypt")
	}

	// Database mode
	switch cfg.Database.Mode {
	case "builtin", "external":
		// ok
	default:
		return fmt.Errorf("database.mode %q is not valid; must be one of: builtin, external", cfg.Database.Mode)
	}

	// External DB requires URLs
	if cfg.Database.Mode == "external" && len(cfg.Database.URLs) == 0 {
		return fmt.Errorf("database.urls must not be empty when database.mode is external")
	}

	// Kubernetes mode
	switch cfg.Kubernetes.Mode {
	case "builtin", "external":
		// ok
	default:
		return fmt.Errorf("kubernetes.mode %q is not valid; must be one of: builtin, external", cfg.Kubernetes.Mode)
	}

	// External K8s requires kubeconfig
	if cfg.Kubernetes.Mode == "external" && cfg.Kubernetes.Kubeconfig == "" {
		return fmt.Errorf("kubernetes.kubeconfig is required when kubernetes.mode is external")
	}

	return nil
}

// formatValidationErrors turns go-playground/validator errors into a
// human-readable message.
func formatValidationErrors(err error) error {
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}
	var msgs []string
	for _, fe := range ve {
		msgs = append(msgs, fmt.Sprintf("%s: failed on %q", fe.Namespace(), fe.Tag()))
	}
	return fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
}

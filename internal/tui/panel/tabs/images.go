package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type ImagesTab struct {
	Registry       string
	APIVersion     string
	UIVersion      string
	RouterVersion  string
	GatewayVersion string
	GiteaVersion   string
	API            string
	UI             string
	Router         string
	Gateway        string
	Gitea          string
}

func NewImagesTab(cfg *config.Config) *ImagesTab {
	registry := cfg.Images.Registry
	if registry == "" {
		registry = config.DefaultImageRegistry
	}
	return &ImagesTab{
		Registry:       registry,
		APIVersion:     cfg.Images.APIVersion,
		UIVersion:      cfg.Images.UIVersion,
		RouterVersion:  cfg.Images.RouterVersion,
		GatewayVersion: cfg.Images.GatewayVersion,
		GiteaVersion:   cfg.Images.GiteaVersion,
		API:            cfg.Images.API,
		UI:             cfg.Images.UI,
		Router:         cfg.Images.Router,
		Gateway:        cfg.Images.Gateway,
		Gitea:          cfg.Images.Gitea,
	}
}

func (t *ImagesTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Image Registry").
				Placeholder(config.DefaultImageRegistry).
				Value(&t.Registry),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("API Version").
				Placeholder(config.DefaultAPIVersion+" (default)").
				Value(&t.APIVersion),
			huh.NewInput().
				Title("UI Version").
				Placeholder(config.DefaultUIVersion+" (default)").
				Value(&t.UIVersion),
			huh.NewInput().
				Title("Router Version").
				Placeholder(config.DefaultRouterVersion+" (default)").
				Value(&t.RouterVersion),
			huh.NewInput().
				Title("Gateway Version").
				Placeholder(config.DefaultGatewayVersion+" (default)").
				Value(&t.GatewayVersion),
			huh.NewInput().
				Title("Gitea Version").
				Placeholder(config.DefaultGiteaVersion+" (default)").
				Value(&t.GiteaVersion),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("API Override").
				Description("Full image ref. Overrides registry+version if set.").
				Value(&t.API),
			huh.NewInput().
				Title("UI Override").
				Description("Full image ref. Overrides registry+version if set.").
				Value(&t.UI),
			huh.NewInput().
				Title("Router Override").
				Description("Full image ref. Overrides registry+version if set.").
				Value(&t.Router),
			huh.NewInput().
				Title("Gateway Override").
				Description("Full image ref. Overrides registry+version if set.").
				Value(&t.Gateway),
			huh.NewInput().
				Title("Gitea Override").
				Description("Full image ref. Overrides registry+version if set.").
				Value(&t.Gitea),
		),
	)
}

func (t *ImagesTab) View() string {
	versionOrDefault := func(v, def string) string {
		if v != "" {
			return v
		}
		return def + " (default)"
	}

	s := fmt.Sprintf("  Registry:  %s\n\n", displayValue(t.Registry))
	s += fmt.Sprintf("  API:       %s\n", versionOrDefault(t.APIVersion, config.DefaultAPIVersion))
	s += fmt.Sprintf("  UI:        %s\n", versionOrDefault(t.UIVersion, config.DefaultUIVersion))
	s += fmt.Sprintf("  Router:    %s\n", versionOrDefault(t.RouterVersion, config.DefaultRouterVersion))
	s += fmt.Sprintf("  Gateway:   %s\n", versionOrDefault(t.GatewayVersion, config.DefaultGatewayVersion))
	s += fmt.Sprintf("  Gitea:     %s\n", versionOrDefault(t.GiteaVersion, config.DefaultGiteaVersion))

	// Show overrides only if any are set
	hasOverride := t.API != "" || t.UI != "" || t.Router != "" || t.Gateway != "" || t.Gitea != ""
	if hasOverride {
		s += "\n  Overrides:\n"
		if t.API != "" {
			s += fmt.Sprintf("    API:     %s\n", t.API)
		}
		if t.UI != "" {
			s += fmt.Sprintf("    UI:      %s\n", t.UI)
		}
		if t.Router != "" {
			s += fmt.Sprintf("    Router:  %s\n", t.Router)
		}
		if t.Gateway != "" {
			s += fmt.Sprintf("    Gateway: %s\n", t.Gateway)
		}
		if t.Gitea != "" {
			s += fmt.Sprintf("    Gitea:   %s\n", t.Gitea)
		}
	}

	return s
}

func (t *ImagesTab) Apply(cfg *config.Config) {
	cfg.Images.Registry = t.Registry
	cfg.Images.APIVersion = t.APIVersion
	cfg.Images.UIVersion = t.UIVersion
	cfg.Images.RouterVersion = t.RouterVersion
	cfg.Images.GatewayVersion = t.GatewayVersion
	cfg.Images.GiteaVersion = t.GiteaVersion
	cfg.Images.API = t.API
	cfg.Images.UI = t.UI
	cfg.Images.Router = t.Router
	cfg.Images.Gateway = t.Gateway
	cfg.Images.Gitea = t.Gitea
}

func (t *ImagesTab) Edit() error {
	return RunFormAsEdit(t.Form())
}

// Package template computes derived values used when rendering deployment
// configuration templates.
package template

import (
	"fmt"
	"strings"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/secrets"
)

// DerivedValues holds all computed values that are derived from a Config and
// Secrets pair and are needed when rendering deployment templates.
type DerivedValues struct {
	// URLs
	DashboardURL     string // https://{subdomain}.{domain}
	GiteaURL         string // https://{subdomain}-gitea.{domain}
	APIURL           string // https://{subdomain}.{domain}:6804
	ExternalDomain   string // {domain} (the base domain, NOT subdomain.domain)
	PublicURL        string // https://{subdomain}.{domain}
	CORSOrigins      string // https://{subdomain}.{domain}
	InternalAPIURL   string // https://{subdomain}.{domain}/api
	InternalAdminURL string // https://{subdomain}.{domain}/admin/adminapi

	// Database
	DBConnStrings map[string]string // per-schema connection strings
	GiteaDBHost   string            // parsed from Gitea DB URL
	GiteaDBPort   string
	GiteaDBName   string
	GiteaDBUser   string

	// Images
	ImageRefs map[string]string // per-service full image references

	// Network
	HostIP         string // dynamically detected host IP (placeholder 127.0.0.1)
	DefaultBackend string // computed for builtin K8s mode
	NodeTLSReject  string // "0" if selfsigned, "" otherwise

	// Paths / derived strings
	CACertPath        string
	RouterLocalURL    string // http://router:6580
	GiteaGitSSH       string // git@{subdomain}-gitea.{domain}:2222
	EntraRedirectURI  string // https://{subdomain}.{domain}/api/v1/auth/microsoft/callback
	ANICodeReleaseURL string // https://{subdomain}-gitea.{domain}/launchpad/ani-code/archive/main.tar.gz
	DomainEscaped     string // domain with dots escaped for regex (e.g. example\.com)
	KubeconfigPath    string // resolved kubeconfig path for docker-compose volume mount
}

// ComputeDerived derives all template values from the given Config and Secrets.
func ComputeDerived(cfg *config.Config, sec *secrets.Secrets) *DerivedValues {
	launchpadDomain := cfg.Subdomain + "." + cfg.Domain
	giteaDomain := cfg.Subdomain + "-gitea." + cfg.Domain

	d := &DerivedValues{}

	// URLs
	d.DashboardURL = "https://" + launchpadDomain
	d.GiteaURL = "https://" + giteaDomain
	d.APIURL = "https://" + launchpadDomain + ":6804"
	d.ExternalDomain = cfg.Domain
	d.PublicURL = "https://" + launchpadDomain
	d.CORSOrigins = "https://" + launchpadDomain
	d.InternalAPIURL = "https://" + launchpadDomain + "/api"
	d.InternalAdminURL = "https://" + launchpadDomain + "/admin/adminapi"

	// Database connection strings
	d.DBConnStrings = computeDBConnStrings(cfg, sec)

	// Gitea DB fields — populated for builtin mode using the builtin host/port
	d.GiteaDBHost = "postgres"
	d.GiteaDBPort = "5432"
	d.GiteaDBName = "gitea"
	d.GiteaDBUser = "gitea"

	// Image refs
	d.ImageRefs = computeImageRefs(cfg)

	// Network
	d.HostIP = "127.0.0.1"
	if cfg.Kubernetes.Mode == "builtin" {
		d.DefaultBackend = d.HostIP + ":30080"
	} else {
		d.DefaultBackend = "localhost:30080"
	}

	// TLS rejection flag
	if cfg.SSL.Mode == "selfsigned" {
		d.NodeTLSReject = "0"
	} else {
		d.NodeTLSReject = ""
	}

	// Paths and derived strings
	d.CACertPath = "/usr/local/share/ca-certificates/ani-selfsigned-ca.crt"
	d.RouterLocalURL = "http://router:6580"
	d.GiteaGitSSH = "git@" + giteaDomain + ":2222"
	d.EntraRedirectURI = "https://" + launchpadDomain + "/api/v1/auth/microsoft/callback"
	d.ANICodeReleaseURL = "https://" + giteaDomain + "/launchpad/ani-code/archive/main.tar.gz"
	d.DomainEscaped = strings.ReplaceAll(cfg.Domain, ".", "\\.")

	// Kubeconfig path
	if cfg.Kubernetes.Mode == "builtin" {
		d.KubeconfigPath = "/etc/rancher/k3s/k3s.yaml"
	} else if cfg.Kubernetes.Kubeconfig != "" {
		d.KubeconfigPath = cfg.Kubernetes.Kubeconfig
	} else {
		d.KubeconfigPath = "/etc/rancher/k3s/k3s.yaml"
	}

	return d
}

// computeDBConnStrings builds per-schema connection strings.
// For builtin mode each schema gets a dedicated user and password from Secrets.
// For external mode the URLs are taken directly from cfg.Database.URLs.
func computeDBConnStrings(cfg *config.Config, sec *secrets.Secrets) map[string]string {
	if cfg.Database.Mode != "builtin" {
		// Deep-copy the external URLs map so callers cannot mutate it.
		m := make(map[string]string, len(cfg.Database.URLs))
		for k, v := range cfg.Database.URLs {
			m[k] = v
		}
		return m
	}

	type schemaInfo struct {
		name     string
		password string
	}

	schemas := []schemaInfo{
		{"main", sec.DBPasswordMain},
		{"monitoring", sec.DBPasswordMonitoring},
		{"events", sec.DBPasswordEvents},
		{"billing", sec.DBPasswordBilling},
		{"stats", sec.DBPasswordStats},
		{"gateway", sec.DBPasswordGateway},
	}

	m := make(map[string]string, len(schemas))
	for _, s := range schemas {
		m[s.name] = fmt.Sprintf(
			"postgresql://launchpad_%suser:%s@postgres:5432/launchpad?schema=launchpad_%s",
			s.name, s.password, s.name,
		)
	}
	return m
}

// computeImageRefs builds per-service full image references.
// If an explicit override is set in cfg.Images the override is used as-is;
// otherwise the reference is composed as {registry}/{service-name}:{version}.
func computeImageRefs(cfg *config.Config) map[string]string {
	registry := cfg.Images.Registry
	version := cfg.Images.DefaultVersion

	ref := func(override, serviceName string) string {
		if override != "" {
			return override
		}
		return registry + "/" + serviceName + ":" + version
	}

	return map[string]string{
		"api":     ref(cfg.Images.API, "launchpad-api"),
		"ui":      ref(cfg.Images.UI, "launchpad-ui"),
		"router":  ref(cfg.Images.Router, "launchpad-router"),
		"gateway": ref(cfg.Images.Gateway, "launchpad-gateway"),
		"gitea":   ref(cfg.Images.Gitea, "launchpad-gitea"),
	}
}

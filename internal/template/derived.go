// Package template computes derived values used when rendering deployment
// configuration templates.
package template

import (
	"fmt"
	"net"
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
	GatewayPublicURL string // https://{subdomain}.{domain}/gatewayproxy
	GatewayURL       string // same as GatewayPublicURL (used as ANI_CODE_GATEWAY_URL)

	// Domains (convenience fields for templates)
	LaunchpadDomain string // {subdomain}.{domain}
	GiteaDomain     string // {subdomain}-gitea.{domain}
	IngressDomain   string // {domain} (same as ExternalDomain but semantic name)

	// Database
	DBConnStrings map[string]string // per-schema connection strings
	GiteaDBHost   string            // parsed from Gitea DB URL
	GiteaDBPort   string
	GiteaDBName   string
	GiteaDBUser   string

	// Images
	ImageRefs map[string]string // per-service full image references

	// Network
	HostIP           string // dynamically detected host IP (placeholder 127.0.0.1)
	DefaultBackend   string // computed for builtin K8s mode
	NodeTLSReject    string // "0" if selfsigned, "" otherwise
	IngressClusterIP string // ingress-nginx ClusterIP for CoreDNS (placeholder 10.43.0.0)

	// Kubernetes
	StorageClass string // storage class name for PVCs

	// Paths / derived strings
	CACertPath        string
	RouterLocalURL    string // http://router:6580
	GiteaGitSSH       string // git@{subdomain}-gitea.{domain}:2222
	EntraRedirectURI  string // https://{subdomain}.{domain}/api/v1/auth/microsoft/callback
	ANICodeReleaseURL string // https://{subdomain}-gitea.{domain}/launchpad/ani-code/archive/main.tar.gz
	DomainEscaped     string   // domain with dots escaped for regex (e.g. example\.com)
	AllowedDomains    []string // registration whitelist domains (Config.Domain + admin email domain)
	KubeconfigPath    string   // resolved kubeconfig path for docker-compose volume mount
}

// ComputeDerived derives all template values from the given Config and Secrets.
func ComputeDerived(cfg *config.Config, sec *secrets.Secrets) *DerivedValues {
	launchpadDomain := cfg.Subdomain + "." + cfg.Domain
	giteaDomain := cfg.Subdomain + "-gitea." + cfg.Domain
	if cfg.GiteaSubdomain != "" {
		giteaDomain = cfg.GiteaSubdomain + "." + cfg.Domain
	}

	d := &DerivedValues{}

	// Domains (convenience fields)
	d.LaunchpadDomain = launchpadDomain
	d.GiteaDomain = giteaDomain
	d.IngressDomain = cfg.ResolvedProjectDomain()

	// URLs
	d.DashboardURL = "https://" + launchpadDomain
	d.GiteaURL = "https://" + giteaDomain
	d.APIURL = "https://" + launchpadDomain + ":6804"
	d.ExternalDomain = cfg.ResolvedProjectDomain()
	d.PublicURL = "https://" + launchpadDomain
	d.CORSOrigins = "https://" + launchpadDomain
	d.InternalAPIURL = "https://" + launchpadDomain + "/api"
	d.InternalAdminURL = "https://" + launchpadDomain + "/admin/adminapi"
	d.GatewayPublicURL = "https://" + launchpadDomain + "/gatewayproxy"
	d.GatewayURL = d.GatewayPublicURL

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
	d.HostIP = detectHostIP()
	if cfg.Kubernetes.Mode == "builtin" {
		d.DefaultBackend = d.HostIP + ":30080"
	} else {
		d.DefaultBackend = "localhost:30080"
	}
	d.IngressClusterIP = "10.43.0.0" // placeholder, populated at deploy time

	// Kubernetes
	if cfg.Kubernetes.StorageClass != "" {
		d.StorageClass = cfg.Kubernetes.StorageClass
	} else {
		d.StorageClass = "local-path"
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
	d.DomainEscaped = strings.ReplaceAll(cfg.ResolvedProjectDomain(), ".", "\\.")

	// Registration allowed domains: Config.Domain + admin email domain (deduplicated)
	d.AllowedDomains = []string{cfg.Domain}
	if parts := strings.SplitN(cfg.AdminEmail, "@", 2); len(parts) == 2 {
		adminDomain := parts[1]
		if adminDomain != cfg.Domain {
			d.AllowedDomains = append(d.AllowedDomains, adminDomain)
		}
	}

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

// detectHostIP returns the first non-loopback IPv4 address, or "127.0.0.1" if none found.
func detectHostIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}
	return "127.0.0.1"
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
// Priority: full override > registry+versionField > registry+compiledDefault.
func computeImageRefs(cfg *config.Config) map[string]string {
	registry := cfg.Images.Registry

	ref := func(override, serviceName, versionField, defaultVersion string) string {
		if override != "" {
			return override
		}
		v := versionField
		if v == "" {
			v = defaultVersion
		}
		return registry + "/" + serviceName + ":" + v
	}

	m := map[string]string{
		"api":     ref(cfg.Images.API, "launchpad-api", cfg.Images.APIVersion, config.DefaultAPIVersion),
		"ui":      ref(cfg.Images.UI, "launchpad-ui", cfg.Images.UIVersion, config.DefaultUIVersion),
		"router":  ref(cfg.Images.Router, "launchpad-router", cfg.Images.RouterVersion, config.DefaultRouterVersion),
		"gateway": ref(cfg.Images.Gateway, "ani-code-gateway", cfg.Images.GatewayVersion, config.DefaultGatewayVersion),
	}

	// Gitea uses Docker Hub (gitea/gitea), not the private registry.
	if cfg.Images.Gitea != "" {
		m["gitea"] = cfg.Images.Gitea
	} else {
		gv := cfg.Images.GiteaVersion
		if gv == "" {
			gv = config.DefaultGiteaVersion
		}
		m["gitea"] = "gitea/gitea:" + gv
	}

	return m
}

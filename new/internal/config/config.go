// Package config defines the Config struct and default values for
// AniLaunchpad deployments.
package config

// Config holds all user-configurable options for an AniLaunchpad deployment.
type Config struct {
	Domain      string         `yaml:"domain" validate:"required,fqdn"`
	Subdomain   string         `yaml:"subdomain"`
	AdminEmail  string         `yaml:"admin_email" validate:"required,email"`
	Images      ImageConfig    `yaml:"images"`
	SSL         SSLConfig      `yaml:"ssl"`
	Database    DBConfig       `yaml:"database"`
	Kubernetes  K8sConfig      `yaml:"kubernetes"`
	SMTP        SMTPConfig     `yaml:"smtp,omitempty"`
	SSO         SSOConfig      `yaml:"sso,omitempty"`
	Storage     StorageConfig  `yaml:"storage"`
	Telegram    TelegramConfig `yaml:"telegram,omitempty"`
	AI          AIConfig       `yaml:"ai,omitempty"`
	Stripe      StripeConfig   `yaml:"stripe,omitempty"`
	Performance PerfConfig     `yaml:"performance"`
}

// ImageConfig specifies container image registry and per-service overrides.
type ImageConfig struct {
	Registry       string `yaml:"registry"`
	DefaultVersion string `yaml:"default_version"`
	API            string `yaml:"api,omitempty"`
	UI             string `yaml:"ui,omitempty"`
	Router         string `yaml:"router,omitempty"`
	Gateway        string `yaml:"gateway,omitempty"`
	Gitea          string `yaml:"gitea,omitempty"`
}

// SSLConfig controls TLS certificate provisioning.
type SSLConfig struct {
	Mode             string `yaml:"mode"`
	DNSProvider      string `yaml:"dns_provider,omitempty"`
	DNSAPIToken      string `yaml:"dns_api_token,omitempty"`
	CertPath         string `yaml:"cert_path,omitempty"`
	KeyPath          string `yaml:"key_path,omitempty"`
	WildcardCertPath string `yaml:"wildcard_cert_path,omitempty"`
	WildcardKeyPath  string `yaml:"wildcard_key_path,omitempty"`
}

// DBConfig controls database provisioning.
type DBConfig struct {
	Mode string            `yaml:"mode"`
	URLs map[string]string `yaml:"urls,omitempty"`
}

// K8sConfig controls Kubernetes cluster provisioning.
type K8sConfig struct {
	Mode           string `yaml:"mode"`
	Kubeconfig     string `yaml:"kubeconfig,omitempty"`
	Context        string `yaml:"context,omitempty"`
	StorageClass   string `yaml:"storage_class,omitempty"`
	DefaultBackend string `yaml:"default_backend,omitempty"`
}

// SMTPConfig holds outbound email settings.
type SMTPConfig struct {
	Host     string `yaml:"host,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	User     string `yaml:"user,omitempty"`
	Password string `yaml:"password,omitempty"`
	From     string `yaml:"from,omitempty"`
}

// SSOConfig holds Microsoft Entra ID (Azure AD) SSO settings.
type SSOConfig struct {
	EntraTenantID string `yaml:"entra_tenant_id,omitempty"`
	EntraClientID string `yaml:"entra_client_id,omitempty"`
	EntraSecret   string `yaml:"entra_secret,omitempty"`
}

// StorageConfig controls object/file storage backend.
type StorageConfig struct {
	Mode           string `yaml:"mode"`
	S3Bucket       string `yaml:"s3_bucket,omitempty"`
	S3Region       string `yaml:"s3_region,omitempty"`
	S3Key          string `yaml:"s3_key,omitempty"`
	S3Secret       string `yaml:"s3_secret,omitempty"`
	AzureConn      string `yaml:"azure_connection_string,omitempty"`
	AzureContainer string `yaml:"azure_container,omitempty"`
}

// TelegramConfig holds Telegram bot notification settings.
type TelegramConfig struct {
	BotToken    string `yaml:"bot_token,omitempty"`
	BotUsername string `yaml:"bot_username,omitempty"`
	ChatID      string `yaml:"chat_id,omitempty"`
}

// AIConfig holds AI service endpoint settings.
type AIConfig struct {
	CRS2Endpoint string `yaml:"crs2_endpoint,omitempty"`
	CRS2Token    string `yaml:"crs2_token,omitempty"`
}

// StripeConfig holds Stripe payment integration settings.
type StripeConfig struct {
	SecretKey      string `yaml:"secret_key,omitempty"`
	WebhookSecret  string `yaml:"webhook_secret,omitempty"`
	PublishableKey string `yaml:"publishable_key,omitempty"`
}

// PerfConfig holds performance tuning parameters.
type PerfConfig struct {
	APIReplicas int `yaml:"api_replicas"`
	DBConnLimit int `yaml:"db_conn_limit"`
}

// DefaultConfig returns a Config populated with sensible defaults for the
// given domain and admin email.
func DefaultConfig(domain, adminEmail string) *Config {
	return &Config{
		Domain:     domain,
		Subdomain:  "launchpad",
		AdminEmail: adminEmail,
		Images: ImageConfig{
			Registry:       "swr.ap-southeast-1.myhuaweicloud.com/ghisha",
			DefaultVersion: "2.0.3",
		},
		SSL: SSLConfig{
			Mode: "selfsigned",
		},
		Database: DBConfig{
			Mode: "builtin",
		},
		Kubernetes: K8sConfig{
			Mode: "builtin",
		},
		Storage: StorageConfig{
			Mode: "local",
		},
		Performance: PerfConfig{
			APIReplicas: 1,
			DBConnLimit: 100,
		},
	}
}

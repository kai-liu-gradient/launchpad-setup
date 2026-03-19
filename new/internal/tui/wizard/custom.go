package wizard

import (
	"github.com/charmbracelet/huh"
)

// Custom form field values — readable after form completion.
var (
	// Group 1 — Basic
	customDomain    string
	customSubdomain = "launchpad"
	customEmail     string
	customRegistry  = "swr.ap-southeast-1.myhuaweicloud.com/ghisha"

	// Group 2 — SSL
	customSSLMode     = "selfsigned"
	customDNSProvider string
	customDNSToken    string
	customCertPath    string
	customKeyPath     string

	// Group 3 — Database
	customDBMode    = "builtin"
	customDBMain    string
	customDBMonitor string
	customDBEvents  string
	customDBBilling string
	customDBStats   string
	customDBGateway string

	// Group 4 — Kubernetes
	customK8sMode    = "builtin"
	customKubeconfig string
	customK8sContext string

	// Group 5 — Advanced: SMTP
	customSMTPHost     string
	customSMTPPort     string
	customSMTPUser     string
	customSMTPPassword string
	customSMTPFrom     string

	// Group 5 — Advanced: SSO
	customEntraTenant string
	customEntraClient string
	customEntraSecret string

	// Group 5 — Advanced: Storage
	customStorageMode    = "local"
	customS3Bucket       string
	customS3Region       string
	customS3Key          string
	customS3Secret       string
	customAzureConn      string
	customAzureContainer string

	// Group 5 — Advanced: Telegram
	customTelegramBotToken string
	customTelegramUsername string
	customTelegramChatID   string

	// Group 5 — Advanced: AI
	customCRS2Endpoint string
	customCRS2Token    string

	// Group 5 — Advanced: Stripe
	customStripeSecret  string
	customStripeWebhook string
	customStripePublish string

	// Group 5 — Advanced: Performance
	customAPIReplicas = "1"
	customDBConnLimit = "100"

	// Group 6 — Review
	customConfirm bool
)

// NewCustomForm creates a huh.Form with groups covering the full configuration.
// Conditional sections are placed in separate groups with WithHideFunc so huh
// can skip them when they are not applicable.
func NewCustomForm() *huh.Form {
	return huh.NewForm(
		// Group 1: Basic
		huh.NewGroup(
			huh.NewInput().
				Title("Domain name").
				Placeholder("example.com").
				Value(&customDomain).
				Validate(domainValidator),

			huh.NewInput().
				Title("Subdomain").
				Placeholder("launchpad").
				Value(&customSubdomain),

			huh.NewInput().
				Title("Admin email").
				Placeholder("admin@example.com").
				Value(&customEmail).
				Validate(emailValidator),

			huh.NewInput().
				Title("Image registry").
				Placeholder("swr.ap-southeast-1.myhuaweicloud.com/ghisha").
				Value(&customRegistry),
		),

		// Group 2a: SSL mode selection
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("SSL mode").
				Options(
					huh.NewOption("Self-signed", "selfsigned"),
					huh.NewOption("Let's Encrypt (DNS-01)", "letsencrypt"),
					huh.NewOption("Custom certificates", "custom"),
				).
				Value(&customSSLMode),
		),

		// Group 2b: Let's Encrypt fields (conditional)
		huh.NewGroup(
			huh.NewInput().
				Title("DNS provider").
				Placeholder("cloudflare").
				Value(&customDNSProvider),

			huh.NewInput().
				Title("DNS API token").
				Value(&customDNSToken),
		).WithHideFunc(func() bool { return customSSLMode != "letsencrypt" }),

		// Group 2c: Custom certificate paths (conditional)
		huh.NewGroup(
			huh.NewInput().
				Title("Certificate path").
				Placeholder("/etc/ssl/cert.pem").
				Value(&customCertPath),

			huh.NewInput().
				Title("Key path").
				Placeholder("/etc/ssl/key.pem").
				Value(&customKeyPath),
		).WithHideFunc(func() bool { return customSSLMode != "custom" }),

		// Group 3a: Database mode selection
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Database mode").
				Options(
					huh.NewOption("Built-in PostgreSQL", "builtin"),
					huh.NewOption("External PostgreSQL", "external"),
				).
				Value(&customDBMode),
		),

		// Group 3b: External database connection URLs (conditional)
		huh.NewGroup(
			huh.NewInput().
				Title("Main DB URL").
				Placeholder("postgres://user:pass@host:5432/main").
				Value(&customDBMain),

			huh.NewInput().
				Title("Monitoring DB URL").
				Placeholder("postgres://user:pass@host:5432/monitoring").
				Value(&customDBMonitor),

			huh.NewInput().
				Title("Events DB URL").
				Placeholder("postgres://user:pass@host:5432/events").
				Value(&customDBEvents),

			huh.NewInput().
				Title("Billing DB URL").
				Placeholder("postgres://user:pass@host:5432/billing").
				Value(&customDBBilling),

			huh.NewInput().
				Title("Stats DB URL").
				Placeholder("postgres://user:pass@host:5432/stats").
				Value(&customDBStats),

			huh.NewInput().
				Title("Gateway DB URL").
				Placeholder("postgres://user:pass@host:5432/gateway").
				Value(&customDBGateway),
		).WithHideFunc(func() bool { return customDBMode != "external" }),

		// Group 4a: Kubernetes mode selection
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Kubernetes mode").
				Options(
					huh.NewOption("Built-in K3s", "builtin"),
					huh.NewOption("External cluster", "external"),
				).
				Value(&customK8sMode),
		),

		// Group 4b: External cluster fields (conditional)
		huh.NewGroup(
			huh.NewInput().
				Title("Kubeconfig path").
				Placeholder("~/.kube/config").
				Value(&customKubeconfig),

			huh.NewInput().
				Title("Kube context").
				Placeholder("my-cluster").
				Value(&customK8sContext),
		).WithHideFunc(func() bool { return customK8sMode != "external" }),

		// Group 5: Advanced — SMTP
		huh.NewGroup(
			huh.NewInput().
				Title("SMTP host").
				Placeholder("smtp.example.com").
				Value(&customSMTPHost),

			huh.NewInput().
				Title("SMTP port").
				Placeholder("587").
				Value(&customSMTPPort),

			huh.NewInput().
				Title("SMTP user").
				Value(&customSMTPUser),

			huh.NewInput().
				Title("SMTP password").
				Value(&customSMTPPassword),

			huh.NewInput().
				Title("SMTP from address").
				Placeholder("noreply@example.com").
				Value(&customSMTPFrom),
		),

		// Group 5: Advanced — SSO
		huh.NewGroup(
			huh.NewInput().
				Title("Entra tenant ID").
				Value(&customEntraTenant),

			huh.NewInput().
				Title("Entra client ID").
				Value(&customEntraClient),

			huh.NewInput().
				Title("Entra client secret").
				Value(&customEntraSecret),
		),

		// Group 5: Advanced — Storage mode
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Storage backend").
				Options(
					huh.NewOption("Local disk", "local"),
					huh.NewOption("Amazon S3 / compatible", "s3"),
					huh.NewOption("Azure Blob Storage", "azure"),
				).
				Value(&customStorageMode),
		),

		// Group 5: Advanced — S3 sub-fields (conditional)
		huh.NewGroup(
			huh.NewInput().
				Title("S3 bucket").
				Value(&customS3Bucket),

			huh.NewInput().
				Title("S3 region").
				Value(&customS3Region),

			huh.NewInput().
				Title("S3 access key").
				Value(&customS3Key),

			huh.NewInput().
				Title("S3 secret key").
				Value(&customS3Secret),
		).WithHideFunc(func() bool { return customStorageMode != "s3" }),

		// Group 5: Advanced — Azure sub-fields (conditional)
		huh.NewGroup(
			huh.NewInput().
				Title("Azure connection string").
				Value(&customAzureConn),

			huh.NewInput().
				Title("Azure container name").
				Value(&customAzureContainer),
		).WithHideFunc(func() bool { return customStorageMode != "azure" }),

		// Group 5: Advanced — Telegram
		huh.NewGroup(
			huh.NewInput().
				Title("Telegram bot token").
				Value(&customTelegramBotToken),

			huh.NewInput().
				Title("Telegram bot username").
				Value(&customTelegramUsername),

			huh.NewInput().
				Title("Telegram chat ID").
				Value(&customTelegramChatID),
		),

		// Group 5: Advanced — AI
		huh.NewGroup(
			huh.NewInput().
				Title("CRS2 endpoint").
				Value(&customCRS2Endpoint),

			huh.NewInput().
				Title("CRS2 token").
				Value(&customCRS2Token),
		),

		// Group 5: Advanced — Stripe
		huh.NewGroup(
			huh.NewInput().
				Title("Stripe secret key").
				Value(&customStripeSecret),

			huh.NewInput().
				Title("Stripe webhook secret").
				Value(&customStripeWebhook),

			huh.NewInput().
				Title("Stripe publishable key").
				Value(&customStripePublish),
		),

		// Group 5: Advanced — Performance
		huh.NewGroup(
			huh.NewInput().
				Title("API replicas").
				Placeholder("1").
				Value(&customAPIReplicas),

			huh.NewInput().
				Title("DB connection limit").
				Placeholder("100").
				Value(&customDBConnLimit),
		),

		// Group 6: Review — final confirmation
		huh.NewGroup(
			huh.NewConfirm().
				Title("Proceed with deployment?").
				Description("Review your settings above and confirm to begin installation.").
				Value(&customConfirm),
		),
	)
}

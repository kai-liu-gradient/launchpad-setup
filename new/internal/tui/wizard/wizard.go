package wizard

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/tui/components"
)

// Mode selects which install wizard to present to the user.
type Mode int

const (
	Express Mode = iota
	Custom
)

// Model is the bubbletea model for the install wizard.
type Model struct {
	mode    Mode
	form    *huh.Form
	result  *config.Config
	done    bool
	aborted bool
}

// New creates a wizard Model for the given mode.
func New(mode Mode) Model {
	var form *huh.Form
	if mode == Express {
		form = NewExpressForm()
	} else {
		form = NewCustomForm()
	}
	return Model{
		mode: mode,
		form: form,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle quit
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "ctrl+c" {
			m.aborted = true
			return m, tea.Quit
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		m.done = true
		m.result = m.buildConfig()
		return m, tea.Quit
	}

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	header := components.TitleStyle.Render("AniLaunchpad Setup")
	subtitle := components.SubtitleStyle.Render(m.modeLabel())
	return fmt.Sprintf("%s\n%s\n\n%s", header, subtitle, m.form.View())
}

func (m Model) modeLabel() string {
	if m.mode == Express {
		return "Express Setup — 2 questions to deploy"
	}
	return "Custom Setup — full configuration"
}

// Result returns the completed Config, or nil if the wizard was aborted or not
// yet finished.
func (m Model) Result() *config.Config {
	return m.result
}

// Done reports whether the wizard completed successfully.
func (m Model) Done() bool {
	return m.done
}

// Aborted reports whether the user pressed ctrl+c to cancel.
func (m Model) Aborted() bool {
	return m.aborted
}

// buildConfig reads the collected form values and constructs a *config.Config.
func (m Model) buildConfig() *config.Config {
	if m.mode == Express {
		return config.DefaultConfig(expressDomain, expressEmail)
	}
	return m.buildCustomConfig()
}

// buildCustomConfig assembles a Config from all custom form variables.
func (m Model) buildCustomConfig() *config.Config {
	apiReplicas, _ := strconv.Atoi(customAPIReplicas)
	if apiReplicas <= 0 {
		apiReplicas = 1
	}
	dbConnLimit, _ := strconv.Atoi(customDBConnLimit)
	if dbConnLimit <= 0 {
		dbConnLimit = 100
	}

	registry := customRegistry
	if registry == "" {
		registry = "swr.ap-southeast-1.myhuaweicloud.com/ghisha"
	}
	subdomain := customSubdomain
	if subdomain == "" {
		subdomain = "launchpad"
	}

	cfg := &config.Config{
		Domain:     customDomain,
		Subdomain:  subdomain,
		AdminEmail: customEmail,

		Images: config.ImageConfig{
			Registry: registry,
		},

		SSL: config.SSLConfig{
			Mode:        customSSLMode,
			DNSProvider: customDNSProvider,
			DNSAPIToken: customDNSToken,
			CertPath:    customCertPath,
			KeyPath:     customKeyPath,
		},

		Database: config.DBConfig{
			Mode: customDBMode,
		},

		Kubernetes: config.K8sConfig{
			Mode:       customK8sMode,
			Kubeconfig: customKubeconfig,
			Context:    customK8sContext,
		},

		SMTP: config.SMTPConfig{
			Host:     customSMTPHost,
			User:     customSMTPUser,
			Password: customSMTPPassword,
			From:     customSMTPFrom,
		},

		SSO: config.SSOConfig{
			EntraTenantID: customEntraTenant,
			EntraClientID: customEntraClient,
			EntraSecret:   customEntraSecret,
		},

		Storage: config.StorageConfig{
			Mode:           customStorageMode,
			S3Bucket:       customS3Bucket,
			S3Region:       customS3Region,
			S3Key:          customS3Key,
			S3Secret:       customS3Secret,
			AzureConn:      customAzureConn,
			AzureContainer: customAzureContainer,
		},

		Telegram: config.TelegramConfig{
			BotToken:    customTelegramBotToken,
			BotUsername: customTelegramUsername,
			ChatID:      customTelegramChatID,
		},

		AI: config.AIConfig{
			CRS2Endpoint: customCRS2Endpoint,
			CRS2Token:    customCRS2Token,
		},

		Stripe: config.StripeConfig{
			SecretKey:      customStripeSecret,
			WebhookSecret:  customStripeWebhook,
			PublishableKey: customStripePublish,
		},

		Performance: config.PerfConfig{
			APIReplicas: apiReplicas,
			DBConnLimit: dbConnLimit,
		},
	}

	// Populate external DB URLs map if needed.
	if customDBMode == "external" {
		cfg.Database.URLs = map[string]string{
			"main":       customDBMain,
			"monitoring": customDBMonitor,
			"events":     customDBEvents,
			"billing":    customDBBilling,
			"stats":      customDBStats,
			"gateway":    customDBGateway,
		}
	}

	// Parse SMTP port.
	if customSMTPPort != "" {
		port, err := strconv.Atoi(customSMTPPort)
		if err == nil {
			cfg.SMTP.Port = port
		}
	}

	return cfg
}

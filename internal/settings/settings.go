package settings

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Settings struct {
	Version          int              `yaml:"version"`
	Registration     Registration     `yaml:"registration"`
	Plans            Plans            `yaml:"plans"`
	Credentials      Credentials      `yaml:"credentials"`
	Projects         Projects         `yaml:"projects"`
	Authentication   Authentication   `yaml:"authentication"`
	WelcomeCredits   WelcomeCredits   `yaml:"welcomeCredits"`
	Menu             Menu             `yaml:"menu"`
	YAMLBuilder      YAMLBuilder      `yaml:"yamlBuilder"`
	AIMarketplace    AIMarketplace    `yaml:"aiMarketplace"`
	OpsReport        OpsReport        `yaml:"operationsReport"`
	EmailSuppression EmailSuppression `yaml:"emailSuppression"`
	Notices          Notices          `yaml:"notices"`
}

type Registration struct {
	RestrictDomain           bool     `yaml:"restrictDomain"`
	AllowedDomains           []string `yaml:"allowedDomains"`
	RequireEmailVerification bool     `yaml:"requireEmailVerification"`
	Notice                   string   `yaml:"notice"`
	BlockedNotice            string   `yaml:"blockedNotice"`
}

type Plans struct {
	EnforceFreePlan bool     `yaml:"enforceFreePlan"`
	Disabled        []string `yaml:"disabled"`
	DisabledNotice  string   `yaml:"disabledNotice"`
}

type Credentials struct {
	ClaudeIncluded CredentialToggle `yaml:"claudeIncluded"`
	ZAIIncluded    CredentialToggle `yaml:"zaiIncluded"`
	ClaudePaygo    CredentialToggle `yaml:"claudePaygo"`
	ZAIPaygo       CredentialToggle `yaml:"zaiPaygo"`
	GeminiPaygo    CredentialToggle `yaml:"geminiPaygo"`
}

type CredentialToggle struct {
	Disabled bool   `yaml:"disabled"`
	Notice   string `yaml:"notice"`
}

type Projects struct {
	CreateDisabled       bool   `yaml:"createDisabled"`
	CreateDisabledNotice string `yaml:"createDisabledNotice"`
}

type Authentication struct {
	Mode           string        `yaml:"mode"`
	PasswordLogin  PasswordLogin `yaml:"passwordLogin"`
	OAuth          OAuthConfig   `yaml:"oauth"`
	ProviderSignup bool          `yaml:"providerSignup"`
	SSOOnlyNotice  string        `yaml:"ssoOnlyNotice"`
}

type PasswordLogin struct {
	Enabled        bool   `yaml:"enabled"`
	EnableSignup   bool   `yaml:"enableSignup"`
	DisabledNotice string `yaml:"disabledNotice"`
}

type OAuthConfig struct {
	Google    OAuthProvider  `yaml:"google"`
	GitHub    OAuthProvider  `yaml:"github"`
	Microsoft MicrosoftOAuth `yaml:"microsoft"`
}

type OAuthProvider struct {
	Enabled     bool `yaml:"enabled"`
	AllowSignup bool `yaml:"allowSignup"`
}

type MicrosoftOAuth struct {
	Enabled     bool `yaml:"enabled"`
	AllowSignup bool `yaml:"allowSignup"`
	EnforceOnly bool `yaml:"enforceOnly"`
}

type WelcomeCredits struct {
	Enabled       bool   `yaml:"enabled"`
	Amount        int    `yaml:"amount"`
	CampaignStart string `yaml:"campaignStart"`
	CampaignEnd   string `yaml:"campaignEnd"`
	Message       string `yaml:"message"`
}

type Menu struct {
	Docs MenuDocs `yaml:"docs"`
}

type MenuDocs struct {
	Visible bool `yaml:"visible"`
}

type YAMLBuilder struct {
	AllowedUsers string `yaml:"allowedUsers"`
}

type AIMarketplace struct {
	Visible          bool     `yaml:"visible"`
	EnabledProviders []string `yaml:"enabledProviders"`
}

type OpsReport struct {
	Enabled             bool     `yaml:"enabled"`
	AdminEmails         []string `yaml:"adminEmails"`
	SendTime            string   `yaml:"sendTime"`
	BalanceForecastDays int      `yaml:"balanceForecastDays"`
	TopResourceCount    int      `yaml:"topResourceCount"`
}

type EmailSuppression struct {
	SuppressedRecipients []string `yaml:"suppressedRecipients"`
}

type Notice struct {
	Enabled     bool   `yaml:"enabled"`
	Severity    string `yaml:"severity"`
	Dismissible bool   `yaml:"dismissible"`
	Title       string `yaml:"title"`
	Message     string `yaml:"message"`
	LinkURL     string `yaml:"linkUrl"`
	LinkText    string `yaml:"linkText"`
}

type Notices struct {
	ProjectList  Notice `yaml:"projectList"`
	CodingAgent  Notice `yaml:"codingAgent"`
	Billing      Notice `yaml:"billing"`
	CreditDrawer Notice `yaml:"creditDrawer"`
}

func Load(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Settings
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func Save(s *Settings, path string) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

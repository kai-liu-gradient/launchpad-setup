package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (e *Engine) generateCerts(ctx context.Context) error {
	certDir := filepath.Join(e.output, "nginx", "certs")
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("creating cert dir: %w", err)
	}

	switch e.cfg.SSL.Mode {
	case "selfsigned":
		return e.generateSelfsignedCerts(ctx, certDir)
	case "letsencrypt":
		return e.generateLetsEncryptCerts(ctx, certDir)
	case "custom":
		return e.copyCustomCerts(certDir)
	default:
		return fmt.Errorf("unknown SSL mode: %s", e.cfg.SSL.Mode)
	}
}

func (e *Engine) generateSelfsignedCerts(ctx context.Context, certDir string) error {
	hostIP := detectHostIP()
	domain := e.cfg.Domain
	launchpadDomain := e.cfg.Subdomain + "." + domain
	giteaDomain := e.cfg.Subdomain + "-gitea." + domain

	// Generate CA key
	if err := RunWithTimeout(ctx, "ca-key", 30*time.Second,
		"openssl", "genrsa", "-out", filepath.Join(certDir, "ca.key"), "2048"); err != nil {
		return fmt.Errorf("generating CA key: %w", err)
	}

	// Generate CA cert
	if err := RunWithTimeout(ctx, "ca-cert", 30*time.Second,
		"openssl", "req", "-new", "-x509", "-days", "3650",
		"-key", filepath.Join(certDir, "ca.key"),
		"-out", filepath.Join(certDir, "ca.pem"),
		"-subj", "/CN=AniLaunchpad Local CA",
		"-addext", "basicConstraints=critical,CA:TRUE",
		"-addext", "keyUsage=critical,keyCertSign,cRLSign"); err != nil {
		return fmt.Errorf("generating CA cert: %w", err)
	}

	// Generate server key
	if err := RunWithTimeout(ctx, "server-key", 30*time.Second,
		"openssl", "genrsa", "-out", filepath.Join(certDir, "privkey.pem"), "2048"); err != nil {
		return fmt.Errorf("generating server key: %w", err)
	}

	// Generate CSR
	if err := RunWithTimeout(ctx, "server-csr", 30*time.Second,
		"openssl", "req", "-new",
		"-key", filepath.Join(certDir, "privkey.pem"),
		"-out", filepath.Join(certDir, "server.csr"),
		"-subj", fmt.Sprintf("/CN=*.%s", domain)); err != nil {
		return fmt.Errorf("generating CSR: %w", err)
	}

	// Build SAN string
	san := fmt.Sprintf("DNS:*.%s,DNS:%s,DNS:%s,DNS:%s,DNS:localhost,IP:127.0.0.1",
		domain, domain, launchpadDomain, giteaDomain)
	if hostIP != "127.0.0.1" {
		san += ",IP:" + hostIP
	}

	// Sign with CA
	if err := RunWithTimeout(ctx, "sign-cert", 30*time.Second,
		"bash", "-c", fmt.Sprintf(
			`openssl x509 -req -days 3650 -in %s -CA %s -CAkey %s -CAcreateserial -out %s -extfile <(printf "subjectAltName=%s")`,
			filepath.Join(certDir, "server.csr"),
			filepath.Join(certDir, "ca.pem"),
			filepath.Join(certDir, "ca.key"),
			filepath.Join(certDir, "fullchain.pem"),
			san)); err != nil {
		return fmt.Errorf("signing cert: %w", err)
	}

	// Copy as wildcard certs
	for _, pair := range [][2]string{
		{"fullchain.pem", "wildcard-fullchain.pem"},
		{"privkey.pem", "wildcard-privkey.pem"},
	} {
		src, _ := os.ReadFile(filepath.Join(certDir, pair[0]))
		if err := os.WriteFile(filepath.Join(certDir, pair[1]), src, 0644); err != nil {
			return err
		}
	}

	// Cleanup temp files
	os.Remove(filepath.Join(certDir, "server.csr"))
	os.Remove(filepath.Join(certDir, "ca.srl"))

	return nil
}

func (e *Engine) generateLetsEncryptCerts(ctx context.Context, certDir string) error {
	domain := e.cfg.Domain
	launchpadDomain := e.cfg.Subdomain + "." + domain
	giteaDomain := e.cfg.Subdomain + "-gitea." + domain

	// Install acme.sh if not present
	acmePath := filepath.Join(os.Getenv("HOME"), ".acme.sh", "acme.sh")
	if _, err := os.Stat(acmePath); os.IsNotExist(err) {
		if err := RunWithTimeout(ctx, "acme-install", 60*time.Second,
			"bash", "-c", fmt.Sprintf("curl -fsSL https://get.acme.sh | sh -s email=%s", e.cfg.AdminEmail)); err != nil {
			return fmt.Errorf("installing acme.sh: %w", err)
		}
	}

	// Determine DNS flag based on provider
	var dnsEnv, dnsFlag string
	switch e.cfg.SSL.DNSProvider {
	case "cloudflare":
		dnsEnv = fmt.Sprintf("CF_Token=%s", e.cfg.SSL.DNSAPIToken)
		dnsFlag = "--dns dns_cf"
	case "aliyun":
		dnsEnv = fmt.Sprintf("Ali_Secret=%s", e.cfg.SSL.DNSAPIToken)
		dnsFlag = "--dns dns_ali"
	default:
		dnsFlag = "--dns --yes-I-know-dns-manual-mode-enough-go-ahead-please"
	}

	// Issue wildcard cert
	issueCmd := fmt.Sprintf("%s %s --issue %s -d '*.%s' -d '%s' -d '%s' -d '%s' --keylength ec-256",
		dnsEnv, acmePath, dnsFlag, domain, domain, launchpadDomain, giteaDomain)
	// Exit code 2 = already renewed (ok)
	RunWithTimeout(ctx, "acme-issue", 300*time.Second, "bash", "-c", issueCmd)

	// Install cert
	composeFile := filepath.Join(e.output, "docker-compose.yml")
	installCmd := fmt.Sprintf("%s --install-cert -d '*.%s' --key-file %s --fullchain-file %s --reloadcmd 'docker compose -f %s exec nginx nginx -s reload'",
		acmePath, domain,
		filepath.Join(certDir, "privkey.pem"),
		filepath.Join(certDir, "fullchain.pem"),
		composeFile)
	if err := RunWithTimeout(ctx, "acme-install-cert", 60*time.Second, "bash", "-c", installCmd); err != nil {
		return fmt.Errorf("installing cert: %w", err)
	}

	// Copy as wildcard
	for _, pair := range [][2]string{
		{"fullchain.pem", "wildcard-fullchain.pem"},
		{"privkey.pem", "wildcard-privkey.pem"},
	} {
		src, _ := os.ReadFile(filepath.Join(certDir, pair[0]))
		os.WriteFile(filepath.Join(certDir, pair[1]), src, 0644)
	}

	return nil
}

func (e *Engine) copyCustomCerts(certDir string) error {
	copies := [][2]string{
		{e.cfg.SSL.CertPath, "fullchain.pem"},
		{e.cfg.SSL.KeyPath, "privkey.pem"},
	}
	if e.cfg.SSL.WildcardCertPath != "" {
		copies = append(copies,
			[2]string{e.cfg.SSL.WildcardCertPath, "wildcard-fullchain.pem"},
			[2]string{e.cfg.SSL.WildcardKeyPath, "wildcard-privkey.pem"},
		)
	}

	for _, pair := range copies {
		if pair[0] == "" {
			continue
		}
		data, err := os.ReadFile(pair[0])
		if err != nil {
			return fmt.Errorf("reading cert %s: %w", pair[0], err)
		}
		if err := os.WriteFile(filepath.Join(certDir, pair[1]), data, 0644); err != nil {
			return err
		}
	}

	// If no wildcard cert provided, copy main cert as wildcard
	if e.cfg.SSL.WildcardCertPath == "" {
		for _, pair := range [][2]string{
			{"fullchain.pem", "wildcard-fullchain.pem"},
			{"privkey.pem", "wildcard-privkey.pem"},
		} {
			src, _ := os.ReadFile(filepath.Join(certDir, pair[0]))
			os.WriteFile(filepath.Join(certDir, pair[1]), src, 0644)
		}
	}

	return nil
}

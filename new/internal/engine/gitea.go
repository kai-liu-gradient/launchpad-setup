package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tmpl "github.com/gradient8/launchpad/internal/template"
)

func (e *Engine) startGitea(ctx context.Context) error {
	if err := RunWithTimeout(ctx, "start-gitea", 30*time.Second,
		"docker", "compose", "-f", e.output+"/docker-compose.yml",
		"up", "-d", "gitea"); err != nil {
		return fmt.Errorf("starting gitea: %w", err)
	}
	return WaitForDocker(ctx, e.output, "gitea", 90*time.Second)
}

func (e *Engine) bootstrapGitea(ctx context.Context) error {
	// Idempotency: check if launchpad org already exists
	status, err := DockerExec(ctx, e.output, "gitea",
		"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		"http://localhost:3000/api/v1/orgs/launchpad")
	if err == nil && strings.TrimSpace(status) == "200" {
		return nil // already bootstrapped
	}

	// Wait a bit for Gitea to be fully ready
	time.Sleep(5 * time.Second)

	giteaAdmin := strings.Split(e.cfg.AdminEmail, "@")[0]
	giteaPass := e.sec.AdminPassword

	// Create admin user
	DockerExec(ctx, e.output, "gitea", //nolint:errcheck
		"gitea", "admin", "user", "create",
		"--username", giteaAdmin,
		"--password", giteaPass,
		"--email", e.cfg.AdminEmail,
		"--admin", "--must-change-password=false")

	// Generate API token via Gitea API
	tokenJSON, err := DockerExec(ctx, e.output, "gitea",
		"curl", "-s", "-X", "POST",
		fmt.Sprintf("http://localhost:3000/api/v1/users/%s/tokens", giteaAdmin),
		"-u", fmt.Sprintf("%s:%s", giteaAdmin, giteaPass),
		"-H", "Content-Type: application/json",
		"-d", `{"name":"launchpad-api","scopes":["all"]}`)
	if err != nil {
		return fmt.Errorf("generating gitea token: %w", err)
	}

	// Parse token from response (either "sha1" or "token" field)
	var tokenResp map[string]interface{}
	if err := json.Unmarshal([]byte(tokenJSON), &tokenResp); err != nil {
		return fmt.Errorf("parsing token response: %w", err)
	}
	var token string
	if t, ok := tokenResp["token"].(string); ok {
		token = t
	} else if t, ok := tokenResp["sha1"].(string); ok {
		token = t
	}
	if token == "" {
		return fmt.Errorf("could not extract token from Gitea response")
	}

	// Create launchpad organization
	DockerExec(ctx, e.output, "gitea", //nolint:errcheck
		"curl", "-s", "-X", "POST",
		"http://localhost:3000/api/v1/orgs",
		"-H", fmt.Sprintf("Authorization: token %s", token),
		"-H", "Content-Type: application/json",
		"-d", `{"username":"launchpad","full_name":"Launchpad","visibility":"public"}`)

	// Save runtime values
	rv := &tmpl.RuntimeValues{
		GiteaAccessToken: token,
		GiteaUser:        giteaAdmin,
	}
	runtimePath := filepath.Join(e.output, ".runtime.yaml")
	if err := rv.Save(runtimePath); err != nil {
		return fmt.Errorf("saving runtime values: %w", err)
	}

	return nil
}

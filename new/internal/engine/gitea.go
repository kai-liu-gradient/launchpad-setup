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
		"up", "-d", "--force-recreate", "gitea"); err != nil {
		return fmt.Errorf("starting gitea: %w", err)
	}
	return WaitForDocker(ctx, e.output, "gitea", 90*time.Second)
}

func (e *Engine) bootstrapGitea(ctx context.Context) error {
	// Idempotency: check if launchpad org already exists AND we have a valid token
	status, err := DockerExec(ctx, e.output, "gitea",
		"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		"http://localhost:3000/api/v1/orgs/launchpad")
	if err == nil && strings.TrimSpace(status) == "200" {
		// Org exists — try to reuse saved runtime values
		runtimePath := filepath.Join(e.output, ".runtime.yaml")
		rv, _ := tmpl.LoadRuntime(runtimePath)
		if rv.GiteaAccessToken != "" {
			return e.renderWithRuntime(rv)
		}
		// Token missing (e.g. re-install) — fall through to re-generate token
	}

	// Wait a bit for Gitea to be fully ready
	time.Sleep(5 * time.Second)

	giteaAdmin := strings.Split(e.cfg.AdminEmail, "@")[0]
	giteaPass := e.sec.AdminPassword

	// Create admin user (ignore error if already exists)
	DockerExec(ctx, e.output, "gitea", //nolint:errcheck
		"gitea", "admin", "user", "create",
		"--username", giteaAdmin,
		"--password", giteaPass,
		"--email", e.cfg.AdminEmail,
		"--admin", "--must-change-password=false")

	// Ensure password is current (handles re-install where user exists with old password)
	DockerExec(ctx, e.output, "gitea", //nolint:errcheck
		"gitea", "admin", "user", "change-password",
		"--username", giteaAdmin,
		"--password", giteaPass,
		"--must-change-password=false")

	// Delete existing token (idempotent — ignore errors if it doesn't exist)
	DockerExec(ctx, e.output, "gitea", //nolint:errcheck
		"curl", "-s", "-X", "DELETE",
		fmt.Sprintf("http://localhost:3000/api/v1/users/%s/tokens/launchpad-api", giteaAdmin),
		"-u", fmt.Sprintf("%s:%s", giteaAdmin, giteaPass))

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
		return fmt.Errorf("could not extract token from Gitea response: %s", tokenJSON)
	}

	// Create launchpad organization
	DockerExec(ctx, e.output, "gitea", //nolint:errcheck
		"curl", "-s", "-X", "POST",
		"http://localhost:3000/api/v1/orgs",
		"-H", fmt.Sprintf("Authorization: token %s", token),
		"-H", "Content-Type: application/json",
		"-d", `{"username":"launchpad","full_name":"Launchpad","visibility":"public"}`)

	// Save runtime values and re-render templates (preserve existing fields like IngressClusterIP)
	runtimePath := filepath.Join(e.output, ".runtime.yaml")
	rv, _ := tmpl.LoadRuntime(runtimePath)
	rv.GiteaAccessToken = token
	rv.GiteaUser = giteaAdmin
	if err := rv.Save(runtimePath); err != nil {
		return fmt.Errorf("saving runtime values: %w", err)
	}

	return e.renderWithRuntime(rv)
}

// ensureRuntimeRendered loads saved runtime values and re-renders templates.
// Used on re-runs where bootstrap was already done but .env may lack the token.
func (e *Engine) ensureRuntimeRendered() error {
	runtimePath := filepath.Join(e.output, ".runtime.yaml")
	rv, err := tmpl.LoadRuntime(runtimePath)
	if err != nil {
		return fmt.Errorf("loading runtime values: %w", err)
	}
	if rv.GiteaAccessToken == "" {
		return fmt.Errorf("runtime values file exists but GITEA_ACCESS_TOKEN is empty — delete %s and re-run", runtimePath)
	}
	return e.renderWithRuntime(rv)
}

// renderWithRuntime re-renders all templates with the given runtime values.
func (e *Engine) renderWithRuntime(rv *tmpl.RuntimeValues) error {
	derived := tmpl.ComputeDerived(e.cfg, e.sec)
	renderCtx := &tmpl.RenderContext{
		Config:  e.cfg,
		Secrets: e.sec,
		Derived: derived,
		Runtime: rv,
	}
	if err := tmpl.RenderAll(renderCtx, e.output); err != nil {
		return fmt.Errorf("re-rendering templates with runtime values: %w", err)
	}
	return nil
}

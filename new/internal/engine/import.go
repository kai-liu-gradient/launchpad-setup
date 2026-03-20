package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (e *Engine) importTemplates(ctx context.Context) error {
	// Need Gitea token from runtime values
	runtimePath := filepath.Join(e.output, ".runtime.yaml")
	runtimeData, err := os.ReadFile(runtimePath)
	if err != nil {
		e.send(StepEvent{Step: "Importing templates", Status: Running, Detail: "No runtime values, skipping"})
		return nil
	}

	// Parse YAML to get token (avoid importing template package)
	token := extractYAMLValue(string(runtimeData), "gitea_access_token")
	if token == "" {
		e.send(StepEvent{Step: "Importing templates", Status: Running, Detail: "No Gitea token, skipping"})
		return nil
	}

	giteaDomain := e.cfg.Subdomain + "-gitea." + e.cfg.Domain
	sslInsecure := e.cfg.SSL.Mode == "selfsigned"

	// Find files directory (look in project root)
	// The "files" directory contains .tar.gz repos and .yaml templates
	filesDir := filepath.Join(filepath.Dir(e.output), "files")
	if _, err := os.Stat(filesDir); os.IsNotExist(err) {
		// Try parent directory
		filesDir = filepath.Join(filepath.Dir(filepath.Dir(e.output)), "files")
		if _, err := os.Stat(filesDir); os.IsNotExist(err) {
			e.send(StepEvent{Step: "Importing templates", Status: Running, Detail: "No files directory"})
			return nil
		}
	}

	// Phase 1: Push git repos from tar.gz files
	tarFiles, _ := filepath.Glob(filepath.Join(filesDir, "*.tar.gz"))
	for _, tarFile := range tarFiles {
		repoName := strings.TrimSuffix(filepath.Base(tarFile), "-main.tar.gz")
		e.send(StepEvent{Step: "Importing templates", Status: Running, Detail: "Repo: " + repoName})

		// Check if repo exists and has commits
		repoCheck, _ := DockerExec(ctx, e.output, "gitea",
			"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
			"-H", fmt.Sprintf("Authorization: token %s", token),
			fmt.Sprintf("http://localhost:3000/api/v1/repos/launchpad/%s", repoName))
		if strings.TrimSpace(repoCheck) == "200" {
			// Check if has commits
			commitsOut, _ := DockerExec(ctx, e.output, "gitea",
				"curl", "-s",
				"-H", fmt.Sprintf("Authorization: token %s", token),
				fmt.Sprintf("http://localhost:3000/api/v1/repos/launchpad/%s/commits?limit=1", repoName))
			if strings.Contains(commitsOut, `"sha"`) {
				continue // already imported
			}
		} else {
			// Create empty repo
			DockerExec(ctx, e.output, "gitea",
				"curl", "-s", "-X", "POST",
				"http://localhost:3000/api/v1/orgs/launchpad/repos",
				"-H", fmt.Sprintf("Authorization: token %s", token),
				"-H", "Content-Type: application/json",
				"-d", fmt.Sprintf(`{"name":"%s","auto_init":false,"default_branch":"main"}`, repoName))
		}

		// Extract and push with retry
		tmpDir, _ := os.MkdirTemp("", "launchpad-import-*")
		RunWithTimeout(ctx, "extract-"+repoName, 30*time.Second,
			"tar", "xzf", tarFile, "-C", tmpDir, "--strip-components=1")

		remoteURL := fmt.Sprintf("https://admin:%s@%s/launchpad/%s.git",
			token, giteaDomain, repoName)

		var pushOK bool
		for attempt := 0; attempt < 3; attempt++ {
			var pushCmd string
			if sslInsecure {
				pushCmd = fmt.Sprintf(
					"cd %s && git init -b main && git add -A && "+
						"GIT_AUTHOR_NAME=Launchpad GIT_AUTHOR_EMAIL=launchpad@%s "+
						"GIT_COMMITTER_NAME=Launchpad GIT_COMMITTER_EMAIL=launchpad@%s "+
						"git commit -m 'Initial import' && "+
						"GIT_SSL_NO_VERIFY=1 git push -f %s main",
					tmpDir, e.cfg.Domain, e.cfg.Domain, remoteURL)
			} else {
				pushCmd = fmt.Sprintf(
					"cd %s && git init -b main && git add -A && "+
						"GIT_AUTHOR_NAME=Launchpad GIT_AUTHOR_EMAIL=launchpad@%s "+
						"GIT_COMMITTER_NAME=Launchpad GIT_COMMITTER_EMAIL=launchpad@%s "+
						"git commit -m 'Initial import' && "+
						"git push -f %s main",
					tmpDir, e.cfg.Domain, e.cfg.Domain, remoteURL)
			}
			if err := RunWithTimeout(ctx, "push-"+repoName, 60*time.Second, "bash", "-c", pushCmd); err == nil {
				pushOK = true
				break
			}
			time.Sleep(3 * time.Second)
		}
		os.RemoveAll(tmpDir)

		if !pushOK {
			e.send(StepEvent{Step: "Importing templates", Status: Running,
				Detail: fmt.Sprintf("Warning: push failed for %s", repoName)})
		}
	}

	// Phase 2: Import YAML templates via admin API
	yamlFiles, _ := filepath.Glob(filepath.Join(filesDir, "*.yaml"))
	for _, yamlFile := range yamlFiles {
		templateName := strings.TrimSuffix(filepath.Base(yamlFile), ".yaml")
		e.send(StepEvent{Step: "Importing templates", Status: Running, Detail: "Template: " + templateName})

		yamlContent, err := os.ReadFile(yamlFile)
		if err != nil {
			continue
		}

		// Substitute $GITLAB_DOMAIN
		content := strings.ReplaceAll(string(yamlContent),
			"$GITLAB_DOMAIN", "https://"+giteaDomain)

		// Escape for JSON
		jsonYAML, _ := json.Marshal(content)

		// Validate via admin API
		validatePayload := fmt.Sprintf(`{"yaml": %s}`, string(jsonYAML))
		DockerExec(ctx, e.output, "api",
			"wget", "-q", "-O-",
			"--post-data="+validatePayload,
			"--header=Content-Type: application/json",
			"http://localhost:6804/api/templates/yaml/validate")

		// Import as official template
		importPayload := fmt.Sprintf(`{"yaml": %s, "isOfficial": true}`, string(jsonYAML))
		DockerExec(ctx, e.output, "api",
			"wget", "-q", "-O-",
			"--post-data="+importPayload,
			"--header=Content-Type: application/json",
			"http://localhost:6804/api/templates/yaml/create")
	}

	return nil
}

// extractYAMLValue is a simple helper to extract a value from YAML without importing yaml package.
func extractYAMLValue(yamlStr, key string) string {
	for _, line := range strings.Split(yamlStr, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+":") {
			val := strings.TrimPrefix(line, key+":")
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			return val
		}
	}
	return ""
}

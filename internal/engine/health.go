package engine

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// WaitForHealth polls url until it returns HTTP 200 or the timeout expires.
func WaitForHealth(ctx context.Context, service, url string, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	client := &http.Client{Timeout: 5 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("health check for %s timed out after %s", service, timeout)
		case <-ticker.C:
			resp, err := client.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}

// WaitForDocker polls docker compose ps until the named service is healthy.
func WaitForDocker(ctx context.Context, composeDir, service string, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("docker health check for %s timed out after %s", service, timeout)
		case <-ticker.C:
			cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			out, err := runDockerComposePsHealth(cmdCtx, composeDir, service)
			cancel()
			if err == nil {
				trimmed := strings.TrimSpace(out)
				if trimmed == "healthy" {
					return nil
				}
			}
		}
	}
}

// runDockerComposePsHealth runs docker compose ps --format {{.Health}} for the
// given service and returns the output.
func runDockerComposePsHealth(ctx context.Context, composeDir, service string) (string, error) {
	cmd := exec.CommandContext(ctx,
		"docker", "compose",
		"-f", composeDir+"/docker-compose.yml",
		"ps", "--format", "{{.Health}}", service,
	)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

package engine

import (
	"context"
	"fmt"
	"time"
)

func (e *Engine) startAppServices(ctx context.Context) error {
	return RunWithTimeout(ctx, "start-services", 60*time.Second,
		"docker", "compose", "-f", e.output+"/docker-compose.yml",
		"up", "-d", "--force-recreate", "api", "ui", "router", "cron", "backup-worker", "gateway")
}

func (e *Engine) waitForAPIHealth(ctx context.Context) error {
	return WaitForDocker(ctx, e.output, "api", 90*time.Second)
}

func (e *Engine) waitForUIHealth(ctx context.Context) error {
	return WaitForDocker(ctx, e.output, "ui", 60*time.Second)
}

func (e *Engine) waitForRouterHealth(ctx context.Context) error {
	return WaitForDocker(ctx, e.output, "router", 60*time.Second)
}

func (e *Engine) startNginx(ctx context.Context) error {
	if err := RunWithTimeout(ctx, "start-nginx", 30*time.Second,
		"docker", "compose", "-f", e.output+"/docker-compose.yml",
		"up", "-d", "--force-recreate", "nginx"); err != nil {
		return fmt.Errorf("starting nginx: %w", err)
	}

	if err := WaitForDocker(ctx, e.output, "nginx", 30*time.Second); err != nil {
		return err
	}

	// Give nginx time to fully initialize HTTPS proxy
	time.Sleep(5 * time.Second)

	return nil
}

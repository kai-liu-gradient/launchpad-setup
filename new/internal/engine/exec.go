// Package engine provides execution helpers and pre-flight checks for
// AniLaunchpad deployments.
package engine

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// RunWithTimeout runs cmd with args under a child context bounded by timeout.
// On failure it returns an error that includes name and the combined output.
func RunWithTimeout(ctx context.Context, name string, timeout time.Duration, cmd string, args ...string) error {
	_, err := RunWithOutput(ctx, name, timeout, cmd, args...)
	return err
}

// RunWithOutput runs cmd with args under a child context bounded by timeout and
// returns the captured stdout. On failure it returns an error that includes name
// and the combined output.
func RunWithOutput(ctx context.Context, name string, timeout time.Duration, cmd string, args ...string) (string, error) {
	child, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	c := exec.CommandContext(child, cmd, args...)
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf

	if err := c.Run(); err != nil {
		return "", fmt.Errorf("%s: %w\n%s", name, err, buf.String())
	}
	return buf.String(), nil
}

// DockerExec runs `docker compose exec -T {service} {cmd...}` inside composeDir
// with a 60-second timeout and returns combined stdout.
func DockerExec(ctx context.Context, composeDir, service string, cmd ...string) (string, error) {
	args := append([]string{
		"compose", "-f", composeDir + "/docker-compose.yml",
		"exec", "-T", service,
	}, cmd...)

	child, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	c := exec.CommandContext(child, "docker", args...)
	c.Dir = composeDir
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf

	if err := c.Run(); err != nil {
		return "", fmt.Errorf("docker exec %s: %w\n%s", service, err, buf.String())
	}
	return buf.String(), nil
}

// DockerRun runs `docker compose run --rm {service} {cmd...}` inside composeDir
// with a 120-second timeout and returns combined stdout.
func DockerRun(ctx context.Context, composeDir, service string, cmd ...string) (string, error) {
	args := append([]string{
		"compose", "-f", composeDir + "/docker-compose.yml",
		"run", "--rm", service,
	}, cmd...)

	child, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	c := exec.CommandContext(child, "docker", args...)
	c.Dir = composeDir
	var buf bytes.Buffer
	c.Stdout = &buf
	c.Stderr = &buf

	if err := c.Run(); err != nil {
		return "", fmt.Errorf("docker run %s: %w\n%s", service, err, buf.String())
	}
	return buf.String(), nil
}

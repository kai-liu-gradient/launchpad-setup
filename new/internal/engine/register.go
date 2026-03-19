package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (e *Engine) registerCluster(ctx context.Context) error {
	launchpadDomain := e.cfg.Subdomain + "." + e.cfg.Domain

	// Download register.sh script
	registerScript := "/tmp/launchpad-register.sh"

	// Try public URL first
	downloaded := false
	curlArgs := []string{"-fsSL", "-k",
		fmt.Sprintf("https://%s/admin/adminapi/k3s/register.sh", launchpadDomain),
		"-o", registerScript}
	if err := RunWithTimeout(ctx, "download-register", 15*time.Second,
		"curl", curlArgs...); err == nil {
		downloaded = true
	}

	// Fallback: via Docker network
	if !downloaded {
		out, err := DockerExec(ctx, e.output, "api",
			"curl", "-fsSL", "http://localhost:6802/admin/adminapi/k3s/register.sh")
		if err != nil {
			return fmt.Errorf("failed to download register.sh: %w", err)
		}
		if err := os.WriteFile(registerScript, []byte(out), 0755); err != nil {
			return err
		}
	}

	// Validate script
	data, _ := os.ReadFile(registerScript)
	if len(data) == 0 || !hasShebang(string(data)) {
		os.Remove(registerScript)
		return fmt.Errorf("downloaded register.sh is invalid")
	}
	os.Chmod(registerScript, 0755)

	// For self-signed, patch curl commands
	if e.cfg.SSL.Mode == "selfsigned" {
		RunWithTimeout(ctx, "patch-register", 5*time.Second,
			"sed", "-i", "s|curl -s |curl -sk |g", registerScript)
	}

	// Determine registration URL
	hostIP := detectHostIP()
	regURL := fmt.Sprintf("https://%s/admin/adminapi/k3s/register", launchpadDomain)

	// Test if public URL is reachable
	if err := RunWithTimeout(ctx, "test-public", 5*time.Second,
		"curl", "-sf", "-k", "--max-time", "5",
		fmt.Sprintf("https://%s", launchpadDomain), "-o", "/dev/null"); err != nil {
		regURL = fmt.Sprintf("https://%s/admin/adminapi/k3s/register", hostIP)
	}

	// Run registration
	var regCmd string
	if e.cfg.Kubernetes.Mode == "builtin" {
		regCmd = fmt.Sprintf("LAUNCHPAD_REGISTRATION_URL=%s CLUSTER_NAME=local-k3s %s --skip-k3s-check",
			regURL, registerScript)
	} else {
		kubeconfigPath := e.cfg.Kubernetes.Kubeconfig
		if kubeconfigPath == "" {
			kubeconfigPath = "/etc/rancher/k3s/k3s.yaml"
		}
		clusterName := "external-k8s"
		regCmd = fmt.Sprintf("LAUNCHPAD_REGISTRATION_URL=%s KUBECONFIG=%s CLUSTER_NAME=%s %s --skip-k3s-check",
			regURL, kubeconfigPath, clusterName, registerScript)
	}

	err := RunWithTimeout(ctx, "register-cluster", 120*time.Second, "bash", "-c", regCmd)
	os.Remove(registerScript)

	if err != nil {
		return fmt.Errorf("cluster registration failed: %w", err)
	}

	// For self-signed, patch heartbeat script
	if e.cfg.SSL.Mode == "selfsigned" {
		heartbeatPath := "/usr/local/bin/k3s-heartbeat.sh"
		if _, err := os.Stat(heartbeatPath); err == nil {
			RunWithTimeout(ctx, "patch-heartbeat", 5*time.Second,
				"sed", "-i", "s|curl -s |curl -sk |g", heartbeatPath)
		}
	}

	return nil
}

func hasShebang(s string) bool {
	return len(s) >= 2 && s[:2] == "#!"
}

// registerScriptPath returns the absolute path to the deploy output directory.
// Used as a helper for locating generated files adjacent to register.go.
func (e *Engine) registerScriptPath(name string) string {
	return filepath.Join(e.output, name)
}

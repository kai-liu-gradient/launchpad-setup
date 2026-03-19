package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BuildStepList returns the ordered list of deployment steps based on the
// current configuration. Steps are conditionally included based on modes.
func (e *Engine) BuildStepList() []Step {
	var steps []Step

	// 1. Always: Pre-flight checks
	steps = append(steps, Step{Name: "Pre-flight checks", Fn: e.preflightCheck})

	// 2. If K8s mode == "builtin": Installing K3s
	if e.cfg.Kubernetes.Mode == "builtin" {
		steps = append(steps, Step{Name: "Installing K3s", Fn: e.installK3s})
	}

	// 3. Always: Generating SSL certificates
	steps = append(steps, Step{Name: "Generating SSL certificates", Fn: e.generateCerts})

	// 4. If K8s mode == "builtin": Installing Ingress-Nginx
	if e.cfg.Kubernetes.Mode == "builtin" {
		steps = append(steps, Step{Name: "Installing Ingress-Nginx", Fn: e.installIngressNginx})
	}

	// 5. Always: Configuring CoreDNS
	steps = append(steps, Step{Name: "Configuring CoreDNS", Fn: e.configureCoreDNS})

	// 6. If SSL mode == "selfsigned": Setting up Kyverno + CA distribution
	if e.cfg.SSL.Mode == "selfsigned" {
		steps = append(steps, Step{Name: "Setting up Kyverno + CA distribution", Fn: e.setupKyverno})
	}

	// 7. If DB mode == "builtin": Starting PostgreSQL, Starting Redis
	if e.cfg.Database.Mode == "builtin" {
		steps = append(steps, Step{Name: "Starting PostgreSQL", Fn: e.startPostgres})
		steps = append(steps, Step{Name: "Starting Redis", Fn: e.startRedis})
	}

	// 8. Always: Initializing databases
	steps = append(steps, Step{Name: "Initializing databases", Fn: e.initDatabases})

	// 9. Always: Starting Gitea
	steps = append(steps, Step{Name: "Starting Gitea", Fn: e.startGitea})

	// 10. Always: Bootstrapping Gitea
	steps = append(steps, Step{Name: "Bootstrapping Gitea", Fn: e.bootstrapGitea})

	// 11. Always: Starting application services
	steps = append(steps, Step{Name: "Starting application services", Fn: e.startAppServices})

	// 12. Always: Waiting for API health
	steps = append(steps, Step{Name: "Waiting for API health", Fn: e.waitForAPIHealth})

	// 13. Always: Waiting for UI health
	steps = append(steps, Step{Name: "Waiting for UI health", Fn: e.waitForUIHealth})

	// 14. Always: Waiting for Router health
	steps = append(steps, Step{Name: "Waiting for Router health", Fn: e.waitForRouterHealth})

	// 15. Always: Starting Nginx
	steps = append(steps, Step{Name: "Starting Nginx", Fn: e.startNginx})

	// 16. Always: Importing templates
	steps = append(steps, Step{Name: "Importing templates", Fn: e.importTemplates})

	// 17. Always: Registering cluster
	steps = append(steps, Step{Name: "Registering cluster", Fn: e.registerCluster})

	// 18. If K8s mode == "builtin": Configuring /etc/hosts, Configuring dnsmasq
	if e.cfg.Kubernetes.Mode == "builtin" {
		steps = append(steps, Step{Name: "Configuring /etc/hosts", Fn: e.configureEtcHosts})
		steps = append(steps, Step{Name: "Configuring dnsmasq", Fn: e.configureDnsmasq})
	}

	return steps
}

// --- Infrastructure step implementations ---

func (e *Engine) preflightCheck(_ context.Context) error {
	return RunPreflightChecks(e.cfg)
}

func (e *Engine) installK3s(ctx context.Context) error {
	// Idempotency: if kubectl works, k3s is already running
	if _, err := RunWithOutput(ctx, "k3s-check", 10*time.Second, "kubectl", "get", "nodes"); err == nil {
		return nil
	}

	// Detect internal IP for TLS SAN
	hostIP := detectHostIP()

	// Download and run k3s installer
	err := RunWithTimeout(ctx, "k3s-install", 180*time.Second,
		"bash", "-c",
		fmt.Sprintf("curl -sfL https://get.k3s.io | sh -s - --disable traefik --tls-san %s --write-kubeconfig-mode 644", hostIP))
	if err != nil {
		return fmt.Errorf("k3s installation failed: %w", err)
	}

	// Wait for k3s to be ready (poll kubectl get nodes)
	deadline := time.After(60 * time.Second)
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("k3s failed to start within 60s")
		case <-tick.C:
			if _, err := RunWithOutput(ctx, "k3s-ready", 5*time.Second, "kubectl", "get", "nodes"); err == nil {
				return nil
			}
		}
	}
}

func (e *Engine) installIngressNginx(ctx context.Context) error {
	// Idempotency
	if _, err := RunWithOutput(ctx, "check-ingress", 10*time.Second,
		"helm", "status", "ingress-nginx", "-n", "ingress-nginx"); err == nil {
		return nil
	}

	// Remove Traefik if present
	if _, err := RunWithOutput(ctx, "check-traefik", 10*time.Second,
		"helm", "status", "traefik", "-n", "kube-system"); err == nil {
		RunWithTimeout(ctx, "remove-traefik", 30*time.Second,
			"helm", "uninstall", "traefik", "-n", "kube-system")
		RunWithTimeout(ctx, "remove-traefik-crd", 30*time.Second,
			"helm", "uninstall", "traefik-crd", "-n", "kube-system")
		os.Remove("/var/lib/rancher/k3s/server/manifests/traefik.yaml")
		os.Remove("/var/lib/rancher/k3s/server/manifests/traefik-config.yaml")
	}

	// Add helm repo
	RunWithTimeout(ctx, "helm-repo", 30*time.Second,
		"helm", "repo", "add", "ingress-nginx", "https://kubernetes.github.io/ingress-nginx")

	// values-builtin.yml rendered to output dir
	valuesPath := filepath.Join(e.output, "values-builtin.yml")

	// Install
	if err := RunWithTimeout(ctx, "helm-install", 120*time.Second,
		"helm", "install", "ingress-nginx", "ingress-nginx/ingress-nginx",
		"-n", "ingress-nginx", "--create-namespace",
		"-f", valuesPath); err != nil {
		return fmt.Errorf("installing ingress-nginx: %w", err)
	}

	// Wait for controller pod
	if err := RunWithTimeout(ctx, "wait-ingress", 120*time.Second,
		"kubectl", "wait", "-n", "ingress-nginx",
		"--for=condition=ready", "pod",
		"--selector=app.kubernetes.io/component=controller",
		"--timeout=120s"); err != nil {
		return fmt.Errorf("waiting for ingress-nginx: %w", err)
	}

	return nil
}

func (e *Engine) configureCoreDNS(ctx context.Context) error {
	// Get ingress-nginx ClusterIP (validates the service exists)
	_, err := RunWithOutput(ctx, "get-ingress-ip", 10*time.Second,
		"kubectl", "get", "svc", "ingress-nginx-controller",
		"-n", "ingress-nginx", "-o", "jsonpath={.spec.clusterIP}")
	if err != nil {
		return fmt.Errorf("getting ingress-nginx ClusterIP: %w", err)
	}

	// Apply the rendered coredns config from the output dir
	corednsPath := filepath.Join(e.output, "coredns-custom.yaml")

	if err := RunWithTimeout(ctx, "apply-coredns", 30*time.Second,
		"kubectl", "apply", "-f", corednsPath); err != nil {
		return fmt.Errorf("applying CoreDNS config: %w", err)
	}

	// Patch default CoreDNS: disable loop, use public DNS
	patchCmd := `kubectl get cm coredns -n kube-system -o yaml | sed '/^[^#]*loop$/s/loop/# loop/' | sed 's|forward \. /etc/resolv\.conf|forward . 223.5.5.5 8.8.8.8|' | kubectl apply -f -`
	RunWithTimeout(ctx, "patch-coredns", 30*time.Second, "bash", "-c", patchCmd)

	// Restart CoreDNS
	if err := RunWithTimeout(ctx, "restart-coredns", 60*time.Second,
		"kubectl", "rollout", "restart", "deploy/coredns", "-n", "kube-system"); err != nil {
		return fmt.Errorf("restarting CoreDNS: %w", err)
	}

	return nil
}

func (e *Engine) setupKyverno(ctx context.Context) error {
	// Install Kyverno via helm (idempotent)
	if _, err := RunWithOutput(ctx, "check-kyverno", 10*time.Second,
		"helm", "status", "kyverno", "-n", "kyverno"); err != nil {
		// Not installed — install it
		RunWithTimeout(ctx, "helm-repo-kyverno", 30*time.Second,
			"helm", "repo", "add", "kyverno", "https://kyverno.github.io/kyverno/")

		if err := RunWithTimeout(ctx, "helm-install-kyverno", 120*time.Second,
			"helm", "install", "kyverno", "kyverno/kyverno",
			"-n", "kyverno", "--create-namespace"); err != nil {
			return fmt.Errorf("installing kyverno: %w", err)
		}

		// Wait for admission controller
		if err := RunWithTimeout(ctx, "wait-kyverno", 120*time.Second,
			"kubectl", "wait", "--namespace", "kyverno",
			"--for=condition=ready", "pod",
			"--selector=app.kubernetes.io/component=admission-controller",
			"--timeout=120s"); err != nil {
			return fmt.Errorf("waiting for kyverno: %w", err)
		}
	}

	// Create CA ConfigMaps in kube-system
	caPath := filepath.Join(e.output, "nginx", "certs", "ca.pem")

	createCmd := fmt.Sprintf("kubectl create configmap launchpad-ca-cert --from-file=ca.pem=%s -n kube-system --dry-run=client -o yaml | kubectl apply -f -", caPath)
	if err := RunWithTimeout(ctx, "ca-configmap", 30*time.Second, "bash", "-c", createCmd); err != nil {
		return fmt.Errorf("creating CA configmap: %w", err)
	}

	// Pre-merge system CA bundle + custom CA
	bundleCmd := fmt.Sprintf(
		`cat /etc/ssl/certs/ca-certificates.crt %s > /tmp/ca-bundle.crt && kubectl create configmap launchpad-ca-bundle --from-file=ca-certificates.crt=/tmp/ca-bundle.crt --from-literal=launchpad-ca.sh='export NODE_EXTRA_CA_CERTS=/etc/ssl/certs/launchpad-ca.pem' -n kube-system --dry-run=client -o yaml | kubectl apply -f - && rm -f /tmp/ca-bundle.crt`,
		caPath)
	if err := RunWithTimeout(ctx, "ca-bundle-configmap", 30*time.Second, "bash", "-c", bundleCmd); err != nil {
		return fmt.Errorf("creating CA bundle configmap: %w", err)
	}

	// Apply Kyverno policies (rendered files in output dir)
	for _, f := range []string{"kyverno-sync-ca.yaml", "kyverno-inject-ca.yaml"} {
		policyPath := filepath.Join(e.output, f)
		if err := RunWithTimeout(ctx, "apply-"+f, 30*time.Second,
			"kubectl", "apply", "-f", policyPath); err != nil {
			return fmt.Errorf("applying %s: %w", f, err)
		}
	}

	return nil
}

// Data/Application steps are implemented in database.go, gitea.go, services.go,
// import.go, and register.go (Tasks 21-23).

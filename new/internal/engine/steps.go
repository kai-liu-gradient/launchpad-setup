package engine

import "context"

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

// Stub implementations — real logic added in Chunk 7.

func (e *Engine) preflightCheck(_ context.Context) error      { return nil }
func (e *Engine) installK3s(_ context.Context) error          { return nil }
func (e *Engine) installIngressNginx(_ context.Context) error { return nil }
func (e *Engine) configureCoreDNS(_ context.Context) error    { return nil }
func (e *Engine) setupKyverno(_ context.Context) error        { return nil }

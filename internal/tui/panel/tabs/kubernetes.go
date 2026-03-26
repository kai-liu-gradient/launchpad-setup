package tabs

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/gradient8/launchpad/internal/config"
)

type KubernetesTab struct {
	Mode           string
	InstallIngress bool
	Kubeconfig     string
	Context        string
}

func NewKubernetesTab(cfg *config.Config) *KubernetesTab {
	return &KubernetesTab{
		Mode:           cfg.Kubernetes.Mode,
		InstallIngress: cfg.Kubernetes.InstallIngress,
		Kubeconfig:     cfg.Kubernetes.Kubeconfig,
		Context:        cfg.Kubernetes.Context,
	}
}

func (t *KubernetesTab) Form() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Kubernetes Mode").
				Options(
					huh.NewOption("Built-in K3s", "builtin"),
					huh.NewOption("External cluster", "external"),
				).
				Value(&t.Mode),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Kubeconfig path").
				Placeholder("~/.kube/config").
				Value(&t.Kubeconfig),
			huh.NewInput().
				Title("Kube context").
				Placeholder("my-cluster").
				Value(&t.Context),
			huh.NewConfirm().
				Title("Install Ingress-Nginx?").
				Description("Required for routing traffic to project pods").
				Value(&t.InstallIngress),
		).WithHideFunc(func() bool { return t.Mode != "external" }),
	)
}

func (t *KubernetesTab) View() string {
	s := fmt.Sprintf("  Mode: %s", t.Mode)
	if t.Mode == "external" {
		s += fmt.Sprintf("\n  Kubeconfig: %s\n  Context:    %s\n  Install Ingress-Nginx: %v",
			displayValue(t.Kubeconfig), displayValue(t.Context), t.InstallIngress)
	}
	return s
}

func (t *KubernetesTab) Apply(cfg *config.Config) {
	cfg.Kubernetes.Mode = t.Mode
	cfg.Kubernetes.Kubeconfig = t.Kubeconfig
	cfg.Kubernetes.Context = t.Context
	if t.Mode == "builtin" {
		cfg.Kubernetes.InstallIngress = true
	} else {
		cfg.Kubernetes.InstallIngress = t.InstallIngress
	}
}

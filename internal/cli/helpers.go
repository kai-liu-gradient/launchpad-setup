package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/gradient8/launchpad/internal/config"
	"github.com/gradient8/launchpad/internal/engine"
	"github.com/gradient8/launchpad/internal/secrets"
	"github.com/gradient8/launchpad/internal/template"
	"github.com/spf13/cobra"
)

func newSetupCertsCmd() *cobra.Command {
	var flagDir string

	cmd := &cobra.Command{
		Use:   "setup-certs",
		Short: "Generate SSL certificates for AniLaunchpad",
		Long:  "Run the certificate generation step against an existing installation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetupCerts(flagDir)
		},
	}

	cmd.Flags().StringVar(&flagDir, "dir", ".", "Installation directory")

	return cmd
}

func runSetupCerts(dir string) error {
	cfg, sec, outputDir, err := loadInstallContext(dir)
	if err != nil {
		return err
	}

	eng := engine.New(cfg, sec, outputDir, nil)
	steps := eng.BuildStepList()

	// Find and run the cert generation step
	for _, step := range steps {
		if step.Name == "Generating SSL certificates" {
			if err := step.Fn(context.Background()); err != nil {
				return fmt.Errorf("setup-certs failed: %w", err)
			}
			fmt.Println("SSL certificates generated successfully.")
			return nil
		}
	}

	return fmt.Errorf("certificate generation step not found in pipeline")
}

func newSetupDBCmd() *cobra.Command {
	var flagDir string

	cmd := &cobra.Command{
		Use:   "setup-db",
		Short: "Initialize databases for AniLaunchpad",
		Long:  "Run the database initialization step against an existing installation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetupDB(flagDir)
		},
	}

	cmd.Flags().StringVar(&flagDir, "dir", ".", "Installation directory")

	return cmd
}

func runSetupDB(dir string) error {
	cfg, sec, outputDir, err := loadInstallContext(dir)
	if err != nil {
		return err
	}

	eng := engine.New(cfg, sec, outputDir, nil)
	steps := eng.BuildStepList()

	// Find and run the database init step
	for _, step := range steps {
		if step.Name == "Initializing databases" {
			if err := step.Fn(context.Background()); err != nil {
				return fmt.Errorf("setup-db failed: %w", err)
			}
			fmt.Println("Databases initialized successfully.")
			return nil
		}
	}

	return fmt.Errorf("database initialization step not found in pipeline")
}

func newImportTemplatesCmd() *cobra.Command {
	var flagDir string

	cmd := &cobra.Command{
		Use:   "import-templates",
		Short: "Import Gitea templates for AniLaunchpad",
		Long:  "Run the template import step against an existing installation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImportTemplates(flagDir)
		},
	}

	cmd.Flags().StringVar(&flagDir, "dir", ".", "Installation directory")

	return cmd
}

func runImportTemplates(dir string) error {
	cfg, sec, outputDir, err := loadInstallContext(dir)
	if err != nil {
		return err
	}

	eng := engine.New(cfg, sec, outputDir, nil)
	steps := eng.BuildStepList()

	// Find and run the import templates step
	for _, step := range steps {
		if step.Name == "Importing templates" {
			if err := step.Fn(context.Background()); err != nil {
				return fmt.Errorf("import-templates failed: %w", err)
			}
			fmt.Println("Templates imported successfully.")
			return nil
		}
	}

	return fmt.Errorf("import templates step not found in pipeline")
}

// loadInstallContext is a shared helper that loads config, secrets, and
// resolves the output directory for helper commands.
func loadInstallContext(dir string) (*config.Config, *secrets.Secrets, string, error) {
	cfgPath := filepath.Join(dir, ".setup.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("loading config from %s: %w", cfgPath, err)
	}

	secPath := filepath.Join(dir, ".secrets.yaml")
	sec, err := secrets.Load(secPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("loading secrets from %s: %w", secPath, err)
	}

	runtimePath := filepath.Join(dir, ".runtime.yaml")
	runtime, err := template.LoadRuntime(runtimePath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("loading runtime values: %w", err)
	}

	outputDir := filepath.Join(dir, "generated")
	derived := template.ComputeDerived(cfg, sec)
	ctx := &template.RenderContext{
		Config:  cfg,
		Secrets: sec,
		Derived: derived,
		Runtime: runtime,
	}
	_ = ctx // runtime is loaded for potential future use; outputDir is returned

	return cfg, sec, outputDir, nil
}

// new/internal/cli/root.go
package cli

import "github.com/spf13/cobra"

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "launchpad",
		Short:   "AniLaunchpad deployment tool",
		Version: version,
	}
	cmd.AddCommand(
		newInstallCmd(),
		newConfigureCmd(),
		newStatusCmd(),
		newUpgradeCmd(),
		newRestartCmd(),
		newUninstallCmd(),
		newSetupCertsCmd(),
		newSetupDBCmd(),
		newImportTemplatesCmd(),
	)
	return cmd
}

func Execute(version string) error {
	return newRootCmd(version).Execute()
}

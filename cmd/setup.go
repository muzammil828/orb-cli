package cmd

import (
	"bufio"
	"log/slog"
	"os"

	"orb/component"
	"orb/executor"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure the deployment",
	Long:  "Collect and validate configuration for all deployment components.",
	RunE: func(cmd *cobra.Command, args []string) error {
		deployment := component.DeploymentConfig{}
		prompter := terminalPrompter{input: os.Stdin, reader: bufio.NewReader(os.Stdin), output: os.Stdout}
		setupComponents, err := infraRegistry.SetupComponents()
		if err != nil {
			return err
		}
		if err := component.GatherInputs(cmd.Context(), setupComponents, &deployment, prompter); err != nil {
			return err
		}
		if err := deployment.Validate(); err != nil {
			return err
		}
		path, err := component.DefaultConfigPath()
		if err != nil {
			return err
		}
		if err := component.SaveConfig(path, deployment); err != nil {
			return err
		}
		if err := component.Prepare(cmd.Context(), setupComponents, deployment, executor.Runner{}); err != nil {
			return err
		}
		slog.Info("deployment setup complete", "config_path", path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

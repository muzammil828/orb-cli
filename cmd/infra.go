package cmd

import (
	"orb/component"
	infraComponents "orb/component/infra"
	"orb/executor"

	"github.com/spf13/cobra"
)

var infraService string
var infraRegistry = infraComponents.NewRegistry()

func init() {
	rootCmd.AddCommand(newLifecycleCommand("infra", "Manage infrastructure services", &infraService, map[string]func(*cobra.Command, []string) error{
		"install": infraInstall,
		"upgrade": infraUpgrade,
		"status":  infraStatus,
		"start":   infraStart,
		"stop":    infraStop,
		"restart": infraRestart,
	}))
}

func infraInstall(cmd *cobra.Command, args []string) error { return runInfraAction(cmd, "install") }
func infraUpgrade(cmd *cobra.Command, args []string) error { return runInfraAction(cmd, "upgrade") }
func infraStatus(cmd *cobra.Command, args []string) error  { return runInfraAction(cmd, "status") }
func infraStart(cmd *cobra.Command, args []string) error   { return runInfraAction(cmd, "start") }
func infraStop(cmd *cobra.Command, args []string) error    { return runInfraAction(cmd, "stop") }
func infraRestart(cmd *cobra.Command, args []string) error { return runInfraAction(cmd, "restart") }

func runInfraAction(cmd *cobra.Command, action string) error {
	path, err := component.DefaultConfigPath()
	if err != nil {
		return err
	}
	deployment, err := component.LoadConfig(path)
	if err != nil {
		return err
	}
	return infraRegistry.Run(cmd.Context(), infraService, action, deployment, executor.Runner{})
}

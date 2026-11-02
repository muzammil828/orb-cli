package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var platformService string

func init() {
	rootCmd.AddCommand(newLifecycleCommand("platform", "Manage platform services", &platformService, map[string]func(*cobra.Command, []string) error{
		"install": platformInstall,
		"upgrade": platformUpgrade,
		"status":  platformStatus,
		"start":   platformStart,
		"stop":    platformStop,
		"restart": platformRestart,
	}))
}

func platformInstall(cmd *cobra.Command, args []string) error { return runPlatformAction("install") }
func platformUpgrade(cmd *cobra.Command, args []string) error { return runPlatformAction("upgrade") }
func platformStatus(cmd *cobra.Command, args []string) error  { return runPlatformAction("status") }
func platformStart(cmd *cobra.Command, args []string) error   { return runPlatformAction("start") }
func platformStop(cmd *cobra.Command, args []string) error    { return runPlatformAction("stop") }
func platformRestart(cmd *cobra.Command, args []string) error { return runPlatformAction("restart") }

func runPlatformAction(action string) error {
	if platformService != "" {
		slog.Info("platform lifecycle command selected", "action", action, "service", platformService)
		return nil
	}

	slog.Info("platform lifecycle command selected", "action", action, "service_scope", "all")
	return nil
}

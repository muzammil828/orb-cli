package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
)

var appService string

func init() {
	rootCmd.AddCommand(newLifecycleCommand("app", "Manage application services", &appService, map[string]func(*cobra.Command, []string) error{
		"install": appInstall,
		"upgrade": appUpgrade,
		"status":  appStatus,
		"start":   appStart,
		"stop":    appStop,
		"restart": appRestart,
	}))
}

func appInstall(cmd *cobra.Command, args []string) error { return runAppAction("install") }
func appUpgrade(cmd *cobra.Command, args []string) error { return runAppAction("upgrade") }
func appStatus(cmd *cobra.Command, args []string) error  { return runAppAction("status") }
func appStart(cmd *cobra.Command, args []string) error   { return runAppAction("start") }
func appStop(cmd *cobra.Command, args []string) error    { return runAppAction("stop") }
func appRestart(cmd *cobra.Command, args []string) error { return runAppAction("restart") }

func runAppAction(action string) error {
	if appService != "" {
		slog.Info("application lifecycle command selected", "action", action, "service", appService)
		return nil
	}

	slog.Info("application lifecycle command selected", "action", action, "service_scope", "all")
	return nil
}

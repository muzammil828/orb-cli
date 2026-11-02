package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "orb",
	Short:         "Manage Orb deployments",
	Long:          "Orb is a CLI for configuring, installing, and managing service deployments.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

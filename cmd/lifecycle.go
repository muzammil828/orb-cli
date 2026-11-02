package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var lifecycleOperations = []struct {
	name  string
	short string
}{
	{name: "install", short: "Install"},
	{name: "upgrade", short: "Upgrade"},
	{name: "status", short: "Show status of"},
	{name: "start", short: "Start"},
	{name: "stop", short: "Stop"},
	{name: "restart", short: "Restart"},
}

// newLifecycleCommand builds the shared CLI shape for a category. The
// category-specific component logic will be connected behind these handlers
// as the service registry and deployment layers are introduced.
func newLifecycleCommand(
	category string,
	description string,
	service *string,
	runHandlers map[string]func(*cobra.Command, []string) error,
) *cobra.Command {
	categoryCmd := &cobra.Command{
		Use:   category,
		Short: description,
	}

	categoryCmd.PersistentFlags().StringVar(service, "service", "", "Name of a specific service")

	for _, operation := range lifecycleOperations {
		runE, ok := runHandlers[operation.name]
		if !ok {
			continue
		}
		categoryCmd.AddCommand(newLifecycleOperationCommand(category, operation.name, operation.short, runE))
	}

	return categoryCmd
}

func newLifecycleOperationCommand(category, operation, verb string, runE func(*cobra.Command, []string) error) *cobra.Command {
	noun := category + " services"
	return &cobra.Command{
		Use:   operation,
		Short: fmt.Sprintf("%s %s", verb, noun),
		Long:  fmt.Sprintf("%s selected or all %s.", verb, noun),
		RunE:  runE,
	}
}

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sg",
	Short: "Saga - Task management for agent workflows",
	Long: `Saga is a task management tool designed for agent workflows.

It supports hierarchical tasks (sub-sagas), project scoping, and
clean integration with automated workflows.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here
}

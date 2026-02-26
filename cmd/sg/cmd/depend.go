package cmd

import (
	"fmt"

	"github.com/hbn/saga/internal/store"
	"github.com/spf13/cobra"
)

var dependCmd = &cobra.Command{
	Use:   "depend <id> <add|remove> <target-id>",
	Short: "Manage saga dependencies",
	Long: `Add or remove hard dependencies between sagas.

A saga with dependencies cannot be marked as done until all
dependencies are completed. This creates a blocking relationship.

Examples:
  sg depend abc123 add def456    # abc123 now depends on def456
  sg depend abc123 remove def456 # Remove dependency`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		sagaID := args[0]
		action := args[1]
		targetID := args[2]

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		sg, err := st.GetByID(sagaID)
		if err != nil {
			return sagaNotFound(sagaID)
		}

		target, err := st.GetByID(targetID)
		if err != nil {
			return sagaNotFound(targetID)
		}

		switch action {
		case "add":
			if sg.HasDependency(targetID) {
				return fmt.Errorf("saga \"%s\" already depends on \"%s\"", sagaID, targetID)
			}

			// Check for circular dependency
			circular, err := st.WouldCreateCircularDependency(sagaID, targetID)
			if err != nil {
				return fmt.Errorf("checking circular dependency: %w", err)
			}
			if circular {
				return circularDependency()
			}

			sg.AddDependency(targetID)
			if err := st.Update(sg); err != nil {
				return fmt.Errorf("updating saga: %w", err)
			}
			fmt.Printf("Added dependency: %s now depends on %s (%s)\n", sagaID, targetID, target.Title)

		case "remove":
			if !sg.HasDependency(targetID) {
				return fmt.Errorf("saga %s does not depend on %s", sagaID, targetID)
			}

			sg.RemoveDependency(targetID)
			if err := st.Update(sg); err != nil {
				return fmt.Errorf("updating saga: %w", err)
			}
			fmt.Printf("Removed dependency: %s no longer depends on %s\n", sagaID, targetID)

		default:
			return fmt.Errorf("unknown action: %s (use 'add' or 'remove')", action)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dependCmd)
}

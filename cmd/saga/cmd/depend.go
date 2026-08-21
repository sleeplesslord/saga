package cmd

import (
	"errors"
	"fmt"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
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

		if _, err := st.GetByID(sagaID); err != nil {
			return sagaNotFound(sagaID)
		}

		target, err := st.GetByID(targetID)
		if err != nil {
			return sagaNotFound(targetID)
		}

		// The circular-dependency check walks the whole graph, so it stays outside
		// Mutate — fn runs with the store lock held and re-entering the store
		// would deadlock. The per-record has/hasn't check moves inside, where it
		// sees the record as stored.
		var apply func(*saga.Saga) error
		switch action {
		case "add":
			circular, err := st.WouldCreateCircularDependency(sagaID, targetID)
			if err != nil {
				return fmt.Errorf("checking circular dependency: %w", err)
			}
			if circular {
				return circularDependency()
			}
			apply = func(sg *saga.Saga) error {
				if sg.HasDependency(targetID) {
					return fmt.Errorf("saga \"%s\" already depends on \"%s\"", sagaID, targetID)
				}
				sg.AddDependency(targetID)
				return nil
			}
		case "remove":
			apply = func(sg *saga.Saga) error {
				if !sg.HasDependency(targetID) {
					return fmt.Errorf("saga %s does not depend on %s", sagaID, targetID)
				}
				sg.RemoveDependency(targetID)
				return nil
			}
		default:
			return fmt.Errorf("unknown action: %s (use 'add' or 'remove')", action)
		}

		if _, err := st.Mutate(sagaID, apply); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return sagaNotFound(sagaID)
			}
			return err
		}

		if action == "add" {
			fmt.Printf("Added dependency: %s now depends on %s (%s)\n", sagaID, targetID, target.Title)
		} else {
			fmt.Printf("Removed dependency: %s no longer depends on %s\n", sagaID, targetID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dependCmd)
}

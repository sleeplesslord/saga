package cmd

import (
	"errors"
	"fmt"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var relateCmd = &cobra.Command{
	Use:   "relate <id> <add|remove> <target-id>",
	Short: "Manage saga relationships",
	Long: `Add or remove soft relationships between sagas.

Relationships are informational only - they don't block completion
or affect saga status. Use this to link related work items.

Examples:
  sg relate abc123 add def456    # Mark abc123 as related to def456
  sg relate abc123 remove def456 # Remove relationship`,
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
			return fmt.Errorf("target saga not found: %s", targetID)
		}

		var apply func(*saga.Saga) error
		switch action {
		case "add":
			apply = func(sg *saga.Saga) error {
				if sg.HasRelationship(targetID) {
					return fmt.Errorf("saga %s is already related to %s", sagaID, targetID)
				}
				sg.AddRelationship(targetID)
				return nil
			}
		case "remove":
			apply = func(sg *saga.Saga) error {
				if !sg.HasRelationship(targetID) {
					return fmt.Errorf("saga %s is not related to %s", sagaID, targetID)
				}
				sg.RemoveRelationship(targetID)
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
			fmt.Printf("Added relationship: %s is now related to %s (%s)\n", sagaID, targetID, target.Title)
		} else {
			fmt.Printf("Removed relationship: %s is no longer related to %s\n", sagaID, targetID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(relateCmd)
}

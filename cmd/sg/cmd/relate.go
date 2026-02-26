package cmd

import (
	"fmt"

	"github.com/hbn/saga/internal/store"
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

		sg, err := st.GetByID(sagaID)
		if err != nil {
			return err
		}

		target, err := st.GetByID(targetID)
		if err != nil {
			return fmt.Errorf("target saga not found: %s", targetID)
		}

		switch action {
		case "add":
			if sg.HasRelationship(targetID) {
				return fmt.Errorf("saga %s is already related to %s", sagaID, targetID)
			}

			sg.AddRelationship(targetID)
			if err := st.Update(sg); err != nil {
				return fmt.Errorf("updating saga: %w", err)
			}
			fmt.Printf("Added relationship: %s is now related to %s (%s)\n", sagaID, targetID, target.Title)

		case "remove":
			if !sg.HasRelationship(targetID) {
				return fmt.Errorf("saga %s is not related to %s", sagaID, targetID)
			}

			sg.RemoveRelationship(targetID)
			if err := st.Update(sg); err != nil {
				return fmt.Errorf("updating saga: %w", err)
			}
			fmt.Printf("Removed relationship: %s is no longer related to %s\n", sagaID, targetID)

		default:
			return fmt.Errorf("unknown action: %s (use 'add' or 'remove')", action)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(relateCmd)
}

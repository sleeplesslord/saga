package cmd

import (
	"fmt"

	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var labelCmd = &cobra.Command{
	Use:   "label <id> <add|remove> <label>",
	Short: "Manage saga labels",
	Long: `Add or remove labels from a saga.

Examples:
  sg label abc123 add bug
  sg label abc123 add urgent
  sg label abc123 remove bug`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		action := args[1]
		label := args[2]

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		sg, err := st.GetByID(id)
		if err != nil {
			return sagaNotFound(id)
		}

		switch action {
		case "add":
			if sg.HasLabel(label) {
				return alreadyHasLabel(id, label)
			}
			sg.AddLabel(label)
			if err := st.Update(sg); err != nil {
				return fmt.Errorf("updating saga: %w", err)
			}
			fmt.Printf("Added label '%s' to saga %s\n", label, id)

		case "remove":
			if !sg.HasLabel(label) {
				return missingLabel(id, label)
			}
			sg.RemoveLabel(label)
			if err := st.Update(sg); err != nil {
				return fmt.Errorf("updating saga: %w", err)
			}
			fmt.Printf("Removed label '%s' from saga %s\n", label, id)

		default:
			return fmt.Errorf("unknown action: %s (use 'add' or 'remove')", action)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(labelCmd)
}

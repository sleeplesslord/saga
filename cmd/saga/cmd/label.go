package cmd

import (
	"errors"
	"fmt"

	"github.com/sleeplesslord/saga/internal/saga"
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

		// The has/hasn't check runs inside Mutate so it is decided against the
		// record as stored, not a copy read beforehand.
		var apply func(*saga.Saga) error
		switch action {
		case "add":
			apply = func(sg *saga.Saga) error {
				if sg.HasLabel(label) {
					return alreadyHasLabel(id, label)
				}
				sg.AddLabel(label)
				return nil
			}
		case "remove":
			apply = func(sg *saga.Saga) error {
				if !sg.HasLabel(label) {
					return missingLabel(id, label)
				}
				sg.RemoveLabel(label)
				return nil
			}
		default:
			return fmt.Errorf("unknown action: %s (use 'add' or 'remove')", action)
		}

		if _, err := st.Mutate(id, apply); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return sagaNotFound(id)
			}
			return err
		}

		if action == "add" {
			fmt.Printf("Added label '%s' to saga %s\n", label, id)
		} else {
			fmt.Printf("Removed label '%s' from saga %s\n", label, id)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(labelCmd)
}

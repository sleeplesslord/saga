package cmd

import (
	"fmt"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var reopenReason string

var reopenCmd = &cobra.Command{
	Use:   "reopen <id>",
	Short: "Reopen a completed saga",
	Long: `Reopen a saga that was previously marked as done.

This is useful when:
- A done saga needs additional work
- Requirements changed after completion
- You accidentally marked the wrong saga as done

The saga will be set back to "active" status and added to history.`,
	Example: `  sg reopen abc123
  sg reopen abc123 --reason "Requirements changed, need to add feature X"
  sg reopen abc123 --reason "Bug found in implementation"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		sg, err := st.GetByID(id)
		if err != nil {
			return sagaNotFound(id)
		}

		if sg.Status != saga.StatusDone {
			return fmt.Errorf("saga %s is not done (current status: %s)", sg.ID, sg.Status)
		}

		sg.SetStatus(saga.StatusActive)

		// Log reason if provided
		if reopenReason != "" {
			sg.AddHistory("reopened", reopenReason)
		} else {
			sg.AddHistory("reopened", "Saga reopened")
		}

		if err := st.Update(sg); err != nil {
			return fmt.Errorf("updating saga: %w", err)
		}

		fmt.Printf("Reopened saga %s: %s\n", sg.ID, sg.Title)
		fmt.Printf("Status: %s\n", sg.Status)
		if reopenReason != "" {
			fmt.Printf("Reason: %s\n", reopenReason)
		}
		return nil
	},
}

func init() {
	reopenCmd.Flags().StringVar(&reopenReason, "reason", "", "Reason for reopening the saga (logged in history)")
	rootCmd.AddCommand(reopenCmd)
}

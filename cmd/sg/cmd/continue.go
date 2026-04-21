package cmd

import (
	"fmt"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var continueCmd = &cobra.Command{
	Use:   "continue <id>",
	Short: "Resume a paused saga",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		sg, err := st.GetByID(id)
		if err != nil {
			return err
		}

		if sg.Status == saga.StatusDone || sg.Status == saga.StatusWontDo {
			return fmt.Errorf("saga %s is in a terminal state (%s); use \"sg reopen\" to resume it", sg.ID, sg.Status)
		}

		sg.SetStatus(saga.StatusActive)

		if err := st.Update(sg); err != nil {
			return fmt.Errorf("updating saga: %w", err)
		}

		fmt.Printf("Continuing saga %s: %s\n", sg.ID, sg.Title)
		fmt.Printf("Status: %s\n", sg.Status)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(continueCmd)
}

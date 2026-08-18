package cmd

import (
	"fmt"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var continueCmd = &cobra.Command{
	Use:     "continue <id>",
	Aliases: []string{"resume"},
	Short:   "Resume a paused saga",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		sg, err := st.Mutate(id, func(sg *saga.Saga) error {
			if sg.Status == saga.StatusDone || sg.Status == saga.StatusWontDo {
				return fmt.Errorf("saga %s is in a terminal state (%s); use \"sg reopen\" to resume it", sg.ID, sg.Status)
			}
			sg.SetStatus(saga.StatusActive)
			return nil
		})
		if err != nil {
			return err
		}

		fmt.Printf("Continuing saga %s: %s\n", sg.ID, sg.Title)
		fmt.Printf("Status: %s\n", sg.Status)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(continueCmd)
}

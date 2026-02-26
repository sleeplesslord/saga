package cmd

import (
	"fmt"

	"github.com/hbn/saga/internal/saga"
	"github.com/hbn/saga/internal/store"
	"github.com/spf13/cobra"
)

var force bool

var doneCmd = &cobra.Command{
	Use:   "done <id>",
	Short: "Mark saga as complete",
	Long: `Mark a saga as done. By default, cannot mark a saga as done if it has active sub-sagas.
Use --force to override this check.`,
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

		// Check for active children
		hasActiveChildren, err := st.HasActiveChildren(id)
		if err != nil {
			return fmt.Errorf("checking children: %w", err)
		}
		if hasActiveChildren && !force {
			return activeChildren(id)
		}

		// Check for incomplete dependencies
		hasIncompleteDeps, incompleteDeps, err := st.HasIncompleteDependencies(id)
		if err != nil {
			return fmt.Errorf("checking dependencies: %w", err)
		}
		if hasIncompleteDeps && !force {
			return incompleteDependencies(id, incompleteDeps)
		}

		sg.SetStatus(saga.StatusDone)

		if err := st.Update(sg); err != nil {
			return fmt.Errorf("updating saga: %w", err)
		}

		fmt.Printf("Marked saga %s as done\n", sg.ID)
		return nil
	},
}

func init() {
	doneCmd.Flags().BoolVar(&force, "force", false, "Force completion even with active children")
	rootCmd.AddCommand(doneCmd)
}

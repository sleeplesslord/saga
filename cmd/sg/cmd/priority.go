package cmd

import (
	"fmt"

	"github.com/hbn/saga/internal/saga"
	"github.com/hbn/saga/internal/store"
	"github.com/spf13/cobra"
)

var priorityCmd = &cobra.Command{
	Use:   "priority <id> <high|normal|low>",
	Short: "Change saga priority",
	Long: `Change the priority level of a saga.

Priority levels:
  high   - High priority (shown first in list)
  normal - Normal priority (default)
  low    - Low priority (shown last)

Examples:
  sg priority abc123 high
  sg priority abc123 low`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		priorityStr := args[1]

		var priority saga.Priority
		switch priorityStr {
		case "high":
			priority = saga.PriorityHigh
		case "normal":
			priority = saga.PriorityNormal
		case "low":
			priority = saga.PriorityLow
		default:
			return invalidPriority(priorityStr)
		}

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		sg, err := st.GetByID(id)
		if err != nil {
			return sagaNotFound(id)
		}

		sg.SetPriority(priority)

		if err := st.Update(sg); err != nil {
			return fmt.Errorf("updating saga: %w", err)
		}

		fmt.Printf("Changed priority of saga %s to %s\n", id, priority)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(priorityCmd)
}

package cmd

import (
	"fmt"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <id>",
	Short: "Show saga details and history",
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

		fmt.Printf("Saga: %s (%s)\n", sg.ID, sg.Status)
		fmt.Printf("Title: %s\n", sg.Title)
		if sg.Description != "" {
			fmt.Printf("Description: %s\n", sg.Description)
		}
		if sg.IsSubSaga() {
			fmt.Printf("Parent: %s\n", sg.ParentID)
		}
		if sg.Priority != saga.PriorityNormal {
			fmt.Printf("Priority: %s\n", sg.Priority)
		}
		if len(sg.Labels) > 0 {
			fmt.Printf("Labels: %v\n", sg.Labels)
		}
		if len(sg.DependsOn) > 0 {
			fmt.Printf("Depends on: %v\n", sg.DependsOn)
		}
		if len(sg.RelatedTo) > 0 {
			fmt.Printf("Related to: %v\n", sg.RelatedTo)
		}
		if sg.IsClaimed() {
			fmt.Printf("Claimed by: %s (expires %s)\n", sg.ClaimedBy, sg.ClaimExpiry().Format("Jan 02 15:04"))
		}
		if sg.Deadline != "" {
			fmt.Printf("Deadline: %s\n", sg.Deadline)
		}
		fmt.Printf("Created: %s\n", sg.CreatedAt.Format("Jan 02, 2006 15:04"))
		fmt.Printf("Updated: %s\n", sg.UpdatedAt.Format("Jan 02, 2006 15:04"))
		fmt.Println()
		fmt.Println("Recent history:")

		start := len(sg.History) - 5
		if start < 0 {
			start = 0
		}
		for i := len(sg.History) - 1; i >= start; i-- {
			entry := sg.History[i]
			fmt.Printf("  %s | %s", entry.Timestamp.Format("15:04"), entry.Action)
			if entry.Note != "" {
				fmt.Printf(" - %s", entry.Note)
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

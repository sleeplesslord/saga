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
		printField("Title:", sg.Title, 0)
		if sg.Description != "" {
			printField("Description:", formatDescription(sg.Description), 0)
		}
		if sg.IsSubSaga() {
			fmt.Printf("Parent: %s\n", sg.ParentID)
		}
		if sg.Priority != saga.PriorityNormal {
			printField("Priority:", string(sg.Priority), 0)
		}
		if len(sg.Labels) > 0 {
			printField("Labels:", fmt.Sprintf("%v", sg.Labels), 0)
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
			printField("Deadline:", sg.Deadline, 0)
		}
		fmt.Printf("Created: %s\n", sg.CreatedAt.Format("Jan 02, 2006 15:04"))
		fmt.Printf("Updated: %s\n", sg.UpdatedAt.Format("Jan 02, 2006 15:04"))
		fmt.Println()
		fmt.Println("Recent history:")

		printHistoryEntries(sg.History, 5, false)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

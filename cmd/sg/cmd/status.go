package cmd

import (
	"fmt"

	"github.com/hbn/saga/internal/store"
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
		if sg.IsSubSaga() {
			fmt.Printf("Parent: %s\n", sg.ParentID)
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

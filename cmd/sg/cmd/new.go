package cmd

import (
	"fmt"

	"github.com/hbn/saga/internal/saga"
	"github.com/hbn/saga/internal/store"
	"github.com/spf13/cobra"
)

var (
	parentID string
	labels   []string
	priority string
)

var newCmd = &cobra.Command{
	Use:   "new <title>",
	Short: "Create a new saga",
	Long: `Create a new saga. Use --parent to create a sub-saga, --label to add labels.

Examples:
  sg new "Implement auth"
  sg new "Add OAuth" --parent abc123
  sg new "Fix bug" --label bug --label urgent
  sg new "Critical fix" --priority high`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := args[0]

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		var sg *saga.Saga
		if parentID != "" {
			// Verify parent exists
			parent, err := st.GetByID(parentID)
			if err != nil {
				return fmt.Errorf("parent saga not found: %s", parentID)
			}

			if parent.Status == saga.StatusDone {
				return fmt.Errorf("cannot create sub-saga under done saga %s", parentID)
			}

			// Generate hierarchical ID
			childID, err := st.GetNextChildID(parentID)
			if err != nil {
				return fmt.Errorf("generating child ID: %w", err)
			}

			sg = saga.NewSubSaga(title, childID, parentID)
			fmt.Printf("Created sub-saga %s under %s: %s\n", sg.ID, parentID, sg.Title)
		} else {
			sg = saga.NewSaga(title)
			fmt.Printf("Created saga %s: %s\n", sg.ID, sg.Title)
		}

		// Add labels
		for _, label := range labels {
			sg.AddLabel(label)
		}
		if len(labels) > 0 {
			fmt.Printf("Labels: %v\n", labels)
		}

		// Set priority if specified
		if priority != "" {
			switch priority {
			case "high":
				sg.SetPriority(saga.PriorityHigh)
			case "low":
				sg.SetPriority(saga.PriorityLow)
			default:
				// normal is default, nothing to do
			}
			fmt.Printf("Priority: %s\n", sg.Priority)
		}

		if err := st.Save(sg); err != nil {
			return fmt.Errorf("saving saga: %w", err)
		}

		return nil
	},
}

func init() {
	newCmd.Flags().StringVar(&parentID, "parent", "", "Parent saga ID (creates sub-saga)")
	newCmd.Flags().StringArrayVar(&labels, "label", nil, "Add label (can specify multiple)")
	newCmd.Flags().StringVar(&priority, "priority", "", "Set priority (high, normal, low)")
	rootCmd.AddCommand(newCmd)
}

package cmd

import (
	"fmt"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var (
	parentID    string
	labels      []string
	priority    string
	description string
	deadline    string
)

var newCmd = &cobra.Command{
	Use:   "new <title>",
	Short: "Create a new saga",
	Long: `Create a new saga. Use --parent to create a sub-saga, --label to add labels.

Examples:
  sg new "Implement auth"
  sg new "Add OAuth" --parent abc123
  sg new "Fix bug" --label bug --label urgent
  sg new "Critical fix" --priority high
  sg new "Refactor" --desc "Clean up the auth module"`,
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
				return parentNotFound(parentID)
			}

		if parent.Status == saga.StatusDone || parent.Status == saga.StatusWontDo {
				return parentDone(parentID)
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
			case "normal":
				sg.SetPriority(saga.PriorityNormal)
			case "low":
				sg.SetPriority(saga.PriorityLow)
			default:
				return fmt.Errorf("invalid priority %q (must be high, normal, or low)", priority)
			}
			fmt.Printf("Priority: %s\n", sg.Priority)
		}

		// Set description if specified
		if description != "" {
			sg.Description = description
		}

		// Set deadline if specified
		if deadline != "" {
			sg.Deadline = deadline
			fmt.Printf("Deadline: %s\n", deadline)
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
	newCmd.Flags().StringVar(&description, "desc", "", "Add description")
	newCmd.Flags().StringVar(&deadline, "deadline", "", "Set deadline (YYYYMMDD format)")
	rootCmd.AddCommand(newCmd)
}

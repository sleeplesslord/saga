package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var editTitle string
var editDesc string
var editPlan string
var editDeadline string
var editPriority string

var editCmd = &cobra.Command{
	Use:     "edit <id>",
	Aliases: []string{"update"},
	Short:   "Edit saga title, description, plan, deadline, or priority",
	Long: `Update a saga's title, description, plan, deadline, or priority after creation.

Use --title to change the title, --desc to change the description,
--plan to change the implementation plan, --deadline to set/clear deadline,
--priority to change priority.
At least one flag must be provided.

Examples:
  sg edit abc123 --title "New title"
  sg edit abc123 --desc "Updated description"
  sg edit abc123 --plan "1. Add migration\n2. Update model\n3. Add tests"
  sg edit abc123 --deadline 20250415
  sg edit abc123 --deadline ""         # clear deadline
  sg edit abc123 --priority high
  sg edit abc123 --title "New" --desc "Desc" --priority low`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Validate at least one edit flag provided
		titleChanged := cmd.Flags().Changed("title")
		descChanged := cmd.Flags().Changed("desc")
		planChanged := cmd.Flags().Changed("plan")
		deadlineChanged := cmd.Flags().Changed("deadline")
		priorityChanged := cmd.Flags().Changed("priority")
		if !titleChanged && !descChanged && !planChanged && !deadlineChanged && !priorityChanged {
			return fmt.Errorf("at least one of --title, --desc, --plan, --deadline, or --priority required")
		}

		// Reject empty title — clearing a title is not allowed
		if titleChanged && editTitle == "" {
			return fmt.Errorf("title cannot be empty")
		}

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		// Applied under the store lock against the freshest copy, so editing one
		// field doesn't roll back a change another process made to a different one.
		sg, err := st.Mutate(id, func(sg *saga.Saga) error {
			if titleChanged {
				sg.Title = editTitle
				sg.UpdatedAt = time.Now()
				sg.AddHistory("edited", "Updated title")
			}
			if descChanged {
				sg.Description = editDesc // empty string clears description
				sg.UpdatedAt = time.Now()
				sg.AddHistory("edited", "Updated description")
			}
			if planChanged {
				sg.Plan = editPlan // empty string clears plan
				sg.UpdatedAt = time.Now()
				sg.AddHistory("edited", "Updated plan")
			}
			if deadlineChanged {
				sg.Deadline = editDeadline // empty string clears deadline
			}
			if priorityChanged {
				switch editPriority {
				case "high":
					sg.SetPriority(saga.PriorityHigh)
				case "normal":
					sg.SetPriority(saga.PriorityNormal)
				case "low":
					sg.SetPriority(saga.PriorityLow)
				default:
					return invalidPriority(editPriority)
				}
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return sagaNotFound(id)
			}
			return err
		}

		fmt.Printf("Updated saga %s\n", sg.ID)
		return nil
	},
}

func init() {
	editCmd.Flags().StringVar(&editTitle, "title", "", "New title")
	editCmd.Flags().StringVar(&editDesc, "desc", "", "New description")
	editCmd.Flags().StringVar(&editPlan, "plan", "", "Implementation plan (empty to clear)")
	editCmd.Flags().StringVar(&editDeadline, "deadline", "", "Set deadline (YYYYMMDD) or empty to clear")
	editCmd.Flags().StringVar(&editPriority, "priority", "", "Set priority (high, normal, low)")
	rootCmd.AddCommand(editCmd)
}

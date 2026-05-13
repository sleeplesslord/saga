package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var planFile string

var planCmd = &cobra.Command{
	Use:   "plan <id> [plan-text]",
	Short: "View or set implementation plan for a saga",
	Long: `View or set the implementation plan for a saga.

The plan field stores how you intend to implement a task, separate from
the description (which stores what and why). This keeps implementation
details from cluttering the task description.

When called with just an ID, shows the current plan.
When called with text arguments, sets the plan.
Use --file to read plan from a file.
Use --clear to remove the plan.

Examples:
  sg plan abc123                              # View current plan
  sg plan abc123 "1. Add migration\\n2. Update model\\n3. Add tests"
  sg plan abc123 --file plan.md               # Set plan from file
  sg plan abc123 --clear                      # Remove plan`,
	Args: cobra.MinimumNArgs(1),
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

		// --clear: remove plan
		if planClear {
			sg.Plan = ""
			sg.UpdatedAt = time.Now()
			sg.AddHistory("edited", "Cleared plan")
			if err := st.Update(sg); err != nil {
				return fmt.Errorf("updating saga: %w", err)
			}
			fmt.Printf("Cleared plan for saga %s\n", sg.ID)
			return nil
		}

		// No extra args and no --file: view mode
		if len(args) < 2 && planFile == "" {
			if sg.Plan == "" {
				fmt.Printf("No plan set for saga %s\n", sg.ID)
			} else {
				fmt.Printf("Plan for %s:\n%s\n", sg.ID, formatDescription(sg.Plan))
			}
			return nil
		}

		// Set plan
		var newPlan string
		if planFile != "" {
			data, err := os.ReadFile(planFile)
			if err != nil {
				return fmt.Errorf("reading plan file: %w", err)
			}
			newPlan = string(data)
		} else {
			newPlan = strings.Join(args[1:], " ")
		}

		sg.Plan = newPlan
		sg.UpdatedAt = time.Now()
		sg.AddHistory("edited", "Updated plan")

		if err := st.Update(sg); err != nil {
			return fmt.Errorf("updating saga: %w", err)
		}

		fmt.Printf("Updated plan for saga %s\n", sg.ID)
		return nil
	},
}

var planClear bool

func init() {
	planCmd.Flags().StringVar(&planFile, "file", "", "Read plan from file")
	planCmd.Flags().BoolVar(&planClear, "clear", false, "Clear the plan")
	rootCmd.AddCommand(planCmd)
}

package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sleeplesslord/saga/internal/saga"
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
Pipe or heredoc to stdin to set multi-line plans.
Use --clear to remove the plan.

Examples:
  sg plan abc123                              # View current plan
  sg plan abc123 "1. Add migration\\n2. Update model\\n3. Add tests"
  sg plan abc123 --file plan.md               # Set plan from file
  sg plan abc123 <<'EOF'                      # Set plan from heredoc
  1. Add migration
  2. Update model
  3. Add tests
  EOF
  sg plan abc123 - < plan.md                  # Set plan from stdin
  sg plan abc123 --clear                      # Remove the plan`,
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
			cleared, err := st.Mutate(id, func(sg *saga.Saga) error {
				sg.Plan = ""
				sg.UpdatedAt = time.Now()
				sg.AddHistory("edited", "Cleared plan")
				return nil
			})
			if err != nil {
				return fmt.Errorf("updating saga: %w", err)
			}
			fmt.Printf("Cleared plan for saga %s\n", cleared.ID)
			return nil
		}

		// Input source priority: --file > `-` (explicit stdin) > positional args
		// > stdin.
		//
		// `saga plan <id> - < plan.md` is documented above, so `-` has to mean
		// "read stdin" rather than being taken as one-character plan text. An
		// unannounced stdin is only probed when no plan text was given, and that
		// read is bounded (see readStdinIfReady) — view mode is the common case
		// and it must not block on a stdin that never reaches EOF.
		hasFile := planFile != ""
		explicitStdin := len(args) >= 2 && args[1] == "-"
		hasArgs := len(args) >= 2 && !explicitStdin

		var stdinData string
		switch {
		case hasFile:
		case explicitStdin:
			data, err := readStdin()
			if err != nil {
				return fmt.Errorf("reading plan from stdin: %w", err)
			}
			if data == "" {
				return fmt.Errorf("stdin was empty")
			}
			stdinData = data
		case !hasArgs:
			data, err := readStdinIfReady()
			if err != nil {
				return fmt.Errorf("reading plan from stdin: %w", err)
			}
			stdinData = data
		}

		// No input at all: view mode
		if !hasFile && stdinData == "" && !hasArgs {
			if sg.Plan == "" {
				fmt.Printf("No plan set for saga %s\n", sg.ID)
			} else {
				fmt.Printf("Plan for %s:\n%s\n", sg.ID, formatDescription(sg.Plan))
			}
			return nil
		}

		// Set plan
		var newPlan string
		switch {
		case hasFile:
			data, err := os.ReadFile(planFile)
			if err != nil {
				return fmt.Errorf("reading plan file: %w", err)
			}
			newPlan = string(data)
		case hasArgs:
			newPlan = strings.Join(args[1:], " ")
		default:
			newPlan = stdinData
		}

		updated, err := st.Mutate(id, func(sg *saga.Saga) error {
			sg.Plan = newPlan
			sg.UpdatedAt = time.Now()
			sg.AddHistory("edited", "Updated plan")
			return nil
		})
		if err != nil {
			return fmt.Errorf("updating saga: %w", err)
		}

		fmt.Printf("Updated plan for saga %s\n", updated.ID)
		return nil
	},
}

var planClear bool

func init() {
	planCmd.Flags().StringVar(&planFile, "file", "", "Read plan from file")
	planCmd.Flags().BoolVar(&planClear, "clear", false, "Clear the plan")
	rootCmd.AddCommand(planCmd)
}

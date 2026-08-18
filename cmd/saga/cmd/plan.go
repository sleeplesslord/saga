package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var planFile string

// isPipedStdin returns true when data is available on stdin
// (piped input or heredoc), as opposed to an interactive terminal.
func isPipedStdin() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&syscall.S_IFMT != syscall.S_IFCHR
}

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

		// Determine input source priority: --file > stdin > positional args
		// Note: isPipedStdin() returns true for /dev/null redirects (e.g. Go's
		// exec.Command with nil Stdin), so we read stdin eagerly and fall
		// through to args if stdin is empty.
		hasFile := planFile != ""
		hasStdin := isPipedStdin()
		hasArgs := len(args) >= 2

		// Read stdin eagerly if available, but don't treat empty stdin as input
		var stdinData string
		if hasStdin {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading plan from stdin: %w", err)
			}
			stdinData = strings.TrimRight(string(data), "\n")
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
		if hasFile {
			data, err := os.ReadFile(planFile)
			if err != nil {
				return fmt.Errorf("reading plan file: %w", err)
			}
			newPlan = string(data)
		} else if stdinData != "" {
			newPlan = stdinData
		} else {
			newPlan = strings.Join(args[1:], " ")
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

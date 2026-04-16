package cmd

import (
	"fmt"
	"os"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var wontdoReason string
var wontdoCascade bool
var wontdoQuiet bool

var wontdoCmd = &cobra.Command{
	Use:   "wontdo <id>",
	Short: "Mark saga as won't-do",
	Long: `Mark a saga as won't-do — a terminal state distinct from "done".

Use for sagas that were abandoned, rejected, or obsoleted.
Reason is optional but recommended for later retrospection.

Use --cascade to also mark all active sub-sagas as wontdo.
Use --quiet to suppress the runes hint (also auto-suppressed when not a TTY).

Examples:
  sg wontdo abc123
  sg wontdo abc123 --reason "Requirements changed"
  sg wontdo abc123 --reason "Superseded by Playwright CLI" --cascade`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ids := args

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		for _, id := range ids {
			sg, err := st.GetByID(id)
			if err != nil {
				return sagaNotFound(id)
			}

			// Check for active children
			hasActiveChildren, err := st.HasActiveChildren(id)
			if err != nil {
				return fmt.Errorf("checking children: %w", err)
			}
			if hasActiveChildren && !wontdoCascade {
				return activeChildren(id)
			}

			// Cascade: mark all active children as wontdo first
			if wontdoCascade && hasActiveChildren {
				children, err := st.GetChildren(id)
				if err != nil {
					return fmt.Errorf("getting children: %w", err)
				}
				for _, child := range children {
					if child.Status == saga.StatusActive || child.Status == saga.StatusPaused {
						child.SetStatus(saga.StatusWontDo)
						if wontdoReason != "" {
							child.AddHistory("wontdo", wontdoReason)
						}
						if err := st.Update(child); err != nil {
							return fmt.Errorf("updating child %s: %w", child.ID, err)
						}
						if !wontdoQuiet && isTerminal() {
							fmt.Printf("Marked sub-saga %s as wontdo\n", child.ID)
						}
					}
				}
			}

			sg.SetStatus(saga.StatusWontDo)

			// Log reason if provided
			if wontdoReason != "" {
				sg.AddHistory("wontdo", wontdoReason)
			}

			if err := st.Update(sg); err != nil {
				return fmt.Errorf("updating saga: %w", err)
			}

			if !wontdoQuiet && isTerminal() {
				fmt.Printf("Marked saga %s as wontdo\n", sg.ID)

				// Hint about runes if installed
				if isRunesInstalled() {
					fmt.Println()
					fmt.Println("💡 This saga involved problem-solving. Capture the knowledge?")
					fmt.Println("   runes add \"<title>\" --problem \"...\" --solution \"...\" --saga " + sg.ID)
					fmt.Println("   runes edit <id> --learned \"<insight>\"")
				}
			}
		}

		return nil
	},
}

// isTerminal returns true if stdout is a terminal
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func init() {
	wontdoCmd.Flags().StringVar(&wontdoReason, "reason", "", "Reason for won't-do (logged in history)")
	wontdoCmd.Flags().BoolVar(&wontdoCascade, "cascade", false, "Also mark all active sub-sagas as wontdo")
	wontdoCmd.Flags().BoolVar(&wontdoQuiet, "quiet", false, "Suppress hints and non-essential output")
	rootCmd.AddCommand(wontdoCmd)
}

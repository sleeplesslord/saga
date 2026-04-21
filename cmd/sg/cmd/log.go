package cmd

import (
	"fmt"
	"os"

	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var logFile string

var logCmd = &cobra.Command{
	Use:   "log <id> <message>",
	Short: "Add a work log entry to a saga",
	Long: `Add a custom log entry to a saga's history. Useful for tracking work progress,
decisions, or notes during development.

Use --file to read the log message from a file instead of command line.

Examples:
  sg log abc123 "Started implementing OAuth"
  sg log abc123 "Fixed the timeout issue in auth flow"
  sg log abc123 --file notes.md`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		var message string
		if logFile != "" {
			// Read from file
			data, err := os.ReadFile(logFile)
			if err != nil {
				return fmt.Errorf("reading log file: %w", err)
			}
			message = string(data)
		} else {
			if len(args) < 2 {
				return fmt.Errorf("message required (or use --file)")
			}
			message = args[1]
		}

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		sg, err := st.GetByID(id)
		if err != nil {
			return sagaNotFound(id)
		}

		sg.AddHistory("log", message)

		if err := st.Update(sg); err != nil {
			return fmt.Errorf("updating saga: %w", err)
		}

		fmt.Printf("Added log to saga %s\n", id)
		return nil
	},
}

func init() {
	logCmd.Flags().StringVar(&logFile, "file", "", "Read log message from file")
	rootCmd.AddCommand(logCmd)
}

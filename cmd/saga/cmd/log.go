package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var logFile string

var logCmd = &cobra.Command{
	Use:     "log <id> [message]",
	Aliases: []string{"comment"},
	Short:   "Add a work log entry to a saga",
	Long: `Add a custom log entry to a saga's history. Useful for tracking work progress,
decisions, or notes during development.

Use --file to read the log message from a file.
Pipe or heredoc to stdin for multi-line messages, or pass "-" to read stdin
explicitly. A message given as an argument always wins over stdin.

Examples:
  sg log abc123 "Started implementing OAuth"
  sg log abc123 "Fixed the timeout issue in auth flow"
  sg log abc123 --file notes.md
  sg log abc123 - < notes.md
  sg log abc123 <<'EOF'
  Investigated the auth timeout issue. Root cause: connection pool
  exhaustion under load. Fix: increase pool size and add backoff.
  EOF`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Input source priority: --file > `-` (explicit stdin) > positional
		// message > stdin.
		//
		// The positional message comes before an unannounced stdin deliberately:
		// an explicit argument is unambiguous, and consulting stdin first meant
		// every call touched a stdin that may never produce EOF. `-` stays ahead
		// of it so a caller can still name stdin outright, and heredoc/pipe
		// callers who pass no message reach the probe.
		var message string
		switch {
		case logFile != "":
			data, err := os.ReadFile(logFile)
			if err != nil {
				return fmt.Errorf("reading log file: %w", err)
			}
			message = string(data)
		case len(args) >= 2 && args[1] == "-":
			data, err := readStdin()
			if err != nil {
				return fmt.Errorf("reading log from stdin: %w", err)
			}
			if data == "" {
				return fmt.Errorf("stdin was empty")
			}
			message = data
		case len(args) >= 2:
			message = args[1]
		default:
			stdinData, err := readStdinIfReady()
			if err != nil {
				return fmt.Errorf("reading log from stdin: %w", err)
			}
			if stdinData == "" {
				return fmt.Errorf("message required (or use --file, `-` to read stdin, or pipe stdin)")
			}
			message = stdinData
		}

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		// Mutate re-reads the record under the store lock, so a concurrent status
		// change (e.g. `saga done`) isn't overwritten by a stale copy.
		if _, err := st.Mutate(id, func(sg *saga.Saga) error {
			sg.AddHistory("log", message)
			return nil
		}); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return sagaNotFound(id)
			}
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

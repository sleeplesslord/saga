package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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
Pipe or heredoc to stdin for multi-line messages.

Examples:
  sg log abc123 "Started implementing OAuth"
  sg log abc123 "Fixed the timeout issue in auth flow"
  sg log abc123 --file notes.md
  sg log abc123 <<'EOF'
  Investigated the auth timeout issue. Root cause: connection pool
  exhaustion under load. Fix: increase pool size and add backoff.
  EOF`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Determine input source priority: --file > stdin > positional args
		// Note: isPipedStdin() returns true for /dev/null redirects (e.g. Go's
		// exec.Command with nil Stdin), so we read stdin eagerly and fall
		// through to args if stdin is empty.
		var message string
		if logFile != "" {
			data, err := os.ReadFile(logFile)
			if err != nil {
				return fmt.Errorf("reading log file: %w", err)
			}
			message = string(data)
		} else if isPipedStdin() {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading log from stdin: %w", err)
			}
			stdinData := strings.TrimRight(string(data), "\n")
			if stdinData != "" {
				message = stdinData
			} else {
				// Empty stdin (/dev/null redirect) — fall through to args
				if len(args) < 2 {
					return fmt.Errorf("message required (or use --file, or pipe stdin)")
				}
				message = args[1]
			}
		} else {
			if len(args) < 2 {
				return fmt.Errorf("message required (or use --file, or pipe stdin)")
			}
			message = args[1]
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

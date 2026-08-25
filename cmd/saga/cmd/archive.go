package cmd

import (
	"fmt"
	"time"

	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var (
	archiveDays       int
	archiveDryRun     bool
	archiveScopeLocal bool
	archiveScopeGlobal bool
)

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Archive old done/wontdo sagas",
	Long: `Move sagas with terminal status (done or wontdo) that haven't been
modified in the specified number of days to a separate archive file.

Archived sagas are moved from sagas.jsonl to archive.jsonl in the same
.saga/ directory. Plan files for archived sagas are also moved.

Archiving takes a saga out of the active store, so it stops showing up in
list, ready, and search. Read the archive with 'saga list --archived',
'saga search <query> --archived', or 'saga status <id>' (which falls back
to the archive when the active store has no such ID).

When a local .saga/ exists, only local sagas are archived by default.
Use --global to also archive global sagas, or --global --local for both.

Use --days to control the age threshold (default: 30).
Use --dry-run to preview what would be archived without making changes.

Examples:
  saga archive
  saga archive --days 60
  saga archive --dry-run
  saga archive --global`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		cutoff := time.Now().AddDate(0, 0, -archiveDays)

		// Determine scopes — same logic as list: local-only by default
		// when local exists, unless --global is specified
		var scopes []store.Scope
		if archiveScopeLocal && archiveScopeGlobal {
			scopes = []store.Scope{store.ScopeLocal, store.ScopeGlobal}
		} else if archiveScopeLocal {
			scopes = []store.Scope{store.ScopeLocal}
		} else if archiveScopeGlobal {
			scopes = []store.Scope{store.ScopeGlobal}
		} else {
			// Default: local-only if local exists, otherwise global
			if st.HasLocal() {
				scopes = []store.Scope{store.ScopeLocal}
			} else {
				scopes = []store.Scope{store.ScopeGlobal}
			}
		}

		totalArchived := 0
		for _, scope := range scopes {
			count, err := st.Archive(scope, cutoff, archiveDryRun)
			if err != nil {
				return fmt.Errorf("archiving %s sagas: %w", scopeName(scope), err)
			}
			if count > 0 {
				fmt.Printf("  %s: %d saga(s)\n", scopeName(scope), count)
			}
			totalArchived += count
		}

		if archiveDryRun {
			if totalArchived == 0 {
				fmt.Printf("Dry run: no sagas older than %d days to archive.\n", archiveDays)
			} else {
				fmt.Printf("Dry run: %d saga(s) would be archived (older than %d days).\n", totalArchived, archiveDays)
			}
		} else {
			if totalArchived == 0 {
				fmt.Printf("No sagas older than %d days to archive.\n", archiveDays)
			} else {
				fmt.Printf("Archived %d saga(s) older than %d days.\n", totalArchived, archiveDays)
				fmt.Println("They no longer appear in list/search — read them with: saga list --archived")
			}
		}

		return nil
	},
}

func scopeName(s store.Scope) string {
	switch s {
	case store.ScopeLocal:
		return "project"
	case store.ScopeGlobal:
		return "global"
	default:
		return "unknown"
	}
}

func init() {
	archiveCmd.Flags().IntVar(&archiveDays, "days", 30, "Archive sagas not modified in this many days")
	archiveCmd.Flags().BoolVar(&archiveDryRun, "dry-run", false, "Preview what would be archived without making changes")
	archiveCmd.Flags().BoolVarP(&archiveScopeLocal, "local", "l", false, "Archive only project sagas")
	archiveCmd.Flags().BoolVarP(&archiveScopeGlobal, "global", "g", false, "Include global sagas (when project exists, archive shows local by default)")
	rootCmd.AddCommand(archiveCmd)
}


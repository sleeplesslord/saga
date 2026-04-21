package cmd

import (
	"fmt"
	"strings"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var (
	searchLabels   []string
	searchStatus   string
	searchPriority string
)

// Column widths for search table (no CLAIM column, compact format)
var searchWidths = []int{10, 42, 7, 5, 5, 5, 13}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search sagas",
	Long: `Search for sagas by title, ID, or content.

The query matches against saga titles and IDs. Use flags to filter by
labels, status, or priority.

Examples:
  sg search "auth"                    # Search for "auth" in titles/IDs
  sg search "" --label bug            # All sagas with bug label
  sg search "" --status active        # All active sagas
  sg search "" --priority high        # All high priority sagas
  sg search "fix" --label urgent      # Search with label filter`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := ""
		if len(args) > 0 {
			query = strings.ToLower(args[0])
		}

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		// Default to local-only when project exists (consistent with sg list)
		var scopes []store.Scope
		if st.HasLocal() {
			scopes = []store.Scope{store.ScopeLocal}
		} else {
			scopes = []store.Scope{store.ScopeGlobal}
		}

		sagas, err := st.LoadAll(scopes...)
		if err != nil {
			return fmt.Errorf("loading sagas: %w", err)
		}

		var matches []*saga.Saga
		for _, sg := range sagas {
			// Text search
			if query != "" {
				titleMatch := strings.Contains(strings.ToLower(sg.Title), query)
				idMatch := strings.Contains(strings.ToLower(sg.ID), query)
				descMatch := strings.Contains(strings.ToLower(sg.Description), query)
				if !titleMatch && !idMatch && !descMatch {
					continue
				}
			}

			// Label filter
			if len(searchLabels) > 0 {
				hasAllLabels := true
				for _, label := range searchLabels {
					if !sg.HasLabel(label) {
						hasAllLabels = false
						break
					}
				}
				if !hasAllLabels {
					continue
				}
			}

			// Status filter
			if searchStatus != "" && string(sg.Status) != searchStatus {
				continue
			}

			// Priority filter
			if searchPriority != "" && string(sg.Priority) != searchPriority {
				continue
			}

			matches = append(matches, sg)
		}

		if len(matches) == 0 {
			fmt.Println("No sagas found matching your search.")
			return nil
		}

		fmt.Printf("Found %d saga(s):\n\n", len(matches))

		// Print table header
		printTableHeader(
			[]string{"ID", "TITLE", "STATUS", "PRI", "DATE", "DUE", "LABELS"},
			searchWidths,
		)

		for _, sg := range matches {
			// Build labels display
			labelsStr := ""
			if len(sg.Labels) > 0 {
				labelsStr = strings.Join(sg.Labels, ",")
			}

			// Build priority display (show only if not normal)
			priorityStr := ""
			if sg.Priority != saga.PriorityNormal {
				priorityStr = string(sg.Priority)
			}

			cells := []string{
				sg.ID,
				sg.Title,
				string(sg.Status),
				priorityStr,
				sg.UpdatedAt.Format("01-02"),
				formatDeadline(sg.Deadline),
				labelsStr,
			}
			printTableRow(cells, searchWidths, "")
		}

		return nil
	},
}

func init() {
	searchCmd.Flags().StringArrayVar(&searchLabels, "label", nil, "Filter by label (can specify multiple)")
	searchCmd.Flags().StringVar(&searchStatus, "status", "", "Filter by status (active, paused, done, wontdo)")
	searchCmd.Flags().StringVar(&searchPriority, "priority", "", "Filter by priority (high, normal, low)")
	rootCmd.AddCommand(searchCmd)
}

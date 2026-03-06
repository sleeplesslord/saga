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

		sagas, err := st.LoadAll()
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

		// Use consistent formatting with sg list
		// Terminal width: 160 chars (modern terminals)
		terminalWidth := 160

		for _, sg := range matches {
			// Build metadata strings
			metaParts := []string{}

			// Status
			metaParts = append(metaParts, string(sg.Status))

			// Labels
			if len(sg.Labels) > 0 {
				labelStr := strings.Join(sg.Labels, ", ")
				if len(labelStr) > 20 {
					labelStr = labelStr[:17] + "..."
				}
				metaParts = append(metaParts, "["+labelStr+"]")
			}

			metaStr := strings.Join(metaParts, " ")

			// Calculate available space for title
			// Format: ID + title + metadata + spacing
			availableWidth := terminalWidth - 6 - len(metaStr) - 3 // 3 for spacing
			if availableWidth < 30 {
				availableWidth = 30 // Minimum title space
			}

			title := sg.Title
			if len(title) > availableWidth {
				title = title[:availableWidth-3] + "..."
			}

			fmt.Printf("%-6s %s %s\n", sg.ID, title, metaStr)
		}

		return nil
	},
}

func init() {
	searchCmd.Flags().StringArrayVar(&searchLabels, "label", nil, "Filter by label (can specify multiple)")
	searchCmd.Flags().StringVar(&searchStatus, "status", "", "Filter by status (active, paused, done)")
	searchCmd.Flags().StringVar(&searchPriority, "priority", "", "Filter by priority (high, normal, low)")
	rootCmd.AddCommand(searchCmd)
}

package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hbn/saga/internal/saga"
	"github.com/hbn/saga/internal/store"
	"github.com/spf13/cobra"
)

var (
	showAll       bool
	scopeLocal    bool
	scopeGlobal   bool
	labelFilter   string
	showUnclaimed bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List sagas",
	Long: `List sagas. By default shows active sagas from both global and local (if in project) scopes.

Use flags to filter by scope or show all statuses.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		// Determine scopes
		var scopes []store.Scope
		if scopeLocal && scopeGlobal {
			scopes = []store.Scope{store.ScopeLocal, store.ScopeGlobal}
		} else if scopeLocal {
			scopes = []store.Scope{store.ScopeLocal}
		} else if scopeGlobal {
			scopes = []store.Scope{store.ScopeGlobal}
		} else {
			scopes = []store.Scope{store.ScopeGlobal}
			if st.HasLocal() {
				scopes = append(scopes, store.ScopeLocal)
			}
		}

		sagas, err := st.LoadAll(scopes...)
		if err != nil {
			return fmt.Errorf("loading sagas: %w", err)
		}

		if len(sagas) == 0 {
			fmt.Println("No sagas found.")
			return nil
		}

		// Sort by priority (high > normal > low), then by updated time
		sort.Slice(sagas, func(i, j int) bool {
			priorityOrder := map[saga.Priority]int{
				saga.PriorityHigh:   0,
				saga.PriorityNormal: 1,
				saga.PriorityLow:    2,
			}
			pi, pj := priorityOrder[sagas[i].Priority], priorityOrder[sagas[j].Priority]
			if pi != pj {
				return pi < pj
			}
			return sagas[i].UpdatedAt.After(sagas[j].UpdatedAt)
		})

		// Show scope info
		if st.HasLocal() && !scopeLocal && !scopeGlobal {
			fmt.Printf("(Showing global + project sagas from %s)\n\n", filepath.Dir(st.LocalPath()))
		}

		fmt.Printf("%-6s %s\n", "ID", "Title Status Updated [labels] [claimed]")
		fmt.Println(strings.Repeat("-", 155))

		// Build parent lookup
		children := make(map[string][]*saga.Saga)
		for _, sg := range sagas {
			if sg.ParentID != "" {
				children[sg.ParentID] = append(children[sg.ParentID], sg)
			}
		}

		// Print root sagas and their children
		for _, sg := range sagas {
			if sg.ParentID != "" {
				continue
			}
			if !showAll && sg.Status != saga.StatusActive {
				continue
			}
			if labelFilter != "" && !sg.HasLabel(labelFilter) {
				continue
			}
			if showUnclaimed && sg.IsClaimed() {
				continue
			}
			printSagaWithIndent(sg, 0, showAll, children, labelFilter)
		}

		return nil
	},
}

const maxDisplayDepth = 50

func printSagaWithIndent(sg *saga.Saga, indent int, showAll bool, children map[string][]*saga.Saga, labelFilter string) {
	if indent > maxDisplayDepth {
		fmt.Printf("%-6s %s[Max depth reached]\n", sg.ID, strings.Repeat("  ", indent))
		return
	}

	indentStr := ""
	if indent > 0 {
		indentStr = strings.Repeat("  ", indent) + "↳ "
	}

	// Build metadata strings
	metaParts := []string{}
	
	// Status
	metaParts = append(metaParts, string(sg.Status))
	
	// Updated time
	metaParts = append(metaParts, sg.UpdatedAt.Format("Jan 02 15:04"))
	
	// Labels (compact)
	if len(sg.Labels) > 0 {
		labelStr := strings.Join(sg.Labels, ",")
		if len(labelStr) > 15 {
			labelStr = labelStr[:12] + "..."
		}
		metaParts = append(metaParts, "["+labelStr+"]")
	}
	
	// Claim status (compact)
	if sg.IsClaimed() {
		metaParts = append(metaParts, "[claimed:"+sg.ClaimedBy+"]")
	}
	
	metaStr := strings.Join(metaParts, " ")
	
	// Calculate available space for title
	// Format: ID + indent + title + metadata
	// Terminal width: 160 chars (modern terminals)
	terminalWidth := 160
	availableWidth := terminalWidth - 6 - len(indentStr) - len(metaStr) - 3 // 3 for spacing
	if availableWidth < 30 {
		availableWidth = 30 // Minimum title space
	}
	
	title := sg.Title
	if len(title) > availableWidth {
		title = title[:availableWidth-3] + "..."
	}
	
	fmt.Printf("%-6s %s%s %s\n", sg.ID, indentStr, title, metaStr)

	for _, child := range children[sg.ID] {
		if !showAll && child.Status != saga.StatusActive {
			continue
		}
		if labelFilter != "" && !child.HasLabel(labelFilter) {
			continue
		}
		printSagaWithIndent(child, indent+1, showAll, children, labelFilter)
	}
}

func init() {
	listCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all sagas including done")
	listCmd.Flags().BoolVarP(&scopeLocal, "local", "l", false, "Show only project sagas")
	listCmd.Flags().BoolVarP(&scopeGlobal, "global", "g", false, "Show only global sagas")
	listCmd.Flags().StringVar(&labelFilter, "label", "", "Filter by label")
	listCmd.Flags().BoolVar(&showUnclaimed, "unclaimed", false, "Show only unclaimed sagas")
	rootCmd.AddCommand(listCmd)
}

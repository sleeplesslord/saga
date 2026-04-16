package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var (
	showAll       bool
	scopeLocal    bool
	scopeGlobal   bool
	labelFilter   string
	showUnclaimed bool
	statusFilter  string
	priorityFilter string
	mineFilter    bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List sagas",
	Long: `List sagas. When a local .saga/ exists, shows local sagas by default.
Use --global to include global sagas. Use flags to filter by scope, status, label, or priority.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		// Determine scopes
		// When local exists, default to local-only (user can add --global)
		var scopes []store.Scope
		if scopeLocal && scopeGlobal {
			scopes = []store.Scope{store.ScopeLocal, store.ScopeGlobal}
		} else if scopeLocal {
			scopes = []store.Scope{store.ScopeLocal}
		} else if scopeGlobal {
			scopes = []store.Scope{store.ScopeGlobal}
		} else {
			// Default: local-only if local exists, otherwise global
			if st.HasLocal() {
				scopes = []store.Scope{store.ScopeLocal}
			} else {
				scopes = []store.Scope{store.ScopeGlobal}
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
		scopeDesc := "global"
		if len(scopes) == 2 {
			scopeDesc = "global + project"
		} else if scopes[0] == store.ScopeLocal {
			scopeDesc = "project"
		}
		fmt.Printf("(Showing %s sagas", scopeDesc)
		if st.HasLocal() {
			fmt.Printf(" from %s", filepath.Dir(st.LocalPath()))
		}
		fmt.Printf(")\n\n")

		fmt.Printf("%-6s %s\n", "ID", "Title Status Updated [labels] [claimed]")
		fmt.Println(strings.Repeat("-", 155))

		// Resolve agent name for --mine filter
		var mineAgent string
		if mineFilter {
			mineAgent = resolveAgentName()
		}

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
			if statusFilter != "" && string(sg.Status) != statusFilter {
				continue
			}
			if priorityFilter != "" && string(sg.Priority) != priorityFilter {
				continue
			}
			if mineFilter && !isMine(sg, mineAgent) {
				continue
			}
			printSagaWithIndent(sg, 0, showAll, children, labelFilter, statusFilter, priorityFilter, mineFilter, mineAgent)
		}

		return nil
	},
}

const maxDisplayDepth = 50

func printSagaWithIndent(sg *saga.Saga, indent int, showAll bool, children map[string][]*saga.Saga, labelFilter string, statusFilter string, priorityFilter string, mineFilter bool, mineAgent string) {
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

	// Deadline
	if sg.Deadline != "" {
		metaParts = append(metaParts, "[due:"+sg.Deadline+"]")
	}

	// Labels (compact)
	if len(sg.Labels) > 0 {
		labelStr := strings.Join(sg.Labels, ",")
		if len(labelStr) > 15 {
			labelStr = labelStr[:12] + "..."
		}
		metaParts = append(metaParts, "["+labelStr+"]")
	}

	// Claim status (compact) with remaining time
	if sg.IsClaimed() {
		remaining := timeUntilExpiry(sg)
		if remaining != "" {
			metaParts = append(metaParts, "[claimed:"+sg.ClaimedBy+", "+remaining+"]")
		} else {
			metaParts = append(metaParts, "[claimed:"+sg.ClaimedBy+"]")
		}
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
		if statusFilter != "" && string(child.Status) != statusFilter {
			continue
		}
		if priorityFilter != "" && string(child.Priority) != priorityFilter {
			continue
		}
		if mineFilter && !isMine(child, mineAgent) {
			continue
		}
		printSagaWithIndent(child, indent+1, showAll, children, labelFilter, statusFilter, priorityFilter, mineFilter, mineAgent)
	}
}

// resolveAgentName returns the current agent identity for --mine filtering
func resolveAgentName() string {
	agent := os.Getenv("USER")
	if agent == "" {
		agent = "unknown"
	}
	return agent
}

// isMine checks if a saga is claimed by the current agent
func isMine(sg *saga.Saga, agent string) bool {
	if !sg.IsClaimed() {
		return false
	}
	// Match on agent name prefix (before @ppid)
	parts := strings.SplitN(sg.ClaimedBy, "@", 2)
	agentParts := strings.SplitN(agent, "@", 2)
	return parts[0] == agentParts[0]
}

// timeUntilExpiry returns a human-readable remaining time string for a claimed saga
func timeUntilExpiry(sg *saga.Saga) string {
	if sg.ClaimedBy == "" {
		return ""
	}
	expiry := sg.ClaimExpiry()
	remaining := time.Until(expiry)
	if remaining <= 0 {
		return "expired"
	}
	hours := int(remaining.Hours())
	minutes := int(remaining.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%dm left", hours, minutes)
	}
	return fmt.Sprintf("%dm left", minutes)
}

func init() {
	listCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all sagas including done/wontdo")
	listCmd.Flags().BoolVarP(&scopeLocal, "local", "l", false, "Show only project sagas")
	listCmd.Flags().BoolVarP(&scopeGlobal, "global", "g", false, "Include global sagas (when project exists, list shows local by default)")
	listCmd.Flags().StringVar(&labelFilter, "label", "", "Filter by label")
	listCmd.Flags().BoolVar(&showUnclaimed, "unclaimed", false, "Show only unclaimed sagas")
	listCmd.Flags().StringVar(&statusFilter, "status", "", "Filter by status (active, paused, done, wontdo)")
	listCmd.Flags().StringVar(&priorityFilter, "priority", "", "Filter by priority (high, normal, low)")
	listCmd.Flags().BoolVar(&mineFilter, "mine", false, "Show only sagas claimed by me")
	rootCmd.AddCommand(listCmd)
}

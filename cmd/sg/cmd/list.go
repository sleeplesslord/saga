package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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

// Column widths for list table
var listWidths = []int{10, 34, 7, 5, 5, 5, 13, 18}

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

		// Sort by deadline (soonest first), then priority (high first), then updated
		sortByDeadlinePriorityUpdated(sagas)

		// Load claim duration from config
		claimDuration := st.ClaimDuration()

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

		// Print table header
		printTableHeader(
			[]string{"ID", "TITLE", "STATUS", "PRI", "DATE", "DUE", "LABELS", "CLAIM"},
			listWidths,
		)

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
			if showUnclaimed && sg.IsClaimedWithDuration(claimDuration) {
				continue
			}
			if statusFilter != "" && string(sg.Status) != statusFilter {
				continue
			}
			if priorityFilter != "" && string(sg.Priority) != priorityFilter {
				continue
			}
			if mineFilter && !isMine(sg, mineAgent, claimDuration) {
				continue
			}
			printSagaWithIndent(sg, 0, showAll, children, labelFilter, statusFilter, priorityFilter, mineFilter, mineAgent, claimDuration)
		}

		return nil
	},
}

const maxDisplayDepth = 50

func printSagaWithIndent(sg *saga.Saga, indent int, showAll bool, children map[string][]*saga.Saga, labelFilter string, statusFilter string, priorityFilter string, mineFilter bool, mineAgent string, claimDuration time.Duration) {
	if indent > maxDisplayDepth {
		titleStr := strings.Repeat("  ", indent) + "↳ " + "[Max depth reached]"
		printTableRow([]string{sg.ID, titleStr, "", "", "", "", "", ""}, listWidths, "")
		return
	}

	// Build title with indent prefix (keeps ID as first field for script parsing)
	titleStr := sg.Title
	if indent > 0 {
		titleStr = strings.Repeat("  ", indent) + "↳ " + titleStr
	}

	// Build claim display (always show full identity in list overview)
	claimStr := ""
	if sg.IsClaimedWithDuration(claimDuration) {
		claimStr = sg.ClaimedBy
		remaining := timeUntilExpiry(sg, claimDuration)
		if remaining != "" {
			claimStr += " " + remaining
		}
	}

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
		titleStr,
		string(sg.Status),
		priorityStr,
		sg.UpdatedAt.Format("01-02"),
		formatDeadline(sg.Deadline),
		labelsStr,
		claimStr,
	}

	printTableRow(cells, listWidths, "")

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
		if mineFilter && !isMine(child, mineAgent, claimDuration) {
			continue
		}
		printSagaWithIndent(child, indent+1, showAll, children, labelFilter, statusFilter, priorityFilter, mineFilter, mineAgent, claimDuration)
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

// isMine checks if a saga is claimed by the current process session
// Comparison is by ppid (process session), not username — same ppid means
// same shell/agent session. Username is just decoration.
func isMine(sg *saga.Saga, _ string, claimDuration time.Duration) bool {
	if !sg.IsClaimedWithDuration(claimDuration) {
		return false
	}
	claimParts := strings.SplitN(sg.ClaimedBy, "@", 2)
	currentPPID := fmt.Sprintf("%d", os.Getppid())
	return len(claimParts) == 2 && claimParts[1] == currentPPID
}

// timeUntilExpiry returns a human-readable remaining time string for a claimed saga
func timeUntilExpiry(sg *saga.Saga, claimDuration time.Duration) string {
	if sg.ClaimedBy == "" {
		return ""
	}
	expiry := sg.ClaimExpiryWithDuration(claimDuration)
	remaining := time.Until(expiry)
	if remaining <= 0 {
		return "expired"
	}
	hours := int(remaining.Hours())
	minutes := int(remaining.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
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

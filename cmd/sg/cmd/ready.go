package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var readyTake bool

// readyWidths returns column widths for the ready table (no STATUS column since all are active),
// using the configurable title width from the store config.
func readyWidths(st *store.Store) []int {
	tw := st.TitleWidth()
	return []int{10, tw, 5, 5, 13, 18}
}

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "List sagas ready to work on",
	Long: `Show active sagas that are not blocked by dependencies, children, or claims from others.

This helps you find what you can start working on right now.
Sagas claimed by you are included; sagas claimed by others are excluded.

Sort order: deadline (soonest first), then priority (high first), then recently updated.

Examples:
  sg ready              # List all ready sagas
  sg ready --take       # Claim the first ready saga`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		agent := os.Getenv("USER")
		if agent == "" {
			agent = "unknown"
		}

		// Load claim duration from config
		claimDuration := st.ClaimDuration()

		// Find ready sagas
		var ready []*saga.Saga
		for _, sg := range sagas {
			if !isReady(sg, st, agent, claimDuration) {
				continue
			}
			ready = append(ready, sg)
		}

		if len(ready) == 0 {
			fmt.Println("No sagas ready to work on.")
			fmt.Println("\nAll active sagas are either:")
			fmt.Println("  - Claimed by someone else")
			fmt.Println("  - Blocked by incomplete dependencies")
			fmt.Println("  - Have active sub-sagas that need completion")
			return nil
		}

		// Sort by deadline (closest first), then priority, then updated
		sortByDeadlinePriorityUpdated(ready)

		// --take: claim the first ready saga
		if readyTake {
			sg := ready[0]
			claimAgent := fmt.Sprintf("%s@%d", agent, os.Getppid())
			sg.ClaimWithDuration(claimAgent, claimDuration)
			if err := st.Update(sg); err != nil {
				return fmt.Errorf("claiming saga: %w", err)
			}
			fmt.Printf("Claimed saga %s: %s\n", sg.ID, sg.Title)
			return nil
		}

		fmt.Printf("Found %d saga(s) ready to work on:\n\n", len(ready))

		// Print table header
		printTableHeader(
			[]string{"ID", "TITLE", "PRI", "DUE", "LABELS", "CLAIM"},
			readyWidths(st),
		)

		for _, sg := range ready {
			// Build priority display (show only if not normal)
			priorityStr := ""
			if sg.Priority != saga.PriorityNormal {
				priorityStr = string(sg.Priority)
			}

			// Build labels display
			labelsStr := ""
			if len(sg.Labels) > 0 {
				labelsStr = strings.Join(sg.Labels, ",")
			}

			// Build claim display
			claimStr := ""
			if sg.IsClaimedWithDuration(claimDuration) {
				claimParts := strings.SplitN(sg.ClaimedBy, "@", 2)
				currentPPID := fmt.Sprintf("%d", os.Getppid())
				if len(claimParts) == 2 && claimParts[1] == currentPPID {
					claimStr = "mine"
				} else {
					claimStr = sg.ClaimedBy
				}
				remaining := timeUntilExpiry(sg, claimDuration)
				if remaining != "" {
					claimStr += " " + remaining
				}
			}

			cells := []string{
				sg.ID,
				sg.Title,
				priorityStr,
				formatDeadline(sg.Deadline),
				labelsStr,
				claimStr,
			}
			printTableRow(cells, readyWidths(st), "")
		}

		return nil
	},
}

// isReady checks if a saga can be worked on by the current process session
func isReady(sg *saga.Saga, st *store.Store, agent string, claimDuration time.Duration) bool {
	// Must be active
	if sg.Status != saga.StatusActive {
		return false
	}

	// Exclude claimed by other sessions (ppid comparison)
	// Same ppid = same shell/agent session = allowed
	// Different ppid = different session = excluded
	if sg.IsClaimedWithDuration(claimDuration) {
		claimParts := strings.SplitN(sg.ClaimedBy, "@", 2)
		currentPPID := fmt.Sprintf("%d", os.Getppid())
		if len(claimParts) == 2 && claimParts[1] != currentPPID {
			return false
		}
	}

	// Check for incomplete dependencies
	hasIncompleteDeps, _, err := st.HasIncompleteDependencies(sg.ID)
	if err == nil && hasIncompleteDeps {
		return false
	}

	// Check for active children
	hasActiveChildren, err := st.HasActiveChildren(sg.ID)
	if err == nil && hasActiveChildren {
		return false
	}

	// Check if parent is blocked
	if sg.ParentID != "" {
		parent, err := st.GetByID(sg.ParentID)
		if err == nil && parent != nil {
			// Parent must be active
			if parent.Status != saga.StatusActive {
				return false
			}
			// Parent must not have incomplete dependencies
			hasIncompleteDeps, _, err := st.HasIncompleteDependencies(parent.ID)
			if err == nil && hasIncompleteDeps {
				return false
			}
		}
	}

	return true
}

// sortByDeadlinePriorityUpdated sorts sagas by deadline (soonest first),
// then priority (high first), then recently updated
func sortByDeadlinePriorityUpdated(sagas []*saga.Saga) {
	priorityOrder := map[saga.Priority]int{
		saga.PriorityHigh:   0,
		saga.PriorityNormal: 1,
		saga.PriorityLow:    2,
	}

	sort.Slice(sagas, func(i, j int) bool {
		// Deadline sorting (empty deadlines go last)
		di, dj := sagas[i].Deadline, sagas[j].Deadline
		if di != "" && dj == "" {
			return true
		}
		if di == "" && dj != "" {
			return false
		}
		if di != "" && dj != "" && di != dj {
			return di < dj // Earlier deadline first (YYYYMMDD sorts lexicographically)
		}

		// Then by priority
		pi, pj := priorityOrder[sagas[i].Priority], priorityOrder[sagas[j].Priority]
		if pi != pj {
			return pi < pj
		}

		return sagas[i].UpdatedAt.After(sagas[j].UpdatedAt)
	})
}

func init() {
	readyCmd.Flags().BoolVar(&readyTake, "take", false, "Claim the first ready saga")
	rootCmd.AddCommand(readyCmd)
}

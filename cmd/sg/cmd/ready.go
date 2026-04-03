package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "List sagas ready to work on",
	Long: `Show active sagas that are not blocked by dependencies, children, or claims.

This helps you find what you can start working on right now.

Examples:
  sg ready              # List all ready sagas
  sg ready --take       # Claim the first ready saga`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		sagas, err := st.LoadAll()
		if err != nil {
			return fmt.Errorf("loading sagas: %w", err)
		}

		// Find ready sagas
		var ready []*saga.Saga
		for _, sg := range sagas {
			if !isReady(sg, st) {
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
		sort.Slice(ready, func(i, j int) bool {
			// Deadline sorting (empty deadlines go last)
			hi, hj := ready[i].Deadline, ready[j].Deadline
			if hi != "" && hj == "" {
				return true
			}
			if hi == "" && hj != "" {
				return false
			}
			if hi != "" && hj != "" && hi != hj {
				return hi < hj // Earlier deadline first (YYYYMMDD sorts lexicographically)
			}

			// Then by priority
			priorityOrder := map[saga.Priority]int{
				saga.PriorityHigh:   0,
				saga.PriorityNormal: 1,
				saga.PriorityLow:    2,
			}
			pi, pj := priorityOrder[ready[i].Priority], priorityOrder[ready[j].Priority]
			if pi != pj {
				return pi < pj
			}
			return ready[i].UpdatedAt.After(ready[j].UpdatedAt)
		})

		fmt.Printf("Found %d saga(s) ready to work on:\n\n", len(ready))

		for _, sg := range ready {
			priorityStr := ""
			if sg.Priority != saga.PriorityNormal {
				priorityStr = fmt.Sprintf(" [%s]", sg.Priority)
			}

			labelStr := ""
			if len(sg.Labels) > 0 {
				labelStr = fmt.Sprintf(" [%s]", strings.Join(sg.Labels, ","))
			}

			deadlineStr := ""
			if sg.Deadline != "" {
				deadlineStr = fmt.Sprintf(" [due:%s]", sg.Deadline)
			}

			fmt.Printf("  %-6s %s%s%s%s\n", sg.ID, sg.Title, priorityStr, labelStr, deadlineStr)
		}

		return nil
	},
}

// isReady checks if a saga can be worked on
func isReady(sg *saga.Saga, st *store.Store) bool {
	// Must be active
	if sg.Status != saga.StatusActive {
		return false
	}

	// Must not be claimed
	if sg.IsClaimed() {
		return false
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

func init() {
	rootCmd.AddCommand(readyCmd)
}

package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hbn/saga/internal/saga"
	"github.com/hbn/saga/internal/store"
	"github.com/spf13/cobra"
)

var (
	showAll     bool
	scopeLocal  bool
	scopeGlobal bool
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

		// Show scope info
		if st.HasLocal() && !scopeLocal && !scopeGlobal {
			fmt.Printf("(Showing global + project sagas from %s)\n\n", filepath.Dir(st.LocalPath()))
		}

		fmt.Printf("%-6s %-20s %-10s %s\n", "ID", "Title", "Status", "Updated")
		fmt.Println("-------------------------------------------")

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
			printSagaWithIndent(sg, 0, showAll, children)
		}

		return nil
	},
}

const maxDisplayDepth = 50

func printSagaWithIndent(sg *saga.Saga, indent int, showAll bool, children map[string][]*saga.Saga) {
	if indent > maxDisplayDepth {
		fmt.Printf("%-6s %s[Max depth reached]\n", sg.ID, strings.Repeat("  ", indent))
		return
	}

	title := sg.Title
	if len(title) > 20 {
		title = title[:17] + "..."
	}

	indentStr := ""
	if indent > 0 {
		indentStr = strings.Repeat("  ", indent) + "↳ "
	}

	updated := sg.UpdatedAt.Format("Jan 02 15:04")
	fmt.Printf("%-6s %s%-18s %-10s %s\n", sg.ID, indentStr, title, sg.Status, updated)

	for _, child := range children[sg.ID] {
		if !showAll && child.Status != saga.StatusActive {
			continue
		}
		printSagaWithIndent(child, indent+1, showAll, children)
	}
}

func init() {
	listCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all sagas including done")
	listCmd.Flags().BoolVarP(&scopeLocal, "local", "l", false, "Show only project sagas")
	listCmd.Flags().BoolVarP(&scopeGlobal, "global", "g", false, "Show only global sagas")
	rootCmd.AddCommand(listCmd)
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hbn/saga/internal/saga"
	"github.com/hbn/saga/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	command := os.Args[1]

	// Initialize store
	st, err := store.New(store.DefaultPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing store: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "new":
		handleNew(st, os.Args[2:])
	case "list":
		handleList(st, os.Args[2:])
	case "status":
		handleStatus(st, os.Args[2:])
	case "done":
		handleDone(st, os.Args[2:])
	case "continue":
		handleContinue(st, os.Args[2:])
	case "sub":
		handleSub(st, os.Args[2:])
	case "init":
		handleInit(st, os.Args[2:])
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func handleNew(st *store.Store, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: sg new <title>")
		os.Exit(1)
	}

	title := args[0]
	sg := saga.NewSaga(title)

	if err := st.Save(sg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving saga: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created saga %s: %s\n", sg.ID, sg.Title)
}

func handleList(st *store.Store, args []string) {
	// Parse flags
	showAll := false
	scopeLocal := false
	scopeGlobal := false

	for _, arg := range args {
		switch arg {
		case "--all", "-a":
			showAll = true
		case "--local", "-l":
			scopeLocal = true
		case "--global", "-g":
			scopeGlobal = true
		}
	}

	// Determine which scopes to load
	var scopes []store.Scope
	if scopeLocal && scopeGlobal {
		scopes = []store.Scope{store.ScopeLocal, store.ScopeGlobal}
	} else if scopeLocal {
		scopes = []store.Scope{store.ScopeLocal}
	} else if scopeGlobal {
		scopes = []store.Scope{store.ScopeGlobal}
	} else {
		// Default: show both local and global if local exists
		scopes = []store.Scope{store.ScopeGlobal}
		if st.HasLocal() {
			scopes = append(scopes, store.ScopeLocal)
		}
	}

	sagas, err := st.LoadAll(scopes...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading sagas: %v\n", err)
		os.Exit(1)
	}

	if len(sagas) == 0 {
		fmt.Println("No sagas found.")
		return
	}

	// Show scope info
	if st.HasLocal() && !scopeLocal && !scopeGlobal {
		fmt.Printf("(Showing global + project sagas from %s)\n", filepath.Dir(st.LocalPath()))
		fmt.Println()
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

	// Print root sagas (no parent) and their children recursively
	for _, sg := range sagas {
		if sg.ParentID != "" {
			continue // Skip children, they'll be printed with parent
		}
		if !showAll && sg.Status != saga.StatusActive {
			continue
		}
		printSagaWithIndent(sg, 0, showAll, children)
	}
}

func printSagaWithIndent(sg *saga.Saga, indent int, showAll bool, children map[string][]*saga.Saga) {
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

	// Print children
	for _, child := range children[sg.ID] {
		if !showAll && child.Status != saga.StatusActive {
			continue
		}
		printSagaWithIndent(child, indent+1, showAll, children)
	}
}

func handleStatus(st *store.Store, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: sg status <id>")
		os.Exit(1)
	}

	id := args[0]
	sg, err := st.GetByID(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Saga: %s (%s)\n", sg.ID, sg.Status)
	fmt.Printf("Title: %s\n", sg.Title)
	fmt.Printf("Created: %s\n", sg.CreatedAt.Format("Jan 02, 2006 15:04"))
	fmt.Printf("Updated: %s\n", sg.UpdatedAt.Format("Jan 02, 2006 15:04"))
	fmt.Println()
	fmt.Println("Recent history:")

	// Show last 5 entries
	start := len(sg.History) - 5
	if start < 0 {
		start = 0
	}
	for i := len(sg.History) - 1; i >= start; i-- {
		entry := sg.History[i]
		fmt.Printf("  %s | %s", entry.Timestamp.Format("15:04"), entry.Action)
		if entry.Note != "" {
			fmt.Printf(" - %s", entry.Note)
		}
		fmt.Println()
	}
}

func handleDone(st *store.Store, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: sg done <id>")
		os.Exit(1)
	}

	id := args[0]
	sg, err := st.GetByID(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check for active children
	hasActiveChildren, err := st.HasActiveChildren(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking children: %v\n", err)
		os.Exit(1)
	}
	if hasActiveChildren {
		fmt.Fprintf(os.Stderr, "Cannot mark saga %s as done: has active sub-sagas\n", id)
		fmt.Fprintf(os.Stderr, "Complete sub-sagas first or use --force\n")
		os.Exit(1)
	}

	sg.SetStatus(saga.StatusDone)

	if err := st.Update(sg); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating saga: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Marked saga %s as done\n", sg.ID)
}

func handleInit(st *store.Store, args []string) {
	if err := st.InitLocal(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing local saga: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Initialized local saga storage in %s\n", filepath.Dir(st.LocalPath()))
}

func handleContinue(st *store.Store, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: sg continue <id>")
		os.Exit(1)
	}

	id := args[0]
	sg, err := st.GetByID(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if sg.Status == saga.StatusDone {
		fmt.Fprintf(os.Stderr, "Saga %s is already done\n", sg.ID)
		os.Exit(1)
	}

	sg.SetStatus(saga.StatusActive)

	if err := st.Update(sg); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating saga: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Continuing saga %s: %s\n", sg.ID, sg.Title)
	fmt.Printf("Status: %s\n", sg.Status)
}

func handleSub(st *store.Store, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: sg sub <parent-id> <title>")
		os.Exit(1)
	}

	parentID := args[0]
	title := args[1]

	// Verify parent exists
	parent, err := st.GetByID(parentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if parent.Status == saga.StatusDone {
		fmt.Fprintf(os.Stderr, "Cannot add sub-saga to done saga %s\n", parentID)
		os.Exit(1)
	}

	// Create sub-saga
	sg := saga.NewSubSaga(title, parentID)

	if err := st.Save(sg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving sub-saga: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created sub-saga %s under %s: %s\n", sg.ID, parentID, sg.Title)
}

func printHelp() {
	help := `Saga - Task management for agent workflows

Usage: sg <command> [args]

Commands:
  init                 Initialize local saga storage (.saga/)
  new <title>          Create a new saga
  sub <parent> <title> Create a sub-saga
  list [flags]         List sagas (default: active only)
  status <id>          Show saga details and history
  done <id>            Mark saga as complete
  continue <id>        Resume a paused saga
  help                 Show this help message

List flags:
  --all, -a            Show all sagas (including done)
  --local, -l          Show only project sagas
  --global, -g         Show only global sagas

Examples:
  sg init
  sg new "Implement auth system"
  sg sub abc123 "Add OAuth provider"
  sg list              # Shows global + project (if in project)
  sg list --local      # Shows only project sagas
  sg list --global     # Shows only global sagas
  sg status a1b2
  sg done a1b2
`
	fmt.Print(help)
}

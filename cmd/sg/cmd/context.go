package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hbn/saga/internal/saga"
	"github.com/hbn/saga/internal/store"
	"github.com/spf13/cobra"
)

var contextFormat string

var contextCmd = &cobra.Command{
	Use:   "context <id>",
	Short: "Show full saga context",
	Long: `Display complete context for a saga including relationships, dependencies, and history.

Useful for agents to understand the full picture before acting.
Use --format json for machine-readable output.

Examples:
  sg context abc123
  sg context abc123 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		sg, err := st.GetByID(id)
		if err != nil {
			return sagaNotFound(id)
		}

		// Build context
		ctx := &SagaContext{
			Saga: sg,
		}

		// Get parent info
		if sg.IsSubSaga() {
			parent, err := st.GetByID(sg.ParentID)
			if err == nil {
				ctx.Parent = &BriefSaga{
					ID:     parent.ID,
					Title:  parent.Title,
					Status: parent.Status,
				}
			}
		}

		// Get children
		children, err := st.GetChildren(sg.ID)
		if err == nil {
			for _, child := range children {
				ctx.Children = append(ctx.Children, BriefSaga{
					ID:     child.ID,
					Title:  child.Title,
					Status: child.Status,
				})
			}
		}

		// Get dependencies with status
		for _, depID := range sg.DependsOn {
			dep, err := st.GetByID(depID)
			if err == nil {
				ctx.Dependencies = append(ctx.Dependencies, DependencyInfo{
					ID:       dep.ID,
					Title:    dep.Title,
					Status:   dep.Status,
					Blocking: dep.Status != saga.StatusDone,
				})
			} else {
				ctx.Dependencies = append(ctx.Dependencies, DependencyInfo{
					ID:       depID,
					Title:    "(not found)",
					Status:   "unknown",
					Blocking: true,
				})
			}
		}

		// Get related sagas
		for _, relID := range sg.RelatedTo {
			rel, err := st.GetByID(relID)
			if err == nil {
				ctx.Related = append(ctx.Related, BriefSaga{
					ID:     rel.ID,
					Title:  rel.Title,
					Status: rel.Status,
				})
			}
		}

		// Get linked runes (knowledge) via runes CLI
		// Run from current directory so runes can find local .runes/
		runesCmd := exec.Command("runes", "search", "--json", "--saga", sg.ID)
		runesCmd.Dir = "." // Explicitly use current directory
		output, err := runesCmd.Output()
		if err == nil && len(output) > 0 {
			// Try parsing as saga-linked runes format: {"runes": [...]}
			var sagaResult struct {
				Runes []struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"runes"`
			}
			if err := json.Unmarshal(output, &sagaResult); err == nil && len(sagaResult.Runes) > 0 {
				for _, r := range sagaResult.Runes {
					ctx.Runes = append(ctx.Runes, BriefRune{
						ID:    r.ID,
						Title: r.Title,
					})
				}
			}
		}

		// Output
		if contextFormat == "json" {
			data, err := json.MarshalIndent(ctx, "", "  ")
			if err != nil {
				return fmt.Errorf("encoding JSON: %w", err)
			}
			fmt.Println(string(data))
		} else {
			printContext(ctx)
		}

		return nil
	},
}

// SagaContext holds full context for a saga
type SagaContext struct {
	Saga         *saga.Saga       `json:"saga"`
	Parent       *BriefSaga       `json:"parent,omitempty"`
	Children     []BriefSaga      `json:"children,omitempty"`
	Dependencies []DependencyInfo `json:"dependencies,omitempty"`
	Related      []BriefSaga      `json:"related,omitempty"`
	Runes        []BriefRune      `json:"runes,omitempty"`
}

// BriefSaga minimal saga info
type BriefSaga struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Status saga.Status `json:"status"`
}

// BriefRune minimal rune info
type BriefRune struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Pattern string `json:"pattern,omitempty"`
}

// DependencyInfo includes blocking status
type DependencyInfo struct {
	ID       string      `json:"id"`
	Title    string      `json:"title"`
	Status   saga.Status `json:"status"`
	Blocking bool        `json:"blocking"`
}

// repeat returns a string repeated n times
func repeat(s string, n int) string {
	var result strings.Builder
	for i := 0; i < n; i++ {
		result.WriteString(s)
	}
	return result.String()
}

func printContext(ctx *SagaContext) {
	sg := ctx.Saga

	fmt.Println(repeat("═", 60))
	fmt.Printf("SAGA: %s (%s)\n", sg.ID, sg.Status)
	fmt.Println(repeat("═", 60))
	fmt.Println()

	// Basic info
	fmt.Printf("Title:       %s\n", sg.Title)
	if sg.Description != "" {
		fmt.Printf("Description: %s\n", sg.Description)
	}
	if sg.Priority != saga.PriorityNormal {
		fmt.Printf("Priority:    %s\n", sg.Priority)
	}
	if len(sg.Labels) > 0 {
		fmt.Printf("Labels:      %s\n", strings.Join(sg.Labels, ", "))
	}
	fmt.Println()

	// Hierarchy (only if has parent or children)
	if ctx.Parent != nil || len(ctx.Children) > 0 {
		fmt.Println(repeat("─", 40))
		fmt.Println("HIERARCHY")
		fmt.Println(repeat("─", 40))
		if ctx.Parent != nil {
			fmt.Printf("Parent:  %s (%s) - %s\n", ctx.Parent.ID, ctx.Parent.Status, ctx.Parent.Title)
		}
		if len(ctx.Children) > 0 {
			fmt.Printf("Children: %d\n", len(ctx.Children))
			for _, child := range ctx.Children {
				fmt.Printf("  • %s (%s) - %s\n", child.ID, child.Status, child.Title)
			}
		}
		fmt.Println()
	}

	// Dependencies (only if exists)
	if len(ctx.Dependencies) > 0 {
		fmt.Println(repeat("─", 40))
		fmt.Println("DEPENDENCIES")
		fmt.Println(repeat("─", 40))
		blocking := 0
		for _, dep := range ctx.Dependencies {
			status := "✓ done"
			if dep.Blocking {
				status = "✗ BLOCKING"
				blocking++
			}
			fmt.Printf("  • %s - %s (%s)\n", dep.ID, dep.Title, status)
		}
		fmt.Println()
		if blocking > 0 {
			fmt.Printf("⚠ %d blocking dependencies\n", blocking)
		} else {
			fmt.Println("✓ All dependencies complete")
		}
		fmt.Println()
	}

	// Related
	if len(ctx.Related) > 0 {
		fmt.Println(repeat("─", 40))
		fmt.Println("RELATED")
		fmt.Println(repeat("─", 40))
		for _, rel := range ctx.Related {
			fmt.Printf("  • %s (%s) - %s\n", rel.ID, rel.Status, rel.Title)
		}
		fmt.Println()
	}

	// Runes (knowledge)
	if len(ctx.Runes) > 0 {
		fmt.Println(repeat("─", 40))
		fmt.Println("KNOWLEDGE (Runes)")
		fmt.Println(repeat("─", 40))
		for _, r := range ctx.Runes {
			pattern := ""
			if r.Pattern != "" {
				pattern = fmt.Sprintf(" [%s]", r.Pattern)
			}
			fmt.Printf("  • %s - %s%s\n", r.ID, r.Title, pattern)
		}
		fmt.Println()
	}

	// Suggested runes (based on saga content)
	suggested := suggestRunes(sg)
	if len(suggested) > 0 {
		fmt.Println(repeat("─", 40))
		fmt.Println("💡 SUGGESTED RUNES")
		fmt.Println(repeat("─", 40))
		fmt.Println("Based on this saga's content, these runes may be relevant:")
		for _, r := range suggested {
			fmt.Printf("  • %s - %s\n", r.ID, r.Title)
		}
		fmt.Println()
		fmt.Println("Search runes: runes search \"<keywords>\"")
		fmt.Println("Show details:  runes show <id>")
		fmt.Println()
	}

	// History
	fmt.Println(repeat("─", 40))
	fmt.Println("RECENT HISTORY")
	fmt.Println(repeat("─", 40))
	start := len(sg.History) - 10
	if start < 0 {
		start = 0
	}
	for i := len(sg.History) - 1; i >= start; i-- {
		entry := sg.History[i]
		fmt.Printf("  %s | %s", entry.Timestamp.Format("Jan 02 15:04"), entry.Action)
		if entry.Note != "" {
			fmt.Printf(" - %s", entry.Note)
		}
		fmt.Println()
	}
	fmt.Println()

	// Summary
	fmt.Println(repeat("═", 60))
	fmt.Println("SUMMARY")
	fmt.Println(repeat("═", 60))
	fmt.Printf("Status:        %s\n", sg.Status)
	fmt.Printf("Can complete:  %v\n", canComplete(ctx))
	if !canComplete(ctx) {
		fmt.Println("\nBlocking items must be resolved before marking as done.")
	}
}

func canComplete(ctx *SagaContext) bool {
	if ctx.Saga.Status == saga.StatusDone {
		return true
	}
	for _, dep := range ctx.Dependencies {
		if dep.Blocking {
			return false
		}
	}
	for _, child := range ctx.Children {
		if child.Status != saga.StatusDone {
			return false
		}
	}
	return true
}

// suggestRunes searches runes based on saga title/description keywords
func suggestRunes(sg *saga.Saga) []BriefRune {
	// Extract keywords from title and description
	keywords := extractKeywords(sg.Title + " " + sg.Description)
	if len(keywords) == 0 {
		return nil
	}

	// Search runes with keywords
	var suggested []BriefRune
	for _, kw := range keywords {
		if len(kw) < 3 {
			continue // Skip short words
		}
		runesCmd := exec.Command("runes", "search", "--json", kw)
		output, err := runesCmd.Output()
		if err != nil || len(output) == 0 {
			continue
		}

		// Parse JSON output
		var result struct {
			Queries []struct {
				Query   string `json:"query"`
				Results []struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"results"`
			} `json:"queries"`
		}
		if err := json.Unmarshal(output, &result); err != nil {
			continue
		}

		for _, q := range result.Queries {
			for _, r := range q.Results {
				// Check if already added
				exists := false
				for _, s := range suggested {
					if s.ID == r.ID {
						exists = true
						break
					}
				}
				if !exists {
					suggested = append(suggested, BriefRune{
						ID:    r.ID,
						Title: r.Title,
					})
					if len(suggested) >= 5 {
						return suggested // Max 5 suggestions
					}
				}
			}
		}
	}

	return suggested
}

// extractKeywords extracts meaningful keywords from text
func extractKeywords(text string) []string {
	// Simple keyword extraction - split and filter
	words := strings.Fields(strings.ToLower(text))
	var keywords []string
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
		"of": true, "with": true, "by": true, "is": true, "are": true,
		"was": true, "be": true, "been": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true,
		"would": true, "should": true, "could": true, "can": true,
		"add": true, "create": true, "fix": true, "implement": true,
		"update": true, "make": true, "use": true, "using": true,
	}

	seen := make(map[string]bool)
	for _, w := range words {
		// Remove punctuation
		w = strings.TrimFunc(w, func(r rune) bool {
			return r < 'a' || r > 'z'
		})
		if len(w) >= 3 && !stopWords[w] && !seen[w] {
			keywords = append(keywords, w)
			seen[w] = true
		}
	}

	return keywords
}

func init() {
	contextCmd.Flags().StringVar(&contextFormat, "format", "", "Output format (json)")
	rootCmd.AddCommand(contextCmd)
}

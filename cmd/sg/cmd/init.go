package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hbn/saga/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize local saga storage",
	Long:  `Creates a .saga directory in the current project for local saga storage.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		if err := st.InitLocal(); err != nil {
			return fmt.Errorf("initializing local saga: %w", err)
		}

		fmt.Printf("Initialized local saga storage in %s\n", filepath.Dir(st.LocalPath()))
		
		// Add saga section to AGENTS.md if it exists
		if err := addSagaToAgents(); err != nil {
			// Non-fatal: just print the help text instead
			fmt.Println()
			fmt.Println("Saga initialized. Basic usage:")
			fmt.Println("  sg ready              # Find work to do")
			fmt.Println("  sg claim <id>         # Claim a saga")
			fmt.Println("  sg context <id>       # Read context")
			fmt.Println("  sg log <id> \"...\"    # Log progress")
			fmt.Println("  sg done <id>          # Mark complete")
		}
		
		return nil
	},
}

func addSagaToAgents() error {
	agentsPath := "AGENTS.md"
	
	// Check if file exists
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		return err
	}
	
	// Read existing content
	content, err := os.ReadFile(agentsPath)
	if err != nil {
		return err
	}
	
	// Check if saga section already exists
	if strings.Contains(string(content), "## Saga") {
		return nil // Already exists, skip
	}
	
	// Append saga section
	sagaSection := "\n## Saga - Task Tracking\n\n" +
		"This project uses [Saga](https://github.com/sleeplesslord/saga) for task management.\n\n" +
		"### Quick Start\n\n" +
		"```bash\n" +
		"sg ready              # Find unblocked, unclaimed work\n" +
		"sg claim <id>         # Claim a saga for yourself\n" +
		"sg context <id>       # Read full context before starting\n" +
		"sg log <id> \"...\"     # Log progress and decisions\n" +
		"sg done <id>          # Mark complete when finished\n" +
		"```\n\n" +
		"### Basic Commands\n\n" +
		"| Command | Description |\n" +
		"|---------|-------------|\n" +
		"| `sg ready` | List sagas ready to work on |\n" +
		"| `sg claim <id>` | Prevent others from working on it |\n" +
		"| `sg context <id>` | Full saga details, dependencies, history |\n" +
		"| `sg log <id> \"msg\"` | Record progress and decisions |\n" +
		"| `sg done <id>` | Mark complete, records completion |\n" +
		"| `sg list` | Show all active sagas |\n" +
		"| `sg search \"query\"` | Find sagas by title/content |\n\n" +
		"See `skills/saga-agent/SKILL.md` for full agent documentation.\n"
	
	// Append to file
	f, err := os.OpenFile(agentsPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	
	if _, err := f.WriteString(sagaSection); err != nil {
		return err
	}
	
	fmt.Println("\n✓ Added Saga section to AGENTS.md")
	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}

package cmd

import (
	"fmt"
	"path/filepath"

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
		fmt.Println()
		fmt.Println("Basic usage:")
		fmt.Println("  sg ready              # Find work to do")
		fmt.Println("  sg claim <id>         # Claim a saga")
		fmt.Println("  sg context <id>       # Read context")
		fmt.Println("  sg log <id> \"...\"    # Log progress")
		fmt.Println("  sg done <id>          # Mark complete")
		fmt.Println()
		fmt.Println("See AGENTS.md for full documentation")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

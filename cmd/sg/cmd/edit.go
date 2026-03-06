package cmd

import (
	"fmt"

	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var editTitle string
var editDesc string

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit saga title or description",
	Long: `Update a saga's title or description after creation.

Use --title to change the title, --desc to change the description.
At least one flag must be provided.

Examples:
  sg edit abc123 --title "New title"
  sg edit abc123 --desc "Updated description"
  sg edit abc123 --title "New title" --desc "New description"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Validate at least one edit flag provided
		if editTitle == "" && editDesc == "" {
			return fmt.Errorf("at least one of --title or --desc required")
		}

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		sg, err := st.GetByID(id)
		if err != nil {
			return sagaNotFound(id)
		}

		// Update fields
		if editTitle != "" {
			sg.Title = editTitle
		}
		if editDesc != "" {
			sg.Description = editDesc
		}

		if err := st.Update(sg); err != nil {
			return fmt.Errorf("updating saga: %w", err)
		}

		fmt.Printf("Updated saga %s\n", sg.ID)
		return nil
	},
}

func init() {
	editCmd.Flags().StringVar(&editTitle, "title", "", "New title")
	editCmd.Flags().StringVar(&editDesc, "desc", "", "New description")
	rootCmd.AddCommand(editCmd)
}

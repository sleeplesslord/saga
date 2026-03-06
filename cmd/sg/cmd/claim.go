package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var (
	claimAgent    string
	claimDuration time.Duration
)

var claimCmd = &cobra.Command{
	Use:   "claim <id>",
	Short: "Claim a saga for work",
	Long: `Mark a saga as claimed by you to prevent others from working on it.

Claims expire after 24 hours by default. Use --duration to set custom time.
Use --agent to specify who is claiming (defaults to USER env var).

The claim ID includes the parent process ID (PPID) for unique session
identification: "agent@ppid" (e.g., "andreas@12345" or "claude@67890").

Examples:
  sg claim abc123                    # Claim for 24h
  sg claim abc123 --duration 4h      # Claim for 4 hours
  sg claim abc123 --agent claude     # Claim as specific agent`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Determine agent name
		agent := claimAgent
		if agent == "" {
			agent = os.Getenv("USER")
			if agent == "" {
				agent = "unknown"
			}
		}
		// Append PPID for unique session identification
		agent = fmt.Sprintf("%s@%d", agent, os.Getppid())

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		sg, err := st.GetByID(id)
		if err != nil {
			return sagaNotFound(id)
		}

		// Check if already claimed
		if sg.IsClaimed() && sg.ClaimedBy != agent {
			return fmt.Errorf("saga %s is already claimed by %s (expires %s)",
				id, sg.ClaimedBy, sg.ClaimExpiry().Format("15:04"))
		}

		// Claim it
		sg.Claim(agent)
		if err := st.Update(sg); err != nil {
			return fmt.Errorf("updating saga: %w", err)
		}

		duration := "24h"
		if claimDuration > 0 {
			duration = claimDuration.String()
		}
		fmt.Printf("Claimed saga %s for %s (expires in %s)\n", id, agent, duration)
		return nil
	},
}

var unclaimCmd = &cobra.Command{
	Use:   "unclaim <id>",
	Short: "Release claim on a saga",
	Long: `Release your claim on a saga so others can work on it.

Example:
  sg unclaim abc123`,
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

		if !sg.IsClaimed() {
			return fmt.Errorf("saga %s is not claimed", id)
		}

		agent := sg.ClaimedBy
		sg.Unclaim()
		if err := st.Update(sg); err != nil {
			return fmt.Errorf("updating saga: %w", err)
		}

		fmt.Printf("Released claim on saga %s (was claimed by %s)\n", id, agent)
		return nil
	},
}

func init() {
	claimCmd.Flags().StringVar(&claimAgent, "agent", "", "Agent name (default: $USER)")
	claimCmd.Flags().DurationVar(&claimDuration, "duration", 24*time.Hour, "Claim duration")
	rootCmd.AddCommand(claimCmd)
	rootCmd.AddCommand(unclaimCmd)
}

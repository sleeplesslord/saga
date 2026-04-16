package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var (
	claimAgent         string
	claimDurationFlag  string
	claimDurationSet   bool
)

var claimCmd = &cobra.Command{
	Use:   "claim <id> [id...]",
	Short: "Claim saga for work",
	Long: `Mark a saga as claimed by you to prevent others from working on it.

Claim duration defaults to the configured value (see .saga/config.json),
falling back to 24h if not configured. Use --duration to override.

The claim ID includes the parent process ID (PPID) for unique session
identification: "agent@ppid" (e.g., "andreas@12345" or "claude@67890").

Multiple IDs can be provided to claim several sagas at once.

Examples:
  sg claim abc123                    # Claim for configured/default duration
  sg claim abc123 def456 ghi789      # Claim multiple
  sg claim abc123 --duration 4h      # Claim for 4 hours
  sg claim abc123 --agent claude     # Claim as specific agent`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ids := args

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

		// Resolve claim duration: flag > config > default (24h)
		var duration time.Duration
		if claimDurationSet {
			d, err := time.ParseDuration(claimDurationFlag)
			if err != nil {
				return fmt.Errorf("invalid duration %q: %w", claimDurationFlag, err)
			}
			duration = d
		} else {
			duration = st.ClaimDuration()
		}

		for _, id := range ids {
			sg, err := st.GetByID(id)
			if err != nil {
				return sagaNotFound(id)
			}

			// Check if already claimed by a different session
			if sg.IsClaimedWithDuration(duration) && sg.ClaimedBy != agent {
				return fmt.Errorf("saga %s is already claimed by %s (expires %s)",
					id, sg.ClaimedBy, sg.ClaimExpiryWithDuration(duration).Format("15:04"))
			}

			// Claim it
			sg.ClaimWithDuration(agent, duration)
			if err := st.Update(sg); err != nil {
				return fmt.Errorf("updating saga: %w", err)
			}

			fmt.Printf("Claimed saga %s for %s (expires in %s)\n", id, agent, duration)
		}
		return nil
	},
}

var unclaimCmd = &cobra.Command{
	Use:   "unclaim <id> [id...]",
	Short: "Release claim on a saga",
	Long: `Release your claim on a saga so others can work on it.

Multiple IDs can be provided to unclaim several sagas at once.

Example:
  sg unclaim abc123
  sg unclaim abc123 def456`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ids := args

		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		for _, id := range ids {
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
		}
		return nil
	},
}

func init() {
	claimCmd.Flags().StringVar(&claimAgent, "agent", "", "Agent name (default: $USER)")
	claimCmd.Flags().StringVar(&claimDurationFlag, "duration", "24h", "Claim duration (e.g. 4h, 30m, 72h). Default: configured or 24h")
	// Track whether --duration was explicitly set
	claimCmd.Flags().Lookup("duration").NoOptDefVal = ""
	claimCmd.Flags().Lookup("duration").Changed = false
	_ = claimDurationSet // will be set after flag parse via hook

	// Post-parse hook to detect if --duration was explicitly provided
	claimCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		claimDurationSet = cmd.Flags().Changed("duration")
		return nil
	}

	rootCmd.AddCommand(claimCmd)
	rootCmd.AddCommand(unclaimCmd)
}

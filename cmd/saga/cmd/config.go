package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/sleeplesslord/saga/internal/store"
	"github.com/spf13/cobra"
)

var (
	configClaimDuration string
	configTitleWidth     int
	configScope         string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or set saga configuration",
	Long: `View or modify saga configuration stored in .saga/config.json.

Without flags, shows current configuration.
Use flags to set values. Changes are written to the local .saga/config.json
by default; use --scope global to write to ~/.saga/config.json.

Examples:
  sg config                           # Show current config
  sg config --claim-duration 4h       # Set claim duration to 4 hours
  sg config --claim-duration 30m      # Set claim duration to 30 minutes
  sg config --claim-duration 72h      # Set claim duration to 3 days
  sg config --title-width 80          # Set title column width to 80 chars
  sg config --scope global --claim-duration 4h  # Set globally`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := store.New(store.DefaultPath())
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}

		cfg, err := st.LoadConfig()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// If no flags changed, just show current config
		if !cmd.Flags().Changed("claim-duration") && !cmd.Flags().Changed("title-width") {
			fmt.Println("Saga configuration:")
			fmt.Printf("  claim_duration: %s", cfg.ParsedClaimDuration())
			if cfg.ClaimDuration == "" {
				fmt.Print(" (default)")
			}
			fmt.Println()
			fmt.Printf("  title_width: %d", cfg.EffectiveTitleWidth())
			if cfg.TitleWidth == 0 {
				fmt.Print(" (default)")
			}
			fmt.Println()
			return nil
		}

		// Set claim duration if flag changed
		if cmd.Flags().Changed("claim-duration") {
			d, err := time.ParseDuration(configClaimDuration)
			if err != nil {
				return fmt.Errorf("invalid duration %q: %w", configClaimDuration, err)
			}
			cfg.ClaimDuration = configClaimDuration
			_ = d // validated above
		}

		// Set title width if flag changed
		if cmd.Flags().Changed("title-width") {
			if configTitleWidth < 10 {
				return fmt.Errorf("title-width must be at least 10, got %d", configTitleWidth)
			}
			cfg.TitleWidth = configTitleWidth
		}

		// Determine scope
		scope := store.ScopeLocal
		if configScope == "global" {
			scope = store.ScopeGlobal
		}

		if err := st.SaveConfig(cfg, scope); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		scopeDesc := "local"
		if scope == store.ScopeGlobal {
			scopeDesc = "global"
		}
		changed := []string{}
		if cmd.Flags().Changed("claim-duration") {
			changed = append(changed, fmt.Sprintf("claim_duration to %s", cfg.ParsedClaimDuration()))
		}
		if cmd.Flags().Changed("title-width") {
			changed = append(changed, fmt.Sprintf("title_width to %d", cfg.TitleWidth))
		}
		fmt.Printf("Set %s (%s config)\n", strings.Join(changed, ", "), scopeDesc)
		return nil
	},
}

func init() {
	configCmd.Flags().StringVar(&configClaimDuration, "claim-duration", "", "Default claim duration (e.g. 4h, 30m, 72h)")
	configCmd.Flags().IntVar(&configTitleWidth, "title-width", 0, "Title column width in list/ready/search tables (default: 60)")
	configCmd.Flags().StringVar(&configScope, "scope", "local", "Config scope: local or global")
	rootCmd.AddCommand(configCmd)
}

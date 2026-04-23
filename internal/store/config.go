package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config holds saga configuration
type Config struct {
	ClaimDuration string `json:"claim_duration,omitempty"` // e.g. "24h", "4h30m", "72h"
	TitleWidth    int    `json:"title_width,omitempty"`    // column width for TITLE in list/ready/search tables
}

// DefaultTitleWidth is the fallback when config is unset or zero
const DefaultTitleWidth = 60

// DefaultClaimDuration is the fallback when config is unset or invalid
const DefaultClaimDuration = 24 * time.Hour

// ParsedClaimDuration returns the configured claim duration as a time.Duration.
// Falls back to 24h if unset or unparseable.
func (c *Config) ParsedClaimDuration() time.Duration {
	if c.ClaimDuration == "" {
		return DefaultClaimDuration
	}
	d, err := time.ParseDuration(c.ClaimDuration)
	if err != nil {
		return DefaultClaimDuration
	}
	return d
}

// EffectiveTitleWidth returns the configured title column width.
// Falls back to DefaultTitleWidth if unset or zero.
func (c *Config) EffectiveTitleWidth() int {
	if c.TitleWidth <= 0 {
		return DefaultTitleWidth
	}
	return c.TitleWidth
}

// LoadConfig reads config from the .saga directory (local first, then global).
// Returns an empty Config (with defaults) if no config file exists.
func (s *Store) LoadConfig() (*Config, error) {
	// Try local config first
	if s.localPath != "" {
		cfgPath := filepath.Join(filepath.Dir(s.localPath), "config.json")
		if data, err := os.ReadFile(cfgPath); err == nil {
			var cfg Config
			if err := json.Unmarshal(data, &cfg); err != nil {
				return nil, fmt.Errorf("parsing local config: %w", err)
			}
			return &cfg, nil
		}
	}

	// Try global config
	if s.globalPath != "" {
		cfgPath := filepath.Join(filepath.Dir(s.globalPath), "config.json")
		if data, err := os.ReadFile(cfgPath); err == nil {
			var cfg Config
			if err := json.Unmarshal(data, &cfg); err != nil {
				return nil, fmt.Errorf("parsing global config: %w", err)
			}
			return &cfg, nil
		}
	}

	// No config file found — use defaults
	return &Config{}, nil
}

// SaveConfig writes config to the given scope's .saga directory
func (s *Store) SaveConfig(cfg *Config, scope Scope) error {
	var dir string
	if scope == ScopeLocal {
		if s.localPath == "" {
			return fmt.Errorf("no local saga directory initialized")
		}
		dir = filepath.Dir(s.localPath)
	} else {
		if s.globalPath == "" {
			return fmt.Errorf("no global saga directory found")
		}
		dir = filepath.Dir(s.globalPath)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// ClaimDuration returns the effective claim duration for this store,
// reading from config with fallback to the 24h default.
func (s *Store) ClaimDuration() time.Duration {
	cfg, err := s.LoadConfig()
	if err != nil {
		return DefaultClaimDuration
	}
	return cfg.ParsedClaimDuration()
}

// TitleWidth returns the effective title column width for this store,
// reading from config with fallback to DefaultTitleWidth.
func (s *Store) TitleWidth() int {
	cfg, err := s.LoadConfig()
	if err != nil {
		return DefaultTitleWidth
	}
	if cfg.TitleWidth <= 0 {
		return DefaultTitleWidth
	}
	return cfg.TitleWidth
}

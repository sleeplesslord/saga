package cmd

import "testing"

// Locks every agent-friendly alias for commands an LLM/user naturally guesses
// but that sg names differently. A refactor that drops one should fail here.
// Covers the full alias set (matching the README table), not just the original
// friction cases, so the README/implementation/test can't silently diverge.
// Evidence: saga t4be0d friction log — `sg add` / `sg show` / `sg release`
// repeatedly hit "unknown command".
func TestFrictionAliasesRegistered(t *testing.T) {
	want := map[string][]string{
		"new":      {"add", "create"},
		"context":  {"show"},
		"claim":    {"assign"},
		"unclaim":  {"unassign", "release"},
		"edit":     {"update"},
		"list":     {"ls"},
		"ready":    {"todo"},
		"continue": {"resume"},
		"log":      {"comment"},
		"done":     {"complete", "finish"},
		"wontdo":   {"cancel", "skip"},
	}
	for canonical, aliases := range want {
		for _, alias := range aliases {
			found, _, err := rootCmd.Find([]string{alias})
			if err != nil {
				t.Errorf("alias %q (-> %q) not resolvable: %v", alias, canonical, err)
				continue
			}
			if found.Name() != canonical {
				t.Errorf("alias %q resolved to %q, want %q", alias, found.Name(), canonical)
			}
		}
	}
}

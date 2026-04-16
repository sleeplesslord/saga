package cmd

import (
	"fmt"
	"strings"

	"github.com/sleeplesslord/saga/internal/saga"
)

// formatDescription fixes double-escaped newlines from shell input.
// Shell arguments like "line1\nline2" get stored as "line1\\nline2";
// this converts them back to real newlines for display.
func formatDescription(desc string) string {
	return strings.ReplaceAll(desc, "\\n", "\n")
}

// printField prints a labeled field value.
// If labelWidth > 0, labels are right-padded for column alignment (context style).
// If labelWidth == 0, a single space separates label and value (status style).
func printField(label, value string, labelWidth int) {
	if labelWidth > 0 {
		fmt.Printf("%-*s%s\n", labelWidth, label, value)
	} else {
		fmt.Printf("%s %s\n", label, value)
	}
}

// printSagaFields prints the common conditional saga fields:
// description, priority, labels, and deadline.
//
// labelWidth controls alignment (0 = compact, >0 = padded).
// labelsJoined controls label formatting:
//   - true  = comma-separated (context style)
//   - false = Go %v format like [a b c] (status style)
func printSagaFields(sg *saga.Saga, labelWidth int, labelsJoined bool) {
	if sg.Description != "" {
		printField("Description:", formatDescription(sg.Description), labelWidth)
	}
	if sg.Priority != saga.PriorityNormal {
		printField("Priority:", string(sg.Priority), labelWidth)
	}
	if len(sg.Labels) > 0 {
		var labelsStr string
		if labelsJoined {
			labelsStr = strings.Join(sg.Labels, ", ")
		} else {
			labelsStr = fmt.Sprintf("%v", sg.Labels)
		}
		printField("Labels:", labelsStr, labelWidth)
	}
	if sg.Deadline != "" {
		printField("Deadline:", sg.Deadline, labelWidth)
	}
}

// printHistoryEntries prints recent history entries in reverse chronological order.
// limit controls the maximum number of entries shown.
// longFormat uses "Jan 02 15:04" timestamps; otherwise "15:04".
func printHistoryEntries(history []saga.HistoryEntry, limit int, longFormat bool) {
	start := len(history) - limit
	if start < 0 {
		start = 0
	}
	timeFormat := "15:04"
	if longFormat {
		timeFormat = "Jan 02 15:04"
	}
	for i := len(history) - 1; i >= start; i-- {
		entry := history[i]
		fmt.Printf("  %s | %s", entry.Timestamp.Format(timeFormat), entry.Action)
		if entry.Note != "" {
			fmt.Printf(" - %s", entry.Note)
		}
		fmt.Println()
	}
}

// repeat returns a string repeated n times.
// Used for decorative separators in display output.
func repeat(s string, n int) string {
	var result strings.Builder
	for i := 0; i < n; i++ {
		result.WriteString(s)
	}
	return result.String()
}

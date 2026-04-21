package cmd

import (
	"fmt"
	"strings"
	"time"

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

// --- Table formatting helpers ---

// runeWidth returns the terminal display width of a single rune.
// CJK and other East Asian wide characters count as 2 columns.
func runeWidth(r rune) int {
	switch {
	case r >= 0x1100 && r <= 0x115F,
		r >= 0x2E80 && r <= 0x9FFF,
		r >= 0xA960 && r <= 0xA97C,
		r >= 0xAC00 && r <= 0xD7A3,
		r >= 0xD7B0 && r <= 0xD7C6,
		r >= 0xF900 && r <= 0xFAFF,
		r >= 0xFE30 && r <= 0xFE6F,
		r >= 0xFF01 && r <= 0xFF60,
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x20000 && r <= 0x2FFFD,
		r >= 0x30000 && r <= 0x3FFFD:
		return 2
	default:
		return 1
	}
}

// displayWidth returns the terminal display width of a string.
// CJK characters count as 2 columns; most others count as 1.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

// truncateToWidth truncates s to fit within the given display width.
// If truncated, appends "…" (ellipsis) to indicate truncation.
func truncateToWidth(s string, width int) string {
	if displayWidth(s) <= width {
		return s
	}
	target := width - 1 // leave room for …
	runes := []rune(s)
	w := 0
	for i, r := range runes {
		rw := runeWidth(r)
		if w+rw > target {
			return string(runes[:i]) + "…"
		}
		w += rw
	}
	return s
}

// padOrTruncate right-pads s to exactly the given display width, or truncates with "…" if too wide.
func padOrTruncate(s string, width int) string {
	s = truncateToWidth(s, width)
	dw := displayWidth(s)
	return s + strings.Repeat(" ", width-dw)
}

// formatDeadline converts a YYYYMMDD deadline to "MM-DD" format for table display.
// Returns the original string if parsing fails, or empty if no deadline.
func formatDeadline(deadline string) string {
	if deadline == "" {
		return ""
	}
	t, err := time.Parse("20060102", deadline)
	if err != nil {
		return deadline
	}
	return t.Format("01-02")
}

// printTableHeader prints a table header with │ separators and ─ divider line.
func printTableHeader(headers []string, widths []int) {
	parts := make([]string, len(headers))
	for i, h := range headers {
		parts[i] = padOrTruncate(h, widths[i])
	}
	fmt.Println(strings.Join(parts, " │ "))

	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = strings.Repeat("─", widths[i])
	}
	fmt.Println(strings.Join(seps, "─┼─"))
}

// printTableRow prints a formatted table row with │ separators.
// indent is prepended before the row for hierarchical indentation.
func printTableRow(cells []string, widths []int, indent string) {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = padOrTruncate(c, widths[i])
	}
	fmt.Printf("%s%s\n", indent, strings.Join(parts, " │ "))
}

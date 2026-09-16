package diff

import (
	"fmt"
	"strings"
)

// --- Line splitting ---

// splitLines splits text into lines, preserving the content without trailing newlines.
// Returns the lines and whether the original text ended with a newline.
func splitLines(text string) ([]string, bool) {
	if text == "" {
		return nil, false
	}
	// Normalize \r\n to \n
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	// If text ends with \n, the last element will be empty
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		return lines[:len(lines)-1], true
	}
	return lines, false
}

// --- LCS-based line diff ---

// lcsTable computes the LCS dynamic programming table for two line slices.
func lcsTable(a, b []string) [][]int {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}
	return dp
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// LineDiff computes a line-level diff between two texts.
// Returns a slice of Line entries describing the operations.
func LineDiff(oldText, newText string) []Line {
	oldLines, _ := splitLines(oldText)
	newLines, _ := splitLines(newText)

	dp := lcsTable(oldLines, newLines)

	var result []Line
	i, j := len(oldLines), len(newLines)

	// Backtrack through the DP table to build the diff
	type entry struct {
		op   Operation
		text string
		oldNo, newNo int
	}
	var entries []entry

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			entries = append(entries, entry{OpEqual, oldLines[i-1], i, j})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			entries = append(entries, entry{OpInsert, newLines[j-1], 0, j})
			j--
		} else if i > 0 {
			entries = append(entries, entry{OpDelete, oldLines[i-1], i, 0})
			i--
		}
	}

	// Reverse entries (we built them backwards)
	for k := len(entries) - 1; k >= 0; k-- {
		e := entries[k]
		result = append(result, Line{
			Op:    e.op,
			Text:  e.text,
			OldNo: e.oldNo,
			NewNo: e.newNo,
		})
	}

	return result
}

// --- Unified diff generation ---

// UnifiedDiff generates a unified diff between two texts.
// contextLines specifies the number of context lines around changes (default 3).
func UnifiedDiff(oldText, newText string, contextLines int) string {
	if contextLines < 0 {
		contextLines = 3
	}

	lines := LineDiff(oldText, newText)
	if lines == nil {
		// Check if texts are identical
		if oldText == newText {
			return ""
		}
	}

	hunks := buildHunks(lines, contextLines)
	if len(hunks) == 0 {
		return ""
	}

	var sb strings.Builder
	// Header
	oldLabel, newLabel := "old", "new"
	sb.WriteString(fmt.Sprintf("--- %s\n", oldLabel))
	sb.WriteString(fmt.Sprintf("+++ %s\n", newLabel))

	for _, hunk := range hunks {
		sb.WriteString(fmt.Sprintf("@@ -%s +%s @@\n", formatRange(hunk.OldStart, hunk.OldCount), formatRange(hunk.NewStart, hunk.NewCount)))
		for _, line := range hunk.Lines {
			switch line.Op {
			case OpEqual:
				sb.WriteString(" " + line.Text + "\n")
			case OpInsert:
				sb.WriteString("+" + line.Text + "\n")
			case OpDelete:
				sb.WriteString("-" + line.Text + "\n")
			}
		}
	}

	return sb.String()
}

// formatRange formats a hunk range for the @@ header.
func formatRange(start, count int) string {
	if count == 1 {
		return fmt.Sprintf("%d", start)
	}
	if count == 0 {
		return fmt.Sprintf("%d,0", start)
	}
	return fmt.Sprintf("%d,%d", start, count)
}

// buildHunks groups diff lines into hunks with context.
func buildHunks(lines []Line, context int) []Hunk {
	if len(lines) == 0 {
		return nil
	}

	// Find indices of changed lines
	var changedIdx []int
	for i, line := range lines {
		if line.Op != OpEqual {
			changedIdx = append(changedIdx, i)
		}
	}

	if len(changedIdx) == 0 {
		return nil
	}

	// Group changes into hunks (merge if within 2*context+1 lines of each other)
	var hunks []Hunk
	hunkStart := max(0, changedIdx[0]-context)
	hunkEnd := min(len(lines)-1, changedIdx[0]+context)

	for i := 1; i < len(changedIdx); i++ {
		ci := changedIdx[i]
		if ci-context <= hunkEnd+1 {
			// Merge into current hunk
			hunkEnd = min(len(lines)-1, ci+context)
		} else {
			// Finalize current hunk
			hunks = append(hunks, buildHunk(lines, hunkStart, hunkEnd))
			hunkStart = max(0, ci-context)
			hunkEnd = min(len(lines)-1, ci+context)
		}
	}
	hunks = append(hunks, buildHunk(lines, hunkStart, hunkEnd))

	return hunks
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// buildHunk creates a Hunk from a slice of diff lines.
func buildHunk(lines []Line, start, end int) Hunk {
	hunkLines := lines[start : end+1]

	// Calculate old/new start and counts
	var oldStart, newStart int
	oldCount, newCount := 0, 0
	oldStart, newStart = 0, 0

	for i, line := range hunkLines {
		if line.OldNo > 0 {
			if oldStart == 0 {
				oldStart = line.OldNo
			}
			oldCount++
		}
		if line.NewNo > 0 {
			if newStart == 0 {
				newStart = line.NewNo
			}
			newCount++
		}
		_ = i
	}

	// Handle empty hunks
	if oldStart == 0 {
		oldStart = 1
	}
	if newStart == 0 {
		newStart = 1
	}

	return Hunk{
		OldStart: oldStart,
		OldCount: oldCount,
		NewStart: newStart,
		NewCount: newCount,
		Lines:    hunkLines,
	}
}

// --- Patch parsing and application ---

// ParsePatch parses a unified diff patch and returns the hunks.
func ParsePatch(patch string) ([]Hunk, error) {
	lines := strings.Split(patch, "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty patch")
	}

	var hunks []Hunk
	i := 0

	// Skip header lines (--- and +++)
	for i < len(lines) {
		if strings.HasPrefix(lines[i], "@@") {
			break
		}
		i++
	}

	for i < len(lines) {
		line := lines[i]

		if !strings.HasPrefix(line, "@@") {
			i++
			continue
		}

		// Parse hunk header: @@ -oldStart,oldCount +newStart,newCount @@
		hunk, _, err := parseHunkHeader(line)
		if err != nil {
			return nil, fmt.Errorf("invalid hunk header at line %d: %w", i+1, err)
		}

		i++

		// Read hunk body lines
		oldNo := hunk.OldStart
		newNo := hunk.NewStart

		for i < len(lines) && !strings.HasPrefix(lines[i], "@@") {
			bodyLine := lines[i]

			if bodyLine == "" {
				// Empty line could be end of patch or a blank context line
				i++
				continue
			}

			prefix := bodyLine[0]
			content := bodyLine[1:]

			switch prefix {
			case ' ':
				hunk.Lines = append(hunk.Lines, Line{Op: OpEqual, Text: content, OldNo: oldNo, NewNo: newNo})
				oldNo++
				newNo++
			case '+':
				hunk.Lines = append(hunk.Lines, Line{Op: OpInsert, Text: content, OldNo: 0, NewNo: newNo})
				newNo++
			case '-':
				hunk.Lines = append(hunk.Lines, Line{Op: OpDelete, Text: content, OldNo: oldNo, NewNo: 0})
				oldNo++
			case '\\':
				// "\ No newline at end of file" — skip
			default:
				// Unknown line, stop
				goto hunkDone
			}
			i++
		}

	hunkDone:
		hunks = append(hunks, hunk)
	}

	return hunks, nil
}

// parseHunkHeader parses a @@ -start,count +start,count @@ line.
func parseHunkHeader(line string) (Hunk, int, error) {
	// Format: @@ -oldStart,oldCount +newStart,newCount @@
	s := strings.TrimPrefix(line, "@@ ")
	// Find the closing @@
	endIdx := strings.Index(s, " @@")
	if endIdx == -1 {
		// Maybe no trailing space
		endIdx = strings.Index(s, "@@")
	}
	if endIdx == -1 {
		return Hunk{}, 0, fmt.Errorf("missing closing @@ in hunk header")
	}
	s = s[:endIdx]

	parts := strings.Fields(s)
	if len(parts) < 2 {
		return Hunk{}, 0, fmt.Errorf("expected old and new ranges, got %d parts", len(parts))
	}

	oldRange := strings.TrimPrefix(parts[0], "-")
	newRange := strings.TrimPrefix(parts[1], "+")

	oldStart, oldCount, err := parseRange(oldRange)
	if err != nil {
		return Hunk{}, 0, fmt.Errorf("invalid old range: %w", err)
	}
	newStart, newCount, err := parseRange(newRange)
	if err != nil {
		return Hunk{}, 0, fmt.Errorf("invalid new range: %w", err)
	}

	return Hunk{
		OldStart: oldStart,
		OldCount: oldCount,
		NewStart: newStart,
		NewCount: newCount,
	}, 0, nil
}

// parseRange parses "start" or "start,count" into start and count.
func parseRange(s string) (int, int, error) {
	if s == "" {
		return 0, 0, fmt.Errorf("empty range")
	}
	if idx := strings.Index(s, ","); idx >= 0 {
		start, err := atoi(s[:idx])
		if err != nil {
			return 0, 0, err
		}
		count, err := atoi(s[idx+1:])
		if err != nil {
			return 0, 0, err
		}
		return start, count, nil
	}
	start, err := atoi(s)
	if err != nil {
		return 0, 0, err
	}
	return start, 1, nil
}

// atoi is a simple string-to-int converter.
func atoi(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty number")
	}
	n := 0
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid digit: %c", c)
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}

// ApplyPatch applies a unified diff patch to the given text.
// Returns the patched text and an error if the patch cannot be applied.
func ApplyPatch(text, patch string) (string, error) {
	hunks, err := ParsePatch(patch)
	if err != nil {
		return "", fmt.Errorf("failed to parse patch: %w", err)
	}
	if len(hunks) == 0 {
		return "", fmt.Errorf("no hunks found in patch")
	}

	lines, hadTrailingNewline := splitLines(text)
	result := make([]string, 0, len(lines))
	srcIdx := 0 // 0-based index into original lines

	for _, hunk := range hunks {
		// Copy lines before the hunk
		targetIdx := hunk.OldStart - 1 // 0-based
		if targetIdx < 0 {
			targetIdx = 0
		}
		for srcIdx < targetIdx && srcIdx < len(lines) {
			result = append(result, lines[srcIdx])
			srcIdx++
		}

		// Apply hunk lines
		for _, hline := range hunk.Lines {
			switch hline.Op {
			case OpEqual:
				if srcIdx >= len(lines) {
					return "", fmt.Errorf("patch context mismatch at line %d: expected line %q but reached end of text", srcIdx+1, hline.Text)
				}
				if lines[srcIdx] != hline.Text {
					return "", fmt.Errorf("patch context mismatch at line %d: expected %q, got %q", srcIdx+1, hline.Text, lines[srcIdx])
				}
				result = append(result, lines[srcIdx])
				srcIdx++
			case OpInsert:
				result = append(result, hline.Text)
			case OpDelete:
				if srcIdx >= len(lines) {
					return "", fmt.Errorf("patch delete mismatch at line %d: expected line %q but reached end of text", srcIdx+1, hline.Text)
				}
				if lines[srcIdx] != hline.Text {
					return "", fmt.Errorf("patch delete mismatch at line %d: expected %q, got %q", srcIdx+1, hline.Text, lines[srcIdx])
				}
				srcIdx++
			}
		}
	}

	// Copy remaining lines after the last hunk
	for srcIdx < len(lines) {
		result = append(result, lines[srcIdx])
		srcIdx++
	}

	output := strings.Join(result, "\n")
	if len(result) > 0 && hadTrailingNewline {
		output += "\n"
	}
	return output, nil
}

// --- Word-level diff ---

// ComputeWordDiff computes a word-level diff between two texts.
// Words are split on whitespace while preserving the whitespace between them.
func ComputeWordDiff(oldText, newText string) WordDiff {
	oldWords := tokenizeWords(oldText)
	newWords := tokenizeWords(newText)

	dp := lcsTable(oldWords, newWords)

	var words []WordOp
	i, j := len(oldWords), len(newWords)

	type entry struct {
		op   Operation
		text string
	}
	var entries []entry

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldWords[i-1] == newWords[j-1] {
			entries = append(entries, entry{OpEqual, oldWords[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			entries = append(entries, entry{OpInsert, newWords[j-1]})
			j--
		} else if i > 0 {
			entries = append(entries, entry{OpDelete, oldWords[i-1]})
			i--
		}
	}

	for k := len(entries) - 1; k >= 0; k-- {
		words = append(words, WordOp{Op: entries[k].op, Text: entries[k].text})
	}

	return WordDiff{Words: words}
}

// tokenizeWords splits text into tokens, preserving whitespace as separate tokens.
func tokenizeWords(text string) []string {
	var tokens []string
	var current strings.Builder
	inWord := false

	for _, ch := range text {
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			if inWord {
				tokens = append(tokens, current.String())
				current.Reset()
				inWord = false
			}
			current.WriteRune(ch)
		} else {
			if !inWord && current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			current.WriteRune(ch)
			inWord = true
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// FormatWordDiff formats a word-level diff as a human-readable string.
// Uses [-old-] for deletions and {+new+} for insertions.
func FormatWordDiff(wd WordDiff) string {
	var sb strings.Builder
	for _, w := range wd.Words {
		switch w.Op {
		case OpEqual:
			sb.WriteString(w.Text)
		case OpDelete:
			sb.WriteString("[-" + w.Text + "-]")
		case OpInsert:
			sb.WriteString("{+" + w.Text + "+}")
		}
	}
	return sb.String()
}

// --- Summary ---

// DiffSummary returns a summary of changes between two texts.
type DiffSummary struct {
	Added    int
	Removed  int
	Changed  int
	Unchanged int
}

// Summarize returns a summary of the line-level diff.
func Summarize(oldText, newText string) DiffSummary {
	lines := LineDiff(oldText, newText)
	var s DiffSummary
	for _, line := range lines {
		switch line.Op {
		case OpEqual:
			s.Unchanged++
		case OpInsert:
			s.Added++
		case OpDelete:
			s.Removed++
		}
	}
	s.Changed = s.Added + s.Removed
	return s
}

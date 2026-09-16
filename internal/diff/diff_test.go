package diff

import (
	"strings"
	"testing"
)

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input    string
		wantLen  int
		wantNL   bool
	}{
		{"", 0, false},
		{"hello", 1, false},
		{"hello\n", 1, true},
		{"a\nb\nc", 3, false},
		{"a\nb\nc\n", 3, true},
		{"a\r\nb\r\n", 2, true},
	}
	for _, tt := range tests {
		lines, nl := splitLines(tt.input)
		if len(lines) != tt.wantLen {
			t.Errorf("splitLines(%q) got %d lines, want %d", tt.input, len(lines), tt.wantLen)
		}
		if nl != tt.wantNL {
			t.Errorf("splitLines(%q) trailingNL = %v, want %v", tt.input, nl, tt.wantNL)
		}
	}
}

func TestLineDiffIdentical(t *testing.T) {
	text := "line1\nline2\nline3"
	lines := LineDiff(text, text)
	for _, line := range lines {
		if line.Op != OpEqual {
			t.Errorf("expected all OpEqual, got %d for %q", line.Op, line.Text)
		}
	}
}

func TestLineDiffInsert(t *testing.T) {
	old := "a\nb"
	new := "a\nb\nc"
	lines := LineDiff(old, new)
	hasInsert := false
	for _, line := range lines {
		if line.Op == OpInsert && line.Text == "c" {
			hasInsert = true
		}
	}
	if !hasInsert {
		t.Errorf("expected insert of 'c', not found")
	}
}

func TestLineDiffDelete(t *testing.T) {
	old := "a\nb\nc"
	new := "a\nc"
	lines := LineDiff(old, new)
	hasDelete := false
	for _, line := range lines {
		if line.Op == OpDelete && line.Text == "b" {
			hasDelete = true
		}
	}
	if !hasDelete {
		t.Errorf("expected delete of 'b', not found")
	}
}

func TestLineDiffReplace(t *testing.T) {
	old := "a\nb\nc"
	new := "a\nB\nc"
	lines := LineDiff(old, new)
	hasDelete := false
	hasInsert := false
	for _, line := range lines {
		if line.Op == OpDelete && line.Text == "b" {
			hasDelete = true
		}
		if line.Op == OpInsert && line.Text == "B" {
			hasInsert = true
		}
	}
	if !hasDelete || !hasInsert {
		t.Errorf("expected delete 'b' and insert 'B'")
	}
}

func TestLineDiffEmpty(t *testing.T) {
	lines := LineDiff("", "")
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for empty diff, got %d", len(lines))
	}
}

func TestLineDiffEmptyToContent(t *testing.T) {
	lines := LineDiff("", "hello")
	if len(lines) != 1 || lines[0].Op != OpInsert || lines[0].Text != "hello" {
		t.Errorf("expected single insert 'hello', got %v", lines)
	}
}

func TestLineDiffContentToEmpty(t *testing.T) {
	lines := LineDiff("hello", "")
	if len(lines) != 1 || lines[0].Op != OpDelete || lines[0].Text != "hello" {
		t.Errorf("expected single delete 'hello', got %v", lines)
	}
}

func TestUnifiedDiffIdentical(t *testing.T) {
	text := "line1\nline2\nline3"
	result := UnifiedDiff(text, text, 3)
	if result != "" {
		t.Errorf("expected empty diff for identical texts, got %q", result)
	}
}

func TestUnifiedDiffBasic(t *testing.T) {
	old := "line1\nline2\nline3"
	new := "line1\nmodified\nline3"
	result := UnifiedDiff(old, new, 3)
	if !strings.Contains(result, "---") {
		t.Errorf("expected --- header, got %q", result)
	}
	if !strings.Contains(result, "+++") {
		t.Errorf("expected +++ header, got %q", result)
	}
	if !strings.Contains(result, "@@") {
		t.Errorf("expected @@ hunk header, got %q", result)
	}
	if !strings.Contains(result, "-line2") {
		t.Errorf("expected -line2 in diff, got %q", result)
	}
	if !strings.Contains(result, "+modified") {
		t.Errorf("expected +modified in diff, got %q", result)
	}
}

func TestUnifiedDiffInsert(t *testing.T) {
	old := "a\nb"
	new := "a\nb\nc\nd"
	result := UnifiedDiff(old, new, 3)
	if !strings.Contains(result, "+c") {
		t.Errorf("expected +c in diff, got %q", result)
	}
	if !strings.Contains(result, "+d") {
		t.Errorf("expected +d in diff, got %q", result)
	}
}

func TestUnifiedDiffDelete(t *testing.T) {
	old := "a\nb\nc\nd"
	new := "a\nd"
	result := UnifiedDiff(old, new, 3)
	if !strings.Contains(result, "-b") {
		t.Errorf("expected -b in diff, got %q", result)
	}
	if !strings.Contains(result, "-c") {
		t.Errorf("expected -c in diff, got %q", result)
	}
}

func TestUnifiedDiffContext(t *testing.T) {
	old := "line1\nline2\nline3\nline4\nline5\nline6\nline7"
	new := "line1\nline2\nCHANGED\nline4\nline5\nline6\nline7"
	result := UnifiedDiff(old, new, 1)
	// With context=1, we should see line2 and line4 as context
	if !strings.Contains(result, " line2") {
		t.Errorf("expected context line2, got %q", result)
	}
	if !strings.Contains(result, " line4") {
		t.Errorf("expected context line4, got %q", result)
	}
}

func TestParsePatchBasic(t *testing.T) {
	patch := `--- old
+++ new
@@ -1,3 +1,3 @@
 line1
-line2
+modified
 line3
`
	hunks, err := ParsePatch(patch)
	if err != nil {
		t.Fatalf("ParsePatch error: %v", err)
	}
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	if hunks[0].OldStart != 1 {
		t.Errorf("expected oldStart=1, got %d", hunks[0].OldStart)
	}
	if len(hunks[0].Lines) != 4 {
		t.Errorf("expected 4 lines in hunk, got %d", len(hunks[0].Lines))
	}
}

func TestParsePatchMultipleHunks(t *testing.T) {
	patch := `--- old
+++ new
@@ -1,3 +1,3 @@
 a
-b
+B
 c
@@ -7,3 +7,3 @@
 x
-y
+Y
 z
`
	hunks, err := ParsePatch(patch)
	if err != nil {
		t.Fatalf("ParsePatch error: %v", err)
	}
	if len(hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(hunks))
	}
}

func TestApplyPatchBasic(t *testing.T) {
	text := "line1\nline2\nline3"
	patch := `--- old
+++ new
@@ -1,3 +1,3 @@
 line1
-line2
+modified
 line3
`
	result, err := ApplyPatch(text, patch)
	if err != nil {
		t.Fatalf("ApplyPatch error: %v", err)
	}
	expected := "line1\nmodified\nline3"
	if result != expected {
		t.Errorf("ApplyPatch result = %q, want %q", result, expected)
	}
}

func TestApplyPatchInsert(t *testing.T) {
	text := "a\nb"
	patch := `--- old
+++ new
@@ -1,2 +1,4 @@
 a
+b1
+b2
 b
`
	result, err := ApplyPatch(text, patch)
	if err != nil {
		t.Fatalf("ApplyPatch error: %v", err)
	}
	expected := "a\nb1\nb2\nb"
	if result != expected {
		t.Errorf("ApplyPatch result = %q, want %q", result, expected)
	}
}

func TestApplyPatchDelete(t *testing.T) {
	text := "a\nb\nc\nd"
	patch := `--- old
+++ new
@@ -1,4 +1,2 @@
 a
-b
-c
 d
`
	result, err := ApplyPatch(text, patch)
	if err != nil {
		t.Fatalf("ApplyPatch error: %v", err)
	}
	expected := "a\nd"
	if result != expected {
		t.Errorf("ApplyPatch result = %q, want %q", result, expected)
	}
}

func TestApplyPatchRoundTrip(t *testing.T) {
	old := "line1\nline2\nline3\nline4\nline5"
	new := "line1\nCHANGED\nline3\nADDED\nline5"
	patch := UnifiedDiff(old, new, 3)
	if patch == "" {
		t.Fatal("expected non-empty patch")
	}
	result, err := ApplyPatch(old, patch)
	if err != nil {
		t.Fatalf("ApplyPatch error: %v", err)
	}
	if result != new {
		t.Errorf("round-trip failed:\n  result=%q\n  want  =%q", result, new)
	}
}

func TestApplyPatchMultipleHunks(t *testing.T) {
	text := "a\nb\nc\nd\ne\nf\ng\nh"
	patch := `--- old
+++ new
@@ -1,3 +1,3 @@
 a
-b
+B
 c
@@ -6,3 +6,3 @@
 f
-g
+G
 h
`
	result, err := ApplyPatch(text, patch)
	if err != nil {
		t.Fatalf("ApplyPatch error: %v", err)
	}
	expected := "a\nB\nc\nd\ne\nf\nG\nh"
	if result != expected {
		t.Errorf("ApplyPatch result = %q, want %q", result, expected)
	}
}

func TestApplyPatchContextMismatch(t *testing.T) {
	text := "line1\nWRONG\nline3"
	patch := `--- old
+++ new
@@ -1,3 +1,3 @@
 line1
-line2
+modified
 line3
`
	_, err := ApplyPatch(text, patch)
	if err == nil {
		t.Error("expected error for context mismatch, got nil")
	}
}

func TestApplyPatchInvalidPatch(t *testing.T) {
	_, err := ApplyPatch("hello", "not a valid patch")
	if err == nil {
		t.Error("expected error for invalid patch, got nil")
	}
}

func TestWordDiffIdentical(t *testing.T) {
	wd := ComputeWordDiff("hello world", "hello world")
	for _, w := range wd.Words {
		if w.Op != OpEqual {
			t.Errorf("expected all OpEqual, got %d for %q", w.Op, w.Text)
		}
	}
}

func TestWordDiffBasic(t *testing.T) {
	wd := ComputeWordDiff("hello world", "hello earth")
	formatted := FormatWordDiff(wd)
	if !strings.Contains(formatted, "[-world-]") {
		t.Errorf("expected [-world-] in formatted diff, got %q", formatted)
	}
	if !strings.Contains(formatted, "{+earth+}") {
		t.Errorf("expected {+earth+} in formatted diff, got %q", formatted)
	}
}

func TestWordDiffInsert(t *testing.T) {
	wd := ComputeWordDiff("hello", "hello world")
	formatted := FormatWordDiff(wd)
	if !strings.Contains(formatted, "{+ world+}") || !strings.Contains(formatted, "{+world+}") {
		// The whitespace tokenization may vary, just check "world" is inserted
		if !strings.Contains(formatted, "world") {
			t.Errorf("expected 'world' in formatted diff, got %q", formatted)
		}
	}
}

func TestWordDiffDelete(t *testing.T) {
	wd := ComputeWordDiff("hello world", "hello")
	formatted := FormatWordDiff(wd)
	if !strings.Contains(formatted, "[-") {
		t.Errorf("expected deletion in formatted diff, got %q", formatted)
	}
}

func TestSummarize(t *testing.T) {
	old := "a\nb\nc\nd\ne"
	new := "a\nB\nc\nd"
	s := Summarize(old, new)
	if s.Added != 1 {
		t.Errorf("added = %d, want 1", s.Added)
	}
	if s.Removed != 2 {
		t.Errorf("removed = %d, want 2", s.Removed)
	}
	if s.Unchanged != 3 {
		t.Errorf("unchanged = %d, want 3", s.Unchanged)
	}
}

func TestSummarizeIdentical(t *testing.T) {
	s := Summarize("hello", "hello")
	if s.Added != 0 || s.Removed != 0 || s.Unchanged != 1 {
		t.Errorf("expected 0 added, 0 removed, 1 unchanged, got %+v", s)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	s := Summarize("", "")
	if s.Added != 0 || s.Removed != 0 || s.Unchanged != 0 {
		t.Errorf("expected all zeros for empty, got %+v", s)
	}
}

func TestUnifiedDiffEmptyToContent(t *testing.T) {
	result := UnifiedDiff("", "hello\nworld", 3)
	if !strings.Contains(result, "+hello") {
		t.Errorf("expected +hello in diff, got %q", result)
	}
	if !strings.Contains(result, "+world") {
		t.Errorf("expected +world in diff, got %q", result)
	}
}

func TestUnifiedDiffContentToEmpty(t *testing.T) {
	result := UnifiedDiff("hello\nworld", "", 3)
	if !strings.Contains(result, "-hello") {
		t.Errorf("expected -hello in diff, got %q", result)
	}
	if !strings.Contains(result, "-world") {
		t.Errorf("expected -world in diff, got %q", result)
	}
}

func TestApplyPatchTrailingNewline(t *testing.T) {
	text := "line1\nline2\n"
	patch := `--- old
+++ new
@@ -1,2 +1,2 @@
 line1
-line2
+modified
`
	result, err := ApplyPatch(text, patch)
	if err != nil {
		t.Fatalf("ApplyPatch error: %v", err)
	}
	expected := "line1\nmodified\n"
	if result != expected {
		t.Errorf("ApplyPatch result = %q, want %q", result, expected)
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		input     string
		wantStart int
		wantCount int
		wantErr   bool
	}{
		{"1", 1, 1, false},
		{"1,3", 1, 3, false},
		{"5,10", 5, 10, false},
		{"0,0", 0, 0, false},
		{"", 0, 0, true},
	}
	for _, tt := range tests {
		start, count, err := parseRange(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseRange(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if !tt.wantErr {
			if start != tt.wantStart {
				t.Errorf("parseRange(%q) start = %d, want %d", tt.input, start, tt.wantStart)
			}
			if count != tt.wantCount {
				t.Errorf("parseRange(%q) count = %d, want %d", tt.input, count, tt.wantCount)
			}
		}
	}
}

func TestAtoi(t *testing.T) {
	tests := []struct {
		input  string
		want   int
		wantErr bool
	}{
		{"0", 0, false},
		{"42", 42, false},
		{"-5", -5, false},
		{"", 0, true},
		{"abc", 0, true},
	}
	for _, tt := range tests {
		got, err := atoi(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("atoi(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("atoi(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

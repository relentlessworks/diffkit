package diff

// Operation describes a diff operation type.
type Operation int

const (
	OpEqual  Operation = 0 // line is present in both texts
	OpInsert Operation = 1 // line is only in the new text
	OpDelete Operation = 2 // line is only in the old text
)

// Line represents a single line in a diff with its operation.
type Line struct {
	Op    Operation
	Text  string
	OldNo int // 1-based line number in old text (0 if not present)
	NewNo int // 1-based line number in new text (0 if not present)
}

// Hunk represents a contiguous group of diff lines with line-number context.
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []Line
}

// WordDiff represents a word-level diff between two texts.
type WordDiff struct {
	Words []WordOp
}

// WordOp represents a single word in a word-level diff.
type WordOp struct {
	Op   Operation
	Text string
}

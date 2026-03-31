package types

// Hunk represents a changed section in a diff.
type Hunk struct {
	StartLine    int    `json:"startLine"`
	EndLine      int    `json:"endLine"`
	OldStartLine int    `json:"oldStartLine"`
	Content      string `json:"content"`
	RawContent   string `json:"rawContent"`
}

// FileChange represents a file that was changed.
type FileChange struct {
	Path  string `json:"path"`
	Hunks []Hunk `json:"hunks"`
}

// Diff represents the differences between two branches.
type Diff struct {
	Branch string       `json:"branch"`
	Base   string       `json:"base"`
	Files  []FileChange `json:"files"`
}

package types

type ScoreStatus string

const (
	ScoreFail         ScoreStatus = "FAIL"
	ScorePass         ScoreStatus = "PASS"
	ScoreNeedsChanges ScoreStatus = "NEEDS_CHANGES"
)

type Score struct {
	Status        ScoreStatus `json:"status"`
	CriticalCount int         `json:"criticalCount"`
	WarningCount  int         `json:"warningCount"`
}

type Metadata struct {
	Branch        string `json:"branch"`
	Base          string `json:"base"`
	Duration      string `json:"duration"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	FilesAnalyzed int    `json:"filesAnalyzed"`
}

type ReviewResult struct {
	Findings []Finding `json:"findings"`
	Score    Score     `json:"score"`
	Metadata Metadata  `json:"metadata"`
}

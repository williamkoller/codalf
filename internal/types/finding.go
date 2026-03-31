package types

type Finding struct {
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Agent      string   `json:"agent"`
	Severity   Severity `json:"severity"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion,omitempty"`
}

package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/williamkoller/codalf/internal/types"
)

func TestWriteJSON(t *testing.T) {
	result := &types.ReviewResult{
		Findings: []types.Finding{
			{File: "test.go", Line: 10, Agent: "general", Severity: types.SeverityWarning, Message: "Test message"},
		},
		Score: types.Score{
			Status:        types.ScorePass,
			CriticalCount: 0,
			WarningCount:  1,
		},
	}

	var buf bytes.Buffer
	err := WriteJSON(&buf, result)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	if !strings.Contains(buf.String(), "test.go") {
		t.Error("JSON output should contain file name")
	}
	if !strings.Contains(buf.String(), "general") {
		t.Error("JSON output should contain agent name")
	}
}

func TestWriteInline_NoFindings(t *testing.T) {
	result := &types.ReviewResult{
		Findings: []types.Finding{},
		Score: types.Score{
			Status:        types.ScorePass,
			CriticalCount: 0,
			WarningCount:  0,
		},
	}

	var buf bytes.Buffer
	err := WriteInline(&buf, result, nil)
	if err != nil {
		t.Fatalf("WriteInline failed: %v", err)
	}

	if !strings.Contains(buf.String(), "No issues found") {
		t.Error("Output should indicate no issues found")
	}
	if !strings.Contains(buf.String(), "Approved") {
		t.Error("Output should contain Approved status")
	}
}

func TestWriteInline_WithFindings(t *testing.T) {
	result := &types.ReviewResult{
		Findings: []types.Finding{
			{File: "a.go", Line: 1, Agent: "general", Severity: types.SeverityWarning, Message: "msg1"},
			{File: "b.go", Line: 5, Agent: "general", Severity: types.SeverityCritical, Message: "msg2"},
		},
		Score: types.Score{
			Status:        types.ScoreFail,
			CriticalCount: 1,
			WarningCount:  1,
		},
	}

	var buf bytes.Buffer
	err := WriteInline(&buf, result, nil)
	if err != nil {
		t.Fatalf("WriteInline failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "a.go") {
		t.Error("Output should contain file a.go")
	}
	if !strings.Contains(out, "b.go") {
		t.Error("Output should contain file b.go")
	}
	if !strings.Contains(out, "Changes required") {
		t.Error("Output should show Changes required status")
	}
}

func TestWriteInline_WithDiff(t *testing.T) {
	diff := &types.Diff{
		Branch: "feature",
		Base:   "main",
		Files: []types.FileChange{
			{
				Path: "main.go",
				Hunks: []types.Hunk{
					{
						OldStartLine: 10,
						StartLine:    10,
						EndLine:      13,
						RawContent:   " func foo() {\n-\tprintln(\"hi\")\n+\tfmt.Println(\"hi\")\n }\n",
					},
				},
			},
		},
	}

	result := &types.ReviewResult{
		Findings: []types.Finding{
			{File: "main.go", Line: 11, Agent: "general", Severity: types.SeverityCritical, Message: "use fmt.Println", Suggestion: "replace println with fmt.Println"},
		},
		Score: types.Score{Status: types.ScoreFail, CriticalCount: 1},
	}

	var buf bytes.Buffer
	err := WriteInline(&buf, result, diff)
	if err != nil {
		t.Fatalf("WriteInline failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "main.go") {
		t.Error("Output should contain file name")
	}
	if !strings.Contains(out, "fmt.Println") {
		t.Error("Output should contain added line content")
	}
	if !strings.Contains(out, "Changes required") {
		t.Error("Output should show Changes required status")
	}
}

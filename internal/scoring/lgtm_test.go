package scoring

import (
	"testing"

	"github.com/williamkoller/codalf/internal/types"
)

func TestCalculate_FailOnCritical(t *testing.T) {
	findings := []types.Finding{
		{File: "test.go", Line: 1, Severity: types.SeverityCritical, Message: "test"},
	}

	score := Calculate(findings)

	if score.Status != types.ScoreFail {
		t.Errorf("expected FAIL, got %s", score.Status)
	}
	if score.CriticalCount != 1 {
		t.Errorf("expected CriticalCount=1, got %d", score.CriticalCount)
	}
}

func TestCalculate_PassOnLowWarnings(t *testing.T) {
	findings := []types.Finding{
		{File: "test.go", Line: 1, Severity: types.SeverityWarning, Message: "test1"},
		{File: "test.go", Line: 2, Severity: types.SeverityWarning, Message: "test2"},
	}

	score := Calculate(findings)

	if score.Status != types.ScorePass {
		t.Errorf("expected PASS, got %s", score.Status)
	}
	if score.WarningCount != 2 {
		t.Errorf("expected WarningCount=2, got %d", score.WarningCount)
	}
}

func TestCalculate_NeedsChangesOnHighWarnings(t *testing.T) {
	findings := []types.Finding{
		{File: "test.go", Line: 1, Severity: types.SeverityWarning, Message: "test1"},
		{File: "test.go", Line: 2, Severity: types.SeverityWarning, Message: "test2"},
		{File: "test.go", Line: 3, Severity: types.SeverityWarning, Message: "test3"},
	}

	score := Calculate(findings)

	if score.Status != types.ScoreNeedsChanges {
		t.Errorf("expected NEEDS_CHANGES, got %s", score.Status)
	}
}

func TestCalculate_InfoNotCounted(t *testing.T) {
	findings := []types.Finding{
		{File: "test.go", Line: 1, Severity: types.SeverityInfo, Message: "test"},
	}

	score := Calculate(findings)

	if score.Status != types.ScorePass {
		t.Errorf("expected PASS for info-only findings, got %s", score.Status)
	}
}

func TestCalculate_EmptyFindings(t *testing.T) {
	findings := []types.Finding{}

	score := Calculate(findings)

	if score.Status != types.ScorePass {
		t.Errorf("expected PASS for empty findings, got %s", score.Status)
	}
}

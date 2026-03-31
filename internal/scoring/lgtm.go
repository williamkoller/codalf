package scoring

import (
	"log/slog"

	"github.com/williamkoller/codalf/internal/types"
)

func Calculate(findings []types.Finding) types.Score {
	slog.Debug("Scoring: Calculating score", "findings_count", len(findings))

	criticalCount := 0
	warningCount := 0

	for _, f := range findings {
		switch f.Severity {
		case types.SeverityCritical:
			criticalCount++
		case types.SeverityWarning:
			warningCount++
		}
	}

	var status types.ScoreStatus
	if criticalCount > 0 {
		status = types.ScoreFail
	} else if warningCount <= 2 {
		status = types.ScorePass
	} else {
		status = types.ScoreNeedsChanges
	}

	slog.Debug("Scoring: Score calculated", "status", status, "critical", criticalCount, "warning", warningCount)

	return types.Score{
		Status:        status,
		CriticalCount: criticalCount,
		WarningCount:  warningCount,
	}
}

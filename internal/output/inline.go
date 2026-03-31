package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/williamkoller/codalf/internal/types"
)

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorBlue   = "\033[34m"
	bgDarkGreen = "\033[48;5;22m"
	bgDarkRed   = "\033[48;5;52m"
)

func WriteInline(w io.Writer, result *types.ReviewResult, diff *types.Diff) error {
	if diff != nil && len(diff.Files) > 0 {
		writeDiff(w, diff, result.Findings)
	} else {
		writeFindings(w, result)
	}

	writeReviewSummary(w, result)
	printSummary(w, result)

	return nil
}

// writeDiff renders a GitHub-style diff with findings annotated inline.
func writeDiff(w io.Writer, diff *types.Diff, findings []types.Finding) {
	findingsByFile := groupByFile(findings)

	for _, file := range diff.Files {
		fileFindingsByLine := make(map[int][]types.Finding)
		for _, f := range findingsByFile[file.Path] {
			fileFindingsByLine[f.Line] = append(fileFindingsByLine[f.Line], f)
		}

		fileFindings := findingsByFile[file.Path]
		fileIssueLabel := formatIssueCount(fileFindings)

		// File header with issue count
		fmt.Fprintf(w, "\n  %s┌─ %s%s",
			colorCyan+colorBold, file.Path, colorReset)
		if fileIssueLabel != "" {
			fmt.Fprintf(w, "  %s·  %s%s", colorDim, fileIssueLabel, colorReset)
		}
		fmt.Fprintln(w)

		for _, hunk := range file.Hunks {
			renderHunk(w, hunk, fileFindingsByLine)
		}

		fmt.Fprintf(w, "  %s└%s%s\n",
			colorCyan+colorBold, strings.Repeat("─", 71), colorReset)
	}
}

func formatIssueCount(findings []types.Finding) string {
	if len(findings) == 0 {
		return ""
	}
	critical, warning, info := 0, 0, 0
	for _, f := range findings {
		switch f.Severity {
		case types.SeverityCritical:
			critical++
		case types.SeverityWarning:
			warning++
		default:
			info++
		}
	}
	var parts []string
	if critical > 0 {
		parts = append(parts, fmt.Sprintf("%s%d critical%s", colorRed+colorBold, critical, colorReset+colorDim))
	}
	if warning > 0 {
		parts = append(parts, fmt.Sprintf("%s%d warning%s", colorYellow+colorBold, warning, colorReset+colorDim))
	}
	if info > 0 {
		parts = append(parts, fmt.Sprintf("%d info", info))
	}
	return strings.Join(parts, ", ")
}

func renderHunk(w io.Writer, hunk types.Hunk, findingsByLine map[int][]types.Finding) {
	lines := strings.Split(strings.TrimRight(hunk.RawContent, "\n"), "\n")

	newLine := hunk.StartLine
	oldLine := hunk.OldStartLine

	// Hunk header
	fmt.Fprintf(w, "  %s│%s  %s@@ -%d +%d @@%s\n",
		colorCyan+colorBold, colorReset,
		colorDim, oldLine, newLine, colorReset)

	for _, raw := range lines {
		if raw == "" {
			continue
		}

		marker := raw[0]
		content := ""
		if len(raw) > 1 {
			content = raw[1:]
		}

		switch marker {
		case '+':
			lineNum := newLine
			newLine++
			fmt.Fprintf(w, "  %s│%s %s+%4d%s %s%s%s\n",
				colorCyan+colorBold, colorReset,
				colorGreen+colorBold, lineNum, colorReset,
				bgDarkGreen+colorGreen, content, colorReset)

			for _, f := range findingsByLine[lineNum] {
				renderFindingAnnotation(w, f)
			}

		case '-':
			lineNum := oldLine
			oldLine++
			fmt.Fprintf(w, "  %s│%s %s-%4d%s %s%s%s\n",
				colorCyan+colorBold, colorReset,
				colorRed+colorBold, lineNum, colorReset,
				bgDarkRed+colorRed, content, colorReset)

		default: // context line (space)
			fmt.Fprintf(w, "  %s│%s  %4d  %s%s%s\n",
				colorCyan+colorBold, colorReset,
				newLine, colorDim, content, colorReset)
			newLine++
			oldLine++
		}
	}
}

func renderFindingAnnotation(w io.Writer, f types.Finding) {
	icon, color := severityStyle(f.Severity)
	// Arrow pointing back to the code line above
	fmt.Fprintf(w, "  %s│%s       %s╰─ %s%s  %s%s\n",
		colorCyan+colorBold, colorReset,
		colorDim, color+colorBold, icon, f.Message, colorReset)
	if f.Suggestion != "" {
		fmt.Fprintf(w, "  %s│%s          %s↳  %s%s\n",
			colorCyan+colorBold, colorReset,
			colorGreen+colorDim, f.Suggestion, colorReset)
	}
}

// writeFindings is used when no diff is available — shows findings grouped by file.
func writeFindings(w io.Writer, result *types.ReviewResult) {
	grouped := groupByFile(result.Findings)

	var files []string
	for f := range grouped {
		files = append(files, f)
	}
	sort.Strings(files)

	if len(files) == 0 {
		fmt.Fprintf(w, "\n  %s✓  No issues found%s\n", colorGreen+colorBold, colorReset)
		return
	}

	for _, file := range files {
		findings := grouped[file]
		sort.Slice(findings, func(i, j int) bool {
			return findings[i].Line < findings[j].Line
		})

		fileIssueLabel := formatIssueCount(findings)
		fmt.Fprintf(w, "\n  %s┌─ %s%s", colorCyan+colorBold, file, colorReset)
		if fileIssueLabel != "" {
			fmt.Fprintf(w, "  %s·  %s%s", colorDim, fileIssueLabel, colorReset)
		}
		fmt.Fprintln(w)

		for _, f := range findings {
			icon, color := severityStyle(f.Severity)
			fmt.Fprintf(w, "  %s│%s  %s%s  L%d%s  %s%s%s\n",
				colorCyan+colorBold, colorReset,
				color+colorBold, icon, f.Line, colorReset,
				colorBold, f.Message, colorReset)
			if f.Suggestion != "" {
				fmt.Fprintf(w, "  %s│%s     %s↳  %s%s\n",
					colorCyan+colorBold, colorReset,
					colorGreen+colorDim, f.Suggestion, colorReset)
			}
		}

		fmt.Fprintf(w, "  %s└%s%s\n", colorCyan+colorBold, strings.Repeat("─", 71), colorReset)
	}
}

// writeReviewSummary prints a grouped, actionable list of all findings after the diff.
func writeReviewSummary(w io.Writer, result *types.ReviewResult) {
	if len(result.Findings) == 0 {
		return
	}

	// Sort: critical first, then warning, then info; within each group by file+line
	sorted := make([]types.Finding, len(result.Findings))
	copy(sorted, result.Findings)
	sort.Slice(sorted, func(i, j int) bool {
		si := severityOrder(sorted[i].Severity)
		sj := severityOrder(sorted[j].Severity)
		if si != sj {
			return si < sj
		}
		if sorted[i].File != sorted[j].File {
			return sorted[i].File < sorted[j].File
		}
		return sorted[i].Line < sorted[j].Line
	})

	fmt.Fprintf(w, "\n  %sReview Summary%s  %s%s%s\n",
		colorBold, colorReset,
		colorDim, strings.Repeat("─", 57), colorReset)

	for _, f := range sorted {
		icon, color := severityStyle(f.Severity)
		location := fmt.Sprintf("%s:%d", f.File, f.Line)
		fmt.Fprintf(w, "\n  %s%s  %s%s%s  %s%s%s\n",
			color+colorBold, icon,
			colorReset+colorDim, location, colorReset,
			colorBold, f.Message, colorReset)
		if f.Suggestion != "" {
			fmt.Fprintf(w, "     %sFix → %s%s\n",
				colorGreen+colorDim, f.Suggestion, colorReset)
		}
	}
}

func severityOrder(s types.Severity) int {
	switch s {
	case types.SeverityCritical:
		return 0
	case types.SeverityWarning:
		return 1
	default:
		return 2
	}
}

func severityStyle(s types.Severity) (icon, color string) {
	switch s {
	case types.SeverityCritical:
		return "✗", colorRed + colorBold
	case types.SeverityWarning:
		return "⚠", colorYellow + colorBold
	default:
		return "ℹ", colorBlue
	}
}

func printSummary(w io.Writer, result *types.ReviewResult) {
	score := result.Score

	var statusColor, statusIcon string
	switch score.Status {
	case types.ScorePass:
		statusColor = colorGreen + colorBold
		statusIcon = "✓  Approved"
	case types.ScoreNeedsChanges:
		statusColor = colorYellow + colorBold
		statusIcon = "⚠  Changes requested"
	default:
		statusColor = colorRed + colorBold
		statusIcon = "✗  Changes required"
	}

	fmt.Fprintf(w, "\n  %s%s%s", statusColor, statusIcon, colorReset)

	if score.CriticalCount > 0 {
		fmt.Fprintf(w, "  %s%d critical%s", colorRed+colorBold, score.CriticalCount, colorReset)
	}
	if score.WarningCount > 0 {
		fmt.Fprintf(w, "  %s%d warning%s", colorYellow, score.WarningCount, colorReset)
	}
	if result.Metadata.FilesAnalyzed > 0 {
		label := "file"
		if result.Metadata.FilesAnalyzed != 1 {
			label = "files"
		}
		fmt.Fprintf(w, "  %s%d %s%s", colorDim, result.Metadata.FilesAnalyzed, label, colorReset)
	}
	if result.Metadata.Duration != "" {
		fmt.Fprintf(w, "  %s%s%s", colorDim, result.Metadata.Duration, colorReset)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w)
}

func groupByFile(findings []types.Finding) map[string][]types.Finding {
	grouped := make(map[string][]types.Finding)
	for _, f := range findings {
		grouped[f.File] = append(grouped[f.File], f)
	}
	return grouped
}

package git

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/williamkoller/codalf/internal/types"
)

var diffHunkRegex = regexp.MustCompile(`@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@`)

var reviewableExtensions = map[string]bool{
	".go":   true,
	".js":   true,
	".jsx":  true,
	".ts":   true,
	".tsx":  true,
	".css":  true,
	".scss": true,
}

var excludedPathPrefixes = []string{
	"vendor/",
	"node_modules/",
	"dist/",
	"build/",
	".gen/",
	"generated/",
	"mocks/",
	"mock/",
	"migrations/",
	"testdata/",
}

var excludedFileSuffixes = []string{
	"_test.go",
	".pb.go",
	".gen.go",
	"_mock.go",
	"mock_",
}

func isExcluded(path string) bool {
	for _, prefix := range excludedPathPrefixes {
		if strings.HasPrefix(path, prefix) || strings.Contains(path, "/"+prefix) {
			return true
		}
	}
	base := filepath.Base(path)
	for _, suffix := range excludedFileSuffixes {
		if strings.HasSuffix(base, suffix) || strings.HasPrefix(base, suffix) {
			return true
		}
	}
	return false
}

// GetAllFiles builds a synthetic Diff from all tracked files in the repo,
// used when reviewing the base branch (e.g. main) in full.
func GetAllFiles(ctx context.Context) (*types.Diff, error) {
	slog.Info("Git: Getting all tracked files for full review")

	cmd := exec.CommandContext(ctx, "git", "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list tracked files: %w", err)
	}

	diff := &types.Diff{Branch: "main", Base: "main", Files: []types.FileChange{}}

	for _, path := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if path == "" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !reviewableExtensions[ext] {
			continue
		}
		if isExcluded(path) {
			slog.Debug("Git: Skipping excluded path", "path", path)
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("Git: Could not read file, skipping", "path", path, "error", err)
			continue
		}

		lines := strings.Split(string(data), "\n")
		const maxLines = 150
		if len(lines) > maxLines {
			slog.Debug("Git: Skipping large file", "path", path, "lines", len(lines))
			continue
		}

		var content, rawContent strings.Builder
		for _, l := range lines {
			content.WriteString(" " + l + "\n")
			rawContent.WriteString(" " + l + "\n")
		}

		hunk := types.Hunk{
			OldStartLine: 1,
			StartLine:    1,
			EndLine:      len(lines),
			Content:      content.String(),
			RawContent:   rawContent.String(),
		}

		diff.Files = append(diff.Files, types.FileChange{
			Path:  path,
			Hunks: []types.Hunk{hunk},
		})
	}

	slog.Info("Git: Full review files collected", "files_count", len(diff.Files))
	return diff, nil
}

func GetCurrentBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return "", fmt.Errorf("repository is in detached HEAD state")
	}
	return branch, nil
}

func GetDiff(ctx context.Context, branch, base string) (*types.Diff, error) {
	slog.Info("Git: Getting diff", "branch", branch, "base", base)

	var output []byte
	var err error

	if branch != base {
		cmd := exec.CommandContext(ctx, "git", "diff", base+"..."+branch)
		output, err = cmd.Output()
		if err != nil {
			slog.Error("Git: Failed to get diff", "error", err)
			return nil, fmt.Errorf("failed to get diff: %w", err)
		}
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		slog.Info("Git: No committed diff found, falling back to uncommitted changes (git diff HEAD)")
		cmd := exec.CommandContext(ctx, "git", "diff", "HEAD")
		output, err = cmd.Output()
		if err != nil {
			slog.Error("Git: Failed to get HEAD diff", "error", err)
			return nil, fmt.Errorf("failed to get diff: %w", err)
		}
	}

	diff := parseDiff(string(output), branch, base)
	slog.Info("Git: Diff parsed", "files_count", len(diff.Files))
	return diff, nil
}

func parseDiff(output, branch, base string) *types.Diff {
	diff := &types.Diff{
		Branch: branch,
		Base:   base,
		Files:  []types.FileChange{},
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	var currentFile *types.FileChange
	var currentHunk *types.Hunk

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "diff --git") {
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
				}
				diff.Files = append(diff.Files, *currentFile)
			}
			path := extractPath(line)
			currentFile = &types.FileChange{Path: path, Hunks: []types.Hunk{}}
			currentHunk = nil
			continue
		}

		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil && currentFile != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}
			matches := diffHunkRegex.FindStringSubmatch(line)
			if len(matches) >= 4 {
				currentHunk = &types.Hunk{
					OldStartLine: parseInt(matches[1]),
					StartLine:    parseInt(matches[3]),
					EndLine:      parseInt(matches[3]) + parseInt(matches[4]) - 1,
					Content:      "",
					RawContent:   "",
				}
			}
			continue
		}

		if currentHunk != nil && !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---") {
			if strings.HasPrefix(line, "+") || strings.HasPrefix(line, " ") {
				currentHunk.Content += line + "\n"
			}
			if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") {
				currentHunk.RawContent += line + "\n"
			}
		}
	}

	if currentFile != nil {
		if currentHunk != nil {
			currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
		}
		diff.Files = append(diff.Files, *currentFile)
	}

	return diff
}

func extractPath(line string) string {
	parts := strings.Split(line, " b/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func parseInt(s string) int {
	if s == "" {
		return 1
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 1
	}
	return n
}

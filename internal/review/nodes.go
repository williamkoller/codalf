package review

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/williamkoller/codalf/internal/agents"
	"github.com/williamkoller/codalf/internal/provider"
	"github.com/williamkoller/codalf/internal/skills"
	"github.com/williamkoller/codalf/internal/types"
)

const (
	fileReviewTimeout = 3 * time.Minute
	batchSize         = 5
	concurrency       = 2
)

type GetDiffNode struct{}

func NewGetDiffNode() *GetDiffNode {
	return &GetDiffNode{}
}

func (n *GetDiffNode) Name() string {
	return "get_diff"
}

func (n *GetDiffNode) Execute(ctx context.Context, input any) (any, error) {
	if diff, ok := input.(*types.Diff); ok && diff != nil {
		return diff, nil
	}
	return input, nil
}

type RunAgentNode struct {
	provider provider.Provider
}

func NewRunAgentNode(p provider.Provider) *RunAgentNode {
	return &RunAgentNode{provider: p}
}

func (n *RunAgentNode) Name() string {
	return "run_agent"
}

func (n *RunAgentNode) Execute(ctx context.Context, input any) (any, error) {
	slog.Info("RunAgentNode: Starting execution", "provider", n.provider.Name())

	diff, ok := input.(*types.Diff)
	if !ok || diff == nil {
		slog.Error("RunAgentNode: Diff is nil or wrong type", "input_type", fmt.Sprintf("%T", input))
		return []types.Finding{}, nil
	}

	slog.Info("RunAgentNode: Processing diff", "files_count", len(diff.Files), "provider", n.provider.Name())

	goDiff, reactDiff := splitDiffByLanguage(diff)

	goBatches := chunkFiles(goDiff.Files, batchSize)
	reactBatches := chunkFiles(reactDiff.Files, batchSize)

	type result struct {
		findings []types.Finding
	}

	total := len(goBatches) + len(reactBatches)
	resultsCh := make(chan result, total)
	sem := make(chan struct{}, concurrency)

	var wg sync.WaitGroup

	goSkillCtx := skills.BuildSkillContext(".go")
	for _, batch := range goBatches {
		wg.Add(1)
		go func(batch []types.FileChange) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			batchCtx, cancel := context.WithTimeout(ctx, fileReviewTimeout)
			defer cancel()

			slog.Info("RunAgentNode: Running Go agent", "files", len(batch))
			batchDiff := &types.Diff{Branch: goDiff.Branch, Base: goDiff.Base, Files: batch}
			findings, err := agents.NewGeneralAgent(n.provider, goSkillCtx).Review(batchCtx, batchDiff)
			if err != nil {
				slog.Warn("RunAgentNode: Go batch skipped", "files", len(batch), "error", err)
				resultsCh <- result{}
				return
			}
			resultsCh <- result{findings: findings}
		}(batch)
	}

	for _, batch := range reactBatches {
		wg.Add(1)
		go func(batch []types.FileChange) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			batchCtx, cancel := context.WithTimeout(ctx, fileReviewTimeout)
			defer cancel()

			slog.Info("RunAgentNode: Running React agent", "files", len(batch))
			batchDiff := &types.Diff{Branch: reactDiff.Branch, Base: reactDiff.Base, Files: batch}
			reactSkillCtx := buildReactSkillContext(batchDiff)
			findings, err := agents.NewReactAgent(n.provider, reactSkillCtx).Review(batchCtx, batchDiff)
			if err != nil {
				slog.Warn("RunAgentNode: React batch skipped", "files", len(batch), "error", err)
				resultsCh <- result{}
				return
			}
			resultsCh <- result{findings: findings}
		}(batch)
	}

	wg.Wait()
	close(resultsCh)

	var allFindings []types.Finding
	for r := range resultsCh {
		allFindings = append(allFindings, r.findings...)
	}

	slog.Info("RunAgentNode: Completed", "findings_count", len(allFindings))
	return allFindings, nil
}

// buildReactSkillContext determines whether the diff has TypeScript and/or React files
// and builds the appropriate combined skill context.
func buildReactSkillContext(diff *types.Diff) string {
	hasTS := false
	hasReact := false
	for _, f := range diff.Files {
		ext := strings.ToLower(filepath.Ext(f.Path))
		if ext == ".ts" {
			hasTS = true
		}
		if ext == ".tsx" || ext == ".jsx" {
			hasReact = true
		}
	}

	var contexts []string
	if hasTS || hasReact {
		if ctx := skills.BuildSkillContext(".tsx"); ctx != "" {
			contexts = append(contexts, ctx)
		}
	}
	if hasTS && !hasReact {
		if ctx := skills.BuildSkillContext(".ts"); ctx != "" {
			contexts = append(contexts, ctx)
		}
	}

	return strings.Join(contexts, "\n")
}

func chunkFiles(files []types.FileChange, size int) [][]types.FileChange {
	var chunks [][]types.FileChange
	for size < len(files) {
		files, chunks = files[size:], append(chunks, files[:size])
	}
	return append(chunks, files)
}

var reactExtensions = map[string]bool{
	".js":   true,
	".jsx":  true,
	".ts":   true,
	".tsx":  true,
	".css":  true,
	".scss": true,
}

func splitDiffByLanguage(diff *types.Diff) (goDiff *types.Diff, reactDiff *types.Diff) {
	goDiff = &types.Diff{Branch: diff.Branch, Base: diff.Base, Files: []types.FileChange{}}
	reactDiff = &types.Diff{Branch: diff.Branch, Base: diff.Base, Files: []types.FileChange{}}

	for _, f := range diff.Files {
		ext := strings.ToLower(filepath.Ext(f.Path))
		if reactExtensions[ext] {
			reactDiff.Files = append(reactDiff.Files, f)
		} else if ext == ".go" {
			goDiff.Files = append(goDiff.Files, f)
		}
	}

	return goDiff, reactDiff
}

type MergeResultsNode struct{}

func NewMergeResultsNode() *MergeResultsNode {
	return &MergeResultsNode{}
}

func (n *MergeResultsNode) Name() string {
	return "merge_results"
}

func (n *MergeResultsNode) Execute(ctx context.Context, input any) (any, error) {
	if findings, ok := input.([]types.Finding); ok {
		return findings, nil
	}
	return []types.Finding{}, nil
}

type ScoreNode struct{}

func NewScoreNode() *ScoreNode {
	return &ScoreNode{}
}

func (n *ScoreNode) Name() string {
	return "score"
}

func (n *ScoreNode) Execute(ctx context.Context, input any) (any, error) {
	return input, nil
}

type OutputNode struct{}

func NewOutputNode() *OutputNode {
	return &OutputNode{}
}

func (n *OutputNode) Name() string {
	return "output"
}

func (n *OutputNode) Execute(ctx context.Context, input any) (any, error) {
	return input, nil
}

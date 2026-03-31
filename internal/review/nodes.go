package review

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/williamkoller/codalf/internal/agents"
	"github.com/williamkoller/codalf/internal/provider"
	"github.com/williamkoller/codalf/internal/skills"
	"github.com/williamkoller/codalf/internal/types"
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

	var allFindings []types.Finding

	if len(goDiff.Files) > 0 {
		slog.Info("RunAgentNode: Running Go agent", "files_count", len(goDiff.Files))
		goSkillCtx := skills.BuildSkillContext(".go")
		goFindings, err := agents.NewGeneralAgent(n.provider, goSkillCtx).Review(ctx, goDiff)
		if err != nil {
			slog.Error("RunAgentNode: Go agent error", "error", err)
			return []types.Finding{}, err
		}
		allFindings = append(allFindings, goFindings...)
	}

	if len(reactDiff.Files) > 0 {
		slog.Info("RunAgentNode: Running React agent", "files_count", len(reactDiff.Files))
		reactSkillCtx := buildReactSkillContext(reactDiff)
		reactFindings, err := agents.NewReactAgent(n.provider, reactSkillCtx).Review(ctx, reactDiff)
		if err != nil {
			slog.Error("RunAgentNode: React agent error", "error", err)
			return []types.Finding{}, err
		}
		allFindings = append(allFindings, reactFindings...)
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

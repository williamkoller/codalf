package review

import (
	"context"

	"github.com/williamkoller/codalf/internal/graph"
	"github.com/williamkoller/codalf/internal/provider"
	"github.com/williamkoller/codalf/internal/types"
)

type Pipeline struct {
	dag      *graph.DAG
	provider provider.Provider
}

func NewPipeline(p provider.Provider) *Pipeline {
	return &Pipeline{
		dag:      graph.NewDAG(),
		provider: p,
	}
}

func (p *Pipeline) Execute(ctx context.Context, diff *types.Diff) (*types.ReviewResult, error) {
	p.buildDAG()

	executor := graph.NewExecutor(p.dag)
	results, err := executor.Execute(ctx, diff)
	if err != nil {
		return nil, err
	}

	return p.aggregateResults(results), nil
}

func (p *Pipeline) buildDAG() {
	p.dag.AddNode(NewGetDiffNode())
	p.dag.AddNode(NewRunAgentNode(p.provider))
	p.dag.AddNode(NewMergeResultsNode())
	p.dag.AddNode(NewScoreNode())
	p.dag.AddNode(NewOutputNode())

	p.dag.AddEdge("get_diff", "run_agent")
	p.dag.AddEdge("run_agent", "merge_results")
	p.dag.AddEdge("merge_results", "score")
	p.dag.AddEdge("score", "output")
}

func (p *Pipeline) aggregateResults(results map[string]any) *types.ReviewResult {
	reviewResult := &types.ReviewResult{
		Findings: []types.Finding{},
	}

	if mergeResults, ok := results["merge_results"]; ok {
		if findings, ok := mergeResults.([]types.Finding); ok {
			reviewResult.Findings = findings
		}
	}

	return reviewResult
}

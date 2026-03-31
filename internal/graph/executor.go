package graph

import (
	"context"
	"log/slog"
	"sync"
)

type Executor struct {
	dag *DAG
}

func NewExecutor(dag *DAG) *Executor {
	return &Executor{dag: dag}
}

func (e *Executor) Execute(ctx context.Context, input any) (map[string]any, error) {
	slog.Info("Executor: Starting DAG execution", "nodes_count", len(e.dag.nodes), "edges_count", len(e.dag.edges))

	if err := e.dag.Validate(); err != nil {
		return nil, err
	}

	order := e.dag.GetExecutionOrder()
	results := make(map[string]any)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(e.dag.nodes))

	nodeInputs := make(map[string]any)
	nodeInputs[order[0]] = input

	dependencyCount := make(map[string]int)
	for _, node := range order[1:] {
		dependencyCount[node] = 0
	}
	for _, edge := range e.dag.edges {
		dependencyCount[edge.To]++
	}

	ready := make(chan string, len(e.dag.nodes))
	for _, node := range order {
		if dependencyCount[node] == 0 {
			ready <- node
		}
	}

	for {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case nodeName := <-ready:
			node := e.dag.nodes[nodeName]
			input := nodeInputs[nodeName]

			wg.Add(1)
			go func(n string, nd Node, inp any) {
				defer wg.Done()

				output, err := nd.Execute(ctx, inp)

				mu.Lock()
				if err != nil {
					errChan <- err
					mu.Unlock()
					return
				}
				results[n] = output

				for _, edge := range e.dag.edges {
					if edge.From == n {
						nodeInputs[edge.To] = output
						dependencyCount[edge.To]--
						if dependencyCount[edge.To] == 0 {
							ready <- edge.To
						}
					}
				}
				mu.Unlock()
			}(nodeName, node, input)

		default:
			if len(results) == len(e.dag.nodes) {
				wg.Wait()
				slog.Info("Executor: DAG execution completed", "results_count", len(results))
				return results, nil
			}

			select {
			case err := <-errChan:
				wg.Wait()
				return results, err
			default:
			}
		}
	}
}

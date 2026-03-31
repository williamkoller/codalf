package graph

import (
	"errors"
	"fmt"
	"log/slog"
)

var ErrCycleDetected = errors.New("cycle detected in DAG")

type DAG struct {
	nodes map[string]Node
	edges []Edge
}

func NewDAG() *DAG {
	return &DAG{
		nodes: make(map[string]Node),
		edges: make([]Edge, 0),
	}
}

func (d *DAG) AddNode(node Node) {
	slog.Debug("DAG: Adding node", "node", node.Name())
	d.nodes[node.Name()] = node
}

func (d *DAG) AddEdge(from, to string) {
	slog.Debug("DAG: Adding edge", "from", from, "to", to)
	d.edges = append(d.edges, Edge{From: from, To: to})
}

func (d *DAG) Validate() error {
	slog.Debug("DAG: Validating", "nodes_count", len(d.nodes), "edges_count", len(d.edges))

	nodeNames := make(map[string]bool)
	for name := range d.nodes {
		nodeNames[name] = true
	}

	for _, edge := range d.edges {
		if !nodeNames[edge.From] {
			slog.Error("DAG: Edge references unknown node", "node", edge.From)
			return fmt.Errorf("edge references unknown node: %s", edge.From)
		}
		if !nodeNames[edge.To] {
			slog.Error("DAG: Edge references unknown node", "node", edge.To)
			return fmt.Errorf("edge references unknown node: %s", edge.To)
		}
	}

	if d.hasCycle() {
		slog.Error("DAG: Cycle detected")
		return ErrCycleDetected
	}

	slog.Debug("DAG: Validation passed")
	return nil
}

func (d *DAG) hasCycle() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, edge := range d.edges {
			if edge.From == node {
				if !visited[edge.To] {
					if dfs(edge.To) {
						return true
					}
				} else if recStack[edge.To] {
					return true
				}
			}
		}

		recStack[node] = false
		return false
	}

	for node := range d.nodes {
		if !visited[node] {
			if dfs(node) {
				return true
			}
		}
	}

	return false
}

func (d *DAG) GetExecutionOrder() []string {
	slog.Debug("DAG: Computing execution order")

	inDegree := make(map[string]int)
	hasIncoming := make(map[string]bool)
	hasOutgoing := make(map[string]bool)

	for _, edge := range d.edges {
		inDegree[edge.To]++
		hasIncoming[edge.To] = true
		hasOutgoing[edge.From] = true
	}

	var queue []string
	for name := range d.nodes {
		if !hasIncoming[name] {
			queue = append(queue, name)
		}
	}

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, edge := range d.edges {
			if edge.From == node {
				inDegree[edge.To]--
				if inDegree[edge.To] == 0 {
					queue = append(queue, edge.To)
				}
			}
		}
	}

	slog.Debug("DAG: Execution order computed", "order", result)
	return result
}

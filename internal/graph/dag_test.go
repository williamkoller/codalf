package graph

import (
	"context"
	"testing"
)

type testNode struct {
	name   string
	output any
	err    error
}

func (n *testNode) Name() string { return n.name }
func (n *testNode) Execute(ctx context.Context, input any) (any, error) {
	return n.output, n.err
}

func TestDAG_AddNode(t *testing.T) {
	dag := NewDAG()
	node := &testNode{name: "test"}
	dag.AddNode(node)

	if len(dag.nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(dag.nodes))
	}
}

func TestDAG_Validate_NoEdges(t *testing.T) {
	dag := NewDAG()
	dag.AddNode(&testNode{name: "a"})

	if err := dag.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDAG_Validate_UnknownNode(t *testing.T) {
	dag := NewDAG()
	dag.AddNode(&testNode{name: "a"})
	dag.AddEdge("a", "b")

	err := dag.Validate()
	if err == nil {
		t.Error("expected error for unknown node")
	}
}

func TestDAG_GetExecutionOrder(t *testing.T) {
	dag := NewDAG()
	dag.AddNode(&testNode{name: "a"})
	dag.AddNode(&testNode{name: "b"})
	dag.AddNode(&testNode{name: "c"})
	dag.AddEdge("a", "b")
	dag.AddEdge("b", "c")

	order := dag.GetExecutionOrder()

	if len(order) != 3 {
		t.Errorf("expected 3 nodes in order, got %d", len(order))
	}
	if order[0] != "a" {
		t.Errorf("expected first node to be 'a', got '%s'", order[0])
	}
}

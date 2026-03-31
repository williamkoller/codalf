package graph

import (
	"context"
)

type Node interface {
	Name() string
	Execute(ctx context.Context, input any) (any, error)
}

package agents

import (
	"context"

	"github.com/williamkoller/codalf/internal/types"
)

type Agent interface {
	Name() string
	Review(ctx context.Context, diff *types.Diff) ([]types.Finding, error)
}

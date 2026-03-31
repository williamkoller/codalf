package output

import (
	"encoding/json"
	"io"
	"log/slog"

	"github.com/williamkoller/codalf/internal/types"
)

func WriteJSON(w io.Writer, result *types.ReviewResult) error {
	slog.Debug("Output: Writing JSON", "findings_count", len(result.Findings))
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		slog.Error("Output: Failed to write JSON", "error", err)
		return err
	}
	slog.Debug("Output: JSON written successfully")
	return nil
}

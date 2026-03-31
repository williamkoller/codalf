package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/williamkoller/codalf/internal/types"
)

type GeneralAgent struct {
	client       Provider
	skillContext string
}

type Provider interface {
	Name() string
	Generate(ctx context.Context, prompt string) (string, error)
}

func NewGeneralAgent(client Provider, skillContext string) *GeneralAgent {
	return &GeneralAgent{client: client, skillContext: skillContext}
}

func (a *GeneralAgent) Name() string {
	return "general"
}

func (a *GeneralAgent) Review(ctx context.Context, diff *types.Diff) ([]types.Finding, error) {
	slog.Info("GeneralAgent: Starting review", "files_count", len(diff.Files))
	prompt := buildGeneralPrompt(diff, a.skillContext)
	slog.Debug("GeneralAgent: Prompt built", "prompt_length", len(prompt))

	response, err := a.client.Generate(ctx, prompt)
	if err != nil {
		slog.Error("GeneralAgent: Failed to generate review", "error", err)
		return nil, err
	}

	slog.Info("GeneralAgent: Review completed", "response_length", len(response))
	return parseFindings(response, "general")
}

func buildGeneralPrompt(diff *types.Diff, skillContext string) string {
	var sb strings.Builder

	sb.WriteString(`You are a senior Go engineer performing a rigorous code review. Your job is to catch real bugs, design issues, and violations of Go best practices — not style nitpicks.

Analyze only the lines marked with "+" (new additions). Context lines (space-prefixed) are for reference only.

`)

	if skillContext != "" {
		sb.WriteString("## Project Review Guidelines\n\n")
		sb.WriteString(skillContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString(`## Severity Rules

Report findings using EXACTLY these severity values (lowercase):

**critical** — code that will cause bugs, crashes, data loss, or security issues:
  - builtin println() or print() used (must use fmt.Println / log package)
  - error return value silently ignored (no "if err != nil" check)
  - variable declared but never used
  - nil pointer dereference risk
  - goroutine leak (goroutine started but never stopped/signaled)
  - race condition (shared variable written without synchronization)

**warning** — code that is likely wrong or will cause subtle bugs:
  - assignment in if init-statement that shadows or modifies an outer variable (e.g. "if x = val; ..." where x was already declared)
  - function declared to return error but always returns nil
  - defer inside a loop (defers accumulate until function returns, not loop iteration)
  - context passed to goroutine but not used for cancellation
  - TODO / FIXME / HACK comments left in code
  - magic numbers or magic strings without named constants

**info** — code smell or convention violation, not a bug:
  - fmt.Println used in non-main packages (prefer structured logging)
  - exported function missing godoc comment
  - overly large function that should be decomposed

## Diff to Review

`)

	for _, file := range diff.Files {
		sb.WriteString(fmt.Sprintf("File: %s\n", file.Path))
		for _, h := range file.Hunks {
			sb.WriteString(fmt.Sprintf("Lines %d-%d:\n%s\n", h.StartLine, h.EndLine, h.Content))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`## Response Format

Respond ONLY with a valid JSON array. No markdown, no explanation, no preamble.
Each object must have:
  - "file": exact file path from the diff
  - "line": the line number of the offending "+" line
  - "severity": exactly "critical", "warning", or "info"
  - "message": one clear sentence describing the problem and why it matters
  - "suggestion": the exact corrected code or a precise fix instruction (be specific — show the fixed code when possible)

Examples:
[
  {"file":"pkg/store/cache.go","line":42,"severity":"critical","message":"error from redis.Get is silently ignored — cache misses will be masked as hits","suggestion":"if val, err := r.client.Get(ctx, key).Result(); err == nil { return val, nil } else if !errors.Is(err, redis.Nil) { return \"\", err }"},
  {"file":"cmd/server/main.go","line":17,"severity":"critical","message":"builtin println() used instead of structured logger","suggestion":"log.Info(\"server starting\", \"addr\", addr)"},
  {"file":"internal/handler.go","line":88,"severity":"warning","message":"defer db.Close() inside a for loop — connections will not be released until the function returns","suggestion":"move db.Close() outside the loop or use a helper function per iteration"}
]

If no issues are found: []
`)

	return sb.String()
}

func parseFindings(response, agentName string) ([]types.Finding, error) {
	response = strings.TrimSpace(response)

	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")

	if start == -1 || end == -1 {
		return []types.Finding{}, nil
	}

	jsonStr := response[start : end+1]
	var findings []types.Finding
	if err := json.Unmarshal([]byte(jsonStr), &findings); err != nil {
		return []types.Finding{}, nil
	}

	for i := range findings {
		findings[i].Agent = agentName
	}

	return findings, nil
}

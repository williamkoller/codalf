package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/williamkoller/codalf/internal/types"
)

type ReactAgent struct {
	client       Provider
	skillContext string
}

func NewReactAgent(client Provider, skillContext string) *ReactAgent {
	return &ReactAgent{client: client, skillContext: skillContext}
}

func (a *ReactAgent) Name() string {
	return "react"
}

func (a *ReactAgent) Review(ctx context.Context, diff *types.Diff) ([]types.Finding, error) {
	prompt := buildReactPrompt(diff, a.skillContext)

	response, err := a.client.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return parseFindings(response, "react")
}

func buildReactPrompt(diff *types.Diff, skillContext string) string {
	var sb strings.Builder

	sb.WriteString(`You are a senior React/TypeScript engineer performing a rigorous code review. Focus on real bugs and correctness issues — not style preferences.

Analyze only the lines marked with "+" (new additions). Context lines (space-prefixed) are for reference only.

`)

	if skillContext != "" {
		sb.WriteString("## Project Review Guidelines\n\n")
		sb.WriteString(skillContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString(`## Severity Rules

Report findings using EXACTLY these severity values (lowercase):

**critical** — will cause runtime errors, infinite loops, or incorrect behavior:
  - useEffect called without a dependency array — causes infinite re-render loop
  - Direct mutation of state object instead of using setState/useState setter
  - Missing "key" prop in list rendering (.map() returning JSX without stable key)
  - Hook called inside a condition, loop, or nested function (violates Rules of Hooks)
  - Unhandled promise in event handler or useEffect (missing .catch or try/catch)
  - Calling setState on an unmounted component

**warning** — likely wrong, causes subtle bugs or performance issues:
  - console.log / console.error / console.warn left in production code
  - TypeScript "any" type used — bypasses type safety
  - useEffect with async function directly as callback (returns Promise, React ignores it)
  - Component defined inside another component (re-created on every parent render)
  - Stale closure: variable from outer scope captured in useCallback/useMemo without deps
  - TODO / FIXME comments left in code

**info** — convention issue, not a bug:
  - Inline styles object created inside render (new reference every render, defeats memoization)
  - Exported component or function missing return type annotation
  - Magic string used where a typed constant or enum would be clearer

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
  - "suggestion": the exact corrected code or a precise fix instruction (show the fixed code when possible)

Examples:
[
  {"file":"src/hooks/useData.ts","line":12,"severity":"critical","message":"useEffect is missing a dependency array — will run after every render, causing an infinite loop","suggestion":"useEffect(() => { fetchData() }, [id]) — add the dependency array with all values used inside"},
  {"file":"src/components/List.tsx","line":34,"severity":"critical","message":"Missing 'key' prop on list item — React cannot efficiently reconcile the list","suggestion":"items.map((item) => <Item key={item.id} {...item} />)"},
  {"file":"src/api/client.ts","line":8,"severity":"warning","message":"TypeScript 'any' type weakens type safety and disables IDE assistance","suggestion":"Replace 'any' with the specific type: ResponseBody or use 'unknown' with a type guard"}
]

If no issues are found: []
`)

	return sb.String()
}

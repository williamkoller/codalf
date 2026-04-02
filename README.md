# Codalf

Local AI code review for Go and React/TypeScript projects, powered by [Ollama](https://ollama.com). No cloud, no telemetry — your code never leaves your machine.

---

## Requirements

- Go 1.21+
- [Ollama](https://ollama.com) running locally

---

## Installation

**macOS / Linux**
```bash
git clone https://github.com/williamkoller/codalf
cd codalf
go install ./cmd/codalf
```

**Windows (PowerShell)**
```powershell
git clone https://github.com/williamkoller/codalf
cd codalf
go install ./cmd/codalf
```

> Make sure `%GOPATH%\bin` (Windows) or `$GOPATH/bin` (macOS/Linux) is in your `PATH`.

---

## Getting Started

### 1. Initialize the vault

```bash
codalf init
```

This sets up a local vault at `~/.codalf/vault.json` that stores your model preference and ensures Ollama is configured to run offline only. The vault is protected with file permission `0600` and a SHA-256 checksum to detect tampering.

### 2. Run a review

```bash
# Review current branch against main (auto-detect)
codalf review

# Review a specific branch against main
codalf review my-branch

# Review against a custom base branch
codalf review my-branch develop
```

When run on the base branch (e.g. `main`) with no arguments, codalf performs a **full repository review** — scanning all tracked files instead of a diff.

---

## Output

### Branch review

```
  codalf  feature/my-branch → main  ollama · qwen3:8b  [local]

  ┌─ src/components/Form.tsx
  │ @@ -12 +12 @@
  │    12    useEffect(() => {
  │ + 13      fetchData()
  │         ✗  useEffect missing dependency array — will cause infinite re-render
  │            ↳ add a second argument: useEffect(() => { ... }, [])
  │   14    }, [])
  └────────────────────────────────────────────────────────────────────────

  ✗ FAIL  1 critical  3.4s
```

### Full repository review

```
  codalf  main  full review  ollama · qwen3:8b  [local]

  ┌─ src/app/api/users/route.ts
  │ @@ -1 +1 @@
  │    ...
  └────────────────────────────────────────────────────────────────────────

  ✓ PASS  47 files  18.3s
```

---

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `-f, --format` | `inline` | Output format: `inline` or `json` |
| `-m, --model` | vault default | Ollama model to use |
| `-b, --base` | `main` | Base branch for comparison |
| `-v, --verbose` | `false` | Show detailed logs |

---

## What it detects

### Go

| Severity | Rule |
|----------|------|
| `critical` | Use of builtin `println()` / `print()` |
| `critical` | Ignored errors |
| `critical` | Declared but unused variables |
| `warning` | Assignment in `if` init-statement modifying an outer variable |
| `warning` | Function returns `error` but never returns non-nil |
| `warning` | `TODO` / `FIXME` comments left in code |
| `info` | `fmt.Println` in non-main packages |

### React / TypeScript

| Severity | Rule |
|----------|------|
| `critical` | `useEffect` missing dependency array |
| `critical` | Direct state mutation instead of setter |
| `critical` | Missing `key` prop in list rendering |
| `critical` | Hook called conditionally or inside a loop |
| `critical` | Unhandled promise in event handler or `useEffect` |
| `warning` | `console.log` / `console.error` left in code |
| `warning` | `TODO` / `FIXME` comments left in code |
| `warning` | TypeScript `any` type used |
| `warning` | `useEffect` with async function as direct callback |
| `warning` | Component defined inside another component |
| `info` | Inline style object defined inside render |
| `info` | Missing return type on exported function/component |

The correct agent is selected automatically based on file extensions in the diff (`.go` → Go agent, `.ts` / `.tsx` / `.js` / `.jsx` / `.css` / `.scss` → React agent). Mixed projects run both agents and merge the findings.

---

## Recommended models

| Model | Size | Min RAM | Min VRAM (GPU) | Best for |
|-------|------|---------|----------------|----------|
| `qwen3:8b` | ~5 GB | 8 GB | 5 GB | Best balance — fast, accurate, **recommended default** |
| `qwen3:14b` | ~9 GB | 16 GB | 9 GB | Higher precision |
| `qwen2.5-coder:7b` | ~4 GB | 8 GB | 4 GB | Limited RAM / fallback |
| `qwen2.5-coder:14b` | ~8 GB | 12 GB | 8 GB | Balanced quality / speed |
| `deepseek-coder-v2` | ~8 GB | 12 GB | 8 GB | Accurate, great for TypeScript |
| `codellama:34b` | ~20 GB | 32 GB | 20 GB | Highest precision |

> **Recommended:** `qwen3:8b` offers the best speed/quality ratio for local reviews. Use `qwen3:14b` if you have 12 GB+ VRAM.
>
> Codalf checks your available RAM before running. If the system does not meet the minimum requirement for the selected model, you will see an error before any review starts.

```bash
ollama pull qwen3:8b
codalf review -m qwen3:8b
```

---

## Security

Codalf is designed to be fully offline:

- All analysis runs locally via Ollama
- Only `localhost` / `127.0.0.1` are accepted as hosts
- No data is sent to any external service
- The vault file is stored at `~/.codalf/vault.json` with restricted permissions (`0600`)
- Vault integrity is verified with a SHA-256 checksum on every run — any tampering is detected

---

## Ollama Setup

**macOS**
```bash
brew install ollama
ollama serve
ollama pull qwen3:8b
```

**Linux**
```bash
curl -fsSL https://ollama.com/install.sh | sh
ollama serve
ollama pull qwen3:8b
```

**Windows**

Download and install from [ollama.com/download](https://ollama.com/download), then in PowerShell:
```powershell
ollama serve
ollama pull qwen3:8b
```

---

## Architecture

The review pipeline runs as a DAG (Directed Acyclic Graph):

```
get_diff → run_agent → merge_results → score → output
```

The agent is selected based on the languages detected in the diff:
- **`GeneralAgent`** — Go files
- **`ReactAgent`** — TypeScript / JavaScript / CSS files

Both agents send their portion of the diff to Ollama and parse the structured JSON response into findings, which are then scored and rendered as a GitHub-style diff in the terminal.

---

## Development

### Make Commands

```bash
make build      # Build the binary
make test       # Run tests with race detector
make lint       # Run go vet and golangci-lint
make lint-fix   # Run golangci-lint with auto-fix
make vuln       # Check for vulnerabilities with govulncheck
make tools      # Install development tools
make clean      # Clean build artifacts
make run        # Build and run locally
make deps       # Download and tidy dependencies
```

### Git Hooks

This project uses [lefthook](https://github.com/evilmartians/lefthook) for git hooks.

**pre-commit:** `go vet`, `go test`, `golangci-lint`  
**pre-push:** `govulncheck`

Install hooks:
```bash
lefthook install
```

---

## License

MIT

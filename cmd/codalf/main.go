package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/williamkoller/codalf/internal/git"
	"github.com/williamkoller/codalf/internal/output"
	"github.com/williamkoller/codalf/internal/provider"
	"github.com/williamkoller/codalf/internal/review"
	"github.com/williamkoller/codalf/internal/scoring"
	"github.com/williamkoller/codalf/internal/skills"
	"github.com/williamkoller/codalf/internal/types"
	"github.com/williamkoller/codalf/internal/vault"
)

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorCyan   = "\033[36m"
	colorBlue   = "\033[34m"
	colorYellow = "\033[33m"
)

var (
	format     string
	model      string
	prov       string
	apiKey     string
	baseBranch string
	verbose    bool
)

func init() {
	flag.StringVar(&format, "format", "inline", "Output format: inline, json")
	flag.StringVar(&format, "f", "inline", "Output format: inline, json (shorthand)")
	flag.StringVar(&model, "model", "", "Model to use (overrides vault)")
	flag.StringVar(&model, "m", "", "Model to use (shorthand)")
	flag.StringVar(&prov, "provider", "", "Provider to use: ollama, openai, anthropic, copilot (overrides vault)")
	flag.StringVar(&prov, "p", "", "Provider to use (shorthand)")
	flag.StringVar(&apiKey, "api-key", "", "API key to use (overrides vault)")
	flag.StringVar(&apiKey, "k", "", "API key to use (shorthand)")
	flag.StringVar(&baseBranch, "base", "main", "Base branch for comparison")
	flag.StringVar(&baseBranch, "b", "main", "Base branch for comparison (shorthand)")
	flag.BoolVar(&verbose, "verbose", false, "Show detailed logs")
	flag.BoolVar(&verbose, "v", false, "Show detailed logs (shorthand)")
	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, `Codalf - AI-powered Code Review

Usage: codalf [options] review <branch> [base]

Providers:
  ollama    - Local Ollama (default)
  openai    - OpenAI GPT-4o
  anthropic - Anthropic Claude
  copilot   - GitHub Copilot

Options:
`)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Examples:
  codalf review feature-branch
  codalf review feature-branch main
  codalf review feature-branch -p openai -k $OPENAI_API_KEY
  codalf review feature-branch -p anthropic -k $ANTHROPIC_API_KEY
  codalf review feature-branch -p copilot -k $GITHUB_TOKEN
  codalf review feature-branch -f json

Report bugs at: https://github.com/williamkoller/codalf
`)
}

func main() {
	flag.Parse()

	level := slog.LevelError
	if verbose {
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	rootDir, _ := os.Getwd()
	skillsDir := filepath.Join(rootDir, ".agents", "skills")
	if _, err := os.Stat(skillsDir); err == nil {
		_, _ = skills.LoadSkills(skillsDir) // nolint:errcheck
	}

	switch os.Args[1] {
	case "init":
		if err := vault.RunInit(); err != nil {
			fmt.Fprintf(os.Stderr, "\n  \033[31mError:\033[0m %s\n\n", err)
			os.Exit(1)
		}
	case "review":
		if err := runReview(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "\n  \033[31mError:\033[0m %s\n\n", err)
			os.Exit(1)
		}
	default:
		flag.Usage()
		os.Exit(1)
	}
}

func runReview(args []string) error {
	cfg, err := vault.Load()
	if err != nil {
		return fmt.Errorf("%w\n\n  Run 'codalf init' to set up your local vault", err)
	}

	activeProvider := cfg.Provider
	if prov != "" {
		activeProvider = prov
	}

	activeModel := cfg.Model
	if model != "" {
		activeModel = model
	}

	apiKey := ""
	baseURL := ""
	ollamaHost := cfg.OllamaHost

	switch activeProvider {
	case "openai":
		if apiKey == "" {
			apiKey, err = vault.DecryptAPIKey(cfg.OpenAIKey, cfg.EncryptionKey)
			if err != nil {
				return fmt.Errorf("failed to decrypt OpenAI key: %w", err)
			}
		}
		baseURL = cfg.OpenAIBaseURL
	case "anthropic":
		if apiKey == "" {
			apiKey, err = vault.DecryptAPIKey(cfg.AnthropicKey, cfg.EncryptionKey)
			if err != nil {
				return fmt.Errorf("failed to decrypt Anthropic key: %w", err)
			}
		}
		baseURL = cfg.AnthropicURL
	case "copilot":
		if apiKey == "" {
			apiKey, err = vault.DecryptAPIKey(cfg.CopilotToken, cfg.EncryptionKey)
			if err != nil {
				return fmt.Errorf("failed to decrypt Copilot token: %w", err)
			}
		}
		baseURL = cfg.CopilotURL
	case "ollama":
		if ollamaHost == "" {
			ollamaHost = "http://localhost:11434"
		}
	}

	if apiKey == "" && activeProvider != "ollama" {
		return fmt.Errorf("API key required for %s. Use -k flag or run 'codalf init'", activeProvider)
	}

	provCfg := provider.Config{
		Provider:      activeProvider,
		Model:         activeModel,
		APIKey:        apiKey,
		BaseURL:       baseURL,
		OllamaHost:    ollamaHost,
		EncryptionKey: cfg.EncryptionKey,
	}

	client, err := provider.New(provCfg)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	ctx := context.Background()

	autoDetected := len(args) == 0
	var branch string
	if !autoDetected {
		branch = args[0]
	} else {
		branch, err = git.GetCurrentBranch(ctx)
		if err != nil {
			return fmt.Errorf("could not detect current branch: %w", err)
		}
	}

	base := baseBranch
	if len(args) > 1 {
		base = args[1]
	}

	fullReview := autoDetected && branch == base

	printHeader(branch, base, activeProvider, activeModel, fullReview)

	if activeProvider == "ollama" {
		if err := checkSystemRequirements(activeModel); err != nil {
			return err
		}
		if err := ensureModel(activeModel, ollamaHost); err != nil {
			return fmt.Errorf("model setup failed: %w", err)
		}
	}

	var diff *types.Diff
	if fullReview {
		diff, err = git.GetAllFiles(ctx)
	} else {
		diff, err = git.GetDiff(ctx, branch, base)
	}
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if len(diff.Files) == 0 {
		fmt.Fprintf(os.Stderr, "\r\033[K")
		fmt.Fprintf(os.Stdout, "\n  %sNo changes to review between '%s' and '%s'.%s\n\n", colorDim, branch, base, colorReset)
		return nil
	}

	done := make(chan struct{})
	go spinner(done, len(diff.Files))

	start := time.Now()
	pipeline := review.NewPipeline(client)
	result, err := pipeline.Execute(ctx, diff)
	elapsed := time.Since(start)
	close(done)
	time.Sleep(10 * time.Millisecond)

	if err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	score := scoring.Calculate(result.Findings)
	result.Score = score
	result.Metadata.Branch = branch
	result.Metadata.Base = base
	result.Metadata.FilesAnalyzed = len(diff.Files)
	result.Metadata.Duration = formatDuration(elapsed)
	result.Metadata.Provider = activeProvider
	result.Metadata.Model = activeModel

	switch format {
	case "json":
		return output.WriteJSON(os.Stdout, result)
	default:
		return output.WriteInline(os.Stdout, result, diff)
	}
}

func printHeader(branch, base, provStr, modelStr string, fullReview bool) {
	var target string
	if fullReview {
		target = fmt.Sprintf("%s%s%s  %sfull review%s", colorBold, branch, colorReset, colorYellow, colorReset)
	} else {
		target = fmt.Sprintf("%s%s%s  →  %s", colorBold, branch, colorReset, base)
	}

	provLabel := provStr
	if modelStr != "" {
		provLabel = provStr + " · " + modelStr
	}

	offline := ""
	if provStr == "ollama" {
		offline = colorDim + "[local]" + colorReset
	}

	fmt.Fprintf(os.Stderr, "\n  %s%scodalf%s  %s  %s%s%s",
		colorCyan, colorBold, colorReset,
		target,
		colorDim, provLabel, colorReset)
	if offline != "" {
		fmt.Fprintf(os.Stderr, "  %s", offline)
	}
	fmt.Fprintf(os.Stderr, "\n\n")
}

func spinner(done <-chan struct{}, files int) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	msg := fmt.Sprintf("Reviewing %d file", files)
	if files != 1 {
		msg += "s"
	}
	i := 0
	for {
		select {
		case <-done:
			fmt.Fprint(os.Stderr, "\r\033[K")
			return
		default:
			fmt.Fprintf(os.Stderr, "\r  %s%s%s  %s%s%s",
				colorCyan, frames[i%len(frames)], colorReset,
				colorDim, msg, colorReset)
			time.Sleep(80 * time.Millisecond)
			i++
		}
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

type modelReq struct {
	minRAMGB int
}

var modelRequirements = map[string]modelReq{
	"qwen3:8b":           {minRAMGB: 8},
	"qwen3:14b":          {minRAMGB: 16},
	"qwen2.5-coder:7b":  {minRAMGB: 8},
	"qwen2.5-coder:14b": {minRAMGB: 12},
	"deepseek-coder-v2": {minRAMGB: 12},
	"codellama:34b":      {minRAMGB: 32},
}

func totalRAMGB() int {
	var out []byte
	var err error
	switch runtime.GOOS {
	case "linux":
		data, e := os.ReadFile("/proc/meminfo")
		if e != nil {
			return 0
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				var kb int
				fmt.Sscanf(strings.TrimPrefix(line, "MemTotal:"), "%d", &kb)
				return kb / 1024 / 1024
			}
		}
	case "darwin":
		out, err = exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			return 0
		}
		var bytes int64
		fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &bytes)
		return int(bytes / 1024 / 1024 / 1024)
	case "windows":
		out, err = exec.Command("wmic", "ComputerSystem", "get", "TotalPhysicalMemory", "/value").Output()
		if err != nil {
			return 0
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "TotalPhysicalMemory=") {
				var bytes int64
				fmt.Sscanf(strings.TrimPrefix(line, "TotalPhysicalMemory="), "%d", &bytes)
				return int(bytes / 1024 / 1024 / 1024)
			}
		}
	}
	return 0
}

func checkSystemRequirements(modelName string) error {
	req, ok := modelRequirements[modelName]
	if !ok {
		return nil
	}
	ramGB := totalRAMGB()
	if ramGB == 0 {
		slog.Warn("Could not detect system RAM, skipping requirement check")
		return nil
	}
	if ramGB < req.minRAMGB {
		return fmt.Errorf(
			"model %s requires at least %d GB RAM, but only %d GB detected — use a smaller model (e.g. qwen3:8b)",
			modelName, req.minRAMGB, ramGB,
		)
	}
	return nil
}

func ensureModel(modelName, host string) error {
	checkCmd := exec.Command("ollama", "list")
	checkOut, err := checkCmd.Output()
	if err != nil {
		return nil
	}

	models := string(checkOut)
	if strings.Contains(models, modelName+":") || strings.Contains(models, modelName+" ") {
		return nil
	}

	fmt.Fprintf(os.Stderr, "  %sPulling model %s...%s\n", colorDim, modelName, colorReset)
	installCmd := exec.Command("ollama", "pull", modelName)
	installCmd.Stdout = os.Stderr
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install model %s: %w", modelName, err)
	}

	return nil
}

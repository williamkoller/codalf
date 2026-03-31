package vault

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	green  = "\033[32m"
	cyan   = "\033[36m"
	yellow = "\033[33m"
	red    = "\033[31m"
)

func RunInit() error {
	printBanner()

	if Exists() {
		cfg, err := Load()
		if err == nil {
			fmt.Printf("  %sWarning:%s Vault already initialized\n", yellow+bold, reset)
			fmt.Printf("  %sProvider:%s %s\n", dim, reset, cfg.Provider)
			fmt.Printf("  %sModel:%s   %s\n", dim, reset, cfg.Model)
			fmt.Printf("  %sCreated:%s %s\n\n", dim, reset, cfg.CreatedAt.Format("2006-01-02 15:04"))
			fmt.Printf("  Reinitialize? [y/N] ")

			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(answer)) != "y" {
				fmt.Printf("\n  %sVault unchanged.%s\n\n", dim, reset)
				return nil
			}
			fmt.Println()
		}
	}

	printPrivacyStatement()

	provider := promptProvider()

	var cfg *Config
	var err error

	switch provider {
	case "ollama":
		cfg, err = initOllama()
	case "openai":
		cfg, err = initOpenAI()
	case "anthropic":
		cfg, err = initAnthropic()
	case "copilot":
		cfg, err = initCopilot()
	default:
		cfg, err = initOllama()
	}

	if err != nil {
		return err
	}

	cfg.Provider = provider
	cfg.CreatedAt = time.Now()

	if err := Save(cfg); err != nil {
		return fmt.Errorf("could not save vault: %w", err)
	}

	p, _ := Path()
	printSuccess(p, cfg)

	return nil
}

func printBanner() {
	fmt.Printf("\n  %s%scodalf init%s\n", cyan, bold, reset)
	fmt.Printf("  %s%s%s\n\n", dim, strings.Repeat("-", 50), reset)
}

func printPrivacyStatement() {
	fmt.Printf("  %s%sSecurity & Privacy%s\n\n", bold, cyan, reset)
	fmt.Printf("  %s[+]%s Vault integrity verified via SHA-256 checksum\n", green, reset)
	fmt.Printf("  %s[+]%s API keys can be encrypted with a password\n", green, reset)
	fmt.Printf("  %s[+]%s Vault stored at %s~/.codalf/vault.json%s (mode 0600)\n\n", green, reset, dim, reset)
}

func promptProvider() string {
	fmt.Printf("  %s%sChoose a provider:%s\n\n", bold, cyan, reset)
	providers := []struct {
		id   string
		name string
		desc string
	}{
		{"ollama", "Ollama (local)", "Runs entirely on your machine"},
		{"openai", "OpenAI", "GPT-4o, GPT-4o-mini"},
		{"anthropic", "Anthropic", "Claude Sonnet 4, Claude Opus 4"},
		{"copilot", "GitHub Copilot", "Uses GitHub's Copilot API"},
	}

	for i, p := range providers {
		marker := dim
		if i == 0 {
			marker = cyan
		}
		fmt.Printf("  %s[%d]%s %s - %s%s%s\n", marker, i+1, reset, bold+p.name, dim, p.desc, reset)
	}
	fmt.Printf("\n  Select [1-%d]: ", len(providers))

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" || input == "1" {
		return "ollama"
	}

	for i, p := range providers {
		if input == fmt.Sprintf("%d", i+1) {
			return p.id
		}
	}

	return "ollama"
}

func initOllama() (*Config, error) {
	host := "http://localhost:11434"

	fmt.Printf("\n  %sVerifying Ollama at %s...%s", dim, host, reset)

	if err := checkOllama(host); err != nil {
		fmt.Printf("\r  %sError:%s Ollama not reachable at %s\n", red+bold, reset, host)
		fmt.Printf("  %sMake sure Ollama is running: ollama serve%s\n\n", dim, reset)
		return nil, fmt.Errorf("ollama not reachable: %w", err)
	}
	fmt.Printf("\r  %s[ok]%s Ollama reachable at %s\n", green+bold, reset, host)

	models := []string{
		"qwen2.5-coder:7b",
		"qwen2.5-coder:14b",
		"deepseek-coder:6.7b",
		"codellama:7b",
		"llama3.3:70b",
		"phi4",
		"mistral:7b",
		"custom...",
	}

	fmt.Printf("\n  %s%sChoose a model:%s\n\n", bold, cyan, reset)
	for i, m := range models {
		if i == 0 {
			fmt.Printf("  %s[%d]%s %s %s(default)%s\n", cyan, i+1, reset, m, dim, reset)
		} else if m == "custom..." {
			fmt.Printf("  %s[%d]%s %s%s%s\n", dim, i+1, reset, bold, m, reset)
		} else {
			fmt.Printf("  %s[%d]%s %s\n", dim, i+1, reset, m)
		}
	}
	fmt.Printf("\n  Select [1-%d] or type a model name (Enter for default): ", len(models))

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	model := models[0]
	if input != "" && input != "1" {
		found := false
		for i, m := range models {
			if input == fmt.Sprintf("%d", i+1) {
				model = m
				found = true
				break
			}
		}
		if !found {
			model = input
		}
	}

	if model == "custom..." {
		fmt.Printf("\n  Enter custom model name (e.g., qwen2.5-coder:7b): ")
		input, _ := reader.ReadString('\n')
		model = strings.TrimSpace(input)
		if model == "" {
			return nil, fmt.Errorf("model name is required")
		}
	}

	fmt.Printf("\n  %sOllama Host%s (default: http://localhost:11434): ", bold, reset)
	hostInput, _ := reader.ReadString('\n')
	hostInput = strings.TrimSpace(hostInput)
	if hostInput != "" {
		host = hostInput
	}

	return &Config{
		Provider:   "ollama",
		Model:      model,
		OllamaHost: host,
		Offline:    true,
		CreatedAt:  time.Now(),
	}, nil
}

func initOpenAI() (*Config, error) {
	fmt.Printf("\n  %s%sOpenAI Configuration%s\n\n", bold, cyan, reset)

	fmt.Printf("  Enter your OpenAI API key (sk-...): ")
	reader := bufio.NewReader(os.Stdin)
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	fmt.Printf("\n  Model (Enter for default gpt-4o): ")
	modelInput, _ := reader.ReadString('\n')
	model := strings.TrimSpace(modelInput)
	if model == "" {
		model = "gpt-4o"
	}

	fmt.Printf("\n  Encryption password (optional, Enter to skip): ")
	passInput, _ := reader.ReadString('\n')
	password := strings.TrimSpace(passInput)

	encryptedKey, err := EncryptAPIKey(apiKey, password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	return &Config{
		Provider:      "openai",
		Model:         model,
		OpenAIKey:     encryptedKey,
		EncryptionKey: password,
		Offline:       false,
	}, nil
}

func initAnthropic() (*Config, error) {
	fmt.Printf("\n  %s%sAnthropic Configuration%s\n\n", bold, cyan, reset)

	fmt.Printf("  Enter your Anthropic API key (sk-ant-...): ")
	reader := bufio.NewReader(os.Stdin)
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	fmt.Printf("\n  Model (Enter for default claude-sonnet-4-20250514): ")
	modelInput, _ := reader.ReadString('\n')
	model := strings.TrimSpace(modelInput)
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	fmt.Printf("\n  Encryption password (optional, Enter to skip): ")
	passInput, _ := reader.ReadString('\n')
	password := strings.TrimSpace(passInput)

	encryptedKey, err := EncryptAPIKey(apiKey, password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}

	return &Config{
		Provider:      "anthropic",
		Model:         model,
		AnthropicKey:  encryptedKey,
		EncryptionKey: password,
		Offline:       false,
	}, nil
}

func initCopilot() (*Config, error) {
	fmt.Printf("\n  %s%sGitHub Copilot Configuration%s\n\n", bold, cyan, reset)
	fmt.Printf("  %sNote:%s Requires GitHub Copilot subscription\n", dim, reset)
	fmt.Printf("  %sToken:%s Use gh auth token or create at github.com/settings/tokens\n\n", dim, reset)

	fmt.Printf("  Enter your GitHub token: ")
	reader := bufio.NewReader(os.Stdin)
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	fmt.Printf("\n  Model (Enter for default gpt-4o): ")
	modelInput, _ := reader.ReadString('\n')
	model := strings.TrimSpace(modelInput)
	if model == "" {
		model = "gpt-4o"
	}

	fmt.Printf("\n  Encryption password (optional, Enter to skip): ")
	passInput, _ := reader.ReadString('\n')
	password := strings.TrimSpace(passInput)

	encryptedKey, err := EncryptAPIKey(apiKey, password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	return &Config{
		Provider:      "copilot",
		Model:         model,
		CopilotToken:  encryptedKey,
		EncryptionKey: password,
		Offline:       false,
	}, nil
}

func checkOllama(host string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(host)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func printSuccess(path string, cfg *Config) {
	fmt.Printf("\n  %s%s%s\n\n", dim, strings.Repeat("-", 50), reset)
	fmt.Printf("  %s%s[vault sealed]%s\n\n", green, bold, reset)
	fmt.Printf("  %sProvider:%s %s\n", dim, reset, cfg.Provider)
	fmt.Printf("  %sModel:%s   %s\n", dim, reset, cfg.Model)
	if cfg.Provider == "ollama" {
		fmt.Printf("  %sHost:%s    %s\n", dim, reset, cfg.OllamaHost)
		fmt.Printf("  %sOffline:%s true\n", dim, reset)
	} else {
		fmt.Printf("  %sOffline:%s false\n", dim, reset)
	}
	fmt.Printf("  %sPath:%s    %s\n", dim, reset, path)
	fmt.Printf("\n  %sRun your first review:%s\n", dim, reset)
	fmt.Printf("  %s%scodalf review <branch>%s\n\n", cyan, bold, reset)
}

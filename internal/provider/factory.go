package provider

import (
	"fmt"

	"github.com/williamkoller/codalf/internal/agents"
)

type ProviderFactory func(cfg Config) (Provider, error)

var providers = make(map[string]ProviderFactory)

func Register(name string, factory ProviderFactory) {
	providers[name] = factory
}

func New(cfg Config) (Provider, error) {
	factory, ok := providers[cfg.Provider]
	if !ok {
		return nil, &UnsupportedProviderError{Provider: cfg.Provider}
	}
	return factory(cfg)
}

func SupportedProviders() []string {
	var names []string
	for name := range providers {
		names = append(names, name)
	}
	return names
}

func IsSupported(provider string) bool {
	_, ok := providers[provider]
	return ok
}

type UnsupportedProviderError struct {
	Provider string
}

func (e *UnsupportedProviderError) Error() string {
	return "unsupported provider: " + e.Provider
}

func init() {
	Register("ollama", func(cfg Config) (Provider, error) {
		return agents.NewOllamaClient(cfg.OllamaHost, cfg.Model), nil
	})
	Register("openai", func(cfg Config) (Provider, error) {
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("openai requires API key")
		}
		model := cfg.Model
		if model == "" {
			model = "gpt-4o"
		}
		return agents.NewOpenAIClient(cfg.APIKey, model, cfg.BaseURL), nil
	})
	Register("anthropic", func(cfg Config) (Provider, error) {
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("anthropic requires API key")
		}
		model := cfg.Model
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
		return agents.NewAnthropicClient(cfg.APIKey, model, cfg.BaseURL), nil
	})
	Register("copilot", func(cfg Config) (Provider, error) {
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("copilot requires GitHub token")
		}
		model := cfg.Model
		if model == "" {
			model = "gpt-4o"
		}
		return agents.NewCopilotClient(cfg.APIKey, model, cfg.BaseURL), nil
	})
}

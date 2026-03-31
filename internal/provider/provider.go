package provider

import "context"

type Provider interface {
	Name() string
	Generate(ctx context.Context, prompt string) (string, error)
}

type Config struct {
	Provider      string
	Model         string
	APIKey        string
	BaseURL       string
	OllamaHost    string
	EncryptionKey string
}

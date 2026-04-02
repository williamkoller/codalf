package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type OllamaClient struct {
	host   string
	model  string
	client *http.Client
}

type OllamaOptions struct {
	Temperature float64 `json:"temperature"`
	Seed        int     `json:"seed"`
}

type OllamaRequest struct {
	Model   string        `json:"model"`
	Prompt  string        `json:"prompt"`
	Stream  bool          `json:"stream"`
	Options OllamaOptions `json:"options"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func NewOllamaClient(host, model string) *OllamaClient {
	if host == "" {
		host = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen2.5-coder:7b"
	}
	return &OllamaClient{
		host:   host,
		model:  model,
		client: &http.Client{Timeout: 3 * time.Minute},
	}
}

func (c *OllamaClient) Name() string {
	return "ollama"
}

func (c *OllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	slog.Info("Sending request to Ollama", "model", c.model, "host", c.host, "prompt_length", len(prompt))

	reqBody := OllamaRequest{
		Model:   c.model,
		Prompt:  prompt,
		Stream:  false,
		Options: OllamaOptions{Temperature: 0, Seed: 42},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	slog.Info("Ollama request completed", "response_length", len(response.Response))
	return response.Response, nil
}

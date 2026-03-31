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

type AnthropicClient struct {
	model   string
	apiKey  string
	baseURL string
	client  *http.Client
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
}

type AnthropicContent struct {
	Text string `json:"text"`
}

type AnthropicResponse struct {
	Content []AnthropicContent `json:"content"`
}

func NewAnthropicClient(apiKey, model, baseURL string) *AnthropicClient {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &AnthropicClient{
		model:   model,
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

func (c *AnthropicClient) Name() string {
	return "anthropic"
}

func (c *AnthropicClient) Generate(ctx context.Context, prompt string) (string, error) {
	slog.Info("Sending request to Anthropic", "model", c.model, "prompt_length", len(prompt))

	reqBody := AnthropicRequest{
		Model: c.model,
		Messages: []AnthropicMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   4096,
		Temperature: 0,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic returned status %d: %s", resp.StatusCode, string(body))
	}

	var response AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	slog.Info("Anthropic request completed", "response_length", len(response.Content[0].Text))
	return response.Content[0].Text, nil
}

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

type OpenAIClient struct {
	model   string
	apiKey  string
	baseURL string
	client  *http.Client
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	Seed        int             `json:"seed"`
}

type OpenAIChoice struct {
	Message OpenAIMessage `json:"message"`
}

type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
}

func NewOpenAIClient(apiKey, model, baseURL string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIClient{
		model:   model,
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

func (c *OpenAIClient) Name() string {
	return "openai"
}

func (c *OpenAIClient) Generate(ctx context.Context, prompt string) (string, error) {
	slog.Info("Sending request to OpenAI", "model", c.model, "prompt_length", len(prompt))

	reqBody := OpenAIRequest{
		Model: c.model,
		Messages: []OpenAIMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0,
		Seed:        42,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	slog.Info("OpenAI request completed", "response_length", len(response.Choices[0].Message.Content))
	return response.Choices[0].Message.Content, nil
}

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

type CopilotClient struct {
	model   string
	apiKey  string
	baseURL string
	client  *http.Client
}

type CopilotMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CopilotRequest struct {
	Model    string           `json:"model"`
	Messages []CopilotMessage `json:"messages"`
}

type CopilotChoice struct {
	Message CopilotMessage `json:"message"`
}

type CopilotResponse struct {
	Choices []CopilotChoice `json:"choices"`
}

func NewCopilotClient(apiKey, model, baseURL string) *CopilotClient {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	if model == "" {
		model = "gpt-4o"
	}
	return &CopilotClient{
		model:   model,
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

func (c *CopilotClient) Name() string {
	return "copilot"
}

func (c *CopilotClient) Generate(ctx context.Context, prompt string) (string, error) {
	slog.Info("Sending request to GitHub Copilot", "model", c.model, "prompt_length", len(prompt))

	reqBody := CopilotRequest{
		Model: c.model,
		Messages: []CopilotMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("copilot returned status %d: %s", resp.StatusCode, string(body))
	}

	var response CopilotResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	slog.Info("Copilot request completed", "response_length", len(response.Choices[0].Message.Content))
	return response.Choices[0].Message.Content, nil
}

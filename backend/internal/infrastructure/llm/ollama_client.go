package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}

type ollamaTagModelDetails struct {
	ParameterSize string `json:"parameter_size"`
}

type ollamaTagModel struct {
	Name    string                `json:"name"`
	Model   string                `json:"model"`
	Size    int64                 `json:"size"`
	Details ollamaTagModelDetails `json:"details"`
}

type ollamaTagsResponse struct {
	Models []ollamaTagModel `json:"models"`
}

func NewOllamaClient(baseURL string, httpClient *http.Client) *OllamaClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &OllamaClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *OllamaClient) Generate(ctx context.Context, model string, prompt string) (string, error) {
	if c == nil || c.baseURL == "" {
		return "", fmt.Errorf("ollama base URL is required")
	}
	if strings.TrimSpace(model) == "" {
		return "", fmt.Errorf("ollama model is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("ollama prompt is required")
	}

	reqBody, err := json.Marshal(ollamaGenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("marshal ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("build ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute ollama request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama request failed with status %d", resp.StatusCode)
	}

	var decoded ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode ollama response: %w", err)
	}
	return strings.TrimSpace(decoded.Response), nil
}

func (c *OllamaClient) ListModels(ctx context.Context) ([]string, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("ollama base URL is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("build ollama tags request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute ollama tags request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama tags request failed with status %d", resp.StatusCode)
	}

	var decoded ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode ollama tags response: %w", err)
	}

	models := make([]string, 0, len(decoded.Models))
	for _, model := range decoded.Models {
		name := strings.TrimSpace(model.Name)
		if name == "" {
			name = strings.TrimSpace(model.Model)
		}
		if name != "" {
			models = append(models, name)
		}
	}
	return models, nil
}

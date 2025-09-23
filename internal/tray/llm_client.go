package tray

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// LLMClient interface for AI model interactions
type LLMClient interface {
	Analyze(prompt string) (string, error)
	GenerateConfig(serverName, documentation, currentConfig string) (string, error)
}

// OpenAIClient implements LLMClient using OpenAI API
type OpenAIClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient() *OpenAIClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		// Try common environment variable names
		apiKey = os.Getenv("OPENAI_KEY")
	}
	
	return &OpenAIClient{
		apiKey: apiKey,
		model:  "gpt-4o-mini", // Cost-effective model
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Analyze performs general analysis using the LLM
func (c *OpenAIClient) Analyze(prompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not found. Set OPENAI_API_KEY environment variable")
	}

	request := OpenAIRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert MCP (Model Context Protocol) server diagnostic agent. Analyze issues and provide clear, actionable solutions.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 1000,
	}

	return c.makeRequest(request)
}

// GenerateConfig generates or fixes configuration based on documentation
func (c *OpenAIClient) GenerateConfig(serverName, documentation, currentConfig string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not found. Set OPENAI_API_KEY environment variable")
	}

	prompt := fmt.Sprintf(`Analyze this MCP server configuration and documentation, then provide a corrected JSON configuration:

SERVER: %s

DOCUMENTATION:
%s

CURRENT CONFIG:
%s

Please provide:
1. Analysis of configuration issues
2. Corrected JSON configuration
3. Explanation of changes made

Format your response as:
## Analysis
[Your analysis here]

## Corrected Configuration
` + "```json" + `
[Corrected JSON config here]
` + "```" + `

## Changes Made
[Explanation of changes]`, serverName, documentation, currentConfig)

	request := OpenAIRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert MCP server configuration specialist. Generate valid JSON configurations based on documentation.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 2000,
	}

	return c.makeRequest(request)
}

// makeRequest makes HTTP request to OpenAI API
func (c *OpenAIClient) makeRequest(request OpenAIRequest) (string, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var response OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return response.Choices[0].Message.Content, nil
}

package tray

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mcpproxy-go/internal/config"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LLMClient interface for AI model interactions
type LLMClient interface {
	Analyze(prompt string) (string, error)
	AnalyzeWithTools(prompt string, tools []Tool, toolExecutor ToolExecutor) (string, []ToolCallRecord, error)
	GenerateConfig(serverName, documentation, currentConfig string) (string, error)
}

// ToolCallRecord tracks a single tool call execution
type ToolCallRecord struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    string                 `json:"result"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
}

// ToolExecutor is a function that executes a tool call and returns the result
type ToolExecutor func(toolName string, arguments map[string]interface{}) (string, error)

// OpenAIClient implements LLMClient using OpenAI API
type OpenAIClient struct {
	apiKey      string
	model       string
	temperature float64
	maxTokens   int
	httpClient  *http.Client
}

// AnthropicClient implements LLMClient using Anthropic API
type AnthropicClient struct {
	apiKey      string
	model       string
	temperature float64
	maxTokens   int
	httpClient  *http.Client
}

// OllamaClient implements LLMClient using Ollama API
type OllamaClient struct {
	baseURL     string
	model       string
	temperature float64
	maxTokens   int
	httpClient  *http.Client
}

type OpenAIRequest struct {
	Model      string     `json:"model"`
	Messages   []Message  `json:"messages"`
	MaxTokens  int        `json:"max_tokens,omitempty"`
	Tools      []Tool     `json:"tools,omitempty"`
	ToolChoice string     `json:"tool_choice,omitempty"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// NewLLMClientFromConfig creates an LLM client based on configuration
func NewLLMClientFromConfig(cfg *config.LLMConfig) LLMClient {
	if cfg == nil {
		// Fallback to environment-based OpenAI client
		return NewOpenAIClient()
	}

	switch strings.ToLower(cfg.Provider) {
	case "anthropic":
		return NewAnthropicClient(cfg)
	case "ollama":
		return NewOllamaClient(cfg)
	case "openai":
		fallthrough
	default:
		return NewOpenAIClientFromConfig(cfg)
	}
}

// NewOpenAIClient creates a new OpenAI client (fallback for environment variables)
func NewOpenAIClient() *OpenAIClient {
	// Try to load from .env file first
	apiKey := ""

	// Try loading from data directory first
	homeDir, err := os.UserHomeDir()
	if err == nil {
		dataDir := filepath.Join(homeDir, ".mcpproxy")
		envPath := filepath.Join(dataDir, ".env")
		envVars, err := config.LoadDotEnv(envPath)
		if err == nil && len(envVars) > 0 {
			if key, ok := envVars["OPENAI_API_KEY"]; ok && key != "" {
				apiKey = key
			} else if key, ok := envVars["OPENAI_KEY"]; ok && key != "" {
				apiKey = key
			}
		}
	}

	// Try current directory .env if not found
	if apiKey == "" {
		envVars, err := config.LoadDotEnv(".env")
		if err == nil && len(envVars) > 0 {
			if key, ok := envVars["OPENAI_API_KEY"]; ok && key != "" {
				apiKey = key
			} else if key, ok := envVars["OPENAI_KEY"]; ok && key != "" {
				apiKey = key
			}
		}
	}

	// Fallback to environment variables
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_KEY")
		}
	}

	// Debug logging to help diagnose environment variable issues
	// This is especially important for GUI applications on macOS which may not
	// inherit shell environment variables
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] OPENAI_API_KEY not found in environment or .env files\n")
		fmt.Fprintf(os.Stderr, "[DEBUG] Available environment variables:\n")
		for _, env := range os.Environ() {
			// Only log variable names, not values (for security)
			if len(env) > 0 {
				parts := []byte(env)
				for i, ch := range parts {
					if ch == '=' {
						fmt.Fprintf(os.Stderr, "  - %s\n", string(parts[:i]))
						break
					}
				}
			}
		}
	} else {
		// Log that key was found (but not the key itself)
		keyPreview := apiKey
		if len(keyPreview) > 7 {
			keyPreview = apiKey[:7] + "..." + apiKey[len(apiKey)-4:]
		}
		fmt.Fprintf(os.Stderr, "[DEBUG] OPENAI_API_KEY found: %s\n", keyPreview)
	}

	return &OpenAIClient{
		apiKey: apiKey,
		model:  "gpt-4o-mini", // Cost-effective model
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewOpenAIClientFromConfig creates OpenAI client from config
func NewOpenAIClientFromConfig(cfg *config.LLMConfig) *OpenAIClient {
	apiKey := cfg.OpenAIKey
	if apiKey == "" {
		// Fallback to environment variables
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_KEY")
		}
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] Creating OpenAI client with model: %s\n", model)
	if apiKey != "" {
		keyPreview := apiKey
		if len(keyPreview) > 7 {
			keyPreview = apiKey[:7] + "..." + apiKey[len(apiKey)-4:]
		}
		fmt.Fprintf(os.Stderr, "[DEBUG] OpenAI API key found: %s\n", keyPreview)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG] OpenAI API key NOT found - calls will fail\n")
	}

	return &OpenAIClient{
		apiKey:      apiKey,
		model:       model,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
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

// AnalyzeWithTools performs analysis with tool calling support
func (c *OpenAIClient) AnalyzeWithTools(prompt string, tools []Tool, toolExecutor ToolExecutor) (string, []ToolCallRecord, error) {
	if c.apiKey == "" {
		return "", nil, fmt.Errorf("OpenAI API key not found. Set OPENAI_API_KEY environment variable")
	}

	messages := []Message{
		{
			Role:    "system",
			Content: "You are an expert MCP (Model Context Protocol) server diagnostic agent. You have access to tools to read configuration files, logs, and GitHub documentation. Use these tools when needed to provide accurate assistance.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Track all tool calls made during this analysis
	var toolCallRecords []ToolCallRecord

	// Maximum iterations to prevent infinite loops
	maxIterations := 5

	for i := 0; i < maxIterations; i++ {
		// Use configured max_tokens or default to 16000 for diagnostic agent
		maxTokens := c.maxTokens
		if maxTokens == 0 {
			maxTokens = 16000 // Higher default for diagnostic tasks with tool calls
		}

		request := OpenAIRequest{
			Model:     c.model,
			Messages:  messages,
			Tools:     tools,
			MaxTokens: maxTokens,
		}

		response, err := c.makeRequestRaw(request)
		if err != nil {
			return "", toolCallRecords, err
		}

		if len(response.Choices) == 0 {
			return "", toolCallRecords, fmt.Errorf("no response from OpenAI")
		}

		choice := response.Choices[0]

		// Check finish reason
		if choice.FinishReason == "stop" {
			// Normal completion, return the content
			return choice.Message.Content, toolCallRecords, nil
		}

		if choice.FinishReason == "tool_calls" {
			// Execute tool calls
			messages = append(messages, choice.Message)

			for _, toolCall := range choice.Message.ToolCalls {
				startTime := time.Now()

				// Parse tool arguments
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					return "", toolCallRecords, fmt.Errorf("failed to parse tool arguments: %w", err)
				}

				// Execute the tool
				result, err := toolExecutor(toolCall.Function.Name, args)
				errorMsg := ""
				if err != nil {
					errorMsg = err.Error()
					result = fmt.Sprintf("Error executing tool: %v", err)
				}

				// Record the tool call
				toolCallRecords = append(toolCallRecords, ToolCallRecord{
					ToolName:  toolCall.Function.Name,
					Arguments: args,
					Result:    result,
					Error:     errorMsg,
					Timestamp: startTime,
					Duration:  time.Since(startTime),
				})

				// CRITICAL FIX: Truncate large tool outputs to prevent context overflow
				// Maximum ~5000 tokens per tool result (roughly 20,000 characters)
				truncatedResult := truncateToolOutput(result, 20000)

				// Add tool result to messages
				messages = append(messages, Message{
					Role:       "tool",
					Content:    truncatedResult,
					ToolCallID: toolCall.ID,
				})
			}

			// Continue to next iteration with tool results
			continue
		}

		// Handle finish_reason: "length" - response was truncated due to max_tokens
		if choice.FinishReason == "length" {
			if choice.Message.Content != "" {
				// Log warning but return the truncated content
				fmt.Printf("⚠️  WARNING: AI response was truncated (finish_reason: length). Consider increasing max_tokens in config.\n")
				return choice.Message.Content + "\n\n[Note: Response was truncated due to token limit. Consider increasing max_tokens for complete responses.]", toolCallRecords, nil
			}
			return "", toolCallRecords, fmt.Errorf("response truncated with no content (finish_reason: length)")
		}

		// If we get here, it's an unexpected finish reason
		if choice.Message.Content != "" {
			return choice.Message.Content, toolCallRecords, nil
		}

		return "", toolCallRecords, fmt.Errorf("unexpected finish reason: %s", choice.FinishReason)
	}

	return "", toolCallRecords, fmt.Errorf("maximum tool call iterations reached")
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

// makeRequest makes HTTP request to OpenAI API and returns the content
func (c *OpenAIClient) makeRequest(request OpenAIRequest) (string, error) {
	response, err := c.makeRequestRaw(request)
	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return response.Choices[0].Message.Content, nil
}

// makeRequestRaw makes HTTP request to OpenAI API and returns the raw response
func (c *OpenAIClient) makeRequestRaw(request OpenAIRequest) (*OpenAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	return &response, nil
}

// ============================================================================
// Anthropic Client Implementation
// ============================================================================

// AnthropicRequest represents Anthropic API request
type AnthropicRequest struct {
	Model       string              `json:"model"`
	Messages    []AnthropicMessage  `json:"messages"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature,omitempty"`
	Tools       []Tool              `json:"tools,omitempty"`
}

// AnthropicMessage represents a message in Anthropic format
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents Anthropic API response
type AnthropicResponse struct {
	ID      string              `json:"id"`
	Type    string              `json:"type"`
	Role    string              `json:"role"`
	Content []AnthropicContent  `json:"content"`
	Model   string              `json:"model"`
	StopReason string           `json:"stop_reason"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *APIError `json:"error,omitempty"`
}

// AnthropicContent represents content in Anthropic response
type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// For tool use
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// NewAnthropicClient creates a new Anthropic client from config
func NewAnthropicClient(cfg *config.LLMConfig) *AnthropicClient {
	apiKey := cfg.AnthropicKey
	if apiKey == "" {
		// Fallback to environment variable
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	model := cfg.Model
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] Creating Anthropic client with model: %s\n", model)
	if apiKey != "" {
		keyPreview := apiKey
		if len(keyPreview) > 7 {
			keyPreview = apiKey[:7] + "..." + apiKey[len(apiKey)-4:]
		}
		fmt.Fprintf(os.Stderr, "[DEBUG] Anthropic API key found: %s\n", keyPreview)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG] Anthropic API key NOT found - calls will fail\n")
	}

	return &AnthropicClient{
		apiKey:      apiKey,
		model:       model,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Analyze performs general analysis using Anthropic
func (c *AnthropicClient) Analyze(prompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("Anthropic API key not found. Set ANTHROPIC_API_KEY in config or environment")
	}

	request := AnthropicRequest{
		Model: c.model,
		Messages: []AnthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
	}

	return c.makeRequest(request)
}

// AnalyzeWithTools performs analysis with tool calling
func (c *AnthropicClient) AnalyzeWithTools(prompt string, tools []Tool, toolExecutor ToolExecutor) (string, []ToolCallRecord, error) {
	if c.apiKey == "" {
		return "", nil, fmt.Errorf("Anthropic API key not found. Set ANTHROPIC_API_KEY in config or environment")
	}

	messages := []AnthropicMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Track all tool calls made during this analysis
	var toolCallRecords []ToolCallRecord

	// Maximum iterations
	maxIterations := 5

	for i := 0; i < maxIterations; i++ {
		request := AnthropicRequest{
			Model:       c.model,
			Messages:    messages,
			MaxTokens:   c.maxTokens,
			Temperature: c.temperature,
			Tools:       tools,
		}

		response, err := c.makeRequestRaw(request)
		if err != nil {
			return "", toolCallRecords, err
		}

		// Check stop reason
		if response.StopReason == "end_turn" {
			// Normal completion
			for _, content := range response.Content {
				if content.Type == "text" {
					return content.Text, toolCallRecords, nil
				}
			}
			return "", toolCallRecords, fmt.Errorf("no text content in response")
		}

		if response.StopReason == "tool_use" {
			// Execute tool calls
			for _, content := range response.Content {
				if content.Type == "tool_use" {
					startTime := time.Now()

					result, err := toolExecutor(content.Name, content.Input)
					errorMsg := ""
					if err != nil {
						errorMsg = err.Error()
						result = fmt.Sprintf("Error executing tool: %v", err)
					}

					// Record the tool call
					toolCallRecords = append(toolCallRecords, ToolCallRecord{
						ToolName:  content.Name,
						Arguments: content.Input,
						Result:    result,
						Error:     errorMsg,
						Timestamp: startTime,
						Duration:  time.Since(startTime),
					})

					// CRITICAL FIX: Truncate large tool outputs to prevent context overflow
					truncatedResult := truncateToolOutput(result, 20000)

					// Add tool result to messages
					messages = append(messages, AnthropicMessage{
						Role:    "assistant",
						Content: fmt.Sprintf("Tool %s executed", content.Name),
					}, AnthropicMessage{
						Role:    "user",
						Content: truncatedResult,
					})
				}
			}
			continue
		}

		// Return any text content
		for _, content := range response.Content {
			if content.Type == "text" && content.Text != "" {
				return content.Text, toolCallRecords, nil
			}
		}

		return "", toolCallRecords, fmt.Errorf("unexpected stop reason: %s", response.StopReason)
	}

	return "", toolCallRecords, fmt.Errorf("maximum tool call iterations reached")
}

// GenerateConfig generates config using Anthropic
func (c *AnthropicClient) GenerateConfig(serverName, documentation, currentConfig string) (string, error) {
	prompt := fmt.Sprintf(`Analyze this MCP server configuration and documentation, then provide a corrected JSON configuration:

SERVER: %s

DOCUMENTATION:
%s

CURRENT CONFIG:
%s

Please provide:
1. Analysis of configuration issues
2. Corrected JSON configuration
3. Explanation of changes made`, serverName, documentation, currentConfig)

	return c.Analyze(prompt)
}

// makeRequest makes HTTP request to Anthropic API
func (c *AnthropicClient) makeRequest(request AnthropicRequest) (string, error) {
	response, err := c.makeRequestRaw(request)
	if err != nil {
		return "", err
	}

	for _, content := range response.Content {
		if content.Type == "text" {
			return content.Text, nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}

// makeRequestRaw makes HTTP request and returns raw response
func (c *AnthropicClient) makeRequestRaw(request AnthropicRequest) (*AnthropicResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response AnthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("Anthropic API error: %s", response.Error.Message)
	}

	return &response, nil
}

// ============================================================================
// Ollama Client Implementation
// ============================================================================

// OllamaRequest represents Ollama API request
type OllamaRequest struct {
	Model    string            `json:"model"`
	Prompt   string            `json:"prompt"`
	Stream   bool              `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// OllamaResponse represents Ollama API response
type OllamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

// NewOllamaClient creates a new Ollama client from config
func NewOllamaClient(cfg *config.LLMConfig) *OllamaClient {
	baseURL := cfg.OllamaURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := cfg.Model
	if model == "" {
		model = "llama2"
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] Creating Ollama client with model: %s, URL: %s\n", model, baseURL)

	return &OllamaClient{
		baseURL:     baseURL,
		model:       model,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Ollama can be slow
		},
	}
}

// Analyze performs general analysis using Ollama
func (c *OllamaClient) Analyze(prompt string) (string, error) {
	request := OllamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": c.temperature,
			"num_predict": c.maxTokens,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var response OllamaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Response, nil
}

// AnalyzeWithTools is not fully supported by Ollama yet
// Falls back to regular analysis
func (c *OllamaClient) AnalyzeWithTools(prompt string, tools []Tool, toolExecutor ToolExecutor) (string, []ToolCallRecord, error) {
	// Ollama doesn't natively support tool calling like OpenAI/Anthropic
	// Fallback to regular analysis with empty tool call records
	fmt.Fprintf(os.Stderr, "[DEBUG] Ollama doesn't support native tool calling, using standard analysis\n")
	response, err := c.Analyze(prompt)
	return response, nil, err
}

// GenerateConfig generates config using Ollama
func (c *OllamaClient) GenerateConfig(serverName, documentation, currentConfig string) (string, error) {
	prompt := fmt.Sprintf(`Analyze this MCP server configuration and documentation, then provide a corrected JSON configuration:

SERVER: %s

DOCUMENTATION:
%s

CURRENT CONFIG:
%s

Please provide:
1. Analysis of configuration issues
2. Corrected JSON configuration
3. Explanation of changes made`, serverName, documentation, currentConfig)

	return c.Analyze(prompt)
}

// ============================================================================
// Helper Functions
// ============================================================================

// truncateToolOutput intelligently truncates tool output to prevent context overflow
// It preserves important parts (beginning and end) and adds a truncation notice
func truncateToolOutput(output string, maxChars int) string {
	if len(output) <= maxChars {
		return output
	}

	// Calculate how much to keep from start and end
	// Keep more from the start as it usually contains important headers/metadata
	startChars := int(float64(maxChars) * 0.7)  // 70% from start
	endChars := maxChars - startChars - 500     // Remaining for end, minus truncation notice

	if endChars < 0 {
		endChars = 0
	}

	start := output[:startChars]
	end := ""
	if endChars > 0 && len(output) > startChars+endChars {
		end = output[len(output)-endChars:]
	}

	truncationNotice := fmt.Sprintf(

		"\n\n[... OUTPUT TRUNCATED: %d characters removed to prevent context overflow ...]\n"+
			"[Original length: %d chars, Truncated to: %d chars]\n"+
			"[Showing first %d and last %d characters]\n\n",
		len(output)-maxChars,
		len(output),
		maxChars,
		startChars,
		endChars,
	)

	return start + truncationNotice + end
}

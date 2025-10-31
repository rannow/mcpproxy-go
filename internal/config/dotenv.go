package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadDotEnv loads environment variables from a .env file
// Returns a map of key-value pairs
func LoadDotEnv(envPath string) (map[string]string, error) {
	env := make(map[string]string)

	// Check if .env file exists
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// .env file doesn't exist, return empty map (not an error)
		return env, nil
	}

	file, err := os.Open(envPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			// Invalid format, skip
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		env[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return env, nil
}

// ApplyDotEnvToLLMConfig applies .env values to LLM config if not already set
func ApplyDotEnvToLLMConfig(llmConfig *LLMConfig, dataDir string) error {
	if llmConfig == nil {
		return nil
	}

	// Try to load .env from data directory
	envPath := filepath.Join(dataDir, ".env")
	envVars, err := LoadDotEnv(envPath)
	if err != nil {
		return err
	}

	loadedFrom := ""

	// If .env file is empty or doesn't exist, try to load from current directory
	if len(envVars) == 0 {
		envPath = ".env"
		envVars, err = LoadDotEnv(envPath)
		if err != nil {
			return err
		}
		if len(envVars) > 0 {
			loadedFrom = "./.env"
		}
	} else {
		loadedFrom = envPath
	}

	// Log if .env was loaded
	if loadedFrom != "" {
		// Count how many LLM-related vars were found
		llmVarCount := 0
		for key := range envVars {
			if key == "OPENAI_API_KEY" || key == "OPENAI_KEY" ||
			   key == "ANTHROPIC_API_KEY" || key == "LLM_PROVIDER" ||
			   key == "LLM_MODEL" || key == "OLLAMA_URL" {
				llmVarCount++
			}
		}
		if llmVarCount > 0 {
			os.Stderr.WriteString(fmt.Sprintf("[INFO] Loaded %d LLM variables from %s\n", llmVarCount, loadedFrom))
		}
	}

	// Apply values to LLM config only if not already set
	// Priority: Config file > .env file > Environment variables

	// OpenAI API Key
	if llmConfig.OpenAIKey == "" {
		if key, ok := envVars["OPENAI_API_KEY"]; ok && key != "" {
			llmConfig.OpenAIKey = key
		} else if key, ok := envVars["OPENAI_KEY"]; ok && key != "" {
			llmConfig.OpenAIKey = key
		}
	}

	// Anthropic API Key
	if llmConfig.AnthropicKey == "" {
		if key, ok := envVars["ANTHROPIC_API_KEY"]; ok && key != "" {
			llmConfig.AnthropicKey = key
		}
	}

	// Provider (only if not set in config)
	if llmConfig.Provider == "" {
		if provider, ok := envVars["LLM_PROVIDER"]; ok && provider != "" {
			llmConfig.Provider = provider
		}
	}

	// Model (only if not set in config)
	if llmConfig.Model == "" {
		if model, ok := envVars["LLM_MODEL"]; ok && model != "" {
			llmConfig.Model = model
		}
	}

	// Ollama URL (only if not set in config)
	if llmConfig.OllamaURL == "" || llmConfig.OllamaURL == "http://localhost:11434" {
		if url, ok := envVars["OLLAMA_URL"]; ok && url != "" {
			llmConfig.OllamaURL = url
		}
	}

	return nil
}

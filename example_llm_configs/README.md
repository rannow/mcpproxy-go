# LLM Configuration Examples

This directory contains example configurations for different LLM providers.

## Quick Start

1. **Choose your provider:**
   - `openai_config.json` - OpenAI GPT models (requires API key)
   - `anthropic_config.json` - Anthropic Claude models (requires API key)
   - `ollama_config.json` - Ollama local models (free, requires Ollama installed)

2. **Copy to your config directory:**
   ```bash
   # For macOS/Linux
   cp openai_config.json ~/.mcpproxy/mcp_config.json

   # Or for Windows
   copy openai_config.json %USERPROFILE%\.mcpproxy\mcp_config.json
   ```

3. **Add your API key** (OpenAI or Anthropic):
   - Edit the config file
   - Replace `YOUR-KEY-HERE` with your actual API key

4. **Restart mcpproxy:**
   ```bash
   pkill mcpproxy
   ./mcpproxy serve
   ```

## Provider Details

### OpenAI (`openai_config.json`)
- **Pros**: Fast, reliable, good general performance
- **Cons**: Requires paid API key
- **Best for**: Most users, production use
- **Get API key**: https://platform.openai.com/api-keys

### Anthropic (`anthropic_config.json`)
- **Pros**: Excellent reasoning, advanced capabilities
- **Cons**: Requires paid API key
- **Best for**: Complex diagnostic tasks, advanced reasoning
- **Get API key**: https://console.anthropic.com/

### Ollama (`ollama_config.json`)
- **Pros**: Free, fully local, privacy-focused
- **Cons**: Requires local installation, slower on weak hardware
- **Best for**: Privacy-sensitive environments, offline use
- **Setup**:
  ```bash
  # Install Ollama
  brew install ollama  # macOS
  # Or download from https://ollama.ai

  # Start Ollama
  ollama serve

  # Pull a model
  ollama pull llama2
  ```

## Customization

All config files support these LLM parameters:

```json
{
  "llm": {
    "provider": "openai|anthropic|ollama",
    "model": "model-name",
    "temperature": 0.7,
    "max_tokens": 2000
  }
}
```

### Temperature
- **0.0**: Deterministic, focused responses
- **0.7**: Balanced (recommended)
- **1.0**: Creative, varied responses

### Max Tokens
- Controls response length
- Higher = longer responses (but more expensive)
- Default: 2000 tokens (~1500 words)

## See Also

- [Complete LLM Configuration Guide](../LLM_CONFIGURATION.md)
- [Main mcpproxy README](../README.md)

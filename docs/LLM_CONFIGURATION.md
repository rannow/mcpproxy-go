# LLM Configuration for AI Diagnostic Agent

The AI Diagnostic Agent in mcpproxy now supports multiple LLM providers: **OpenAI**, **Anthropic**, and **Ollama**.

## Configuration

Add the `llm` section to your `~/.mcpproxy/mcp_config.json`:

### OpenAI (Default)

```json
{
  "llm": {
    "provider": "openai",
    "model": "gpt-4o-mini",
    "openai_api_key": "sk-proj-your-key-here",
    "temperature": 0.7,
    "max_tokens": 2000
  }
}
```

**Available Models:**
- `gpt-4o-mini` (recommended, cost-effective)
- `gpt-4o`
- `gpt-4-turbo`
- `gpt-3.5-turbo`

### Anthropic Claude

```json
{
  "llm": {
    "provider": "anthropic",
    "model": "claude-3-5-sonnet-20241022",
    "anthropic_api_key": "sk-ant-your-key-here",
    "temperature": 0.7,
    "max_tokens": 2000
  }
}
```

**Available Models:**
- `claude-3-5-sonnet-20241022` (recommended)
- `claude-3-5-haiku-20241022`
- `claude-3-opus-20240229`

### Ollama (Local LLM)

```json
{
  "llm": {
    "provider": "ollama",
    "model": "llama2",
    "ollama_url": "http://localhost:11434",
    "temperature": 0.7,
    "max_tokens": 2000
  }
}
```

**Available Models:** (any model you have installed in Ollama)
- `llama2`
- `codellama`
- `mistral`
- `mixtral`
- `neural-chat`
- etc.

## Configuration Priority

mcpproxy loads API keys in the following priority order:

**1. Config File** (`mcp_config.json`) - **Highest Priority**
```json
{
  "llm": {
    "openai_api_key": "sk-proj-..."
  }
}
```

**2. .env File** - **Medium Priority**
```bash
# ~/.mcpproxy/.env or .env in current directory
OPENAI_API_KEY=sk-proj-your-key-here
ANTHROPIC_API_KEY=sk-ant-your-key-here
```

**3. Environment Variables** - **Lowest Priority**
```bash
export OPENAI_API_KEY="sk-proj-your-key-here"
```

## Using .env Files (Recommended for Development)

Create a `.env` file in your mcpproxy data directory or current directory:

```bash
# Create .env file
cp .env.example ~/.mcpproxy/.env

# Edit and add your keys
vi ~/.mcpproxy/.env
```

**Example `.env` file:**
```bash
# OpenAI
OPENAI_API_KEY=sk-proj-your-openai-key-here

# Anthropic
ANTHROPIC_API_KEY=sk-ant-your-anthropic-key-here

# Ollama (optional)
OLLAMA_URL=http://localhost:11434

# General settings (optional)
LLM_PROVIDER=openai
LLM_MODEL=gpt-4o-mini
```

**Advantages of .env files:**
- ✅ Works with GUI apps (unlike `~/.zshrc`)
- ✅ Easy to manage multiple environments
- ✅ Keeps secrets out of config files
- ✅ Standard pattern for API keys
- ✅ Can be shared (using `.env.example`)

⚠️ **Security**: Never commit `.env` files with real keys to version control!

## Environment Variable Fallback

If you don't set API keys in config or `.env`, mcpproxy falls back to system environment variables:

- **OpenAI**: `OPENAI_API_KEY` or `OPENAI_KEY`
- **Anthropic**: `ANTHROPIC_API_KEY`

⚠️ **Important for macOS GUI Apps**: Environment variables in `~/.zshrc` are **NOT** available to GUI applications like mcpproxy's system tray.

**Solutions:**
1. **Recommended**: Use `.env` file (works for GUI apps!)
2. Set in `mcp_config.json` (most reliable)
3. Set systemwide environment variables:
   ```bash
   # In ~/.zprofile (loaded at login for GUI apps)
   export OPENAI_API_KEY="sk-proj-your-key-here"
   export ANTHROPIC_API_KEY="sk-ant-your-key-here"
   ```
4. Use launchctl (for GUI apps):
   ```bash
   launchctl setenv OPENAI_API_KEY "sk-proj-your-key-here"
   ```

## Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `provider` | string | `"openai"` | LLM provider: `openai`, `anthropic`, or `ollama` |
| `model` | string | Provider-specific | Model name to use |
| `openai_api_key` | string | `""` | OpenAI API key (or use env `OPENAI_API_KEY`) |
| `anthropic_api_key` | string | `""` | Anthropic API key (or use env `ANTHROPIC_API_KEY`) |
| `ollama_url` | string | `"http://localhost:11434"` | Ollama server URL |
| `temperature` | float | `0.7` | Response randomness (0.0-1.0) |
| `max_tokens` | int | `2000` | Maximum tokens in response |

## Complete Configuration Example

```json
{
  "listen": ":8080",
  "data_dir": "~/.mcpproxy",
  "enable_tray": true,

  "llm": {
    "provider": "openai",
    "model": "gpt-4o-mini",
    "openai_api_key": "sk-proj-your-key-here",
    "temperature": 0.7,
    "max_tokens": 2000
  },

  "mcpServers": [
    {
      "name": "github-server",
      "url": "https://api.github.com/mcp",
      "protocol": "http",
      "enabled": true
    }
  ]
}
```

## Testing Your Configuration

### Option 1: Using .env File (Recommended)

1. **Create .env file**:
   ```bash
   # Copy example
   cp .env.example ~/.mcpproxy/.env

   # Edit and add your API key
   echo "OPENAI_API_KEY=sk-proj-your-real-key-here" > ~/.mcpproxy/.env
   ```

2. **Create minimal config** (optional):
   ```bash
   cat > ~/.mcpproxy/mcp_config.json << 'EOF'
   {
     "listen": ":8080",
     "llm": {
       "provider": "openai",
       "model": "gpt-4o-mini"
     }
   }
   EOF
   ```

3. **Restart mcpproxy**:
   ```bash
   pkill mcpproxy
   ./mcpproxy serve
   ```

4. **Check the logs**:
   ```
   [DEBUG] Loading .env from: ~/.mcpproxy/.env
   [DEBUG] Creating OpenAI client with model: gpt-4o-mini
   [DEBUG] OpenAI API key found: sk-proj...xyz
   INFO: LLM Agent initialized (provider=openai, model=gpt-4o-mini)
   ```

### Option 2: Using Config File Only

1. **Create config with API key**:
   ```bash
   cat > ~/.mcpproxy/mcp_config.json << 'EOF'
   {
     "listen": ":8080",
     "llm": {
       "provider": "openai",
       "model": "gpt-4o-mini",
       "openai_api_key": "sk-proj-your-real-key-here"
     }
   }
   EOF
   ```

2. **Restart mcpproxy**:
   ```bash
   pkill mcpproxy
   ./mcpproxy serve
   ```

3. **Check the logs** for confirmation

4. **Use the AI Diagnostic Agent** via the system tray menu

## Provider Comparison

| Feature | OpenAI | Anthropic | Ollama |
|---------|--------|-----------|--------|
| **Cost** | Paid API | Paid API | Free (local) |
| **Speed** | Fast | Fast | Varies (hardware-dependent) |
| **Privacy** | Cloud | Cloud | Fully local |
| **Tool Calling** | ✅ Full support | ✅ Full support | ❌ Limited* |
| **Best For** | General use | Complex reasoning | Privacy-sensitive environments |

*Ollama doesn't natively support tool calling, so advanced features like config file reading/writing may be limited.

## Troubleshooting

### "OpenAI API key not found" Error

**Problem**: mcpproxy can't find your API key
**Solution**:
1. Set it in `mcp_config.json` (recommended)
2. Or add to `~/.zprofile` (for GUI apps on macOS)

### "Failed to make request to Ollama" Error

**Problem**: Ollama server not running
**Solution**:
```bash
# Install Ollama
brew install ollama

# Start Ollama service
ollama serve

# Pull a model
ollama pull llama2
```

### Rate Limiting

If you hit rate limits:
1. **OpenAI**: Upgrade your API plan
2. **Anthropic**: Check your usage tier
3. **Ollama**: No limits (local)

## Migration Guide

### From Environment Variables Only

**Before:**
```bash
# In ~/.zshrc (doesn't work for GUI apps!)
export OPENAI_API_KEY="sk-proj-..."
```

**After:**
```json
{
  "llm": {
    "provider": "openai",
    "openai_api_key": "sk-proj-your-key-here"
  }
}
```

### Switching Providers

Simply change the `provider` field and add the appropriate API key:

```bash
# Edit your config
vi ~/.mcpproxy/mcp_config.json

# Restart mcpproxy
pkill mcpproxy && ./mcpproxy serve
```

## Security Best Practices

1. **Never commit API keys** to version control
2. **Use environment variables** for CI/CD pipelines
3. **Use config files** for local development
4. **Rotate keys regularly**
5. **Monitor usage** through provider dashboards

# MCP Code API - Multi-Provider Code Generation Server

[![CI](https://github.com/cecil-the-coder/mcp-code-api/actions/workflows/ci.yml/badge.svg)](https://github.com/cecil-the-coder/mcp-code-api/actions/workflows/ci.yml)
[![Release](https://github.com/cecil-the-coder/mcp-code-api/actions/workflows/release.yml/badge.svg)](https://github.com/cecil-the-coder/mcp-code-api/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/cecil-the-coder/mcp-code-api)](https://github.com/cecil-the-coder/mcp-code-api/releases/latest)

A high-performance **Model Context Protocol (MCP) server** supporting multiple AI providers (Cerebras, OpenRouter, OpenAI, Anthropic, Gemini, and more). Designed for **planning with Claude Code, Cline, or Cursor** while leveraging fast providers like Cerebras for code generation to maximize speed and avoid API limits.

## 🚀 Why Go?

The Go implementation offers significant advantages over the Node.js version:
- **10x faster performance** for large code generation tasks
- **Single binary deployment** - no Node.js runtime required
- **Lower memory footprint** and better resource utilization
- **Cross-platform compilation** for easy deployment
- **Type safety** and better error handling
- **Concurrent processing** for handling multiple requests

## ✨ Features

- 🎯 **Smart API Routing** with automatic fallback between Cerebras and OpenRouter
- 🔧 **Single 'write' Tool** for ALL code operations (creation, editing, generation)
- 🎨 **Enhanced Visual Diffs** with emoji indicators (✅ additions, ❌ removals, 🔍 changes)
- 🔄 **Auto-Instruction System** that enforces proper MCP tool usage
- 📁 **Context-Aware Processing** with multiple file support
- 💻 **Multi-IDE Support** - Claude Code, Cursor, Cline, VS Code
- ⚙️ **Interactive Configuration Wizard** for easy setup
- 📝 **Comprehensive Logging** with debug support

## 📋 System Requirements

- **Go 1.21+** (for building from source)
- **Cerebras API Key** (primary) or **OpenRouter API Key** (fallback)
- **Supported IDE**: Claude Code, Cursor, Cline, or VS Code

## 🚀 Quick Start

### Option 1: Install from Binary (Recommended)

```bash
# Download the latest release for your platform
curl -L https://github.com/cecil-the-coder/mcp-code-api/releases/latest/download/mcp-code-api-$(uname -s)-$(uname -m) -o mcp-code-api

# Make it executable
chmod +x mcp-code-api

# Move to your PATH
sudo mv mcp-code-api /usr/local/bin/
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/cecil-the-coder/mcp-code-api.git
cd mcp-code-api

# Build the binary
make build

# Install to system PATH
make install
```

## 📱 Configuration

### 1. Run the Configuration Wizard

```bash
mcp-code-api config
```

The wizard will guide you through:
- Setting up API keys for Cerebras and/or OpenRouter
- Configuring your preferred IDE
- Testing API connections
- Generating configuration files

### 2. Set API Keys (Optional Manual Setup)

```bash
# Cerebras API (Primary)
export CEREBRAS_API_KEY="your_cerebras_api_key"

# OpenRouter API (Optional Fallback)
export OPENROUTER_API_KEY="your_openrouter_api_key"

# Set model preferences (optional)
export CEREBRAS_MODEL="zai-glm-4.6"
export OPENROUTER_MODEL="qwen/qwen3-coder"
```

### 3. Start the MCP Server

```bash
mcp-code-api server
```

## 💻 IDE Integration

### Claude Code

The configuration wizard automatically sets up Claude Code. After configuration:

1. Restart Claude Code
2. The `write` tool will appear in your tool list
3. Use it for all code operations

### Cursor

1. Run the configuration wizard
2. Copy the generated rules to Cursor → Settings → Developer → User Rules
3. Restart Cursor

### Cline

1. Run the configuration wizard
2. Restart Cline
3. The `write` tool will be available

### VS Code

1. Install an MCP extension for VS Code
2. Run the configuration wizard
3. Restart VS Code
4. The `write` tool will be available via MCP

## 🔧 Usage

The MCP tool provides a single `write` tool that handles ALL code operations:

### Basic Usage

```bash
# In your IDE, use natural language:

"Create a REST API with Express.js that handles user authentication"

"Add input validation to the login function in auth.js"

"Generate a Python script that processes CSV files and outputs to JSON"
```

### Advanced Usage with Context Files

```bash
"Refactor the database connection in models.js using the pattern from utils.js"

# The tool will automatically read context files:
# - models.js (existing file to modify)
# - utils.js (context for patterns)
```

### Parameters

The `write` tool accepts:

- **file_path** (required): Absolute path to the target file
- **prompt** (required): Detailed description of what to create/modify
- **context_files** (optional): Array of file paths for context

## 🎨 Visual Diffs

The Go implementation enhances visual diffs with:

- ✅ **Green indicators** for new lines
- ❌ **Red indicators** for removed lines
- 🔍 **Change indicators** for modified content
- 📊 **Summary statistics** (additions, removals, modifications)
- 📁 **Full file paths** for clarity

## 🔒 Auto-Instruction System

The Go implementation includes an enhanced auto-instruction system that:

- Automatically enforces MCP tool usage
- Prevents direct file editing
- Provides clear instructions to AI models
- Ensures consistent behavior across all IDEs

## 🏗️ Development

### Building

```bash
# Build for current platform
make build

# Build for Linux (cross-compile)
make linux

# Build all platforms
make release
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make coverage
```

### Code Quality

```bash
# Format code
make format

# Run linter
make lint
```

### Docker

```bash
# Build Docker image
make docker-build

# Run Docker container
make docker-run
```

## 📁 Project Structure

```
mcp-code-api/
├── 📄 go.mod              # Go module definition
├── 📄 main.go              # Entry point
├── 📁 cmd/                 # CLI commands
│   ├── 📜 root.go          # Root command
│   ├── 📜 server.go        # Server command
│   └── 📜 config.go        # Configuration command
├── 📁 internal/            # Internal packages
│   ├── 📁 api/             # API integrations
│   │   ├── 📜 router.go    # API router
│   │   ├── 📜 cerebras.go  # Cerebras client
│   │   └── 📜 openrouter.go # OpenRouter client
│   ├── 📁 config/          # Configuration management
│   │   ├── 📜 config.go     # Configuration types
│   │   ├── 📜 constants.go # Constants
│   │   ├── 📜 utils.go      # Utility functions
│   │   └── 📁 interactive/ # Interactive wizards
│   ├── 📁 mcp/             # MCP server implementation
│   │   ├── 📜 server.go    # Main MCP server
│   │   └── 📜 write_tool.go # Write tool handler
│   ├── 📁 utils/           # General utilities
│   │   └── 📜 file_utils.go # File operations
│   ├── 📁 formatting/      # Response formatting
│   │   └── 📜 response_formatter.go # Visual diffs
│   └── 📁 logger/          # Logging system
│       └── 📜 logger.go      # Logger implementation
├── 📄 Makefile             # Build automation
├── 📄 README.md            # This file
└── 📄 LICENSE              # MIT License
```

## 🔧 Configuration Options

### Environment Variables

```bash
# Cerebras Configuration
CEREBRAS_API_KEY=your_key
CEREBRAS_MODEL=zai-glm-4.6
CEREBRAS_TEMPERATURE=0.6
CEREBRAS_MAX_TOKENS=4096

# OpenRouter Configuration
OPENROUTER_API_KEY=your_key
OPENROUTER_MODEL=qwen/qwen3-coder
OPENROUTER_SITE_URL=https://github.com/your-repo
OPENROUTER_SITE_NAME=Your Project

# Server Configuration
CEREBRAS_MCP_LOG_LEVEL=info
CEREBRAS_MCP_LOG_FILE=/path/to/logfile
CEREBRAS_MCP_DEBUG=false
CEREBRAS_MCP_VERBOSE=false
```

### Configuration File

You can also use a YAML configuration file at `~/.mcp-code-api/config.yaml`:

```yaml
cerebras:
  api_key: "your_key"
  model: "zai-glm-4.6"
  temperature: 0.6
  max_tokens: 4096

openrouter:
  api_key: "your_key"
  model: "qwen/qwen3-coder"
  site_url: "https://github.com/your-repo"
  site_name: "Your Project"

logging:
  level: "info"
  verbose: false
  debug: false
  file: "/path/to/logfile"
```

### Load Balancing & Failover

The server supports multiple API keys per provider for automatic load balancing and failover:

#### Multiple API Keys Configuration

```yaml
providers:
  cerebras:
    # Multiple keys - automatically load balanced
    api_keys:
      - "${CEREBRAS_API_KEY_1}"
      - "${CEREBRAS_API_KEY_2}"
      - "${CEREBRAS_API_KEY_3}"
    model: "zai-glm-4.6"

  openrouter:
    # Single key - backward compatible
    api_key: "${OPENROUTER_API_KEY}"
    model: "qwen/qwen3-coder"
```

#### How It Works

- **Round-robin load balancing**: Requests are evenly distributed across all configured keys
- **Automatic failover**: If one key fails (rate limit, error), automatically tries the next available key
- **Exponential backoff**: Failed keys enter backoff period: 1s → 2s → 4s → 8s → max 60s
- **Health tracking**: System monitors each key's health and skips unhealthy keys
- **Auto-recovery**: Keys automatically recover and rejoin rotation after backoff period

#### Benefits

- **Rate limit avoidance**: Multiply your effective rate limit by using multiple keys
- **High availability**: Service continues even if some keys fail or are rate limited
- **Better throughput**: Distribute load across multiple keys for higher concurrency
- **Fault tolerance**: Automatic recovery from transient failures

#### Recommended Setup

- **Light usage**: 1 key is sufficient
- **Production**: 2-3 keys recommended for failover capability
- **High volume**: 3-5 keys for optimal performance and resilience

#### Example with Environment Variables

```bash
# Set multiple keys
export CEREBRAS_API_KEY_1="csk-primary-xxxxx"
export CEREBRAS_API_KEY_2="csk-secondary-xxxxx"
export CEREBRAS_API_KEY_3="csk-tertiary-xxxxx"

# Start server - will automatically use all configured keys
mcp-code-api server
```

For a complete example configuration, see [config.example.yaml](config.example.yaml).

## 🔌 Using API-Compatible Providers

The server supports **API-compatible providers** - third-party services that implement the same API format as the major providers. This includes:

- **Anthropic-compatible** (e.g., z.ai with GLM-4.6, local proxies)
- **OpenAI-compatible** (e.g., LM Studio, Ollama, LocalAI)
- **Custom self-hosted endpoints**

### Anthropic-Compatible Providers (z.ai)

The MCP Code API supports any provider that implements the Anthropic Messages API format.

#### Configuration File Method

Add to your `~/.mcp-code-api/config.yaml`:

```yaml
providers:
  anthropic:
    # z.ai's authentication token
    api_key: "your-zai-api-key"
    # z.ai's Anthropic-compatible endpoint
    base_url: "https://api.z.ai/api/anthropic"
    # Use Z.ai's GLM-4.6 model (200K context, optimized for coding)
    model: "glm-4.6"

  enabled:
    - anthropic

  preferred_order:
    - anthropic
```

**Available Z.ai Models:**
- `glm-4.6` - Latest flagship model (200K context, best for coding/reasoning)
- `glm-4.5-air` - Lighter/faster variant for quick tasks

#### Environment Variables Method

```bash
# z.ai example
export ANTHROPIC_AUTH_TOKEN="your-zai-api-key"
export ANTHROPIC_BASE_URL="https://api.z.ai/api/anthropic"

# Start the server
./mcp-code-api server
```

**Note**: Both `ANTHROPIC_API_KEY` and `ANTHROPIC_AUTH_TOKEN` environment variables are supported.

#### Multiple Anthropic Providers (Advanced)

If you want to use both standard Anthropic AND a compatible provider:

```yaml
providers:
  # Standard Anthropic
  anthropic:
    api_key: "sk-ant-..."
    base_url: "https://api.anthropic.com"
    model: "claude-3-5-sonnet-20241022"

  # Custom provider: z.ai
  custom:
    zai:
      type: "anthropic"
      name: "Z.ai"
      api_key: "your-zai-api-key"
      base_url: "https://api.z.ai/api/anthropic"
      default_model: "glm-4.6"
      supports_streaming: true
      supports_tool_calling: true
      tool_format: "anthropic"

  enabled:
    - anthropic
    - zai

  preferred_order:
    - zai         # Try z.ai first
    - anthropic   # Fall back to official Anthropic
```

### OpenAI-Compatible Providers (LM Studio, Ollama)

```yaml
providers:
  openai:
    api_key: "lm-studio"  # Can be any value for LM Studio
    base_url: "http://localhost:1234/v1"
    model: "local-model"
```

Or using environment variables:

```bash
export OPENAI_API_KEY="lm-studio"
export OPENAI_BASE_URL="http://localhost:1234/v1"
```

### Supported Environment Variables

All providers now support custom base URLs via environment variables:

| Provider   | API Key Env Var(s)                    | Base URL Env Var       |
|-----------|---------------------------------------|------------------------|
| Anthropic | `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` | `ANTHROPIC_BASE_URL` |
| OpenAI    | `OPENAI_API_KEY`                      | `OPENAI_BASE_URL`     |
| Gemini    | `GEMINI_API_KEY`                      | `GEMINI_BASE_URL`     |
| Qwen      | `QWEN_API_KEY`                        | `QWEN_BASE_URL`       |
| Cerebras  | `CEREBRAS_API_KEY`                    | `CEREBRAS_BASE_URL`   |
| OpenRouter| `OPENROUTER_API_KEY`                  | `OPENROUTER_BASE_URL` |

**Examples:**

```bash
# Use an OpenAI-compatible endpoint (like LM Studio)
export OPENAI_API_KEY="lm-studio-key"
export OPENAI_BASE_URL="http://localhost:1234/v1"

# Use a custom Anthropic-compatible endpoint (z.ai)
export ANTHROPIC_AUTH_TOKEN="your-token"
export ANTHROPIC_BASE_URL="https://api.z.ai/api/anthropic"
```

### Troubleshooting

**Authentication fails:**
- Verify your token/API key is correct
- Check if the base URL includes the correct API version path
- Some providers require specific headers - check their documentation

**Different API format:**
If the provider uses a slightly different format, you may need to create a custom provider adapter.

**Rate limiting:**
Some compatible providers have different rate limits than the official APIs. Adjust your usage accordingly.

## 🤝 Contributing

Contributions are welcome! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- 📧 **Email**: support@cerebras.ai
- 🐛 **Issues**: [GitHub Issues](https://github.com/cecil-the-coder/mcp-code-api/issues)
- 📚 **Documentation**: [Wiki](https://github.com/cecil-the-coder/mcp-code-api/wiki)
- 💬 **Community**: [Discussions](https://github.com/cecil-the-coder/mcp-code-api/discussions)

## 🔗 Related Projects

- [Cerebras Node.js MCP Server](https://github.com/cerebras/cerebras-mcp) - Original Node.js implementation
- [Cerebras AI Platform](https://cloud.cerebras.ai) - AI platform
- [Model Context Protocol](https://modelcontextprotocol.io) - MCP specification

## 🎯 Roadmap

- [ ] **Real-time streaming** for large code generation
- [ ] **Plugin system** for custom tools
- [ ] **Workspace management** for project-level operations
- [ ] **Performance monitoring** and metrics
- [ ] **Advanced caching** for faster responses
- [ ] **Multi-model support** with automatic selection

---

**⚡ Built with Go for maximum performance and reliability**
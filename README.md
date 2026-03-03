# AI Gateway

A unified API gateway for multiple AI service providers with intelligent rate limiting, caching, and routing capabilities.

## Features

- **Multi-Provider Support**: Unified interface for OpenAI, Anthropic, Azure OpenAI, and more
- **Intelligent Rate Limiting**: Per-user and global rate limiting with configurable quotas
- **Smart Caching**: Response caching to reduce API costs and improve latency
- **Flexible Routing**: Route requests to different providers based on model, cost, or availability
- **RESTful API**: OpenAI-compatible API for easy integration
- **Web Dashboard**: Built-in management console for monitoring and configuration
- **Docker Ready**: One-click deployment with Docker Compose
- **Intelligent Routing**: Smart model selection based on task type and difficulty (requires local Ollama + qwen2.5:0.5b-instruct)
## Quick Start

### Prerequisites

- Go 1.21 or higher
- Redis (optional, for distributed caching)
- Docker (optional, for containerized deployment)

### Local Development

1. Clone the repository:
   ```bash
   git clone https://github.com/wenkezhi8/ai-gateway.git
   cd ai-gateway
   ```

2. Install dependencies:
   ```bash
   make deps
   ```

3. Copy the example configuration:
   ```bash
   cp configs/config.example.json configs/config.json
   ```

4. Edit `configs/config.json` and add your API keys

5. Run the application:
   ```bash
   make run
   ```

### Docker Deployment

1. Build and run with Docker Compose:
   ```bash
   make docker-build
   make docker-up
   ```

2. Access the gateway at `http://localhost:8566`

## Configuration

Configuration is managed via `configs/config.json`. Environment variables can override file settings.

### Edition Management

The dashboard supports 3 editions: `basic`, `standard`, and `enterprise`.

- Switch path: `Settings -> 版本管理`
- Default: `standard`
- Guide: `docs/EDITION-GUIDE.md`
- `Ollama 管理` 页面：标准版与企业版可见（侧边栏菜单）
- `向量管理` / `知识库` 入口：仅企业版可见（Header 入口）

Related admin APIs:

- `GET /api/admin/edition`
- `PUT /api/admin/edition`
- `GET /api/admin/edition/definitions`
- `GET /api/admin/edition/dependencies`

### Environment Variables

| Variable | Description |
|----------|-------------|
| `CONFIG_PATH` | Path to config file (default: `./configs/config.json`) |
| `SERVER_PORT` | Server port (default: `8566`) |
| `GIN_MODE` | Gin mode: `debug` or `release` |
| `REDIS_HOST` | Redis host |
| `OPENAI_API_KEY` | OpenAI API key |
| `ANTHROPIC_API_KEY` | Anthropic API key |

## API Endpoints

### Chat Completions
```
POST /api/v1/chat/completions
```

### Completions
```
POST /api/v1/completions
```

### Embeddings
```
POST /api/v1/embeddings
```

### List Providers
```
GET /api/v1/providers
```

### Health Check
```
GET /health
```

## Project Structure

```
ai-gateway/
├── cmd/
│   └── gateway/          # Application entry point
│       └── main.go
├── internal/
│   ├── config/           # Configuration management
│   ├── handler/          # HTTP handlers
│   ├── middleware/       # HTTP middleware
│   ├── router/           # Router setup
│   ├── provider/         # AI provider adapters
│   ├── limiter/          # Rate limiting
│   └── cache/            # Response caching
├── pkg/                  # Public packages
├── configs/              # Configuration files
├── scripts/              # Utility scripts
├── web/                  # Web dashboard
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── README.md
```

## Development

### Running Tests

```bash
make test
```

### Code Formatting

```bash
make fmt
```

### Linting

```bash
make lint
```

## License

MIT License

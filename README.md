# Point Cloud Annotator

A web application for loading, viewing, and annotating 3D point clouds with persistent storage.

## Features

- **3D Point Cloud Visualization**: Built on Potree WebGL viewer for efficient rendering of large point cloud datasets
- **Interactive Annotations**: Click on any point in the 3D scene to create annotation markers
- **Persistent Storage**: Annotations are saved to PostgreSQL with Redis caching for optimal performance
- **Delete Functionality**: Easily remove annotations with a hover-to-reveal delete button
- **Microservices Architecture**: API Gateway and Handler services with role-based configuration

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Frontend     │────▶│   API Gateway   │────▶│    Handler      │
│  (Potree/Nginx) │     │   (Go + Fx)     │     │   (Go + Fx)     │
└─────────────────┘     └─────────────────┘     └────────┬────────┘
                                                         │
                               ┌─────────────────────────┼─────────────────────────┐
                               │                         │                         │
                        ┌──────▼──────┐          ┌───────▼───────┐
                        │   Redis     │          │  PostgreSQL   │
                        │   (Cache)   │          │  (Database)   │
                        └─────────────┘          └───────────────┘
```

### Backend Services

The backend is a single Go binary that can run in different modes:

- **Gateway Mode** (`-role gateway`): Routes incoming requests to handler services
- **Handler Mode** (`-role handler`): Processes annotation CRUD operations with database/cache

Built with:
- [Uber Fx](https://github.com/uber-go/fx) for dependency injection
- [Gin](https://github.com/gin-gonic/gin) for HTTP routing
- [pgx](https://github.com/jackc/pgx) for PostgreSQL
- [go-redis](https://github.com/redis/go-redis) for Redis

### Frontend

- Built on [Potree](https://github.com/potree/potree) WebGL point cloud viewer
- Custom annotation toolbar and dialog
- Real-time annotation management

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Start all services
make docker-up

# Or directly with docker compose
docker compose up -d
```

Access the application:
- **Frontend**: http://localhost:3000
- **API Gateway**: http://localhost:8080

### Local Development

```bash
# Install Go dependencies
make deps

# Build the binary
make build

# Run tests
make test

# Start PostgreSQL and Redis locally, then:
make run-handler  # In terminal 1
make run-gateway  # In terminal 2
```

## API Endpoints

All endpoints are prefixed with `/api/v1`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/annotations` | List all annotations |
| GET | `/annotations/:id` | Get annotation by ID |
| POST | `/annotations` | Create new annotation |
| PUT | `/annotations/:id` | Update annotation |
| DELETE | `/annotations/:id` | Delete annotation |

### Request/Response Examples

**Create Annotation**
```json
POST /api/v1/annotations
{
  "x": 1.5,
  "y": 2.5,
  "z": 3.5,
  "title": "Point of Interest",
  "description": "Optional description (max 256 bytes)"
}
```

**Response**
```json
{
  "data": {
    "id": "uuid",
    "x": 1.5,
    "y": 2.5,
    "z": 3.5,
    "title": "Point of Interest",
    "description": "Optional description",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

## Configuration

The service can be configured via environment variables or command-line flags:

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `SERVICE_ROLE` | `-role` | `gateway` | Service role: `gateway` or `handler` |
| `SERVER_PORT` | `-port` | `8080` | HTTP server port |
| `HANDLER_URL` | - | `http://handler:8081` | Handler service URL (gateway only) |
| `DATABASE_URL` | - | `postgres://...` | PostgreSQL connection string |
| `REDIS_URL` | - | `redis://redis:6379` | Redis connection string |
| `ENVIRONMENT` | - | `development` | Environment: `development` or `production` |

## Project Structure

```
.
├── backend/
│   ├── cmd/
│   │   └── main.go              # Application entry point
│   └── internal/
│       ├── cache/               # Redis cache implementation
│       ├── config/              # Configuration management
│       ├── database/            # PostgreSQL repository
│       ├── gateway/             # API Gateway proxy logic
│       ├── handler/             # Request handlers
│       └── models/              # Data models
├── frontend/
│   ├── index.html               # Main HTML page
│   ├── css/
│   │   └── annotator.css        # Custom styles
│   ├── js/
│   │   └── annotator.js         # Application logic
│   └── nginx.conf               # Nginx configuration
├── .tmp/
│   └── potree/                  # Potree library and assets
├── docker-compose.yml           # Docker Compose configuration
├── Makefile                     # Build and development commands
└── README.md                    # This file
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Code Quality

```bash
# Lint code
make lint

# Format code
make fmt
```

### Docker Commands

```bash
# Build images
make docker-build

# Start services
make docker-up

# View logs
make docker-logs

# View specific service logs
make docker-logs-handler

# Stop services
make docker-down

# Clean up (including volumes)
make docker-clean
```

## License

See [LICENSE](LICENSE) file.

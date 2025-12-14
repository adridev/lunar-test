# Running the Application

## Quick Start

### Using Docker Compose

```bash
docker-compose -f docker/docker-compose.yml up
```

This starts:

- MongoDB on port 27017
- Application on port 8088

### Using Make

```bash
make docker-up    # Start all services
make docker-down  # Stop all services
make docker-logs  # View logs
```

## Running Tests

```bash
go test ./...                    # All tests
```

## API Endpoints

- `POST /messages` - Submit rocket messages
- `GET /rockets` - List all rockets
- `GET /rockets/{channel}` - Get specific rocket by channel ID
- `GET /health` - Health check

## Code Generation

If you modify `docs/openapi.yaml`:

```bash
make generate
```

This regenerates `internal/api/api.gen.go` using oapi-codegen.

.PHONY: generate install-tools clean tidy build test docker-build docker-up docker-down docker-logs

generate:
	@echo "Generating API code from OpenAPI spec..."
	@mkdir -p internal/api
	@go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest -config oapi-codegen.yaml docs/openapi.yaml
	@echo "Code generation complete!"
	@$(MAKE) tidy

install-tools:
	@echo "Installing oapi-codegen..."
	@go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

tidy:
	@echo "Tidying go modules..."
	@go mod tidy
	@echo "Tidy complete!"

clean:
	@echo "Cleaning generated files..."
	@rm -f internal/api/*.gen.go
	@echo "Clean complete!"

build: generate
	@echo "Building application..."
	@go build -o bin/rockets ./cmd
	@echo "Build complete!"

test:
	@echo "Running tests..."
	@go test -v ./...

docker-build:
	@echo "Building Docker images..."
	@docker-compose -f docker/docker-compose.yml build
	@echo "Docker build complete!"

docker-up:
	@echo "Starting services..."
	@docker-compose -f docker/docker-compose.yml up -d
	@echo "Services started!"

docker-down:
	@echo "Stopping services..."
	@docker-compose -f docker/docker-compose.yml down
	@echo "Services stopped!"

docker-logs:
	@docker-compose -f docker/docker-compose.yml logs -f

.DEFAULT_GOAL := generate

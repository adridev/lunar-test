# Code Generation with oapi-codegen

This project uses [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to generate Go server code from the OpenAPI specification.

## Configuration

The code generation is configured through `oapi-codegen.yaml`:

- **Package**: `api` - Generated code is placed in the `internal/api` package
- **Output**: `internal/api/api.gen.go` - Combined generated file
- **Generates**:
  - Type definitions for all API models
  - Chi router handlers
  - Embedded OpenAPI spec

## Usage

### Generate Code

To generate or regenerate the API code:

```bash
make generate
```

This will:
1. Create the `internal/api` directory if it doesn't exist
2. Run oapi-codegen with the configuration
3. Run `go mod tidy` to update dependencies

### Clean Generated Files

To remove generated code:

```bash
make clean
```

### Build Application

To build the application (includes code generation):

```bash
make build
```

## Generated Files

- `internal/api/api.gen.go` - Generated types, server interface, and Chi router

## Modifying the API

1. Edit `docs/openapi.yaml` with your API changes
2. Run `make generate` to regenerate the code
3. Implement the new endpoints in your handler implementation

## Manual Installation

If you want to install oapi-codegen globally:

```bash
make install-tools
```

Or directly:

```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

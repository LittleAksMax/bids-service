# Bids Service

A clean, testable Go service with concurrent HTTP server and scheduler components.

## Architecture

This project follows clean architecture principles with clear separation of concerns:

```
bids-service/
├── main.go                      # Application entry point and orchestration
├── internal/
│   ├── domain/                  # Core business entities and interfaces
│   │   └── config.go           # Configuration entity and repository interface
│   ├── repository/              # Data access implementations
│   │   └── config_repository.go # In-memory repository implementation
│   ├── service/                 # Business logic layer
│   │   ├── config_service.go   # Configuration service
│   │   └── config_service_test.go
│   ├── handler/                 # HTTP request handlers
│   │   ├── config_handler.go   # HTTP handler for configuration endpoints
│   │   └── config_handler_test.go
│   ├── scheduler/               # Background scheduler
│   │   └── scheduler.go        # Polls for due configurations
│   └── server/                  # HTTP server wrapper
│       └── server.go           # Server setup and routing
└── api/                         # Legacy API code (can be removed)
```

## Components

### 1. Domain Layer (`internal/domain`)
- Defines core business entities (`Configuration`)
- Defines repository interfaces
- No external dependencies
- Pure business logic

### 2. Repository Layer (`internal/repository`)
- Implements data access interfaces
- Currently uses in-memory storage (scaffold)
- Can be replaced with database implementations

### 3. Service Layer (`internal/service`)
- Contains business logic
- Depends only on domain interfaces
- Fully testable with mock repositories

### 4. Handler Layer (`internal/handler`)
- HTTP request/response handling
- Delegates to service layer
- Testable with httptest

### 5. Scheduler (`internal/scheduler`)
- Runs in a separate goroutine
- Polls repository for due configurations
- Processes them via service layer
- Graceful shutdown support

### 6. Server (`internal/server`)
- HTTP server wrapper
- Runs in a separate goroutine
- Graceful shutdown support
- Chi router with middleware

## Concurrency Model

The application runs two main goroutines:

1. **HTTP Server Goroutine**: Handles incoming HTTP requests
   - POST / - Schedule update endpoint
   - GET /ping - Heartbeat endpoint

2. **Scheduler Goroutine**: Polls for due configurations
   - Runs at configurable intervals (default: 30s)
   - Processes configurations via service layer

Both goroutines:
- Use context for cancellation
- Support graceful shutdown
- Coordinate via WaitGroup

## Configuration

Environment variables:
- `PORT`: HTTP server port (default: 8080)

## Running the Application

```bash
# Set port (optional)
export PORT=8080

# Run the application
go run main.go
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/service/
```

## Graceful Shutdown

The application handles SIGINT and SIGTERM signals:
1. Receives shutdown signal
2. Cancels context
3. HTTP server stops accepting new connections
4. Scheduler completes current iteration
5. Waits for all goroutines to finish
6. Exits cleanly

## Extensibility

### Adding a New Repository Implementation

```go
type PostgresConfigRepository struct {
    db *sql.DB
}

func (r *PostgresConfigRepository) GetDueConfigurations() ([]*domain.Configuration, error) {
    // Implement database query
}
```

### Adding More Endpoints

Add routes in `internal/server/server.go`:

```go
mux.Get("/configs/{id}", s.handler.GetConfiguration)
```

### Customizing Poll Interval

Modify `pollInterval` in `main.go` or make it configurable via environment variable.

## Testing Strategy

- **Unit Tests**: Test each layer in isolation using mocks
- **Integration Tests**: Test handler + service + repository together
- **Mock Implementations**: Use interface-based mocking for clean tests

## TODO

This is a scaffold. Implement:
- [ ] Actual repository logic (database integration)
- [ ] Request/response DTOs for handlers
- [ ] Configuration validation
- [ ] Error handling and logging improvements
- [ ] Metrics and monitoring
- [ ] More comprehensive tests
- [ ] API documentation (OpenAPI/Swagger)


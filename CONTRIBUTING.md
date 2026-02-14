# Contributing to goto

Thank you for your interest in contributing to `goto`! This guide will help you
get started.

## Development Setup

1. **Prerequisites**
   - Go 1.22 or later
   - Git

2. **Clone the repository**
   ```bash
   git clone https://github.com/riyanimam/goto.git
   cd goto
   ```

3. **Run tests**
   ```bash
   make test
   ```

## Adding a New AWS Service

Each AWS service is implemented as a separate package under `services/`. To add
support for a new service:

1. Create a new directory under `services/` (e.g., `services/dynamodb/`).

2. Implement the `awsmock.Service` interface:

   ```go
   package dynamodb

   import "net/http"

   type Service struct {
       // in-memory state
   }

   func New() *Service {
       return &Service{}
   }

   func (s *Service) Name() string         { return "dynamodb" }
   func (s *Service) Handler() http.Handler { return http.HandlerFunc(s.handle) }
   func (s *Service) Reset()               { /* clear state */ }

   func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
       // Parse request and return mock response
   }
   ```

3. Register the service in `builtin.go`:

   ```go
   func builtinServices() []Service {
       return []Service{
           sts.New(),
           s3.New(),
           sqs.New(),
           dynamodb.New(), // Add your service here
       }
   }
   ```

4. Add tests in `awsmock_test.go` using the real AWS SDK client.

5. Update the README with the new service's supported operations.

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`).
- Use `sync.RWMutex` for thread-safe state management.
- Keep service implementations self-contained in their own packages.
- Write table-driven tests where applicable.

## Pull Requests

1. Fork the repository and create a feature branch.
2. Make your changes with clear commit messages.
3. Ensure all tests pass: `make ci`
4. Open a pull request with a description of your changes.

## Reporting Issues

Please open an issue on GitHub with:
- A clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Go version and OS

## License

By contributing, you agree that your contributions will be licensed under the
Apache License 2.0.

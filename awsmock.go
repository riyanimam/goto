package awsmock

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// Service represents an AWS service mock that can handle HTTP requests.
type Service interface {
	// Name returns the AWS service identifier (e.g., "s3", "sqs", "sts").
	Name() string

	// Handler returns the HTTP handler for this service.
	Handler() http.Handler

	// Reset clears all in-memory state for the service.
	Reset()
}

// MockServer is a mock AWS server that routes requests to service handlers.
type MockServer struct {
	server   *httptest.Server
	services map[string]Service
	mu       sync.RWMutex
}

// Start creates and starts a new mock AWS server with all built-in services.
// The server is automatically stopped when the test completes.
func Start(t testing.TB, opts ...Option) *MockServer {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	m := &MockServer{
		services: make(map[string]Service),
	}

	// Register built-in services.
	for _, svc := range builtinServices() {
		m.Register(svc)
	}

	// Register any additional user-provided services.
	for _, svc := range cfg.services {
		m.Register(svc)
	}

	m.server = httptest.NewServer(m)
	t.Cleanup(m.Stop)

	return m
}

// Register adds a service to the mock server.
// If a service with the same name already exists, it is replaced.
func (m *MockServer) Register(svc Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services[svc.Name()] = svc
}

// URL returns the base URL of the mock server.
func (m *MockServer) URL() string {
	return m.server.URL
}

// AWSConfig returns an [aws.Config] pre-configured to route all requests
// to the mock server with static test credentials.
func (m *MockServer) AWSConfig(ctx context.Context) (aws.Config, error) {
	endpoint := m.server.URL

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"AKIAIOSFODNN7EXAMPLE",
			"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"testing",
		)),
	)
	if err != nil {
		return aws.Config{}, err
	}

	cfg.BaseEndpoint = aws.String(endpoint)

	return cfg, nil
}

// Stop shuts down the mock server and resets all services.
func (m *MockServer) Stop() {
	if m.server != nil {
		m.server.Close()
	}
	m.Reset()
}

// Reset clears all in-memory state across all registered services.
func (m *MockServer) Reset() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, svc := range m.services {
		svc.Reset()
	}
}

// ServeHTTP routes incoming requests to the appropriate service handler.
// It determines the target service by inspecting the Authorization header's
// credential scope (e.g., ".../s3/aws4_request").
func (m *MockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serviceName := m.identifyService(r)

	m.mu.RLock()
	svc, ok := m.services[serviceName]
	m.mu.RUnlock()

	if !ok {
		http.Error(w, "unknown service: "+serviceName, http.StatusBadRequest)
		return
	}

	svc.Handler().ServeHTTP(w, r)
}

// identifyService extracts the AWS service name from the request.
// It checks (in order):
//  1. The Authorization header credential scope
//  2. The X-Amz-Target header prefix
//  3. Falls back to "s3" for unsigned requests (S3 presigned URLs, etc.)
func (m *MockServer) identifyService(r *http.Request) string {
	// Try Authorization header: AWS4-HMAC-SHA256 Credential=.../region/SERVICE/aws4_request
	if auth := r.Header.Get("Authorization"); auth != "" {
		if idx := strings.Index(auth, "Credential="); idx >= 0 {
			parts := strings.Split(auth[idx:], "/")
			if len(parts) >= 4 {
				return parts[3]
			}
		}
	}

	// Try X-Amz-Target header for JSON protocol services (e.g., DynamoDB).
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		parts := strings.SplitN(target, ".", 2)
		if len(parts) >= 1 {
			name := strings.ToLower(parts[0])
			// Map known target prefixes to service names.
			switch {
			case strings.Contains(name, "dynamodb"):
				return "dynamodb"
			case strings.Contains(name, "kinesis"):
				return "kinesis"
			case strings.Contains(name, "secretsmanager"):
				return "secretsmanager"
			}
		}
	}

	// Default to s3 for requests without auth (e.g., presigned URLs).
	return "s3"
}

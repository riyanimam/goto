// Package lambda provides a mock implementation of AWS Lambda.
//
// Supported actions:
//   - CreateFunction
//   - GetFunction
//   - DeleteFunction
//   - ListFunctions
//   - Invoke
//   - UpdateFunctionCode
//   - UpdateFunctionConfiguration
package lambda

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultAccountID = "123456789012"

// Service implements the Lambda mock.
type Service struct {
	mu        sync.RWMutex
	functions map[string]*function // keyed by function name
}

type function struct {
	name         string
	arn          string
	runtime      string
	role         string
	handler      string
	description  string
	timeout      int
	memorySize   int
	codeSize     int64
	codeSHA256   string
	version      string
	lastModified string
	environment  map[string]string
}

// New creates a new Lambda mock service.
func New() *Service {
	return &Service{
		functions: make(map[string]*function),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "lambda" }

// Handler returns the HTTP handler for Lambda requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all functions.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.functions = make(map[string]*function)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case strings.HasSuffix(path, "/functions") && r.Method == http.MethodGet:
		s.listFunctions(w, r)
	case strings.HasSuffix(path, "/functions") && r.Method == http.MethodPost:
		s.createFunction(w, r)
	case strings.Contains(path, "/functions/") && strings.HasSuffix(path, "/invocations"):
		name := extractFunctionName(path, "/invocations")
		s.invoke(w, r, name)
	case strings.Contains(path, "/functions/") && strings.HasSuffix(path, "/code"):
		name := extractFunctionName(path, "/code")
		s.updateFunctionCode(w, r, name)
	case strings.Contains(path, "/functions/") && strings.HasSuffix(path, "/configuration") && r.Method == http.MethodPut:
		name := extractFunctionName(path, "/configuration")
		s.updateFunctionConfiguration(w, r, name)
	case strings.Contains(path, "/functions/") && r.Method == http.MethodGet:
		name := extractLastSegment(path)
		s.getFunction(w, r, name)
	case strings.Contains(path, "/functions/") && r.Method == http.MethodDelete:
		name := extractLastSegment(path)
		s.deleteFunction(w, r, name)
	default:
		writeJSONError(w, "InvalidAction", "unsupported operation", http.StatusBadRequest)
	}
}

func extractFunctionName(path, suffix string) string {
	path = strings.TrimSuffix(path, suffix)
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func extractLastSegment(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func (s *Service) createFunction(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, "ServiceException", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &params); err != nil {
		writeJSONError(w, "InvalidParameterValueException", "could not parse request body", http.StatusBadRequest)
		return
	}

	name := getString(params, "FunctionName")
	if name == "" {
		writeJSONError(w, "InvalidParameterValueException", "FunctionName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.functions[name]; exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceConflictException", "Function already exist: "+name, http.StatusConflict)
		return
	}

	fn := &function{
		name:         name,
		arn:          fmt.Sprintf("arn:aws:lambda:us-east-1:%s:function:%s", defaultAccountID, name),
		runtime:      getString(params, "Runtime"),
		role:         getString(params, "Role"),
		handler:      getString(params, "Handler"),
		description:  getString(params, "Description"),
		timeout:      getInt(params, "Timeout", 3),
		memorySize:   getInt(params, "MemorySize", 128),
		codeSize:     1024,
		codeSHA256:   "abc123def456",
		version:      "$LATEST",
		lastModified: time.Now().UTC().Format(time.RFC3339),
	}

	if env, ok := params["Environment"].(map[string]interface{}); ok {
		if vars, ok := env["Variables"].(map[string]interface{}); ok {
			fn.environment = make(map[string]string)
			for k, v := range vars {
				if sv, ok := v.(string); ok {
					fn.environment[k] = sv
				}
			}
		}
	}

	s.functions[name] = fn
	s.mu.Unlock()

	writeJSON(w, http.StatusCreated, s.functionConfig(fn))
}

func (s *Service) getFunction(w http.ResponseWriter, _ *http.Request, name string) {
	s.mu.RLock()
	fn, exists := s.functions[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Function not found: arn:aws:lambda:us-east-1:"+defaultAccountID+":function:"+name, http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Configuration": s.functionConfig(fn),
		"Code": map[string]interface{}{
			"RepositoryType": "S3",
			"Location":       "https://awslambda-us-east-1-tasks.s3.us-east-1.amazonaws.com/...",
		},
	})
}

func (s *Service) deleteFunction(w http.ResponseWriter, _ *http.Request, name string) {
	s.mu.Lock()
	if _, exists := s.functions[name]; !exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "Function not found: arn:aws:lambda:us-east-1:"+defaultAccountID+":function:"+name, http.StatusNotFound)
		return
	}
	delete(s.functions, name)
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) listFunctions(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var fns []map[string]interface{}
	for _, fn := range s.functions {
		fns = append(fns, s.functionConfig(fn))
	}
	s.mu.RUnlock()

	sort.Slice(fns, func(i, j int) bool {
		return fns[i]["FunctionName"].(string) < fns[j]["FunctionName"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Functions": fns,
	})
}

func (s *Service) invoke(w http.ResponseWriter, r *http.Request, name string) {
	s.mu.RLock()
	_, exists := s.functions[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Function not found: arn:aws:lambda:us-east-1:"+defaultAccountID+":function:"+name, http.StatusNotFound)
		return
	}

	// Read the payload.
	payload, _ := io.ReadAll(r.Body)
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	// Return the payload as the response (echo function behavior).
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amz-Executed-Version", "$LATEST")
	w.Header().Set("X-Amz-Function-Error", "")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}

func (s *Service) updateFunctionCode(w http.ResponseWriter, r *http.Request, name string) {
	s.mu.Lock()
	fn, exists := s.functions[name]
	if !exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "Function not found: "+name, http.StatusNotFound)
		return
	}
	fn.lastModified = time.Now().UTC().Format(time.RFC3339)
	fn.codeSHA256 = "updated-sha256"
	config := s.functionConfig(fn)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, config)
}

func (s *Service) updateFunctionConfiguration(w http.ResponseWriter, r *http.Request, name string) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &params)
	}

	s.mu.Lock()
	fn, exists := s.functions[name]
	if !exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "Function not found: "+name, http.StatusNotFound)
		return
	}

	if v := getString(params, "Description"); v != "" {
		fn.description = v
	}
	if v := getString(params, "Handler"); v != "" {
		fn.handler = v
	}
	if v := getString(params, "Runtime"); v != "" {
		fn.runtime = v
	}
	if v := getInt(params, "Timeout", 0); v > 0 {
		fn.timeout = v
	}
	if v := getInt(params, "MemorySize", 0); v > 0 {
		fn.memorySize = v
	}
	fn.lastModified = time.Now().UTC().Format(time.RFC3339)
	config := s.functionConfig(fn)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, config)
}

func (s *Service) functionConfig(fn *function) map[string]interface{} {
	cfg := map[string]interface{}{
		"FunctionName":     fn.name,
		"FunctionArn":      fn.arn,
		"Runtime":          fn.runtime,
		"Role":             fn.role,
		"Handler":          fn.handler,
		"Description":      fn.description,
		"Timeout":          fn.timeout,
		"MemorySize":       fn.memorySize,
		"CodeSize":         fn.codeSize,
		"CodeSha256":       fn.codeSHA256,
		"Version":          fn.version,
		"LastModified":     fn.lastModified,
		"State":            "Active",
		"LastUpdateStatus": "Successful",
	}
	if fn.environment != nil {
		cfg["Environment"] = map[string]interface{}{
			"Variables": fn.environment,
		}
	}
	return cfg
}

// Helper functions.

func getString(params map[string]interface{}, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(params map[string]interface{}, key string, defaultVal int) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"Type":    code,
		"Message": message,
	})
}

func newRequestID() string {
	const chars = "abcdef0123456789"
	b := make([]byte, 36)
	sections := []int{8, 4, 4, 4, 12}
	pos := 0
	for i, l := range sections {
		if i > 0 {
			b[pos] = '-'
			pos++
		}
		for j := 0; j < l; j++ {
			b[pos] = chars[rand.Intn(len(chars))]
			pos++
		}
	}
	return string(b[:pos])
}

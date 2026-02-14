// Package ssm provides a mock implementation of AWS Systems Manager Parameter Store.
//
// Supported actions:
//   - PutParameter
//   - GetParameter
//   - GetParameters
//   - DeleteParameter
//   - DescribeParameters
//   - GetParametersByPath
package ssm

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

// Service implements the SSM Parameter Store mock.
type Service struct {
	mu     sync.RWMutex
	params map[string]*parameter // keyed by name
}

type parameter struct {
	name         string
	paramType    string // String, StringList, SecureString
	value        string
	description  string
	version      int64
	lastModified time.Time
	arn          string
}

// New creates a new SSM mock service.
func New() *Service {
	return &Service{
		params: make(map[string]*parameter),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "ssm" }

// Handler returns the HTTP handler for SSM requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all parameters.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.params = make(map[string]*parameter)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, "InternalServerError", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			writeJSONError(w, "ValidationException", "could not parse request body", http.StatusBadRequest)
			return
		}
	}
	if params == nil {
		params = make(map[string]interface{})
	}

	action := ""
	if target != "" {
		parts := strings.SplitN(target, ".", 2)
		if len(parts) == 2 {
			action = parts[1]
		}
	}

	switch action {
	case "PutParameter":
		s.putParameter(w, params)
	case "GetParameter":
		s.getParameter(w, params)
	case "GetParameters":
		s.getParameters(w, params)
	case "DeleteParameter":
		s.deleteParameter(w, params)
	case "DescribeParameters":
		s.describeParameters(w, params)
	case "GetParametersByPath":
		s.getParametersByPath(w, params)
	default:
		writeJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) putParameter(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "Name")
	if name == "" {
		writeJSONError(w, "ValidationException", "Name is required", http.StatusBadRequest)
		return
	}

	value := getString(params, "Value")
	paramType := getString(params, "Type")
	if paramType == "" {
		paramType = "String"
	}
	description := getString(params, "Description")
	overwrite := getBool(params, "Overwrite")

	s.mu.Lock()
	existing, exists := s.params[name]
	if exists && !overwrite {
		s.mu.Unlock()
		writeJSONError(w, "ParameterAlreadyExists", "The parameter already exists.", http.StatusBadRequest)
		return
	}

	var version int64 = 1
	if exists {
		version = existing.version + 1
	}

	s.params[name] = &parameter{
		name:         name,
		paramType:    paramType,
		value:        value,
		description:  description,
		version:      version,
		lastModified: time.Now().UTC(),
		arn:          fmt.Sprintf("arn:aws:ssm:us-east-1:%s:parameter%s", defaultAccountID, name),
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Version": version,
		"Tier":    "Standard",
	})
}

func (s *Service) getParameter(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "Name")

	s.mu.RLock()
	p, exists := s.params[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ParameterNotFound", "Parameter "+name+" not found.", http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Parameter": parameterResponse(p),
	})
}

func (s *Service) getParameters(w http.ResponseWriter, params map[string]interface{}) {
	names, _ := params["Names"].([]interface{})

	s.mu.RLock()
	var found []map[string]interface{}
	var invalid []string
	for _, n := range names {
		name, ok := n.(string)
		if !ok {
			continue
		}
		if p, exists := s.params[name]; exists {
			found = append(found, parameterResponse(p))
		} else {
			invalid = append(invalid, name)
		}
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Parameters":        found,
		"InvalidParameters": invalid,
	})
}

func (s *Service) deleteParameter(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "Name")

	s.mu.Lock()
	if _, exists := s.params[name]; !exists {
		s.mu.Unlock()
		writeJSONError(w, "ParameterNotFound", "Parameter "+name+" not found.", http.StatusBadRequest)
		return
	}
	delete(s.params, name)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) describeParameters(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var paramList []map[string]interface{}
	for _, p := range s.params {
		paramList = append(paramList, map[string]interface{}{
			"Name":             p.name,
			"Type":             p.paramType,
			"Description":      p.description,
			"Version":          p.version,
			"LastModifiedDate": float64(p.lastModified.Unix()),
			"Tier":             "Standard",
		})
	}
	s.mu.RUnlock()

	sort.Slice(paramList, func(i, j int) bool {
		return paramList[i]["Name"].(string) < paramList[j]["Name"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Parameters": paramList,
	})
}

func (s *Service) getParametersByPath(w http.ResponseWriter, params map[string]interface{}) {
	path := getString(params, "Path")
	recursive := getBool(params, "Recursive")

	s.mu.RLock()
	var found []map[string]interface{}
	for _, p := range s.params {
		if recursive {
			if strings.HasPrefix(p.name, path) {
				found = append(found, parameterResponse(p))
			}
		} else {
			// Non-recursive: only direct children.
			if strings.HasPrefix(p.name, path) {
				rest := strings.TrimPrefix(p.name, path)
				rest = strings.TrimPrefix(rest, "/")
				if !strings.Contains(rest, "/") {
					found = append(found, parameterResponse(p))
				}
			}
		}
	}
	s.mu.RUnlock()

	sort.Slice(found, func(i, j int) bool {
		return found[i]["Name"].(string) < found[j]["Name"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Parameters": found,
	})
}

func parameterResponse(p *parameter) map[string]interface{} {
	return map[string]interface{}{
		"Name":             p.name,
		"Type":             p.paramType,
		"Value":            p.value,
		"Version":          p.version,
		"LastModifiedDate": float64(p.lastModified.Unix()),
		"ARN":              p.arn,
	}
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

func getBool(params map[string]interface{}, key string) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"__type":  code,
		"message": message,
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

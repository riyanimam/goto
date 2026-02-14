// Package secretsmanager provides a mock implementation of AWS Secrets Manager.
//
// Supported actions:
//   - CreateSecret
//   - GetSecretValue
//   - PutSecretValue
//   - DeleteSecret
//   - ListSecrets
//   - DescribeSecret
//   - UpdateSecret
package secretsmanager

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

// Service implements the Secrets Manager mock.
type Service struct {
	mu      sync.RWMutex
	secrets map[string]*secret // keyed by name
}

type secret struct {
	name         string
	arn          string
	description  string
	secretString string
	secretBinary []byte
	versionID    string
	created      time.Time
	lastChanged  time.Time
	deleted      bool
}

// New creates a new Secrets Manager mock service.
func New() *Service {
	return &Service{
		secrets: make(map[string]*secret),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "secretsmanager" }

// Handler returns the HTTP handler for Secrets Manager requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all secrets.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets = make(map[string]*secret)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, "InternalServiceError", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			writeJSONError(w, "InvalidRequestException", "could not parse request body", http.StatusBadRequest)
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
	case "CreateSecret":
		s.createSecret(w, params)
	case "GetSecretValue":
		s.getSecretValue(w, params)
	case "PutSecretValue":
		s.putSecretValue(w, params)
	case "DeleteSecret":
		s.deleteSecret(w, params)
	case "ListSecrets":
		s.listSecrets(w, params)
	case "DescribeSecret":
		s.describeSecret(w, params)
	case "UpdateSecret":
		s.updateSecret(w, params)
	default:
		writeJSONError(w, "InvalidAction", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createSecret(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "Name")
	if name == "" {
		writeJSONError(w, "InvalidParameterException", "Name is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.secrets[name]; exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceExistsException", "The operation failed because the secret "+name+" already exists.", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	versionID := newRequestID()
	sec := &secret{
		name:        name,
		arn:         fmt.Sprintf("arn:aws:secretsmanager:us-east-1:%s:secret:%s-%s", defaultAccountID, name, randomSuffix()),
		description: getString(params, "Description"),
		versionID:   versionID,
		created:     now,
		lastChanged: now,
	}

	if v := getString(params, "SecretString"); v != "" {
		sec.secretString = v
	}

	s.secrets[name] = sec
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ARN":       sec.arn,
		"Name":      sec.name,
		"VersionId": sec.versionID,
	})
}

func (s *Service) getSecretValue(w http.ResponseWriter, params map[string]interface{}) {
	secretID := getString(params, "SecretId")

	s.mu.RLock()
	sec := s.findSecret(secretID)
	s.mu.RUnlock()

	if sec == nil || sec.deleted {
		writeJSONError(w, "ResourceNotFoundException", "Secrets Manager can't find the specified secret.", http.StatusBadRequest)
		return
	}

	resp := map[string]interface{}{
		"ARN":            sec.arn,
		"Name":           sec.name,
		"VersionId":      sec.versionID,
		"CreatedDate":    float64(sec.created.Unix()),
		"VersionStages":  []string{"AWSCURRENT"},
	}
	if sec.secretString != "" {
		resp["SecretString"] = sec.secretString
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) putSecretValue(w http.ResponseWriter, params map[string]interface{}) {
	secretID := getString(params, "SecretId")

	s.mu.Lock()
	sec := s.findSecret(secretID)
	if sec == nil {
		s.mu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "Secrets Manager can't find the specified secret.", http.StatusBadRequest)
		return
	}

	versionID := newRequestID()
	if v := getString(params, "SecretString"); v != "" {
		sec.secretString = v
	}
	sec.versionID = versionID
	sec.lastChanged = time.Now().UTC()
	arn := sec.arn
	name := sec.name
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ARN":           arn,
		"Name":          name,
		"VersionId":     versionID,
		"VersionStages": []string{"AWSCURRENT"},
	})
}

func (s *Service) deleteSecret(w http.ResponseWriter, params map[string]interface{}) {
	secretID := getString(params, "SecretId")

	s.mu.Lock()
	sec := s.findSecret(secretID)
	if sec == nil {
		s.mu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "Secrets Manager can't find the specified secret.", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	sec.deleted = true
	arn := sec.arn
	name := sec.name
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ARN":          arn,
		"Name":         name,
		"DeletionDate": float64(now.Unix()),
	})
}

func (s *Service) listSecrets(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var secretList []map[string]interface{}
	for _, sec := range s.secrets {
		if sec.deleted {
			continue
		}
		secretList = append(secretList, map[string]interface{}{
			"ARN":              sec.arn,
			"Name":             sec.name,
			"Description":      sec.description,
			"CreatedDate":      float64(sec.created.Unix()),
			"LastChangedDate":  float64(sec.lastChanged.Unix()),
		})
	}
	s.mu.RUnlock()

	sort.Slice(secretList, func(i, j int) bool {
		return secretList[i]["Name"].(string) < secretList[j]["Name"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"SecretList": secretList,
	})
}

func (s *Service) describeSecret(w http.ResponseWriter, params map[string]interface{}) {
	secretID := getString(params, "SecretId")

	s.mu.RLock()
	sec := s.findSecret(secretID)
	s.mu.RUnlock()

	if sec == nil {
		writeJSONError(w, "ResourceNotFoundException", "Secrets Manager can't find the specified secret.", http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ARN":             sec.arn,
		"Name":            sec.name,
		"Description":     sec.description,
		"CreatedDate":     float64(sec.created.Unix()),
		"LastChangedDate": float64(sec.lastChanged.Unix()),
		"VersionIdsToStages": map[string][]string{
			sec.versionID: {"AWSCURRENT"},
		},
	})
}

func (s *Service) updateSecret(w http.ResponseWriter, params map[string]interface{}) {
	secretID := getString(params, "SecretId")

	s.mu.Lock()
	sec := s.findSecret(secretID)
	if sec == nil {
		s.mu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "Secrets Manager can't find the specified secret.", http.StatusBadRequest)
		return
	}

	if v := getString(params, "Description"); v != "" {
		sec.description = v
	}
	if v := getString(params, "SecretString"); v != "" {
		sec.secretString = v
		sec.versionID = newRequestID()
	}
	sec.lastChanged = time.Now().UTC()
	arn := sec.arn
	name := sec.name
	versionID := sec.versionID
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ARN":       arn,
		"Name":      name,
		"VersionId": versionID,
	})
}

// findSecret looks up a secret by name or ARN. Caller must hold s.mu.
func (s *Service) findSecret(secretID string) *secret {
	// Try direct name lookup.
	if sec, ok := s.secrets[secretID]; ok {
		return sec
	}
	// Try ARN lookup.
	for _, sec := range s.secrets {
		if sec.arn == secretID {
			return sec
		}
	}
	return nil
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

func randomSuffix() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

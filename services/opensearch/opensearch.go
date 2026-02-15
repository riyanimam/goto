// Package opensearch provides a mock implementation of Amazon OpenSearch Service.
//
// Supported actions:
//   - CreateDomain
//   - DescribeDomain
//   - DeleteDomain
//   - ListDomainNames
//   - UpdateDomainConfig
package opensearch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the OpenSearch mock.
type Service struct {
	mu      sync.RWMutex
	domains map[string]*domain
}

type domain struct {
	name          string
	arn           string
	domainID      string
	engineVersion string
	endpoint      string
	clusterConfig interface{}
	processing    bool
	created       time.Time
}

// New creates a new OpenSearch mock service.
func New() *Service {
	return &Service{
		domains: make(map[string]*domain),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "es" }

// Handler returns the HTTP handler for OpenSearch requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.domains = make(map[string]*domain)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	switch {
	// UpdateDomainConfig: POST /2021-01-01/opensearch/domain/{name}/config
	case strings.HasSuffix(path, "/config") && strings.Contains(path, "/2021-01-01/opensearch/domain/") && method == http.MethodPost:
		s.updateDomainConfig(w, r, path)

	// DescribeDomain: GET /2021-01-01/opensearch/domain/{name}
	case strings.HasPrefix(path, "/2021-01-01/opensearch/domain/") && method == http.MethodGet:
		s.describeDomain(w, r, path)

	// DeleteDomain: DELETE /2021-01-01/opensearch/domain/{name}
	case strings.HasPrefix(path, "/2021-01-01/opensearch/domain/") && method == http.MethodDelete:
		s.deleteDomain(w, r, path)

	// CreateDomain: POST /2021-01-01/opensearch/domain
	case path == "/2021-01-01/opensearch/domain" && method == http.MethodPost:
		s.createDomain(w, r)

	// ListDomainNames: GET /2021-01-01/domain
	case path == "/2021-01-01/domain" && method == http.MethodGet:
		s.listDomainNames(w, r)

	default:
		h.WriteJSONError(w, "ResourceNotFoundException", "unsupported operation", http.StatusNotFound)
	}
}

func extractDomainName(path string) string {
	const prefix = "/2021-01-01/opensearch/domain/"
	rest := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(rest, "/"); idx >= 0 {
		return rest[:idx]
	}
	return rest
}

func (s *Service) createDomain(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	name := h.GetString(params, "DomainName")
	if name == "" {
		h.WriteJSONError(w, "ValidationException", "DomainName is required", http.StatusBadRequest)
		return
	}

	engineVersion := h.GetString(params, "EngineVersion")
	if engineVersion == "" {
		engineVersion = "OpenSearch_2.11"
	}

	var clusterConfig interface{}
	if cc, ok := params["ClusterConfig"]; ok {
		clusterConfig = cc
	}

	s.mu.Lock()
	if _, exists := s.domains[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceAlreadyExistsException", "Domain "+name+" already exists", http.StatusConflict)
		return
	}

	domainID := h.RandomHex(12)
	arn := fmt.Sprintf("arn:aws:es:us-east-1:%s:domain/%s", h.DefaultAccountID, name)
	endpoint := fmt.Sprintf("search-%s-%s.us-east-1.es.amazonaws.com", name, h.RandomHex(28))

	d := &domain{
		name:          name,
		arn:           arn,
		domainID:      domainID,
		engineVersion: engineVersion,
		endpoint:      endpoint,
		clusterConfig: clusterConfig,
		processing:    false,
		created:       time.Now().UTC(),
	}
	s.domains[name] = d
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"DomainStatus": domainResp(d),
	})
}

func (s *Service) describeDomain(w http.ResponseWriter, _ *http.Request, path string) {
	name := extractDomainName(path)

	s.mu.RLock()
	d, exists := s.domains[name]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Domain "+name+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"DomainStatus": domainResp(d),
	})
}

func (s *Service) deleteDomain(w http.ResponseWriter, _ *http.Request, path string) {
	name := extractDomainName(path)

	s.mu.Lock()
	d, exists := s.domains[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Domain "+name+" not found", http.StatusNotFound)
		return
	}
	resp := domainResp(d)
	delete(s.domains, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"DomainStatus": resp,
	})
}

func (s *Service) listDomainNames(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var names []string
	for name := range s.domains {
		names = append(names, name)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	domainNames := make([]map[string]interface{}, len(names))
	for i, name := range names {
		domainNames[i] = map[string]interface{}{
			"DomainName": name,
			"EngineType": "OpenSearch",
		}
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"DomainNames": domainNames,
	})
}

func (s *Service) updateDomainConfig(w http.ResponseWriter, r *http.Request, path string) {
	name := extractDomainName(path)

	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	s.mu.Lock()
	d, exists := s.domains[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Domain "+name+" not found", http.StatusNotFound)
		return
	}

	if v := h.GetString(params, "EngineVersion"); v != "" {
		d.engineVersion = v
	}
	if cc, ok := params["ClusterConfig"]; ok {
		d.clusterConfig = cc
	}
	d.processing = true
	resp := domainResp(d)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"DomainStatus": resp,
	})
}

func domainResp(d *domain) map[string]interface{} {
	resp := map[string]interface{}{
		"DomainName":    d.name,
		"ARN":           d.arn,
		"DomainId":      d.domainID,
		"EngineVersion": d.engineVersion,
		"Endpoint":      d.endpoint,
		"Processing":    d.processing,
		"Created":       true,
		"CreatedAt":     float64(d.created.Unix()),
	}
	if d.clusterConfig != nil {
		resp["ClusterConfig"] = d.clusterConfig
	}
	return resp
}

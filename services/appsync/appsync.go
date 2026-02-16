// Package appsync provides a mock implementation of AWS AppSync.
//
// Supported actions:
//   - CreateGraphqlApi
//   - GetGraphqlApi
//   - DeleteGraphqlApi
//   - ListGraphqlApis
//   - CreateDataSource
//   - GetDataSource
//   - DeleteDataSource
package appsync

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the AppSync mock.
type Service struct {
	mu   sync.RWMutex
	apis map[string]*graphqlAPI
}

type graphqlAPI struct {
	apiID              string
	name               string
	arn                string
	authenticationType string
	logConfig          interface{}
	tags               map[string]interface{}
	created            time.Time
	dataSources        map[string]*dataSource
}

type dataSource struct {
	name           string
	dataSourceArn  string
	dsType         string
	serviceRoleArn string
}

// New creates a new AppSync mock service.
func New() *Service {
	return &Service{
		apis: make(map[string]*graphqlAPI),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "appsync" }

// Handler returns the HTTP handler for AppSync requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apis = make(map[string]*graphqlAPI)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	switch {
	// DataSource by name: /v1/apis/{apiId}/datasources/{name}
	case strings.Contains(path, "/datasources/") && method == http.MethodGet:
		s.getDataSource(w, r, path)
	case strings.Contains(path, "/datasources/") && method == http.MethodDelete:
		s.deleteDataSource(w, r, path)

	// DataSources list: /v1/apis/{apiId}/datasources
	case strings.HasSuffix(path, "/datasources") && method == http.MethodPost:
		s.createDataSource(w, r, path)

	// Single API: /v1/apis/{apiId}
	case strings.HasPrefix(path, "/v1/apis/") && !strings.Contains(path, "/datasources") && method == http.MethodGet:
		s.getGraphqlAPI(w, r, path)
	case strings.HasPrefix(path, "/v1/apis/") && !strings.Contains(path, "/datasources") && method == http.MethodDelete:
		s.deleteGraphqlAPI(w, r, path)

	// APIs list: /v1/apis
	case path == "/v1/apis" && method == http.MethodPost:
		s.createGraphqlAPI(w, r)
	case path == "/v1/apis" && method == http.MethodGet:
		s.listGraphqlApis(w, r)

	default:
		h.WriteJSONError(w, "NotFoundException", "unsupported operation", http.StatusNotFound)
	}
}

func extractAPIID(path string) string {
	// path: /v1/apis/{apiId}...
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func (s *Service) createGraphqlAPI(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	name := h.GetString(params, "name")
	if name == "" {
		h.WriteJSONError(w, "BadRequestException", "name is required", http.StatusBadRequest)
		return
	}

	authType := h.GetString(params, "authenticationType")
	if authType == "" {
		authType = "API_KEY"
	}

	apiID := h.RandomHex(8)
	arn := fmt.Sprintf("arn:aws:appsync:us-east-1:%s:apis/%s", h.DefaultAccountID, apiID)

	var tags map[string]interface{}
	if t, ok := params["tags"].(map[string]interface{}); ok {
		tags = t
	}

	s.mu.Lock()
	api := &graphqlAPI{
		apiID:              apiID,
		name:               name,
		arn:                arn,
		authenticationType: authType,
		logConfig:          params["logConfig"],
		tags:               tags,
		created:            time.Now().UTC(),
		dataSources:        make(map[string]*dataSource),
	}
	s.apis[apiID] = api
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"graphqlApi": apiResp(api),
	})
}

func (s *Service) getGraphqlAPI(w http.ResponseWriter, _ *http.Request, path string) {
	apiID := extractAPIID(path)

	s.mu.RLock()
	api, exists := s.apis[apiID]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "NotFoundException", "GraphQL API "+apiID+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"graphqlApi": apiResp(api),
	})
}

func (s *Service) deleteGraphqlAPI(w http.ResponseWriter, _ *http.Request, path string) {
	apiID := extractAPIID(path)

	s.mu.Lock()
	_, exists := s.apis[apiID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "GraphQL API "+apiID+" not found", http.StatusNotFound)
		return
	}
	delete(s.apis, apiID)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listGraphqlApis(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var apis []map[string]interface{}
	for _, api := range s.apis {
		apis = append(apis, apiResp(api))
	}
	s.mu.RUnlock()

	if apis == nil {
		apis = []map[string]interface{}{}
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"graphqlApis": apis,
	})
}

func (s *Service) createDataSource(w http.ResponseWriter, r *http.Request, path string) {
	apiID := extractAPIID(path)

	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	name := h.GetString(params, "name")
	if name == "" {
		h.WriteJSONError(w, "BadRequestException", "name is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "GraphQL API "+apiID+" not found", http.StatusNotFound)
		return
	}

	dsArn := fmt.Sprintf("arn:aws:appsync:us-east-1:%s:apis/%s/datasources/%s",
		h.DefaultAccountID, apiID, name)

	ds := &dataSource{
		name:           name,
		dataSourceArn:  dsArn,
		dsType:         h.GetString(params, "type"),
		serviceRoleArn: h.GetString(params, "serviceRoleArn"),
	}
	api.dataSources[name] = ds
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"dataSource": dataSourceResp(ds),
	})
}

func (s *Service) getDataSource(w http.ResponseWriter, _ *http.Request, path string) {
	apiID := extractAPIID(path)
	// path: /v1/apis/{apiId}/datasources/{name}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 5 {
		h.WriteJSONError(w, "NotFoundException", "invalid path", http.StatusNotFound)
		return
	}
	dsName := parts[4]

	s.mu.RLock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "NotFoundException", "GraphQL API "+apiID+" not found", http.StatusNotFound)
		return
	}
	ds, exists := api.dataSources[dsName]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "NotFoundException", "DataSource "+dsName+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"dataSource": dataSourceResp(ds),
	})
}

func (s *Service) deleteDataSource(w http.ResponseWriter, _ *http.Request, path string) {
	apiID := extractAPIID(path)
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 5 {
		h.WriteJSONError(w, "NotFoundException", "invalid path", http.StatusNotFound)
		return
	}
	dsName := parts[4]

	s.mu.Lock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "GraphQL API "+apiID+" not found", http.StatusNotFound)
		return
	}
	_, exists = api.dataSources[dsName]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "DataSource "+dsName+" not found", http.StatusNotFound)
		return
	}
	delete(api.dataSources, dsName)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func apiResp(api *graphqlAPI) map[string]interface{} {
	resp := map[string]interface{}{
		"apiId":              api.apiID,
		"name":               api.name,
		"arn":                api.arn,
		"authenticationType": api.authenticationType,
		"uris": map[string]interface{}{
			"GRAPHQL": fmt.Sprintf("https://%s.appsync-api.us-east-1.amazonaws.com/graphql", api.apiID),
		},
	}
	if api.logConfig != nil {
		resp["logConfig"] = api.logConfig
	}
	if api.tags != nil {
		resp["tags"] = api.tags
	}
	return resp
}

func dataSourceResp(ds *dataSource) map[string]interface{} {
	return map[string]interface{}{
		"dataSourceArn":  ds.dataSourceArn,
		"name":           ds.name,
		"type":           ds.dsType,
		"serviceRoleArn": ds.serviceRoleArn,
	}
}

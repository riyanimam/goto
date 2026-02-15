// Package apigateway provides a mock implementation of AWS API Gateway v1 (REST APIs).
//
// Supported actions:
//   - CreateRestApi
//   - GetRestApi
//   - DeleteRestApi
//   - GetRestApis
//   - CreateResource
//   - GetResources
//   - PutMethod
//   - PutIntegration
package apigateway

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the API Gateway v1 (REST APIs) mock.
type Service struct {
	mu   sync.RWMutex
	apis map[string]*restApi
}

type restApi struct {
	id          string
	name        string
	description string
	createdDate time.Time
	resources   map[string]*resource
}

type resource struct {
	id       string
	parentId string
	pathPart string
	path     string
	methods  map[string]*method
}

type method struct {
	httpMethod        string
	authorizationType string
	integration       *integration
}

type integration struct {
	integrationType string
	uri             string
	httpMethod      string
}

// New creates a new API Gateway v1 mock service.
func New() *Service {
	return &Service{
		apis: make(map[string]*restApi),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "apigateway" }

// Handler returns the HTTP handler for API Gateway v1 requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apis = make(map[string]*restApi)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	switch {
	// PutIntegration: PUT /restapis/{id}/resources/{rid}/methods/{httpMethod}/integration
	case strings.HasSuffix(path, "/integration") && method == http.MethodPut:
		s.putIntegration(w, r, path)

	// PutMethod: PUT /restapis/{id}/resources/{rid}/methods/{httpMethod}
	case strings.Contains(path, "/methods/") && method == http.MethodPut:
		s.putMethod(w, r, path)

	// CreateResource: POST /restapis/{id}/resources/{parentId}
	case strings.Contains(path, "/resources/") && !strings.Contains(path, "/methods/") && method == http.MethodPost:
		s.createResource(w, r, path)

	// GetResources: GET /restapis/{id}/resources
	case strings.HasSuffix(path, "/resources") && method == http.MethodGet:
		s.getResources(w, path)

	// GetRestApi: GET /restapis/{id}
	case strings.Count(path, "/") == 2 && strings.HasPrefix(path, "/restapis/") && method == http.MethodGet:
		s.getRestApi(w, path)

	// DeleteRestApi: DELETE /restapis/{id}
	case strings.Count(path, "/") == 2 && strings.HasPrefix(path, "/restapis/") && method == http.MethodDelete:
		s.deleteRestApi(w, path)

	// CreateRestApi: POST /restapis
	case path == "/restapis" && method == http.MethodPost:
		s.createRestApi(w, r)

	// GetRestApis: GET /restapis
	case path == "/restapis" && method == http.MethodGet:
		s.getRestApis(w)

	default:
		h.WriteJSONError(w, "NotFoundException", "unsupported operation", http.StatusNotFound)
	}
}

func extractRestApiID(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func (s *Service) createRestApi(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	name := h.GetString(params, "name")
	if name == "" {
		h.WriteJSONError(w, "BadRequestException", "Name is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	apiID := h.RandomHex(10)
	rootID := h.RandomHex(10)
	api := &restApi{
		id:          apiID,
		name:        name,
		description: h.GetString(params, "description"),
		createdDate: time.Now().UTC(),
		resources: map[string]*resource{
			rootID: {
				id:       rootID,
				parentId: "",
				pathPart: "",
				path:     "/",
				methods:  make(map[string]*method),
			},
		},
	}
	s.apis[apiID] = api
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusCreated, restApiResp(api))
}

func (s *Service) getRestApi(w http.ResponseWriter, path string) {
	apiID := extractRestApiID(path)

	s.mu.RLock()
	api, exists := s.apis[apiID]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "NotFoundException", "REST API "+apiID+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, restApiResp(api))
}

func (s *Service) deleteRestApi(w http.ResponseWriter, path string) {
	apiID := extractRestApiID(path)

	s.mu.Lock()
	if _, exists := s.apis[apiID]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "REST API "+apiID+" not found", http.StatusNotFound)
		return
	}
	delete(s.apis, apiID)
	s.mu.Unlock()

	w.WriteHeader(http.StatusAccepted)
}

func (s *Service) getRestApis(w http.ResponseWriter) {
	s.mu.RLock()
	var items []map[string]interface{}
	for _, api := range s.apis {
		items = append(items, restApiResp(api))
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i]["name"].(string) < items[j]["name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"item": items,
	})
}

func (s *Service) createResource(w http.ResponseWriter, r *http.Request, path string) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 4 {
		h.WriteJSONError(w, "NotFoundException", "invalid path", http.StatusNotFound)
		return
	}
	apiID := parts[1]
	parentID := parts[3]

	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	pathPart := h.GetString(params, "pathPart")
	if pathPart == "" {
		h.WriteJSONError(w, "BadRequestException", "pathPart is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "REST API "+apiID+" not found", http.StatusNotFound)
		return
	}

	parent, exists := api.resources[parentID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "Resource "+parentID+" not found", http.StatusNotFound)
		return
	}

	resourceID := h.RandomHex(10)
	resourcePath := parent.path
	if resourcePath == "/" {
		resourcePath = "/" + pathPart
	} else {
		resourcePath = resourcePath + "/" + pathPart
	}

	res := &resource{
		id:       resourceID,
		parentId: parentID,
		pathPart: pathPart,
		path:     resourcePath,
		methods:  make(map[string]*method),
	}
	api.resources[resourceID] = res
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusCreated, resourceResp(res))
}

func (s *Service) getResources(w http.ResponseWriter, path string) {
	apiID := extractRestApiID(path)

	s.mu.RLock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "NotFoundException", "REST API "+apiID+" not found", http.StatusNotFound)
		return
	}

	var items []map[string]interface{}
	for _, res := range api.resources {
		items = append(items, resourceResp(res))
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i]["path"].(string) < items[j]["path"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"item": items,
	})
}

func (s *Service) putMethod(w http.ResponseWriter, r *http.Request, path string) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 6 {
		h.WriteJSONError(w, "NotFoundException", "invalid path", http.StatusNotFound)
		return
	}
	apiID := parts[1]
	resourceID := parts[3]
	httpMethod := parts[5]

	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	authType := h.GetString(params, "authorizationType")
	if authType == "" {
		authType = "NONE"
	}

	s.mu.Lock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "REST API "+apiID+" not found", http.StatusNotFound)
		return
	}

	res, exists := api.resources[resourceID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "Resource "+resourceID+" not found", http.StatusNotFound)
		return
	}

	m := &method{
		httpMethod:        httpMethod,
		authorizationType: authType,
	}
	res.methods[httpMethod] = m
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusCreated, methodResp(m))
}

func (s *Service) putIntegration(w http.ResponseWriter, r *http.Request, path string) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 7 {
		h.WriteJSONError(w, "NotFoundException", "invalid path", http.StatusNotFound)
		return
	}
	apiID := parts[1]
	resourceID := parts[3]
	httpMethod := parts[5]

	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	intType := h.GetString(params, "type")
	if intType == "" {
		h.WriteJSONError(w, "BadRequestException", "type is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "REST API "+apiID+" not found", http.StatusNotFound)
		return
	}

	res, exists := api.resources[resourceID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "Resource "+resourceID+" not found", http.StatusNotFound)
		return
	}

	m, exists := res.methods[httpMethod]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "Method "+httpMethod+" not found", http.StatusNotFound)
		return
	}

	intg := &integration{
		integrationType: intType,
		uri:             h.GetString(params, "uri"),
		httpMethod:      h.GetString(params, "httpMethod"),
	}
	m.integration = intg
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusCreated, integrationResp(intg))
}

func restApiResp(api *restApi) map[string]interface{} {
	return map[string]interface{}{
		"id":          api.id,
		"name":        api.name,
		"description": api.description,
		"createdDate": api.createdDate.Format(time.RFC3339),
	}
}

func resourceResp(res *resource) map[string]interface{} {
	resp := map[string]interface{}{
		"id":       res.id,
		"parentId": res.parentId,
		"pathPart": res.pathPart,
		"path":     res.path,
	}
	if len(res.methods) > 0 {
		methods := make(map[string]interface{})
		for k, m := range res.methods {
			methods[k] = methodResp(m)
		}
		resp["resourceMethods"] = methods
	}
	return resp
}

func methodResp(m *method) map[string]interface{} {
	resp := map[string]interface{}{
		"httpMethod":        m.httpMethod,
		"authorizationType": m.authorizationType,
	}
	if m.integration != nil {
		resp["methodIntegration"] = integrationResp(m.integration)
	}
	return resp
}

func integrationResp(intg *integration) map[string]interface{} {
	return map[string]interface{}{
		"type":       intg.integrationType,
		"uri":        intg.uri,
		"httpMethod": intg.httpMethod,
	}
}

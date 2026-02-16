// Package apigatewayv2 provides a mock implementation of AWS API Gateway V2 (HTTP/WebSocket APIs).
//
// Supported actions:
//   - CreateApi
//   - GetApi
//   - DeleteApi
//   - GetApis
//   - CreateStage
//   - GetStages
//   - DeleteStage
//   - CreateRoute
//   - GetRoutes
//   - DeleteRoute
package apigatewayv2

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

// Service implements the API Gateway V2 mock.
type Service struct {
	mu   sync.RWMutex
	apis map[string]*apiGw
}

type apiGw struct {
	apiID        string
	name         string
	protocolType string
	description  string
	endpoint     string
	created      time.Time
	stages       map[string]*stage
	routes       map[string]*route
}

type stage struct {
	stageName   string
	description string
	autoDeploy  bool
	created     time.Time
}

type route struct {
	routeID  string
	routeKey string
	target   string
}

// New creates a new API Gateway V2 mock service.
func New() *Service {
	return &Service{
		apis: make(map[string]*apiGw),
	}
}

// Name returns the service identifier. Both API Gateway V1 and V2 sign
// requests with the "apigateway" credential scope; we use "apigatewayv2" as
// the internal key so identifyService can disambiguate via the /v2/ URL prefix.
func (s *Service) Name() string { return "apigatewayv2" }

// Handler returns the HTTP handler for API Gateway V2 requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apis = make(map[string]*apiGw)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	switch {
	// Routes: /v2/apis/{apiId}/routes/{routeId}
	case strings.Contains(path, "/routes/") && method == http.MethodDelete:
		s.deleteRoute(w, r, path)

	// Routes: /v2/apis/{apiId}/routes
	case strings.HasSuffix(path, "/routes") && method == http.MethodPost:
		s.createRoute(w, r, path)
	case strings.HasSuffix(path, "/routes") && method == http.MethodGet:
		s.getRoutes(w, r, path)

	// Stages: /v2/apis/{apiId}/stages/{stageName}
	case strings.Contains(path, "/stages/") && method == http.MethodDelete:
		s.deleteStage(w, r, path)

	// Stages: /v2/apis/{apiId}/stages
	case strings.HasSuffix(path, "/stages") && method == http.MethodPost:
		s.createStage(w, r, path)
	case strings.HasSuffix(path, "/stages") && method == http.MethodGet:
		s.getStages(w, r, path)

	// APIs: /v2/apis/{apiId}
	case strings.Count(path, "/") == 3 && strings.HasPrefix(path, "/v2/apis/") && method == http.MethodGet:
		s.getApi(w, r, path)
	case strings.Count(path, "/") == 3 && strings.HasPrefix(path, "/v2/apis/") && method == http.MethodDelete:
		s.deleteApi(w, r, path)

	// APIs list: /v2/apis
	case path == "/v2/apis" && method == http.MethodPost:
		s.createApi(w, r)
	case path == "/v2/apis" && method == http.MethodGet:
		s.getApis(w, r)

	default:
		h.WriteJSONError(w, "NotFoundException", "unsupported operation", http.StatusNotFound)
	}
}

func extractAPIID(path string) string {
	// path: /v2/apis/{apiId}/...
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func (s *Service) createApi(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	name := h.GetString(params, "name")
	if name == "" {
		h.WriteJSONError(w, "BadRequestException", "Name is required", http.StatusBadRequest)
		return
	}
	protocolType := h.GetString(params, "protocolType")
	if protocolType == "" {
		protocolType = "HTTP"
	}

	s.mu.Lock()
	apiID := h.RandomHex(10)
	endpoint := "https://" + apiID + ".execute-api.us-east-1.amazonaws.com"
	api := &apiGw{
		apiID:        apiID,
		name:         name,
		protocolType: protocolType,
		description:  h.GetString(params, "description"),
		endpoint:     endpoint,
		created:      time.Now().UTC(),
		stages:       make(map[string]*stage),
		routes:       make(map[string]*route),
	}
	s.apis[apiID] = api
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusCreated, apiResp(api))
}

func (s *Service) getApi(w http.ResponseWriter, _ *http.Request, path string) {
	apiID := extractAPIID(path)

	s.mu.RLock()
	api, exists := s.apis[apiID]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "NotFoundException", "API "+apiID+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, apiResp(api))
}

func (s *Service) deleteApi(w http.ResponseWriter, _ *http.Request, path string) {
	apiID := extractAPIID(path)

	s.mu.Lock()
	if _, exists := s.apis[apiID]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "API "+apiID+" not found", http.StatusNotFound)
		return
	}
	delete(s.apis, apiID)
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) getApis(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var items []map[string]interface{}
	for _, api := range s.apis {
		items = append(items, apiResp(api))
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i]["name"].(string) < items[j]["name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func (s *Service) createStage(w http.ResponseWriter, r *http.Request, path string) {
	apiID := extractAPIID(path)
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	stageName := h.GetString(params, "stageName")
	if stageName == "" {
		h.WriteJSONError(w, "BadRequestException", "StageName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "API "+apiID+" not found", http.StatusNotFound)
		return
	}

	stg := &stage{
		stageName:   stageName,
		description: h.GetString(params, "description"),
		autoDeploy:  h.GetBool(params, "autoDeploy"),
		created:     time.Now().UTC(),
	}
	api.stages[stageName] = stg
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusCreated, stageResp(stg))
}

func (s *Service) getStages(w http.ResponseWriter, _ *http.Request, path string) {
	apiID := extractAPIID(path)

	s.mu.RLock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "NotFoundException", "API "+apiID+" not found", http.StatusNotFound)
		return
	}

	var items []map[string]interface{}
	for _, stg := range api.stages {
		items = append(items, stageResp(stg))
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i]["stageName"].(string) < items[j]["stageName"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func (s *Service) deleteStage(w http.ResponseWriter, _ *http.Request, path string) {
	apiID := extractAPIID(path)
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 5 {
		h.WriteJSONError(w, "NotFoundException", "invalid path", http.StatusNotFound)
		return
	}
	stageName := parts[4]

	s.mu.Lock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "API "+apiID+" not found", http.StatusNotFound)
		return
	}
	delete(api.stages, stageName)
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) createRoute(w http.ResponseWriter, r *http.Request, path string) {
	apiID := extractAPIID(path)
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	routeKey := h.GetString(params, "routeKey")
	if routeKey == "" {
		h.WriteJSONError(w, "BadRequestException", "RouteKey is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "API "+apiID+" not found", http.StatusNotFound)
		return
	}

	routeID := h.RandomHex(7)
	rt := &route{
		routeID:  routeID,
		routeKey: routeKey,
		target:   h.GetString(params, "target"),
	}
	api.routes[routeID] = rt
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusCreated, routeResp(rt))
}

func (s *Service) getRoutes(w http.ResponseWriter, _ *http.Request, path string) {
	apiID := extractAPIID(path)

	s.mu.RLock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "NotFoundException", "API "+apiID+" not found", http.StatusNotFound)
		return
	}

	var items []map[string]interface{}
	for _, rt := range api.routes {
		items = append(items, routeResp(rt))
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func (s *Service) deleteRoute(w http.ResponseWriter, _ *http.Request, path string) {
	apiID := extractAPIID(path)
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 5 {
		h.WriteJSONError(w, "NotFoundException", "invalid path", http.StatusNotFound)
		return
	}
	routeID := parts[4]

	s.mu.Lock()
	api, exists := s.apis[apiID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "API "+apiID+" not found", http.StatusNotFound)
		return
	}
	delete(api.routes, routeID)
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func apiResp(api *apiGw) map[string]interface{} {
	return map[string]interface{}{
		"apiId":        api.apiID,
		"name":         api.name,
		"protocolType": api.protocolType,
		"description":  api.description,
		"apiEndpoint":  api.endpoint,
		"createdDate":  api.created.Format(time.RFC3339),
	}
}

func stageResp(stg *stage) map[string]interface{} {
	return map[string]interface{}{
		"stageName":   stg.stageName,
		"description": stg.description,
		"autoDeploy":  stg.autoDeploy,
		"createdDate": stg.created.Format(time.RFC3339),
	}
}

func routeResp(rt *route) map[string]interface{} {
	return map[string]interface{}{
		"routeId":  rt.routeID,
		"routeKey": rt.routeKey,
		"target":   rt.target,
	}
}

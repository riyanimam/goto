// Package codebuild provides a mock implementation of AWS CodeBuild.
//
// Supported actions:
//   - CreateProject
//   - BatchGetProjects
//   - ListProjects
//   - DeleteProject
//   - StartBuild
//   - BatchGetBuilds
package codebuild

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

// Service implements the CodeBuild mock.
type Service struct {
	mu       sync.RWMutex
	projects map[string]*project
	builds   map[string]*build
	buildSeq map[string]int
}

type project struct {
	name         string
	arn          string
	source       sourceInfo
	environment  environmentInfo
	serviceRole  string
	created      time.Time
	lastModified time.Time
}

type sourceInfo struct {
	sourceType string
	location   string
}

type environmentInfo struct {
	envType     string
	image       string
	computeType string
}

type build struct {
	id          string
	arn         string
	projectName string
	buildNumber int
	buildStatus string
	startTime   time.Time
	source      sourceInfo
	environment environmentInfo
}

// New creates a new CodeBuild mock service.
func New() *Service {
	return &Service{
		projects: make(map[string]*project),
		builds:   make(map[string]*build),
		buildSeq: make(map[string]int),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "codebuild" }

// Handler returns the HTTP handler for CodeBuild requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects = make(map[string]*project)
	s.builds = make(map[string]*build)
	s.buildSeq = make(map[string]int)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteJSONError(w, "InternalFailure", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &params)
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
	case "CreateProject":
		s.createProject(w, params)
	case "BatchGetProjects":
		s.batchGetProjects(w, params)
	case "ListProjects":
		s.listProjects(w, params)
	case "DeleteProject":
		s.deleteProject(w, params)
	case "StartBuild":
		s.startBuild(w, params)
	case "BatchGetBuilds":
		s.batchGetBuilds(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createProject(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "name")
	if name == "" {
		h.WriteJSONError(w, "InvalidInputException", "name is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.projects[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceAlreadyExistsException", "Project already exists: "+name, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	p := &project{
		name:         name,
		arn:          fmt.Sprintf("arn:aws:codebuild:us-east-1:%s:project/%s", h.DefaultAccountID, name),
		serviceRole:  h.GetString(params, "serviceRole"),
		created:      now,
		lastModified: now,
	}

	if src, ok := params["source"].(map[string]interface{}); ok {
		p.source.sourceType = h.GetString(src, "type")
		p.source.location = h.GetString(src, "location")
	}

	if env, ok := params["environment"].(map[string]interface{}); ok {
		p.environment.envType = h.GetString(env, "type")
		p.environment.image = h.GetString(env, "image")
		p.environment.computeType = h.GetString(env, "computeType")
	}

	s.projects[name] = p
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"project": projectResp(p),
	})
}

func (s *Service) batchGetProjects(w http.ResponseWriter, params map[string]interface{}) {
	names := getStringSlice(params, "names")

	s.mu.RLock()
	var found []map[string]interface{}
	var notFound []string
	for _, name := range names {
		if p, exists := s.projects[name]; exists {
			found = append(found, projectResp(p))
		} else {
			notFound = append(notFound, name)
		}
	}
	s.mu.RUnlock()

	resp := map[string]interface{}{
		"projects":         found,
		"projectsNotFound": notFound,
	}
	h.WriteJSON(w, http.StatusOK, resp)
}

func (s *Service) listProjects(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var names []string
	for name := range s.projects {
		names = append(names, name)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"projects": names,
	})
}

func (s *Service) deleteProject(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "name")

	s.mu.Lock()
	if _, exists := s.projects[name]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Project not found: "+name, http.StatusBadRequest)
		return
	}
	delete(s.projects, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) startBuild(w http.ResponseWriter, params map[string]interface{}) {
	projectName := h.GetString(params, "projectName")
	if projectName == "" {
		h.WriteJSONError(w, "InvalidInputException", "projectName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	p, exists := s.projects[projectName]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Project not found: "+projectName, http.StatusBadRequest)
		return
	}

	s.buildSeq[projectName]++
	buildNumber := s.buildSeq[projectName]
	buildID := fmt.Sprintf("%s:%s", projectName, h.NewRequestID())
	now := time.Now().UTC()

	b := &build{
		id:          buildID,
		arn:         fmt.Sprintf("arn:aws:codebuild:us-east-1:%s:build/%s", h.DefaultAccountID, buildID),
		projectName: projectName,
		buildNumber: buildNumber,
		buildStatus: "IN_PROGRESS",
		startTime:   now,
		source:      p.source,
		environment: p.environment,
	}
	s.builds[buildID] = b
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"build": buildResp(b),
	})
}

func (s *Service) batchGetBuilds(w http.ResponseWriter, params map[string]interface{}) {
	ids := getStringSlice(params, "ids")

	s.mu.RLock()
	var found []map[string]interface{}
	var notFound []string
	for _, id := range ids {
		if b, exists := s.builds[id]; exists {
			found = append(found, buildResp(b))
		} else {
			notFound = append(notFound, id)
		}
	}
	s.mu.RUnlock()

	resp := map[string]interface{}{
		"builds":         found,
		"buildsNotFound": notFound,
	}
	h.WriteJSON(w, http.StatusOK, resp)
}

func projectResp(p *project) map[string]interface{} {
	return map[string]interface{}{
		"name": p.name,
		"arn":  p.arn,
		"source": map[string]interface{}{
			"type":     p.source.sourceType,
			"location": p.source.location,
		},
		"environment": map[string]interface{}{
			"type":        p.environment.envType,
			"image":       p.environment.image,
			"computeType": p.environment.computeType,
		},
		"serviceRole":  p.serviceRole,
		"created":      float64(p.created.Unix()),
		"lastModified": float64(p.lastModified.Unix()),
	}
}

func buildResp(b *build) map[string]interface{} {
	return map[string]interface{}{
		"id":          b.id,
		"arn":         b.arn,
		"projectName": b.projectName,
		"buildNumber": b.buildNumber,
		"buildStatus": b.buildStatus,
		"startTime":   float64(b.startTime.Unix()),
		"source": map[string]interface{}{
			"type":     b.source.sourceType,
			"location": b.source.location,
		},
		"environment": map[string]interface{}{
			"type":        b.environment.envType,
			"image":       b.environment.image,
			"computeType": b.environment.computeType,
		},
	}
}

func getStringSlice(params map[string]interface{}, key string) []string {
	items, ok := params[key].([]interface{})
	if !ok {
		return nil
	}
	var result []string
	for _, item := range items {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

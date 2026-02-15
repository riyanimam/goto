// Package codepipeline provides a mock implementation of AWS CodePipeline.
//
// Supported actions:
//   - CreatePipeline
//   - GetPipeline
//   - DeletePipeline
//   - ListPipelines
//   - UpdatePipeline
package codepipeline

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

// Service implements the CodePipeline mock.
type Service struct {
	mu        sync.RWMutex
	pipelines map[string]*pipeline
}

type pipeline struct {
	name    string
	arn     string
	roleArn string
	stages  interface{}
	version int
	created time.Time
	updated time.Time
}

// New creates a new CodePipeline mock service.
func New() *Service {
	return &Service{
		pipelines: make(map[string]*pipeline),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "codepipeline" }

// Handler returns the HTTP handler for CodePipeline requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pipelines = make(map[string]*pipeline)
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
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			h.WriteJSONError(w, "SerializationException", "could not parse request body", http.StatusBadRequest)
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
	case "CreatePipeline":
		s.createPipeline(w, params)
	case "GetPipeline":
		s.getPipeline(w, params)
	case "DeletePipeline":
		s.deletePipeline(w, params)
	case "ListPipelines":
		s.listPipelines(w)
	case "UpdatePipeline":
		s.updatePipeline(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createPipeline(w http.ResponseWriter, params map[string]interface{}) {
	pipelineObj, ok := params["pipeline"].(map[string]interface{})
	if !ok {
		h.WriteJSONError(w, "InvalidParameterException", "pipeline is required", http.StatusBadRequest)
		return
	}

	name := h.GetString(pipelineObj, "name")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterException", "pipeline name is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.pipelines[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "PipelineNameInUseException", "Pipeline already exists: "+name, http.StatusConflict)
		return
	}

	now := time.Now().UTC()
	p := &pipeline{
		name:    name,
		arn:     fmt.Sprintf("arn:aws:codepipeline:us-east-1:%s:%s", h.DefaultAccountID, name),
		roleArn: h.GetString(pipelineObj, "roleArn"),
		stages:  pipelineObj["stages"],
		version: 1,
		created: now,
		updated: now,
	}
	s.pipelines[name] = p
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"pipeline": pipelineResp(p),
	})
}

func (s *Service) getPipeline(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "name")

	s.mu.RLock()
	p, exists := s.pipelines[name]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "PipelineNotFoundException", "Pipeline not found: "+name, http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"pipeline": pipelineResp(p),
		"metadata": map[string]interface{}{
			"pipelineArn": p.arn,
			"created":     float64(p.created.Unix()),
			"updated":     float64(p.updated.Unix()),
		},
	})
}

func (s *Service) deletePipeline(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "name")

	s.mu.Lock()
	if _, exists := s.pipelines[name]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "PipelineNotFoundException", "Pipeline not found: "+name, http.StatusBadRequest)
		return
	}
	delete(s.pipelines, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listPipelines(w http.ResponseWriter) {
	s.mu.RLock()
	var list []map[string]interface{}
	for _, p := range s.pipelines {
		list = append(list, map[string]interface{}{
			"name":    p.name,
			"version": p.version,
			"created": float64(p.created.Unix()),
			"updated": float64(p.updated.Unix()),
		})
	}
	s.mu.RUnlock()

	sort.Slice(list, func(i, j int) bool {
		return list[i]["name"].(string) < list[j]["name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"pipelines": list,
	})
}

func (s *Service) updatePipeline(w http.ResponseWriter, params map[string]interface{}) {
	pipelineObj, ok := params["pipeline"].(map[string]interface{})
	if !ok {
		h.WriteJSONError(w, "InvalidParameterException", "pipeline is required", http.StatusBadRequest)
		return
	}

	name := h.GetString(pipelineObj, "name")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterException", "pipeline name is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	p, exists := s.pipelines[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "PipelineNotFoundException", "Pipeline not found: "+name, http.StatusBadRequest)
		return
	}

	p.roleArn = h.GetString(pipelineObj, "roleArn")
	p.stages = pipelineObj["stages"]
	p.version++
	p.updated = time.Now().UTC()
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"pipeline": pipelineResp(p),
	})
}

func pipelineResp(p *pipeline) map[string]interface{} {
	return map[string]interface{}{
		"name":    p.name,
		"roleArn": p.roleArn,
		"stages":  p.stages,
		"version": p.version,
	}
}

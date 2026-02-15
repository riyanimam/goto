// Package batch provides a mock implementation of AWS Batch.
//
// Supported actions:
//   - CreateComputeEnvironment
//   - DescribeComputeEnvironments
//   - DeleteComputeEnvironment
//   - CreateJobQueue
//   - DescribeJobQueues
//   - DeleteJobQueue
//   - SubmitJob
//   - DescribeJobs
package batch

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

// Service implements the AWS Batch mock.
type Service struct {
	mu           sync.RWMutex
	computeEnvs  map[string]*computeEnvironment
	jobQueues    map[string]*jobQueue
	jobs         map[string]*job
}

type computeEnvironment struct {
	name   string
	arn    string
	ceType string
	state  string
	status string
}

type jobQueue struct {
	name     string
	arn      string
	state    string
	priority int
	status   string
}

type job struct {
	id         string
	name       string
	arn        string
	queue      string
	definition string
	status     string
	createdAt  time.Time
}

// New creates a new AWS Batch mock service.
func New() *Service {
	return &Service{
		computeEnvs: make(map[string]*computeEnvironment),
		jobQueues:   make(map[string]*jobQueue),
		jobs:        make(map[string]*job),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "batch" }

// Handler returns the HTTP handler for Batch requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.computeEnvs = make(map[string]*computeEnvironment)
	s.jobQueues = make(map[string]*jobQueue)
	s.jobs = make(map[string]*job)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if r.Method != http.MethodPost {
		h.WriteJSONError(w, "ClientException", "unsupported method", http.StatusBadRequest)
		return
	}

	switch {
	case strings.HasSuffix(path, "/v1/createcomputeenvironment"):
		s.createComputeEnvironment(w, r)
	case strings.HasSuffix(path, "/v1/describecomputeenvironments"):
		s.describeComputeEnvironments(w, r)
	case strings.HasSuffix(path, "/v1/deletecomputeenvironment"):
		s.deleteComputeEnvironment(w, r)
	case strings.HasSuffix(path, "/v1/createjobqueue"):
		s.createJobQueue(w, r)
	case strings.HasSuffix(path, "/v1/describejobqueues"):
		s.describeJobQueues(w, r)
	case strings.HasSuffix(path, "/v1/deletejobqueue"):
		s.deleteJobQueue(w, r)
	case strings.HasSuffix(path, "/v1/submitjob"):
		s.submitJob(w, r)
	case strings.HasSuffix(path, "/v1/describejobs"):
		s.describeJobs(w, r)
	default:
		h.WriteJSONError(w, "ClientException", "unsupported operation", http.StatusBadRequest)
	}
}

func readBody(r *http.Request) (map[string]interface{}, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			return nil, err
		}
	}
	return params, nil
}

func (s *Service) createComputeEnvironment(w http.ResponseWriter, r *http.Request) {
	params, err := readBody(r)
	if err != nil {
		h.WriteJSONError(w, "ClientException", "invalid request body", http.StatusBadRequest)
		return
	}

	name := h.GetString(params, "computeEnvironmentName")
	if name == "" {
		h.WriteJSONError(w, "ClientException", "computeEnvironmentName is required", http.StatusBadRequest)
		return
	}

	ceType := h.GetString(params, "type")
	if ceType == "" {
		ceType = "MANAGED"
	}

	state := h.GetString(params, "state")
	if state == "" {
		state = "ENABLED"
	}

	arn := fmt.Sprintf("arn:aws:batch:us-east-1:%s:compute-environment/%s", h.DefaultAccountID, name)

	s.mu.Lock()
	s.computeEnvs[name] = &computeEnvironment{
		name:   name,
		arn:    arn,
		ceType: ceType,
		state:  state,
		status: "VALID",
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"computeEnvironmentName": name,
		"computeEnvironmentArn":  arn,
	})
}

func (s *Service) describeComputeEnvironments(w http.ResponseWriter, r *http.Request) {
	params, err := readBody(r)
	if err != nil {
		h.WriteJSONError(w, "ClientException", "invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	var envs []map[string]interface{}

	// If specific names are requested, filter by them.
	if names, ok := params["computeEnvironments"].([]interface{}); ok && len(names) > 0 {
		for _, n := range names {
			name, _ := n.(string)
			if ce, exists := s.computeEnvs[name]; exists {
				envs = append(envs, ceToMap(ce))
			}
		}
	} else {
		for _, ce := range s.computeEnvs {
			envs = append(envs, ceToMap(ce))
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"computeEnvironments": envs,
	})
}

func ceToMap(ce *computeEnvironment) map[string]interface{} {
	return map[string]interface{}{
		"computeEnvironmentName": ce.name,
		"computeEnvironmentArn":  ce.arn,
		"type":                   ce.ceType,
		"state":                  ce.state,
		"status":                 ce.status,
	}
}

func (s *Service) deleteComputeEnvironment(w http.ResponseWriter, r *http.Request) {
	params, err := readBody(r)
	if err != nil {
		h.WriteJSONError(w, "ClientException", "invalid request body", http.StatusBadRequest)
		return
	}

	name := h.GetString(params, "computeEnvironment")
	if name == "" {
		h.WriteJSONError(w, "ClientException", "computeEnvironment is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	delete(s.computeEnvs, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) createJobQueue(w http.ResponseWriter, r *http.Request) {
	params, err := readBody(r)
	if err != nil {
		h.WriteJSONError(w, "ClientException", "invalid request body", http.StatusBadRequest)
		return
	}

	name := h.GetString(params, "jobQueueName")
	if name == "" {
		h.WriteJSONError(w, "ClientException", "jobQueueName is required", http.StatusBadRequest)
		return
	}

	state := h.GetString(params, "state")
	if state == "" {
		state = "ENABLED"
	}

	priority := h.GetInt(params, "priority", 0)

	arn := fmt.Sprintf("arn:aws:batch:us-east-1:%s:job-queue/%s", h.DefaultAccountID, name)

	s.mu.Lock()
	s.jobQueues[name] = &jobQueue{
		name:     name,
		arn:      arn,
		state:    state,
		priority: priority,
		status:   "VALID",
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"jobQueueName": name,
		"jobQueueArn":  arn,
	})
}

func (s *Service) describeJobQueues(w http.ResponseWriter, r *http.Request) {
	params, err := readBody(r)
	if err != nil {
		h.WriteJSONError(w, "ClientException", "invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	var queues []map[string]interface{}

	if names, ok := params["jobQueues"].([]interface{}); ok && len(names) > 0 {
		for _, n := range names {
			name, _ := n.(string)
			if jq, exists := s.jobQueues[name]; exists {
				queues = append(queues, jqToMap(jq))
			}
		}
	} else {
		for _, jq := range s.jobQueues {
			queues = append(queues, jqToMap(jq))
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"jobQueues": queues,
	})
}

func jqToMap(jq *jobQueue) map[string]interface{} {
	return map[string]interface{}{
		"jobQueueName": jq.name,
		"jobQueueArn":  jq.arn,
		"state":        jq.state,
		"priority":     jq.priority,
		"status":       jq.status,
	}
}

func (s *Service) deleteJobQueue(w http.ResponseWriter, r *http.Request) {
	params, err := readBody(r)
	if err != nil {
		h.WriteJSONError(w, "ClientException", "invalid request body", http.StatusBadRequest)
		return
	}

	name := h.GetString(params, "jobQueue")
	if name == "" {
		h.WriteJSONError(w, "ClientException", "jobQueue is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	delete(s.jobQueues, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) submitJob(w http.ResponseWriter, r *http.Request) {
	params, err := readBody(r)
	if err != nil {
		h.WriteJSONError(w, "ClientException", "invalid request body", http.StatusBadRequest)
		return
	}

	jobName := h.GetString(params, "jobName")
	if jobName == "" {
		h.WriteJSONError(w, "ClientException", "jobName is required", http.StatusBadRequest)
		return
	}

	jobQueue := h.GetString(params, "jobQueue")
	jobDefinition := h.GetString(params, "jobDefinition")

	jobID := h.NewRequestID()
	arn := fmt.Sprintf("arn:aws:batch:us-east-1:%s:job/%s", h.DefaultAccountID, jobID)

	s.mu.Lock()
	s.jobs[jobID] = &job{
		id:         jobID,
		name:       jobName,
		arn:        arn,
		queue:      jobQueue,
		definition: jobDefinition,
		status:     "SUBMITTED",
		createdAt:  time.Now().UTC(),
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"jobId":   jobID,
		"jobName": jobName,
	})
}

func (s *Service) describeJobs(w http.ResponseWriter, r *http.Request) {
	params, err := readBody(r)
	if err != nil {
		h.WriteJSONError(w, "ClientException", "invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	var jobs []map[string]interface{}

	if ids, ok := params["jobs"].([]interface{}); ok {
		for _, id := range ids {
			jobID, _ := id.(string)
			if j, exists := s.jobs[jobID]; exists {
				jobs = append(jobs, jobToMap(j))
			}
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"jobs": jobs,
	})
}

func jobToMap(j *job) map[string]interface{} {
	return map[string]interface{}{
		"jobId":         j.id,
		"jobName":       j.name,
		"jobArn":        j.arn,
		"jobQueue":      j.queue,
		"jobDefinition": j.definition,
		"status":        j.status,
		"createdAt":     j.createdAt.Unix(),
	}
}

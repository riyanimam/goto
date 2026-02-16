// Package stepfunctions provides a mock implementation of AWS Step Functions.
//
// Supported actions:
//   - CreateStateMachine
//   - DeleteStateMachine
//   - DescribeStateMachine
//   - ListStateMachines
//   - StartExecution
//   - DescribeExecution
//   - ListExecutions
//   - StopExecution
package stepfunctions

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

// Service implements the Step Functions mock.
type Service struct {
	mu            sync.RWMutex
	stateMachines map[string]*stateMachine
	executions    map[string]*execution
}

type stateMachine struct {
	name       string
	arn        string
	definition string
	roleArn    string
	status     string
	smType     string
	created    time.Time
}

type execution struct {
	arn             string
	name            string
	stateMachineArn string
	status          string
	input           string
	output          string
	startDate       time.Time
	stopDate        *time.Time
}

// New creates a new Step Functions mock service.
func New() *Service {
	return &Service{
		stateMachines: make(map[string]*stateMachine),
		executions:    make(map[string]*execution),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "states" }

// Handler returns the HTTP handler for Step Functions requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stateMachines = make(map[string]*stateMachine)
	s.executions = make(map[string]*execution)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteJSONError(w, "InternalError", "could not read request body", http.StatusInternalServerError)
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
	case "CreateStateMachine":
		s.createStateMachine(w, params)
	case "DeleteStateMachine":
		s.deleteStateMachine(w, params)
	case "DescribeStateMachine":
		s.describeStateMachine(w, params)
	case "ListStateMachines":
		s.listStateMachines(w, params)
	case "StartExecution":
		s.startExecution(w, params)
	case "DescribeExecution":
		s.describeExecution(w, params)
	case "ListExecutions":
		s.listExecutions(w, params)
	case "StopExecution":
		s.stopExecution(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createStateMachine(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "name")
	if name == "" {
		h.WriteJSONError(w, "InvalidName", "name is required", http.StatusBadRequest)
		return
	}

	definition := h.GetString(params, "definition")
	roleArn := h.GetString(params, "roleArn")
	smType := h.GetString(params, "type")
	if smType == "" {
		smType = "STANDARD"
	}

	s.mu.Lock()
	arn := fmt.Sprintf("arn:aws:states:us-east-1:%s:stateMachine:%s", h.DefaultAccountID, name)
	if _, exists := s.stateMachines[arn]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "StateMachineAlreadyExists", "State machine already exists: "+arn, http.StatusBadRequest)
		return
	}

	sm := &stateMachine{
		name:       name,
		arn:        arn,
		definition: definition,
		roleArn:    roleArn,
		status:     "ACTIVE",
		smType:     smType,
		created:    time.Now().UTC(),
	}
	s.stateMachines[arn] = sm
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"stateMachineArn": arn,
		"creationDate":    float64(sm.created.Unix()),
	})
}

func (s *Service) deleteStateMachine(w http.ResponseWriter, params map[string]interface{}) {
	arn := h.GetString(params, "stateMachineArn")

	s.mu.Lock()
	if _, exists := s.stateMachines[arn]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "StateMachineDoesNotExist", "State machine does not exist: "+arn, http.StatusBadRequest)
		return
	}
	delete(s.stateMachines, arn)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) describeStateMachine(w http.ResponseWriter, params map[string]interface{}) {
	arn := h.GetString(params, "stateMachineArn")

	s.mu.RLock()
	sm, exists := s.stateMachines[arn]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "StateMachineDoesNotExist", "State machine does not exist: "+arn, http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"stateMachineArn": sm.arn,
		"name":            sm.name,
		"definition":      sm.definition,
		"roleArn":         sm.roleArn,
		"status":          sm.status,
		"type":            sm.smType,
		"creationDate":    float64(sm.created.Unix()),
	})
}

func (s *Service) listStateMachines(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var machines []map[string]interface{}
	for _, sm := range s.stateMachines {
		machines = append(machines, map[string]interface{}{
			"stateMachineArn": sm.arn,
			"name":            sm.name,
			"type":            sm.smType,
			"creationDate":    float64(sm.created.Unix()),
		})
	}
	s.mu.RUnlock()

	sort.Slice(machines, func(i, j int) bool {
		return machines[i]["name"].(string) < machines[j]["name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"stateMachines": machines,
	})
}

func (s *Service) startExecution(w http.ResponseWriter, params map[string]interface{}) {
	smArn := h.GetString(params, "stateMachineArn")
	name := h.GetString(params, "name")
	if name == "" {
		name = h.NewRequestID()
	}
	input := h.GetString(params, "input")

	s.mu.RLock()
	_, exists := s.stateMachines[smArn]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "StateMachineDoesNotExist", "State machine does not exist: "+smArn, http.StatusBadRequest)
		return
	}

	execArn := fmt.Sprintf("arn:aws:states:us-east-1:%s:execution:%s:%s",
		h.DefaultAccountID,
		smArn[strings.LastIndex(smArn, ":")+1:],
		name)

	s.mu.Lock()
	exec := &execution{
		arn:             execArn,
		name:            name,
		stateMachineArn: smArn,
		status:          "RUNNING",
		input:           input,
		startDate:       time.Now().UTC(),
	}
	s.executions[execArn] = exec
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"executionArn": execArn,
		"startDate":    float64(exec.startDate.Unix()),
	})
}

func (s *Service) describeExecution(w http.ResponseWriter, params map[string]interface{}) {
	execArn := h.GetString(params, "executionArn")

	s.mu.RLock()
	exec, exists := s.executions[execArn]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ExecutionDoesNotExist", "Execution does not exist: "+execArn, http.StatusBadRequest)
		return
	}

	result := map[string]interface{}{
		"executionArn":    exec.arn,
		"name":            exec.name,
		"stateMachineArn": exec.stateMachineArn,
		"status":          exec.status,
		"input":           exec.input,
		"startDate":       float64(exec.startDate.Unix()),
	}
	if exec.output != "" {
		result["output"] = exec.output
	}
	if exec.stopDate != nil {
		result["stopDate"] = float64(exec.stopDate.Unix())
	}

	h.WriteJSON(w, http.StatusOK, result)
}

func (s *Service) listExecutions(w http.ResponseWriter, params map[string]interface{}) {
	smArn := h.GetString(params, "stateMachineArn")

	s.mu.RLock()
	var execs []map[string]interface{}
	for _, exec := range s.executions {
		if smArn == "" || exec.stateMachineArn == smArn {
			execs = append(execs, map[string]interface{}{
				"executionArn":    exec.arn,
				"name":            exec.name,
				"stateMachineArn": exec.stateMachineArn,
				"status":          exec.status,
				"startDate":       float64(exec.startDate.Unix()),
			})
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"executions": execs,
	})
}

func (s *Service) stopExecution(w http.ResponseWriter, params map[string]interface{}) {
	execArn := h.GetString(params, "executionArn")

	s.mu.Lock()
	exec, exists := s.executions[execArn]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ExecutionDoesNotExist", "Execution does not exist: "+execArn, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	exec.status = "ABORTED"
	exec.stopDate = &now
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"stopDate": float64(now.Unix()),
	})
}

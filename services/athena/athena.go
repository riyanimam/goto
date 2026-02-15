// Package athena provides a mock implementation of AWS Athena.
//
// Supported actions:
//   - StartQueryExecution
//   - GetQueryExecution
//   - GetQueryResults
//   - ListQueryExecutions
//   - CreateWorkGroup
//   - GetWorkGroup
//   - DeleteWorkGroup
//   - ListWorkGroups
package athena

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

// Service implements the Athena mock.
type Service struct {
	mu         sync.RWMutex
	executions map[string]*queryExecution
	workgroups map[string]*workGroup
}

type queryExecution struct {
	id        string
	query     string
	database  string
	workgroup string
	outputLoc string
	status    string
	submitted time.Time
	completed time.Time
}

type workGroup struct {
	name        string
	state       string
	description string
	created     time.Time
}

// New creates a new Athena mock service.
func New() *Service {
	return &Service{
		executions: make(map[string]*queryExecution),
		workgroups: map[string]*workGroup{
			"primary": {name: "primary", state: "ENABLED", created: time.Now().UTC()},
		},
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "athena" }

// Handler returns the HTTP handler for Athena requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executions = make(map[string]*queryExecution)
	s.workgroups = map[string]*workGroup{
		"primary": {name: "primary", state: "ENABLED", created: time.Now().UTC()},
	}
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
	case "StartQueryExecution":
		s.startQueryExecution(w, params)
	case "GetQueryExecution":
		s.getQueryExecution(w, params)
	case "GetQueryResults":
		s.getQueryResults(w, params)
	case "ListQueryExecutions":
		s.listQueryExecutions(w, params)
	case "CreateWorkGroup":
		s.createWorkGroup(w, params)
	case "GetWorkGroup":
		s.getWorkGroup(w, params)
	case "DeleteWorkGroup":
		s.deleteWorkGroup(w, params)
	case "ListWorkGroups":
		s.listWorkGroups(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) startQueryExecution(w http.ResponseWriter, params map[string]interface{}) {
	query := h.GetString(params, "QueryString")
	if query == "" {
		h.WriteJSONError(w, "InvalidRequestException", "QueryString is required", http.StatusBadRequest)
		return
	}

	database := ""
	if qCtx, ok := params["QueryExecutionContext"].(map[string]interface{}); ok {
		database = h.GetString(qCtx, "Database")
	}

	outputLoc := ""
	if resCfg, ok := params["ResultConfiguration"].(map[string]interface{}); ok {
		outputLoc = h.GetString(resCfg, "OutputLocation")
	}

	wg := h.GetString(params, "WorkGroup")
	if wg == "" {
		wg = "primary"
	}

	now := time.Now().UTC()
	id := h.NewRequestID()

	s.mu.Lock()
	s.executions[id] = &queryExecution{
		id:        id,
		query:     query,
		database:  database,
		workgroup: wg,
		outputLoc: outputLoc,
		status:    "SUCCEEDED",
		submitted: now,
		completed: now,
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"QueryExecutionId": id,
	})
}

func (s *Service) getQueryExecution(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "QueryExecutionId")

	s.mu.RLock()
	exec, exists := s.executions[id]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "InvalidRequestException", "Query execution "+id+" not found", http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"QueryExecution": execResp(exec),
	})
}

func (s *Service) getQueryResults(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "QueryExecutionId")

	s.mu.RLock()
	_, exists := s.executions[id]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "InvalidRequestException", "Query execution "+id+" not found", http.StatusBadRequest)
		return
	}

	// Return empty result set.
	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ResultSet": map[string]interface{}{
			"Rows": []interface{}{},
			"ResultSetMetadata": map[string]interface{}{
				"ColumnInfo": []interface{}{},
			},
		},
	})
}

func (s *Service) listQueryExecutions(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var ids []string
	for id := range s.executions {
		ids = append(ids, id)
	}
	s.mu.RUnlock()

	sort.Strings(ids)

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"QueryExecutionIds": ids,
	})
}

func (s *Service) createWorkGroup(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")
	if name == "" {
		h.WriteJSONError(w, "InvalidRequestException", "Name is required", http.StatusBadRequest)
		return
	}

	desc := h.GetString(params, "Description")

	s.mu.Lock()
	if _, exists := s.workgroups[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "InvalidRequestException", "WorkGroup "+name+" already exists", http.StatusConflict)
		return
	}

	s.workgroups[name] = &workGroup{
		name:        name,
		state:       "ENABLED",
		description: desc,
		created:     time.Now().UTC(),
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) getWorkGroup(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "WorkGroup")

	s.mu.RLock()
	wg, exists := s.workgroups[name]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "InvalidRequestException", "WorkGroup "+name+" not found", http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"WorkGroup": map[string]interface{}{
			"Name":         wg.name,
			"State":        wg.state,
			"Description":  wg.description,
			"CreationTime": float64(wg.created.Unix()),
		},
	})
}

func (s *Service) deleteWorkGroup(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "WorkGroup")

	s.mu.Lock()
	if _, exists := s.workgroups[name]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "InvalidRequestException", "WorkGroup "+name+" not found", http.StatusBadRequest)
		return
	}
	delete(s.workgroups, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listWorkGroups(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var groups []map[string]interface{}
	for _, wg := range s.workgroups {
		groups = append(groups, map[string]interface{}{
			"Name":         wg.name,
			"State":        wg.state,
			"Description":  wg.description,
			"CreationTime": float64(wg.created.Unix()),
		})
	}
	s.mu.RUnlock()

	sort.Slice(groups, func(i, j int) bool {
		return groups[i]["Name"].(string) < groups[j]["Name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"WorkGroups": groups,
	})
}

func execResp(exec *queryExecution) map[string]interface{} {
	return map[string]interface{}{
		"QueryExecutionId": exec.id,
		"Query":            exec.query,
		"QueryExecutionContext": map[string]interface{}{
			"Database": exec.database,
		},
		"ResultConfiguration": map[string]interface{}{
			"OutputLocation": exec.outputLoc,
		},
		"WorkGroup": exec.workgroup,
		"Status": map[string]interface{}{
			"State":              exec.status,
			"SubmissionDateTime": float64(exec.submitted.Unix()),
			"CompletionDateTime": float64(exec.completed.Unix()),
		},
	}
}

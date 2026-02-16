// Package scheduler provides a mock implementation of Amazon EventBridge Scheduler.
//
// Supported actions:
//   - CreateSchedule
//   - GetSchedule
//   - DeleteSchedule
//   - ListSchedules
//   - UpdateSchedule
package scheduler

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

// Service implements the EventBridge Scheduler mock.
type Service struct {
	mu        sync.RWMutex
	schedules map[string]*schedule
}

type schedule struct {
	name               string
	arn                string
	scheduleExpression string
	target             interface{}
	flexibleTimeWindow interface{}
	state              string
	groupName          string
	description        string
	created            time.Time
	modified           time.Time
}

// New creates a new Scheduler mock service.
func New() *Service {
	return &Service{
		schedules: make(map[string]*schedule),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "scheduler" }

// Handler returns the HTTP handler for Scheduler requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.schedules = make(map[string]*schedule)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	switch {
	case path == "/schedules" && method == http.MethodGet:
		s.listSchedules(w, r)
	case strings.HasPrefix(path, "/schedules/") && method == http.MethodPost:
		s.createSchedule(w, r, path)
	case strings.HasPrefix(path, "/schedules/") && method == http.MethodGet:
		s.getSchedule(w, r, path)
	case strings.HasPrefix(path, "/schedules/") && method == http.MethodPut:
		s.updateSchedule(w, r, path)
	case strings.HasPrefix(path, "/schedules/") && method == http.MethodDelete:
		s.deleteSchedule(w, r, path)
	default:
		h.WriteJSONError(w, "NotFoundException", "unsupported operation", http.StatusNotFound)
	}
}

func extractName(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func (s *Service) createSchedule(w http.ResponseWriter, r *http.Request, path string) {
	name := extractName(path)
	if name == "" {
		h.WriteJSONError(w, "ValidationException", "name is required", http.StatusBadRequest)
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	s.mu.Lock()
	if _, exists := s.schedules[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ConflictException", "Schedule "+name+" already exists", http.StatusConflict)
		return
	}

	now := time.Now().UTC()
	arn := fmt.Sprintf("arn:aws:scheduler:us-east-1:%s:schedule/default/%s", h.DefaultAccountID, name)

	state := h.GetString(params, "State")
	if state == "" {
		state = "ENABLED"
	}

	sched := &schedule{
		name:               name,
		arn:                arn,
		scheduleExpression: h.GetString(params, "ScheduleExpression"),
		target:             params["Target"],
		flexibleTimeWindow: params["FlexibleTimeWindow"],
		state:              state,
		groupName:          h.GetString(params, "GroupName"),
		description:        h.GetString(params, "Description"),
		created:            now,
		modified:           now,
	}
	s.schedules[name] = sched
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ScheduleArn": sched.arn,
	})
}

func (s *Service) getSchedule(w http.ResponseWriter, _ *http.Request, path string) {
	name := extractName(path)

	s.mu.RLock()
	sched, exists := s.schedules[name]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Schedule "+name+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, scheduleResp(sched))
}

func (s *Service) deleteSchedule(w http.ResponseWriter, _ *http.Request, path string) {
	name := extractName(path)

	s.mu.Lock()
	_, exists := s.schedules[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Schedule "+name+" not found", http.StatusNotFound)
		return
	}
	delete(s.schedules, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listSchedules(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var items []map[string]interface{}
	var names []string
	for name := range s.schedules {
		names = append(names, name)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	s.mu.RLock()
	for _, name := range names {
		if sched, ok := s.schedules[name]; ok {
			items = append(items, scheduleResp(sched))
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Schedules": items,
	})
}

func (s *Service) updateSchedule(w http.ResponseWriter, r *http.Request, path string) {
	name := extractName(path)

	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	s.mu.Lock()
	sched, exists := s.schedules[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Schedule "+name+" not found", http.StatusNotFound)
		return
	}

	if v := h.GetString(params, "ScheduleExpression"); v != "" {
		sched.scheduleExpression = v
	}
	if v, ok := params["Target"]; ok {
		sched.target = v
	}
	if v, ok := params["FlexibleTimeWindow"]; ok {
		sched.flexibleTimeWindow = v
	}
	if v := h.GetString(params, "State"); v != "" {
		sched.state = v
	}
	if v := h.GetString(params, "GroupName"); v != "" {
		sched.groupName = v
	}
	if _, ok := params["Description"]; ok {
		sched.description = h.GetString(params, "Description")
	}
	sched.modified = time.Now().UTC()
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ScheduleArn": sched.arn,
	})
}

func scheduleResp(sched *schedule) map[string]interface{} {
	resp := map[string]interface{}{
		"Name":                 sched.name,
		"Arn":                  sched.arn,
		"ScheduleExpression":   sched.scheduleExpression,
		"State":                sched.state,
		"CreationDate":         float64(sched.created.Unix()),
		"LastModificationDate": float64(sched.modified.Unix()),
	}
	if sched.target != nil {
		resp["Target"] = sched.target
	}
	if sched.flexibleTimeWindow != nil {
		resp["FlexibleTimeWindow"] = sched.flexibleTimeWindow
	}
	if sched.groupName != "" {
		resp["GroupName"] = sched.groupName
	}
	if sched.description != "" {
		resp["Description"] = sched.description
	}
	return resp
}

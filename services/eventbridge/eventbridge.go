// Package eventbridge provides a mock implementation of AWS EventBridge.
//
// Supported actions:
//   - CreateEventBus
//   - DeleteEventBus
//   - ListEventBuses
//   - PutRule
//   - DeleteRule
//   - ListRules
//   - PutTargets
//   - RemoveTargets
//   - ListTargetsByRule
//   - PutEvents
package eventbridge

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
)

const defaultAccountID = "123456789012"

// Service implements the EventBridge mock.
type Service struct {
	mu      sync.RWMutex
	buses   map[string]*eventBus // keyed by name
	rules   map[string]*rule     // keyed by name
	targets map[string][]*target // keyed by rule name
}

type eventBus struct {
	name string
	arn  string
}

type rule struct {
	name         string
	arn          string
	eventBusName string
	eventPattern string
	scheduleExpr string
	state        string
	description  string
}

type target struct {
	id       string
	arn      string
	ruleName string
}

// New creates a new EventBridge mock service.
func New() *Service {
	s := &Service{
		buses:   make(map[string]*eventBus),
		rules:   make(map[string]*rule),
		targets: make(map[string][]*target),
	}
	// Create the default event bus.
	s.buses["default"] = &eventBus{
		name: "default",
		arn:  fmt.Sprintf("arn:aws:events:us-east-1:%s:event-bus/default", defaultAccountID),
	}
	return s
}

// Name returns the service identifier.
func (s *Service) Name() string { return "events" }

// Handler returns the HTTP handler for EventBridge requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buses = make(map[string]*eventBus)
	s.rules = make(map[string]*rule)
	s.targets = make(map[string][]*target)
	s.buses["default"] = &eventBus{
		name: "default",
		arn:  fmt.Sprintf("arn:aws:events:us-east-1:%s:event-bus/default", defaultAccountID),
	}
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, "InternalException", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			writeJSONError(w, "InvalidRequestException", "could not parse request body", http.StatusBadRequest)
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
	case "CreateEventBus":
		s.createEventBus(w, params)
	case "DeleteEventBus":
		s.deleteEventBus(w, params)
	case "ListEventBuses":
		s.listEventBuses(w, params)
	case "PutRule":
		s.putRule(w, params)
	case "DeleteRule":
		s.deleteRule(w, params)
	case "ListRules":
		s.listRules(w, params)
	case "PutTargets":
		s.putTargets(w, params)
	case "RemoveTargets":
		s.removeTargets(w, params)
	case "ListTargetsByRule":
		s.listTargetsByRule(w, params)
	case "PutEvents":
		s.putEvents(w, params)
	default:
		writeJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createEventBus(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "Name")
	if name == "" {
		writeJSONError(w, "ValidationException", "Name is required", http.StatusBadRequest)
		return
	}

	arn := fmt.Sprintf("arn:aws:events:us-east-1:%s:event-bus/%s", defaultAccountID, name)

	s.mu.Lock()
	if _, exists := s.buses[name]; exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceAlreadyExistsException", "Event bus "+name+" already exists.", http.StatusBadRequest)
		return
	}
	s.buses[name] = &eventBus{name: name, arn: arn}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"EventBusArn": arn,
	})
}

func (s *Service) deleteEventBus(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "Name")

	s.mu.Lock()
	delete(s.buses, name)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listEventBuses(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var buses []map[string]interface{}
	for _, b := range s.buses {
		buses = append(buses, map[string]interface{}{
			"Name": b.name,
			"Arn":  b.arn,
		})
	}
	s.mu.RUnlock()

	sort.Slice(buses, func(i, j int) bool {
		return buses[i]["Name"].(string) < buses[j]["Name"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"EventBuses": buses,
	})
}

func (s *Service) putRule(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "Name")
	if name == "" {
		writeJSONError(w, "ValidationException", "Name is required", http.StatusBadRequest)
		return
	}

	arn := fmt.Sprintf("arn:aws:events:us-east-1:%s:rule/%s", defaultAccountID, name)
	busName := getString(params, "EventBusName")
	if busName == "" {
		busName = "default"
	}

	s.mu.Lock()
	s.rules[name] = &rule{
		name:         name,
		arn:          arn,
		eventBusName: busName,
		eventPattern: getString(params, "EventPattern"),
		scheduleExpr: getString(params, "ScheduleExpression"),
		state:        "ENABLED",
		description:  getString(params, "Description"),
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"RuleArn": arn,
	})
}

func (s *Service) deleteRule(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "Name")

	s.mu.Lock()
	delete(s.rules, name)
	delete(s.targets, name)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listRules(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var rulesList []map[string]interface{}
	for _, rl := range s.rules {
		rulesList = append(rulesList, map[string]interface{}{
			"Name":         rl.name,
			"Arn":          rl.arn,
			"State":        rl.state,
			"Description":  rl.description,
			"EventBusName": rl.eventBusName,
		})
	}
	s.mu.RUnlock()

	sort.Slice(rulesList, func(i, j int) bool {
		return rulesList[i]["Name"].(string) < rulesList[j]["Name"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Rules": rulesList,
	})
}

func (s *Service) putTargets(w http.ResponseWriter, params map[string]interface{}) {
	ruleName := getString(params, "Rule")

	s.mu.Lock()
	if _, exists := s.rules[ruleName]; !exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "Rule "+ruleName+" does not exist.", http.StatusBadRequest)
		return
	}

	if targetsRaw, ok := params["Targets"].([]interface{}); ok {
		for _, t := range targetsRaw {
			if tm, ok := t.(map[string]interface{}); ok {
				tgt := &target{
					id:       getString(tm, "Id"),
					arn:      getString(tm, "Arn"),
					ruleName: ruleName,
				}
				s.targets[ruleName] = append(s.targets[ruleName], tgt)
			}
		}
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"FailedEntryCount": 0,
		"FailedEntries":    []interface{}{},
	})
}

func (s *Service) removeTargets(w http.ResponseWriter, params map[string]interface{}) {
	ruleName := getString(params, "Rule")

	s.mu.Lock()
	if ids, ok := params["Ids"].([]interface{}); ok {
		idSet := make(map[string]bool)
		for _, id := range ids {
			if sid, ok := id.(string); ok {
				idSet[sid] = true
			}
		}
		var remaining []*target
		for _, t := range s.targets[ruleName] {
			if !idSet[t.id] {
				remaining = append(remaining, t)
			}
		}
		s.targets[ruleName] = remaining
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"FailedEntryCount": 0,
		"FailedEntries":    []interface{}{},
	})
}

func (s *Service) listTargetsByRule(w http.ResponseWriter, params map[string]interface{}) {
	ruleName := getString(params, "Rule")

	s.mu.RLock()
	var targetsList []map[string]interface{}
	for _, t := range s.targets[ruleName] {
		targetsList = append(targetsList, map[string]interface{}{
			"Id":  t.id,
			"Arn": t.arn,
		})
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Targets": targetsList,
	})
}

func (s *Service) putEvents(w http.ResponseWriter, params map[string]interface{}) {
	entries, _ := params["Entries"].([]interface{})
	count := len(entries)
	if count == 0 {
		count = 1
	}

	var resultEntries []map[string]interface{}
	for i := 0; i < count; i++ {
		resultEntries = append(resultEntries, map[string]interface{}{
			"EventId": newRequestID(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Entries":          resultEntries,
		"FailedEntryCount": 0,
	})
}

// Helper functions.

func getString(params map[string]interface{}, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"__type":  code,
		"message": message,
	})
}

func newRequestID() string {
	const chars = "abcdef0123456789"
	b := make([]byte, 36)
	sections := []int{8, 4, 4, 4, 12}
	pos := 0
	for i, l := range sections {
		if i > 0 {
			b[pos] = '-'
			pos++
		}
		for j := 0; j < l; j++ {
			b[pos] = chars[rand.Intn(len(chars))]
			pos++
		}
	}
	return string(b[:pos])
}

// Package cloudwatchlogs provides a mock implementation of AWS CloudWatch Logs.
//
// Supported actions:
//   - CreateLogGroup
//   - DeleteLogGroup
//   - DescribeLogGroups
//   - CreateLogStream
//   - DeleteLogStream
//   - DescribeLogStreams
//   - PutLogEvents
//   - GetLogEvents
//   - FilterLogEvents
package cloudwatchlogs

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultAccountID = "123456789012"

// Service implements the CloudWatch Logs mock.
type Service struct {
	mu        sync.RWMutex
	logGroups map[string]*logGroup // keyed by log group name
}

type logGroup struct {
	name      string
	arn       string
	created   int64
	streams   map[string]*logStream
	streamsMu sync.Mutex
}

type logStream struct {
	name    string
	arn     string
	created int64
	events  []*logEvent
}

type logEvent struct {
	timestamp int64
	message   string
	ingested  int64
}

// New creates a new CloudWatch Logs mock service.
func New() *Service {
	return &Service{
		logGroups: make(map[string]*logGroup),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "logs" }

// Handler returns the HTTP handler for CloudWatch Logs requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all log groups, streams, and events.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logGroups = make(map[string]*logGroup)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, "InternalFailure", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			writeJSONError(w, "InvalidParameterException", "could not parse request body", http.StatusBadRequest)
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
	case "CreateLogGroup":
		s.createLogGroup(w, params)
	case "DeleteLogGroup":
		s.deleteLogGroup(w, params)
	case "DescribeLogGroups":
		s.describeLogGroups(w, params)
	case "CreateLogStream":
		s.createLogStream(w, params)
	case "DeleteLogStream":
		s.deleteLogStream(w, params)
	case "DescribeLogStreams":
		s.describeLogStreams(w, params)
	case "PutLogEvents":
		s.putLogEvents(w, params)
	case "GetLogEvents":
		s.getLogEvents(w, params)
	case "FilterLogEvents":
		s.filterLogEvents(w, params)
	default:
		writeJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createLogGroup(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "logGroupName")
	if name == "" {
		writeJSONError(w, "InvalidParameterException", "logGroupName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.logGroups[name]; exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceAlreadyExistsException", "The specified log group already exists", http.StatusBadRequest)
		return
	}

	s.logGroups[name] = &logGroup{
		name:    name,
		arn:     fmt.Sprintf("arn:aws:logs:us-east-1:%s:log-group:%s:*", defaultAccountID, name),
		created: time.Now().UnixMilli(),
		streams: make(map[string]*logStream),
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) deleteLogGroup(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "logGroupName")

	s.mu.Lock()
	if _, exists := s.logGroups[name]; !exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "The specified log group does not exist", http.StatusBadRequest)
		return
	}
	delete(s.logGroups, name)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) describeLogGroups(w http.ResponseWriter, params map[string]interface{}) {
	prefix := getString(params, "logGroupNamePrefix")

	s.mu.RLock()
	var groups []map[string]interface{}
	for _, lg := range s.logGroups {
		if prefix != "" && !strings.HasPrefix(lg.name, prefix) {
			continue
		}
		groups = append(groups, map[string]interface{}{
			"logGroupName":      lg.name,
			"arn":               lg.arn,
			"creationTime":      lg.created,
			"storedBytes":       0,
			"metricFilterCount": 0,
		})
	}
	s.mu.RUnlock()

	sort.Slice(groups, func(i, j int) bool {
		return groups[i]["logGroupName"].(string) < groups[j]["logGroupName"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logGroups": groups,
	})
}

func (s *Service) createLogStream(w http.ResponseWriter, params map[string]interface{}) {
	groupName := getString(params, "logGroupName")
	streamName := getString(params, "logStreamName")

	s.mu.RLock()
	lg, exists := s.logGroups[groupName]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "The specified log group does not exist", http.StatusBadRequest)
		return
	}

	lg.streamsMu.Lock()
	if _, exists := lg.streams[streamName]; exists {
		lg.streamsMu.Unlock()
		writeJSONError(w, "ResourceAlreadyExistsException", "The specified log stream already exists", http.StatusBadRequest)
		return
	}
	lg.streams[streamName] = &logStream{
		name:    streamName,
		arn:     fmt.Sprintf("arn:aws:logs:us-east-1:%s:log-group:%s:log-stream:%s", defaultAccountID, groupName, streamName),
		created: time.Now().UnixMilli(),
	}
	lg.streamsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) deleteLogStream(w http.ResponseWriter, params map[string]interface{}) {
	groupName := getString(params, "logGroupName")
	streamName := getString(params, "logStreamName")

	s.mu.RLock()
	lg, exists := s.logGroups[groupName]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "The specified log group does not exist", http.StatusBadRequest)
		return
	}

	lg.streamsMu.Lock()
	delete(lg.streams, streamName)
	lg.streamsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) describeLogStreams(w http.ResponseWriter, params map[string]interface{}) {
	groupName := getString(params, "logGroupName")
	prefix := getString(params, "logStreamNamePrefix")

	s.mu.RLock()
	lg, exists := s.logGroups[groupName]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "The specified log group does not exist", http.StatusBadRequest)
		return
	}

	lg.streamsMu.Lock()
	var streams []map[string]interface{}
	for _, ls := range lg.streams {
		if prefix != "" && !strings.HasPrefix(ls.name, prefix) {
			continue
		}
		streams = append(streams, map[string]interface{}{
			"logStreamName": ls.name,
			"arn":           ls.arn,
			"creationTime":  ls.created,
			"storedBytes":   0,
		})
	}
	lg.streamsMu.Unlock()

	sort.Slice(streams, func(i, j int) bool {
		return streams[i]["logStreamName"].(string) < streams[j]["logStreamName"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logStreams": streams,
	})
}

func (s *Service) putLogEvents(w http.ResponseWriter, params map[string]interface{}) {
	groupName := getString(params, "logGroupName")
	streamName := getString(params, "logStreamName")

	s.mu.RLock()
	lg, exists := s.logGroups[groupName]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "The specified log group does not exist", http.StatusBadRequest)
		return
	}

	lg.streamsMu.Lock()
	ls, exists := lg.streams[streamName]
	if !exists {
		lg.streamsMu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "The specified log stream does not exist", http.StatusBadRequest)
		return
	}

	now := time.Now().UnixMilli()
	if events, ok := params["logEvents"].([]interface{}); ok {
		for _, e := range events {
			if em, ok := e.(map[string]interface{}); ok {
				ts := int64(0)
				if v, ok := em["timestamp"].(float64); ok {
					ts = int64(v)
				}
				msg := ""
				if v, ok := em["message"].(string); ok {
					msg = v
				}
				ls.events = append(ls.events, &logEvent{
					timestamp: ts,
					message:   msg,
					ingested:  now,
				})
			}
		}
	}
	lg.streamsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"nextSequenceToken": newRequestID(),
	})
}

func (s *Service) getLogEvents(w http.ResponseWriter, params map[string]interface{}) {
	groupName := getString(params, "logGroupName")
	streamName := getString(params, "logStreamName")

	s.mu.RLock()
	lg, exists := s.logGroups[groupName]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "The specified log group does not exist", http.StatusBadRequest)
		return
	}

	lg.streamsMu.Lock()
	ls, exists := lg.streams[streamName]
	if !exists {
		lg.streamsMu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "The specified log stream does not exist", http.StatusBadRequest)
		return
	}

	var events []map[string]interface{}
	for _, e := range ls.events {
		events = append(events, map[string]interface{}{
			"timestamp":     e.timestamp,
			"message":       e.message,
			"ingestionTime": e.ingested,
		})
	}
	lg.streamsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events":            events,
		"nextForwardToken":  "f/00000000000000000000000000000000000000000000000000000000",
		"nextBackwardToken": "b/00000000000000000000000000000000000000000000000000000000",
	})
}

func (s *Service) filterLogEvents(w http.ResponseWriter, params map[string]interface{}) {
	groupName := getString(params, "logGroupName")
	filterPattern := getString(params, "filterPattern")

	s.mu.RLock()
	lg, exists := s.logGroups[groupName]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "The specified log group does not exist", http.StatusBadRequest)
		return
	}

	lg.streamsMu.Lock()
	var events []map[string]interface{}
	for streamName, ls := range lg.streams {
		for _, e := range ls.events {
			if filterPattern == "" || strings.Contains(e.message, filterPattern) {
				events = append(events, map[string]interface{}{
					"timestamp":     e.timestamp,
					"message":       e.message,
					"ingestionTime": e.ingested,
					"logStreamName": streamName,
					"eventId":       newRequestID(),
				})
			}
		}
	}
	lg.streamsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events":             events,
		"searchedLogStreams": []interface{}{},
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

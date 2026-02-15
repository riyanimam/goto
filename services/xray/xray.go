// Package xray provides a mock implementation of AWS X-Ray.
//
// Supported actions:
//   - PutTraceSegments
//   - GetTraceSummaries
//   - BatchGetTraces
//   - CreateGroup
//   - GetGroup
//   - DeleteGroup
//   - GetGroups
package xray

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the X-Ray mock.
type Service struct {
	mu       sync.RWMutex
	segments map[string]*traceSegment // keyed by segmentId
	groups   map[string]*group        // keyed by group name
}

type traceSegment struct {
	traceId   string
	segmentId string
	document  string
	storedAt  time.Time
}

type group struct {
	name             string
	arn              string
	filterExpression string
}

// New creates a new X-Ray mock service.
func New() *Service {
	return &Service{
		segments: make(map[string]*traceSegment),
		groups:   make(map[string]*group),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "xray" }

// Handler returns the HTTP handler for X-Ray requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.segments = make(map[string]*traceSegment)
	s.groups = make(map[string]*group)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if r.Method != http.MethodPost {
		h.WriteJSONError(w, "InvalidAction", "unsupported method", http.StatusBadRequest)
		return
	}

	switch path {
	case "/TraceSegments":
		s.putTraceSegments(w, r)
	case "/TraceSummaries":
		s.getTraceSummaries(w, r)
	case "/Traces":
		s.batchGetTraces(w, r)
	case "/CreateGroup":
		s.createGroup(w, r)
	case "/GetGroup":
		s.getGroup(w, r)
	case "/DeleteGroup":
		s.deleteGroup(w, r)
	case "/Groups":
		s.getGroups(w, r)
	default:
		h.WriteJSONError(w, "InvalidAction", "unsupported operation", http.StatusBadRequest)
	}
}

func (s *Service) putTraceSegments(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteJSONError(w, "ServiceException", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &params); err != nil {
		h.WriteJSONError(w, "InvalidRequestException", "could not parse request body", http.StatusBadRequest)
		return
	}

	docs, _ := params["TraceSegmentDocuments"].([]interface{})

	s.mu.Lock()
	for _, doc := range docs {
		docStr, ok := doc.(string)
		if !ok {
			continue
		}
		// Parse the segment document to extract trace and segment IDs.
		var segDoc map[string]interface{}
		if err := json.Unmarshal([]byte(docStr), &segDoc); err != nil {
			continue
		}
		traceID := h.GetString(segDoc, "trace_id")
		segmentID := h.GetString(segDoc, "id")
		if traceID == "" || segmentID == "" {
			continue
		}
		s.segments[segmentID] = &traceSegment{
			traceId:   traceID,
			segmentId: segmentID,
			document:  docStr,
			storedAt:  time.Now().UTC(),
		}
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"UnprocessedTraceSegments": []interface{}{},
	})
}

func (s *Service) getTraceSummaries(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &params)
	}

	s.mu.RLock()
	// Collect unique trace IDs.
	traceMap := make(map[string]time.Time)
	for _, seg := range s.segments {
		if existing, ok := traceMap[seg.traceId]; !ok || seg.storedAt.Before(existing) {
			traceMap[seg.traceId] = seg.storedAt
		}
	}
	s.mu.RUnlock()

	var summaries []map[string]interface{}
	for traceID, storedAt := range traceMap {
		summaries = append(summaries, map[string]interface{}{
			"Id": traceID,
			"Http": map[string]interface{}{
				"HttpURL":    "https://example.com",
				"HttpMethod": "GET",
				"HttpStatus": 200,
			},
			"Duration":      0.1,
			"ResponseTime":  storedAt.Unix(),
			"HasFault":      false,
			"HasError":      false,
			"HasThrottle":   false,
			"IsPartial":     false,
			"Revision":      0,
			"MatchedEventTime": storedAt.Unix(),
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i]["Id"].(string) < summaries[j]["Id"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"TraceSummaries":    summaries,
		"ApproximateTime":   time.Now().UTC().Unix(),
		"TracesProcessedCount": len(summaries),
	})
}

func (s *Service) batchGetTraces(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &params)
	}

	traceIDs, _ := params["TraceIds"].([]interface{})

	requested := make(map[string]bool)
	for _, id := range traceIDs {
		if s, ok := id.(string); ok {
			requested[s] = true
		}
	}

	s.mu.RLock()
	// Group segments by trace ID.
	traceSegments := make(map[string][]map[string]interface{})
	for _, seg := range s.segments {
		if !requested[seg.traceId] {
			continue
		}
		traceSegments[seg.traceId] = append(traceSegments[seg.traceId], map[string]interface{}{
			"Id":       seg.segmentId,
			"Document": seg.document,
		})
	}
	s.mu.RUnlock()

	var traces []map[string]interface{}
	for traceID, segs := range traceSegments {
		traces = append(traces, map[string]interface{}{
			"Id":       traceID,
			"Duration": 0.1,
			"Segments": segs,
		})
	}

	sort.Slice(traces, func(i, j int) bool {
		return traces[i]["Id"].(string) < traces[j]["Id"].(string)
	})

	// Unprocessed trace IDs are those requested but not found.
	var unprocessed []string
	for _, id := range traceIDs {
		if s, ok := id.(string); ok {
			if _, found := traceSegments[s]; !found {
				unprocessed = append(unprocessed, s)
			}
		}
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Traces":                traces,
		"UnprocessedTraceIds":   unprocessed,
	})
}

func (s *Service) createGroup(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteJSONError(w, "ServiceException", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &params); err != nil {
		h.WriteJSONError(w, "InvalidRequestException", "could not parse request body", http.StatusBadRequest)
		return
	}

	name := h.GetString(params, "GroupName")
	if name == "" {
		h.WriteJSONError(w, "InvalidRequestException", "GroupName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.groups[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "InvalidRequestException", "Group already exists: "+name, http.StatusConflict)
		return
	}

	g := &group{
		name:             name,
		arn:              fmt.Sprintf("arn:aws:xray:us-east-1:%s:group/%s/%s", h.DefaultAccountID, name, h.RandomHex(16)),
		filterExpression: h.GetString(params, "FilterExpression"),
	}
	s.groups[name] = g
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Group": groupResp(g),
	})
}

func (s *Service) getGroup(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &params)
	}

	name := h.GetString(params, "GroupName")
	arn := h.GetString(params, "GroupARN")

	s.mu.RLock()
	g := s.findGroup(name, arn)
	s.mu.RUnlock()

	if g == nil {
		h.WriteJSONError(w, "InvalidRequestException", "Group not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Group": groupResp(g),
	})
}

func (s *Service) deleteGroup(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &params)
	}

	name := h.GetString(params, "GroupName")
	arn := h.GetString(params, "GroupARN")

	s.mu.Lock()
	g := s.findGroup(name, arn)
	if g == nil {
		s.mu.Unlock()
		h.WriteJSONError(w, "InvalidRequestException", "Group not found", http.StatusNotFound)
		return
	}
	delete(s.groups, g.name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) getGroups(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var items []map[string]interface{}
	for _, g := range s.groups {
		items = append(items, groupResp(g))
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i]["GroupName"].(string) < items[j]["GroupName"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Groups": items,
	})
}

// findGroup looks up a group by name or ARN. Must be called with s.mu held.
func (s *Service) findGroup(name, arn string) *group {
	if name != "" {
		return s.groups[name]
	}
	for _, g := range s.groups {
		if g.arn == arn {
			return g
		}
	}
	return nil
}

func groupResp(g *group) map[string]interface{} {
	return map[string]interface{}{
		"GroupName":        g.name,
		"GroupARN":         g.arn,
		"FilterExpression": g.filterExpression,
	}
}

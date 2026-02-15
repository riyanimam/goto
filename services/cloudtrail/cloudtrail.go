// Package cloudtrail provides a mock implementation of AWS CloudTrail.
//
// Supported actions:
//   - CreateTrail
//   - GetTrail
//   - DeleteTrail
//   - DescribeTrails
//   - StartLogging
//   - StopLogging
//   - GetTrailStatus
//   - LookupEvents
package cloudtrail

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

// Service implements the CloudTrail mock.
type Service struct {
	mu     sync.RWMutex
	trails map[string]*trail
}

type trail struct {
	name                string
	arn                 string
	s3BucketName        string
	isMultiRegion       bool
	isOrganizationTrail bool
	isLogging           bool
	homeRegion          string
	created             time.Time
}

// New creates a new CloudTrail mock service.
func New() *Service {
	return &Service{
		trails: make(map[string]*trail),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "cloudtrail" }

// Handler returns the HTTP handler for CloudTrail requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trails = make(map[string]*trail)
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
	case "CreateTrail":
		s.createTrail(w, params)
	case "GetTrail":
		s.getTrail(w, params)
	case "DeleteTrail":
		s.deleteTrail(w, params)
	case "DescribeTrails":
		s.describeTrails(w, params)
	case "StartLogging":
		s.startLogging(w, params)
	case "StopLogging":
		s.stopLogging(w, params)
	case "GetTrailStatus":
		s.getTrailStatus(w, params)
	case "LookupEvents":
		s.lookupEvents(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createTrail(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterException", "Name is required", http.StatusBadRequest)
		return
	}

	s3Bucket := h.GetString(params, "S3BucketName")
	if s3Bucket == "" {
		h.WriteJSONError(w, "InvalidParameterException", "S3BucketName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.trails[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "TrailAlreadyExistsException", "Trail already exists: "+name, http.StatusBadRequest)
		return
	}

	arn := fmt.Sprintf("arn:aws:cloudtrail:us-east-1:%s:trail/%s", h.DefaultAccountID, name)
	t := &trail{
		name:                name,
		arn:                 arn,
		s3BucketName:        s3Bucket,
		isMultiRegion:       h.GetBool(params, "IsMultiRegionTrail"),
		isOrganizationTrail: h.GetBool(params, "IsOrganizationTrail"),
		homeRegion:          "us-east-1",
		created:             time.Now().UTC(),
	}
	s.trails[name] = t
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, trailResp(t))
}

func (s *Service) getTrail(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")

	s.mu.RLock()
	t, exists := s.trails[name]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "TrailNotFoundException", "Trail not found: "+name, http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Trail": trailResp(t),
	})
}

func (s *Service) deleteTrail(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")

	s.mu.Lock()
	if _, exists := s.trails[name]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "TrailNotFoundException", "Trail not found: "+name, http.StatusBadRequest)
		return
	}
	delete(s.trails, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) describeTrails(w http.ResponseWriter, params map[string]interface{}) {
	var filter map[string]bool
	if names, ok := params["trailNameList"].([]interface{}); ok && len(names) > 0 {
		filter = make(map[string]bool, len(names))
		for _, n := range names {
			if name, ok := n.(string); ok {
				filter[name] = true
			}
		}
	}

	s.mu.RLock()
	var list []map[string]interface{}
	for _, t := range s.trails {
		if filter != nil && !filter[t.name] {
			continue
		}
		list = append(list, trailResp(t))
	}
	s.mu.RUnlock()

	if list == nil {
		list = []map[string]interface{}{}
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"trailList": list,
	})
}

func (s *Service) startLogging(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")

	s.mu.Lock()
	t, exists := s.trails[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "TrailNotFoundException", "Trail not found: "+name, http.StatusBadRequest)
		return
	}
	t.isLogging = true
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) stopLogging(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")

	s.mu.Lock()
	t, exists := s.trails[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "TrailNotFoundException", "Trail not found: "+name, http.StatusBadRequest)
		return
	}
	t.isLogging = false
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) getTrailStatus(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")

	s.mu.RLock()
	t, exists := s.trails[name]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "TrailNotFoundException", "Trail not found: "+name, http.StatusBadRequest)
		return
	}

	resp := map[string]interface{}{
		"IsLogging":          t.isLogging,
		"LatestDeliveryTime": float64(time.Now().UTC().Unix()),
	}

	h.WriteJSON(w, http.StatusOK, resp)
}

func (s *Service) lookupEvents(w http.ResponseWriter, _ map[string]interface{}) {
	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Events": []interface{}{},
	})
}

func trailResp(t *trail) map[string]interface{} {
	return map[string]interface{}{
		"Name":                    t.name,
		"TrailARN":                t.arn,
		"S3BucketName":            t.s3BucketName,
		"IsMultiRegionTrail":      t.isMultiRegion,
		"IsOrganizationTrail":     t.isOrganizationTrail,
		"HomeRegion":              t.homeRegion,
		"LogFileValidationEnabled": true,
	}
}

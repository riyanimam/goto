// Package guardduty provides a mock implementation of AWS GuardDuty.
//
// Supported actions:
//   - CreateDetector
//   - GetDetector
//   - DeleteDetector
//   - ListDetectors
//   - UpdateDetector
package guardduty

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

// Service implements the GuardDuty mock.
type Service struct {
	mu        sync.RWMutex
	detectors map[string]*detector
}

type detector struct {
	id                         string
	status                     string
	findingPublishingFrequency string
	serviceRole                string
	created                    time.Time
	updated                    time.Time
}

// New creates a new GuardDuty mock service.
func New() *Service {
	return &Service{
		detectors: make(map[string]*detector),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "guardduty" }

// Handler returns the HTTP handler for GuardDuty requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.detectors = make(map[string]*detector)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	switch {
	// Single detector: /detector/{detectorId}
	case strings.HasPrefix(path, "/detector/") && method == http.MethodGet:
		s.getDetector(w, r, path)
	case strings.HasPrefix(path, "/detector/") && method == http.MethodDelete:
		s.deleteDetector(w, r, path)
	case strings.HasPrefix(path, "/detector/") && method == http.MethodPost:
		s.updateDetector(w, r, path)

	// Detector collection: /detector
	case path == "/detector" && method == http.MethodPost:
		s.createDetector(w, r)
	case path == "/detector" && method == http.MethodGet:
		s.listDetectors(w, r)

	default:
		h.WriteJSONError(w, "NotFoundException", "unsupported operation", http.StatusNotFound)
	}
}

func extractDetectorID(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func (s *Service) createDetector(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	frequency := h.GetString(params, "findingPublishingFrequency")
	if frequency == "" {
		frequency = "SIX_HOURS"
	}

	id := h.RandomHex(32)
	now := time.Now().UTC()

	d := &detector{
		id:                         id,
		status:                     "ENABLED",
		findingPublishingFrequency: frequency,
		serviceRole:                fmt.Sprintf("arn:aws:iam::%s:role/aws-service-role/guardduty.amazonaws.com/AWSServiceRoleForAmazonGuardDuty", h.DefaultAccountID),
		created:                    now,
		updated:                    now,
	}

	s.mu.Lock()
	s.detectors[id] = d
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"detectorId": id,
	})
}

func (s *Service) getDetector(w http.ResponseWriter, _ *http.Request, path string) {
	id := extractDetectorID(path)

	s.mu.RLock()
	d, exists := s.detectors[id]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "BadRequestException", "The request is rejected because the input detectorId is not owned by the current account.", http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, detectorResp(d))
}

func (s *Service) deleteDetector(w http.ResponseWriter, _ *http.Request, path string) {
	id := extractDetectorID(path)

	s.mu.Lock()
	_, exists := s.detectors[id]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "BadRequestException", "The request is rejected because the input detectorId is not owned by the current account.", http.StatusBadRequest)
		return
	}
	delete(s.detectors, id)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listDetectors(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var ids []string
	for id := range s.detectors {
		ids = append(ids, id)
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"detectorIds": ids,
	})
}

func (s *Service) updateDetector(w http.ResponseWriter, r *http.Request, path string) {
	id := extractDetectorID(path)

	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	s.mu.Lock()
	d, exists := s.detectors[id]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "BadRequestException", "The request is rejected because the input detectorId is not owned by the current account.", http.StatusBadRequest)
		return
	}

	if freq := h.GetString(params, "findingPublishingFrequency"); freq != "" {
		d.findingPublishingFrequency = freq
	}
	d.updated = time.Now().UTC()
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func detectorResp(d *detector) map[string]interface{} {
	return map[string]interface{}{
		"createdAt":                  d.created.Format(time.RFC3339),
		"updatedAt":                  d.updated.Format(time.RFC3339),
		"status":                     d.status,
		"findingPublishingFrequency": d.findingPublishingFrequency,
		"serviceRole":                d.serviceRole,
	}
}

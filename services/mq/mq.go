// Package mq provides a mock implementation of Amazon MQ.
//
// Supported actions:
//   - CreateBroker
//   - DescribeBroker
//   - DeleteBroker
//   - ListBrokers
//   - UpdateBroker
package mq

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

// Service implements the Amazon MQ mock.
type Service struct {
	mu      sync.RWMutex
	brokers map[string]*broker
}

type broker struct {
	brokerID           string
	brokerArn          string
	brokerName         string
	brokerState        string
	engineType         string
	engineVersion      string
	hostInstanceType   string
	deploymentMode     string
	publiclyAccessible bool
	created            time.Time
}

// New creates a new Amazon MQ mock service.
func New() *Service {
	return &Service{
		brokers: make(map[string]*broker),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "mq" }

// Handler returns the HTTP handler for Amazon MQ requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.brokers = make(map[string]*broker)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	switch {
	// Single broker: /v1/brokers/{brokerId}
	case strings.HasPrefix(path, "/v1/brokers/") && method == http.MethodGet:
		s.describeBroker(w, r, path)
	case strings.HasPrefix(path, "/v1/brokers/") && method == http.MethodDelete:
		s.deleteBroker(w, r, path)
	case strings.HasPrefix(path, "/v1/brokers/") && method == http.MethodPut:
		s.updateBroker(w, r, path)

	// Brokers list: /v1/brokers
	case path == "/v1/brokers" && method == http.MethodPost:
		s.createBroker(w, r)
	case path == "/v1/brokers" && method == http.MethodGet:
		s.listBrokers(w, r)

	default:
		h.WriteJSONError(w, "NotFoundException", "unsupported operation", http.StatusNotFound)
	}
}

func extractBrokerID(path string) string {
	// path: /v1/brokers/{brokerId}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func (s *Service) createBroker(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	brokerName := h.GetString(params, "brokerName")
	if brokerName == "" {
		h.WriteJSONError(w, "BadRequestException", "brokerName is required", http.StatusBadRequest)
		return
	}

	engineType := h.GetString(params, "engineType")
	if engineType == "" {
		engineType = "ACTIVEMQ"
	}
	engineVersion := h.GetString(params, "engineVersion")
	hostInstanceType := h.GetString(params, "hostInstanceType")
	deploymentMode := h.GetString(params, "deploymentMode")
	publiclyAccessible := h.GetBool(params, "publiclyAccessible")

	brokerID := h.RandomHex(36)
	arn := fmt.Sprintf("arn:aws:mq:us-east-1:%s:broker:%s:%s", h.DefaultAccountID, brokerName, brokerID)
	now := time.Now().UTC()

	b := &broker{
		brokerID:           brokerID,
		brokerArn:          arn,
		brokerName:         brokerName,
		brokerState:        "RUNNING",
		engineType:         engineType,
		engineVersion:      engineVersion,
		hostInstanceType:   hostInstanceType,
		deploymentMode:     deploymentMode,
		publiclyAccessible: publiclyAccessible,
		created:            now,
	}

	s.mu.Lock()
	s.brokers[brokerID] = b
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"brokerId":  brokerID,
		"brokerArn": arn,
	})
}

func (s *Service) describeBroker(w http.ResponseWriter, _ *http.Request, path string) {
	brokerID := extractBrokerID(path)

	s.mu.RLock()
	b, exists := s.brokers[brokerID]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "NotFoundException", "Broker "+brokerID+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, brokerResp(b))
}

func (s *Service) deleteBroker(w http.ResponseWriter, _ *http.Request, path string) {
	brokerID := extractBrokerID(path)

	s.mu.Lock()
	_, exists := s.brokers[brokerID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "Broker "+brokerID+" not found", http.StatusNotFound)
		return
	}
	delete(s.brokers, brokerID)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"brokerId": brokerID,
	})
}

func (s *Service) listBrokers(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	summaries := make([]map[string]interface{}, 0, len(s.brokers))
	for _, b := range s.brokers {
		summaries = append(summaries, map[string]interface{}{
			"brokerId":       b.brokerID,
			"brokerArn":      b.brokerArn,
			"brokerName":     b.brokerName,
			"brokerState":    b.brokerState,
			"engineType":     b.engineType,
			"deploymentMode": b.deploymentMode,
			"created":        b.created.Format(time.RFC3339),
		})
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"brokerSummaries": summaries,
	})
}

func (s *Service) updateBroker(w http.ResponseWriter, r *http.Request, path string) {
	brokerID := extractBrokerID(path)

	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	s.mu.Lock()
	b, exists := s.brokers[brokerID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "Broker "+brokerID+" not found", http.StatusNotFound)
		return
	}

	if v := h.GetString(params, "engineVersion"); v != "" {
		b.engineVersion = v
	}
	if v := h.GetString(params, "hostInstanceType"); v != "" {
		b.hostInstanceType = v
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"brokerId":  brokerID,
		"brokerArn": b.brokerArn,
	})
}

func brokerResp(b *broker) map[string]interface{} {
	return map[string]interface{}{
		"brokerId":         b.brokerID,
		"brokerArn":        b.brokerArn,
		"brokerName":       b.brokerName,
		"brokerState":      b.brokerState,
		"engineType":       b.engineType,
		"engineVersion":    b.engineVersion,
		"hostInstanceType": b.hostInstanceType,
		"deploymentMode":   b.deploymentMode,
		"created":          b.created.Format(time.RFC3339),
	}
}

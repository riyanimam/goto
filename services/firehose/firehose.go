// Package firehose provides a mock implementation of AWS Kinesis Data Firehose.
//
// Supported actions:
//   - CreateDeliveryStream
//   - DeleteDeliveryStream
//   - DescribeDeliveryStream
//   - ListDeliveryStreams
//   - PutRecord
package firehose

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

// Service implements the Firehose mock.
type Service struct {
	mu      sync.RWMutex
	streams map[string]*deliveryStream
}

type deliveryStream struct {
	name    string
	arn     string
	status  string
	destID  string
	created time.Time
	records [][]byte
}

// New creates a new Firehose mock service.
func New() *Service {
	return &Service{
		streams: make(map[string]*deliveryStream),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "firehose" }

// Handler returns the HTTP handler for Firehose requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams = make(map[string]*deliveryStream)
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
	case "CreateDeliveryStream":
		s.createDeliveryStream(w, params)
	case "DeleteDeliveryStream":
		s.deleteDeliveryStream(w, params)
	case "DescribeDeliveryStream":
		s.describeDeliveryStream(w, params)
	case "ListDeliveryStreams":
		s.listDeliveryStreams(w, params)
	case "PutRecord":
		s.putRecord(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createDeliveryStream(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "DeliveryStreamName")
	if name == "" {
		h.WriteJSONError(w, "InvalidArgumentException", "DeliveryStreamName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.streams[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceInUseException", "Delivery stream "+name+" already exists", http.StatusConflict)
		return
	}

	arn := fmt.Sprintf("arn:aws:firehose:us-east-1:%s:deliverystream/%s", h.DefaultAccountID, name)
	ds := &deliveryStream{
		name:    name,
		arn:     arn,
		status:  "ACTIVE",
		destID:  "destinationId-" + h.RandomHex(12),
		created: time.Now().UTC(),
	}
	s.streams[name] = ds
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"DeliveryStreamARN": arn,
	})
}

func (s *Service) deleteDeliveryStream(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "DeliveryStreamName")

	s.mu.Lock()
	if _, exists := s.streams[name]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Delivery stream "+name+" not found", http.StatusNotFound)
		return
	}
	delete(s.streams, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) describeDeliveryStream(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "DeliveryStreamName")

	s.mu.RLock()
	ds, exists := s.streams[name]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Delivery stream "+name+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"DeliveryStreamDescription": map[string]interface{}{
			"DeliveryStreamName":   ds.name,
			"DeliveryStreamARN":    ds.arn,
			"DeliveryStreamStatus": ds.status,
			"DeliveryStreamType":   "DirectPut",
			"CreateTimestamp":      float64(ds.created.Unix()),
			"HasMoreDestinations":  false,
			"Destinations": []map[string]interface{}{
				{
					"DestinationId": ds.destID,
				},
			},
		},
	})
}

func (s *Service) listDeliveryStreams(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var names []string
	for name := range s.streams {
		names = append(names, name)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"DeliveryStreamNames":    names,
		"HasMoreDeliveryStreams": false,
	})
}

func (s *Service) putRecord(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "DeliveryStreamName")

	s.mu.Lock()
	ds, exists := s.streams[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Delivery stream "+name+" not found", http.StatusNotFound)
		return
	}

	if record, ok := params["Record"].(map[string]interface{}); ok {
		if data, ok := record["Data"].(string); ok {
			ds.records = append(ds.records, []byte(data))
		}
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"RecordId":  h.NewRequestID(),
		"Encrypted": false,
	})
}

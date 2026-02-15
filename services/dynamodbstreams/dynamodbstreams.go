// Package dynamodbstreams provides a mock implementation of AWS DynamoDB Streams.
//
// Supported actions:
//   - ListStreams
//   - DescribeStream
//   - GetShardIterator
//   - GetRecords
package dynamodbstreams

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the DynamoDB Streams mock.
type Service struct {
	mu             sync.RWMutex
	streams        map[string]*stream
	shardIterators map[string]*shardIterator
}

type stream struct {
	arn       string
	label     string
	tableName string
	status    string
	shards    []shard
}

type shard struct {
	shardID       string
	parentShardID string
}

type shardIterator struct {
	streamArn string
	shardID   string
}

// New creates a new DynamoDB Streams mock service.
func New() *Service {
	return &Service{
		streams:        make(map[string]*stream),
		shardIterators: make(map[string]*shardIterator),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "streams.dynamodb" }

// Handler returns the HTTP handler for DynamoDB Streams requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams = make(map[string]*stream)
	s.shardIterators = make(map[string]*shardIterator)
}

// AddStream adds a stream programmatically (e.g. from the DynamoDB service).
func (s *Service) AddStream(arn, label, tableName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams[arn] = &stream{
		arn:       arn,
		label:     label,
		tableName: tableName,
		status:    "ENABLED",
		shards: []shard{
			{shardID: "shardId-" + h.RandomHex(32)},
		},
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
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			h.WriteJSONError(w, "SerializationException", "could not parse request body", http.StatusBadRequest)
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
	case "ListStreams":
		s.listStreams(w, params)
	case "DescribeStream":
		s.describeStream(w, params)
	case "GetShardIterator":
		s.getShardIterator(w, params)
	case "GetRecords":
		s.getRecords(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) listStreams(w http.ResponseWriter, params map[string]interface{}) {
	tableFilter := h.GetString(params, "TableName")

	s.mu.RLock()
	var result []map[string]interface{}
	for _, st := range s.streams {
		if tableFilter != "" && st.tableName != tableFilter {
			continue
		}
		result = append(result, map[string]interface{}{
			"StreamArn":   st.arn,
			"StreamLabel": st.label,
			"TableName":   st.tableName,
		})
	}
	s.mu.RUnlock()

	if result == nil {
		result = []map[string]interface{}{}
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Streams": result,
	})
}

func (s *Service) describeStream(w http.ResponseWriter, params map[string]interface{}) {
	arn := h.GetString(params, "StreamArn")
	if arn == "" {
		h.WriteJSONError(w, "ValidationException", "StreamArn is required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	st, exists := s.streams[arn]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Requested resource not found: Stream: "+arn+" not found", http.StatusBadRequest)
		return
	}

	shards := make([]map[string]interface{}, len(st.shards))
	for i, sh := range st.shards {
		entry := map[string]interface{}{
			"ShardId": sh.shardID,
		}
		if sh.parentShardID != "" {
			entry["ParentShardId"] = sh.parentShardID
		}
		shards[i] = entry
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"StreamDescription": map[string]interface{}{
			"StreamArn":    st.arn,
			"StreamLabel":  st.label,
			"StreamStatus": st.status,
			"TableName":    st.tableName,
			"Shards":       shards,
		},
	})
}

func (s *Service) getShardIterator(w http.ResponseWriter, params map[string]interface{}) {
	arn := h.GetString(params, "StreamArn")
	shardID := h.GetString(params, "ShardId")
	iterType := h.GetString(params, "ShardIteratorType")

	if arn == "" || shardID == "" || iterType == "" {
		h.WriteJSONError(w, "ValidationException", "StreamArn, ShardId, and ShardIteratorType are required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	st, exists := s.streams[arn]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Requested resource not found: Stream: "+arn+" not found", http.StatusBadRequest)
		return
	}

	found := false
	for _, sh := range st.shards {
		if sh.shardID == shardID {
			found = true
			break
		}
	}
	if !found {
		h.WriteJSONError(w, "ResourceNotFoundException", "Requested resource not found: Shard: "+shardID+" in Stream: "+arn, http.StatusBadRequest)
		return
	}

	iterToken := h.RandomHex(64)

	s.mu.Lock()
	s.shardIterators[iterToken] = &shardIterator{
		streamArn: arn,
		shardID:   shardID,
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ShardIterator": iterToken,
	})
}

func (s *Service) getRecords(w http.ResponseWriter, params map[string]interface{}) {
	iter := h.GetString(params, "ShardIterator")
	if iter == "" {
		h.WriteJSONError(w, "ValidationException", "ShardIterator is required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	_, exists := s.shardIterators[iter]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ExpiredIteratorException", "Iterator expired or not found", http.StatusBadRequest)
		return
	}

	nextToken := h.RandomHex(64)

	s.mu.Lock()
	si := s.shardIterators[iter]
	s.shardIterators[nextToken] = &shardIterator{
		streamArn: si.streamArn,
		shardID:   si.shardID,
	}
	delete(s.shardIterators, iter)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Records":           []interface{}{},
		"NextShardIterator": nextToken,
	})
}

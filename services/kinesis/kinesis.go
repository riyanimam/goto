// Package kinesis provides a mock implementation of AWS Kinesis Data Streams.
//
// Supported actions:
//   - CreateStream
//   - DeleteStream
//   - DescribeStream
//   - ListStreams
//   - PutRecord
//   - GetRecords
//   - GetShardIterator
package kinesis

import (
	"encoding/base64"
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

// Service implements the Kinesis mock.
type Service struct {
	mu      sync.RWMutex
	streams map[string]*stream
}

type stream struct {
	name       string
	arn        string
	status     string
	shardCount int
	records    []*record
	created    time.Time
	mu         sync.Mutex
}

type record struct {
	sequenceNumber string
	partitionKey   string
	data           []byte
	timestamp      time.Time
}

// New creates a new Kinesis mock service.
func New() *Service {
	return &Service{
		streams: make(map[string]*stream),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "kinesis" }

// Handler returns the HTTP handler for Kinesis requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all streams and records.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams = make(map[string]*stream)
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
			writeJSONError(w, "SerializationException", "could not parse request body", http.StatusBadRequest)
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
	case "CreateStream":
		s.createStream(w, params)
	case "DeleteStream":
		s.deleteStream(w, params)
	case "DescribeStream":
		s.describeStream(w, params)
	case "ListStreams":
		s.listStreams(w, params)
	case "PutRecord":
		s.putRecord(w, params)
	case "GetRecords":
		s.getRecords(w, params)
	case "GetShardIterator":
		s.getShardIterator(w, params)
	default:
		writeJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createStream(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "StreamName")
	if name == "" {
		writeJSONError(w, "ValidationException", "StreamName is required", http.StatusBadRequest)
		return
	}

	shardCount := getInt(params, "ShardCount", 1)

	s.mu.Lock()
	if _, exists := s.streams[name]; exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceInUseException", "Stream "+name+" under account "+defaultAccountID+" already exists.", http.StatusBadRequest)
		return
	}

	s.streams[name] = &stream{
		name:       name,
		arn:        fmt.Sprintf("arn:aws:kinesis:us-east-1:%s:stream/%s", defaultAccountID, name),
		status:     "ACTIVE",
		shardCount: shardCount,
		created:    time.Now().UTC(),
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) deleteStream(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "StreamName")
	if name == "" {
		name = getString(params, "StreamARN")
	}

	s.mu.Lock()
	if _, exists := s.streams[name]; !exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "Stream "+name+" under account "+defaultAccountID+" not found.", http.StatusBadRequest)
		return
	}
	delete(s.streams, name)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) describeStream(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "StreamName")

	s.mu.RLock()
	st, exists := s.streams[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Stream "+name+" under account "+defaultAccountID+" not found.", http.StatusBadRequest)
		return
	}

	var shards []map[string]interface{}
	for i := 0; i < st.shardCount; i++ {
		shards = append(shards, map[string]interface{}{
			"ShardId": fmt.Sprintf("shardId-%012d", i),
			"HashKeyRange": map[string]interface{}{
				"StartingHashKey": "0",
				"EndingHashKey":   "340282366920938463463374607431768211455",
			},
			"SequenceNumberRange": map[string]interface{}{
				"StartingSequenceNumber": "0",
			},
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"StreamDescription": map[string]interface{}{
			"StreamName":              st.name,
			"StreamARN":               st.arn,
			"StreamStatus":            st.status,
			"Shards":                  shards,
			"HasMoreShards":           false,
			"RetentionPeriodHours":    24,
			"StreamCreationTimestamp": float64(st.created.Unix()),
		},
	})
}

func (s *Service) listStreams(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var names []string
	for name := range s.streams {
		names = append(names, name)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"StreamNames":    names,
		"HasMoreStreams": false,
	})
}

func (s *Service) putRecord(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "StreamName")
	partKey := getString(params, "PartitionKey")
	dataB64 := getString(params, "Data")

	s.mu.RLock()
	st, exists := s.streams[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Stream "+name+" under account "+defaultAccountID+" not found.", http.StatusBadRequest)
		return
	}

	data, _ := base64.StdEncoding.DecodeString(dataB64)

	seqNum := fmt.Sprintf("%020d", time.Now().UnixNano())
	rec := &record{
		sequenceNumber: seqNum,
		partitionKey:   partKey,
		data:           data,
		timestamp:      time.Now().UTC(),
	}

	st.mu.Lock()
	st.records = append(st.records, rec)
	st.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ShardId":        "shardId-000000000000",
		"SequenceNumber": seqNum,
	})
}

func (s *Service) getShardIterator(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "StreamName")

	s.mu.RLock()
	_, exists := s.streams[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Stream "+name+" under account "+defaultAccountID+" not found.", http.StatusBadRequest)
		return
	}

	// Return a simple iterator token.
	iterator := base64.StdEncoding.EncodeToString([]byte(name + ":0"))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ShardIterator": iterator,
	})
}

func (s *Service) getRecords(w http.ResponseWriter, params map[string]interface{}) {
	iteratorToken := getString(params, "ShardIterator")

	// Decode the stream name from the iterator.
	decoded, err := base64.StdEncoding.DecodeString(iteratorToken)
	if err != nil {
		writeJSONError(w, "InvalidArgumentException", "Invalid ShardIterator", http.StatusBadRequest)
		return
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	name := parts[0]

	s.mu.RLock()
	st, exists := s.streams[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Stream not found.", http.StatusBadRequest)
		return
	}

	st.mu.Lock()
	var records []map[string]interface{}
	for _, rec := range st.records {
		records = append(records, map[string]interface{}{
			"SequenceNumber":              rec.sequenceNumber,
			"PartitionKey":                rec.partitionKey,
			"Data":                        base64.StdEncoding.EncodeToString(rec.data),
			"ApproximateArrivalTimestamp": float64(rec.timestamp.Unix()),
		})
	}
	st.mu.Unlock()

	nextIterator := base64.StdEncoding.EncodeToString([]byte(name + ":" + fmt.Sprintf("%d", len(records))))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Records":            records,
		"NextShardIterator":  nextIterator,
		"MillisBehindLatest": 0,
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

func getInt(params map[string]interface{}, key string, defaultVal int) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
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

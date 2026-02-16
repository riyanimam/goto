// Package kafka provides a mock implementation of Amazon Managed Streaming for Apache Kafka (MSK).
//
// Supported actions:
//   - CreateCluster
//   - DescribeCluster
//   - DeleteCluster
//   - ListClusters
//   - UpdateBrokerCount
package kafka

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

// Service implements the MSK mock.
type Service struct {
	mu       sync.RWMutex
	clusters map[string]*cluster // keyed by ARN
}

type cluster struct {
	name            string
	arn             string
	state           string
	kafkaVersion    string
	numberOfBrokers int
	instanceType    string
	currentVersion  string
	created         time.Time
}

// New creates a new MSK mock service.
func New() *Service {
	return &Service{
		clusters: make(map[string]*cluster),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "kafka" }

// Handler returns the HTTP handler for MSK requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clusters = make(map[string]*cluster)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	switch {
	// UpdateBrokerCount: PUT /v1/clusters/{clusterArn}/nodes/count
	case strings.HasSuffix(path, "/nodes/count") && method == http.MethodPut:
		s.updateBrokerCount(w, r, path)

	// Single cluster: /v1/clusters/{clusterArn}
	case strings.HasPrefix(path, "/v1/clusters/") && method == http.MethodGet:
		s.describeCluster(w, r, path)
	case strings.HasPrefix(path, "/v1/clusters/") && method == http.MethodDelete:
		s.deleteCluster(w, r, path)

	// Cluster collection: /v1/clusters
	case path == "/v1/clusters" && method == http.MethodPost:
		s.createCluster(w, r)
	case path == "/v1/clusters" && method == http.MethodGet:
		s.listClusters(w, r)

	default:
		h.WriteJSONError(w, "NotFoundException", "unsupported operation", http.StatusNotFound)
	}
}

func extractClusterArn(path string) string {
	// Path format: /v1/clusters/{clusterArn...}
	// Strip the /v1/clusters/ prefix and any trailing segments like /nodes/count.
	trimmed := strings.TrimPrefix(path, "/v1/clusters/")
	if idx := strings.Index(trimmed, "/nodes/"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return trimmed
}

func (s *Service) createCluster(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	name := h.GetString(params, "clusterName")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterException", "clusterName is required", http.StatusBadRequest)
		return
	}

	kafkaVersion := h.GetString(params, "kafkaVersion")
	if kafkaVersion == "" {
		kafkaVersion = "3.5.1"
	}

	numberOfBrokers := h.GetInt(params, "numberOfBrokerNodes", 3)

	instanceType := ""
	if bng, ok := params["brokerNodeGroupInfo"].(map[string]interface{}); ok {
		instanceType = h.GetString(bng, "instanceType")
	}
	if instanceType == "" {
		instanceType = "kafka.m5.large"
	}

	s.mu.Lock()
	// Check for duplicate name.
	for _, c := range s.clusters {
		if c.name == name {
			s.mu.Unlock()
			h.WriteJSONError(w, "ConflictException", "Cluster "+name+" already exists", http.StatusConflict)
			return
		}
	}

	clusterID := h.RandomHex(12)
	arn := fmt.Sprintf("arn:aws:kafka:us-east-1:%s:cluster/%s/%s", h.DefaultAccountID, name, clusterID)
	now := time.Now().UTC()

	c := &cluster{
		name:            name,
		arn:             arn,
		state:           "CREATING",
		kafkaVersion:    kafkaVersion,
		numberOfBrokers: numberOfBrokers,
		instanceType:    instanceType,
		currentVersion:  "K1" + h.RandomHex(6),
		created:         now,
	}
	s.clusters[arn] = c
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"clusterArn":  arn,
		"clusterName": name,
		"state":       "CREATING",
	})
}

func (s *Service) describeCluster(w http.ResponseWriter, _ *http.Request, path string) {
	arnSegment := extractClusterArn(path)

	s.mu.RLock()
	c := s.findClusterByArnSuffix(arnSegment)
	s.mu.RUnlock()

	if c == nil {
		h.WriteJSONError(w, "NotFoundException", "Cluster not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"clusterInfo": clusterInfoResp(c),
	})
}

func (s *Service) deleteCluster(w http.ResponseWriter, _ *http.Request, path string) {
	arnSegment := extractClusterArn(path)

	s.mu.Lock()
	c := s.findClusterByArnSuffix(arnSegment)
	if c == nil {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "Cluster not found", http.StatusNotFound)
		return
	}
	c.state = "DELETING"
	arn := c.arn
	delete(s.clusters, arn)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"clusterArn": arn,
		"state":      "DELETING",
	})
}

func (s *Service) listClusters(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var list []map[string]interface{}
	for _, c := range s.clusters {
		list = append(list, clusterInfoResp(c))
	}
	s.mu.RUnlock()

	if list == nil {
		list = []map[string]interface{}{}
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"clusterInfoList": list,
	})
}

func (s *Service) updateBrokerCount(w http.ResponseWriter, r *http.Request, path string) {
	arnSegment := extractClusterArn(path)

	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	targetCount := h.GetInt(params, "targetNumberOfBrokerNodes", 0)
	if targetCount == 0 {
		h.WriteJSONError(w, "InvalidParameterException", "targetNumberOfBrokerNodes is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	c := s.findClusterByArnSuffix(arnSegment)
	if c == nil {
		s.mu.Unlock()
		h.WriteJSONError(w, "NotFoundException", "Cluster not found", http.StatusNotFound)
		return
	}
	c.numberOfBrokers = targetCount
	arn := c.arn
	s.mu.Unlock()

	operationArn := fmt.Sprintf("arn:aws:kafka:us-east-1:%s:cluster-operation/%s", h.DefaultAccountID, h.RandomHex(12))

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"clusterArn":          arn,
		"clusterOperationArn": operationArn,
	})
}

// findClusterByArnSuffix looks up a cluster whose ARN ends with the given segment.
// Caller must hold at least s.mu.RLock.
func (s *Service) findClusterByArnSuffix(segment string) *cluster {
	// Try exact match first.
	if c, ok := s.clusters[segment]; ok {
		return c
	}
	// Fall back to suffix match for URL-encoded or partial ARN paths.
	for arn, c := range s.clusters {
		if strings.HasSuffix(arn, segment) {
			return c
		}
	}
	return nil
}

func clusterInfoResp(c *cluster) map[string]interface{} {
	return map[string]interface{}{
		"clusterArn":          c.arn,
		"clusterName":         c.name,
		"state":               c.state,
		"currentVersion":      c.currentVersion,
		"clusterType":         "PROVISIONED",
		"creationTime":        c.created.Format(time.RFC3339),
		"numberOfBrokerNodes": c.numberOfBrokers,
		"brokerNodeGroupInfo": map[string]interface{}{
			"instanceType": c.instanceType,
		},
		"currentBrokerSoftwareInfo": map[string]interface{}{
			"kafkaVersion": c.kafkaVersion,
		},
	}
}

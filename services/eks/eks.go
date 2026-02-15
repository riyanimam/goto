// Package eks provides a mock implementation of AWS Elastic Kubernetes Service.
//
// Supported actions:
//   - CreateCluster
//   - DescribeCluster
//   - DeleteCluster
//   - ListClusters
//   - CreateNodegroup
//   - DescribeNodegroup
//   - DeleteNodegroup
//   - ListNodegroups
package eks

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

// Service implements the EKS mock.
type Service struct {
	mu       sync.RWMutex
	clusters map[string]*cluster
}

type cluster struct {
	name       string
	arn        string
	status     string
	version    string
	roleArn    string
	endpoint   string
	created    time.Time
	nodegroups map[string]*nodegroup
}

type nodegroup struct {
	name     string
	arn      string
	status   string
	nodeRole string
	capacity int32
	minSize  int32
	maxSize  int32
	subnets  []string
	created  time.Time
}

// New creates a new EKS mock service.
func New() *Service {
	return &Service{
		clusters: make(map[string]*cluster),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "eks" }

// Handler returns the HTTP handler for EKS requests.
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
	// Nodegroups: /clusters/{name}/node-groups/{ngName}
	case strings.Contains(path, "/node-groups/") && method == http.MethodGet:
		s.describeNodegroup(w, r, path)
	case strings.Contains(path, "/node-groups/") && method == http.MethodDelete:
		s.deleteNodegroup(w, r, path)

	// Nodegroups list: /clusters/{name}/node-groups
	case strings.HasSuffix(path, "/node-groups") && method == http.MethodPost:
		s.createNodegroup(w, r, path)
	case strings.HasSuffix(path, "/node-groups") && method == http.MethodGet:
		s.listNodegroups(w, r, path)

	// Clusters: /clusters/{name}
	case strings.HasPrefix(path, "/clusters/") && !strings.Contains(path, "/node-groups") && method == http.MethodGet:
		s.describeCluster(w, r, path)
	case strings.HasPrefix(path, "/clusters/") && !strings.Contains(path, "/node-groups") && method == http.MethodDelete:
		s.deleteCluster(w, r, path)

	// Clusters list: /clusters
	case path == "/clusters" && method == http.MethodPost:
		s.createCluster(w, r)
	case path == "/clusters" && method == http.MethodGet:
		s.listClusters(w, r)

	default:
		h.WriteJSONError(w, "NotFoundException", "unsupported operation", http.StatusNotFound)
	}
}

func extractClusterName(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func (s *Service) createCluster(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	name := h.GetString(params, "name")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterException", "name is required", http.StatusBadRequest)
		return
	}

	version := h.GetString(params, "version")
	if version == "" {
		version = "1.29"
	}

	roleArn := h.GetString(params, "roleArn")

	s.mu.Lock()
	if _, exists := s.clusters[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceInUseException", "Cluster "+name+" already exists", http.StatusConflict)
		return
	}

	arn := fmt.Sprintf("arn:aws:eks:us-east-1:%s:cluster/%s", h.DefaultAccountID, name)
	endpoint := fmt.Sprintf("https://%s.gr7.us-east-1.eks.amazonaws.com", h.RandomHex(32))
	now := time.Now().UTC()

	c := &cluster{
		name:       name,
		arn:        arn,
		status:     "ACTIVE",
		version:    version,
		roleArn:    roleArn,
		endpoint:   endpoint,
		created:    now,
		nodegroups: make(map[string]*nodegroup),
	}
	s.clusters[name] = c
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"cluster": clusterResp(c),
	})
}

func (s *Service) describeCluster(w http.ResponseWriter, _ *http.Request, path string) {
	name := extractClusterName(path)

	s.mu.RLock()
	c, exists := s.clusters[name]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Cluster "+name+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"cluster": clusterResp(c),
	})
}

func (s *Service) deleteCluster(w http.ResponseWriter, _ *http.Request, path string) {
	name := extractClusterName(path)

	s.mu.Lock()
	c, exists := s.clusters[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Cluster "+name+" not found", http.StatusNotFound)
		return
	}
	c.status = "DELETING"
	resp := clusterResp(c)
	delete(s.clusters, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"cluster": resp,
	})
}

func (s *Service) listClusters(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var names []string
	for name := range s.clusters {
		names = append(names, name)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"clusters": names,
	})
}

func (s *Service) createNodegroup(w http.ResponseWriter, r *http.Request, path string) {
	clusterName := extractClusterName(path)
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	ngName := h.GetString(params, "nodegroupName")
	if ngName == "" {
		h.WriteJSONError(w, "InvalidParameterException", "nodegroupName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	c, exists := s.clusters[clusterName]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Cluster "+clusterName+" not found", http.StatusNotFound)
		return
	}

	var subnets []string
	if subs, ok := params["subnets"].([]interface{}); ok {
		for _, sub := range subs {
			if str, ok := sub.(string); ok {
				subnets = append(subnets, str)
			}
		}
	}

	arn := fmt.Sprintf("arn:aws:eks:us-east-1:%s:nodegroup/%s/%s/%s",
		h.DefaultAccountID, clusterName, ngName, h.RandomHex(17))

	ng := &nodegroup{
		name:     ngName,
		arn:      arn,
		status:   "ACTIVE",
		nodeRole: h.GetString(params, "nodeRole"),
		capacity: int32(h.GetInt(params, "desiredSize", 2)),
		minSize:  int32(h.GetInt(params, "minSize", 1)),
		maxSize:  int32(h.GetInt(params, "maxSize", 3)),
		subnets:  subnets,
		created:  time.Now().UTC(),
	}
	c.nodegroups[ngName] = ng
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"nodegroup": nodegroupResp(ng, clusterName),
	})
}

func (s *Service) describeNodegroup(w http.ResponseWriter, _ *http.Request, path string) {
	clusterName := extractClusterName(path)
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 4 {
		h.WriteJSONError(w, "NotFoundException", "invalid path", http.StatusNotFound)
		return
	}
	ngName := parts[3]

	s.mu.RLock()
	c, exists := s.clusters[clusterName]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Cluster "+clusterName+" not found", http.StatusNotFound)
		return
	}
	ng, exists := c.nodegroups[ngName]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Nodegroup "+ngName+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"nodegroup": nodegroupResp(ng, clusterName),
	})
}

func (s *Service) deleteNodegroup(w http.ResponseWriter, _ *http.Request, path string) {
	clusterName := extractClusterName(path)
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 4 {
		h.WriteJSONError(w, "NotFoundException", "invalid path", http.StatusNotFound)
		return
	}
	ngName := parts[3]

	s.mu.Lock()
	c, exists := s.clusters[clusterName]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Cluster "+clusterName+" not found", http.StatusNotFound)
		return
	}
	ng, exists := c.nodegroups[ngName]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Nodegroup "+ngName+" not found", http.StatusNotFound)
		return
	}
	ng.status = "DELETING"
	resp := nodegroupResp(ng, clusterName)
	delete(c.nodegroups, ngName)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"nodegroup": resp,
	})
}

func (s *Service) listNodegroups(w http.ResponseWriter, _ *http.Request, path string) {
	clusterName := extractClusterName(path)

	s.mu.RLock()
	c, exists := s.clusters[clusterName]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Cluster "+clusterName+" not found", http.StatusNotFound)
		return
	}

	var names []string
	for name := range c.nodegroups {
		names = append(names, name)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"nodegroups": names,
	})
}

func clusterResp(c *cluster) map[string]interface{} {
	return map[string]interface{}{
		"name":            c.name,
		"arn":             c.arn,
		"status":          c.status,
		"version":         c.version,
		"roleArn":         c.roleArn,
		"endpoint":        c.endpoint,
		"createdAt":       float64(c.created.Unix()),
		"platformVersion": "eks.1",
	}
}

func nodegroupResp(ng *nodegroup, clusterName string) map[string]interface{} {
	return map[string]interface{}{
		"nodegroupName": ng.name,
		"nodegroupArn":  ng.arn,
		"clusterName":   clusterName,
		"status":        ng.status,
		"nodeRole":      ng.nodeRole,
		"subnets":       ng.subnets,
		"scalingConfig": map[string]interface{}{
			"desiredSize": ng.capacity,
			"minSize":     ng.minSize,
			"maxSize":     ng.maxSize,
		},
		"createdAt": float64(ng.created.Unix()),
	}
}

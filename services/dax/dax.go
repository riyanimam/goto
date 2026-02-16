// Package dax provides a mock implementation of Amazon DAX.
//
// Supported actions:
//   - CreateCluster
//   - DescribeClusters
//   - DeleteCluster
//   - ListTags
//   - CreateSubnetGroup
//   - DescribeSubnetGroups
//   - DeleteSubnetGroup
package dax

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the DAX mock.
type Service struct {
	mu           sync.RWMutex
	clusters     map[string]*cluster
	subnetGroups map[string]*subnetGroup
	tags         map[string][]map[string]string
}

type cluster struct {
	name              string
	arn               string
	status            string
	nodeType          string
	replicationFactor int
	iamRoleArn        string
	description       string
}

type subnetGroup struct {
	name        string
	description string
	subnetIDs   []string
}

// New creates a new DAX mock service.
func New() *Service {
	return &Service{
		clusters:     make(map[string]*cluster),
		subnetGroups: make(map[string]*subnetGroup),
		tags:         make(map[string][]map[string]string),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "dax" }

// Handler returns the HTTP handler for DAX requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clusters = make(map[string]*cluster)
	s.subnetGroups = make(map[string]*subnetGroup)
	s.tags = make(map[string][]map[string]string)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteJSONError(w, "InternalServerError", "could not read request body", http.StatusInternalServerError)
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
	case "CreateCluster":
		s.createCluster(w, params)
	case "DescribeClusters":
		s.describeClusters(w, params)
	case "DeleteCluster":
		s.deleteCluster(w, params)
	case "ListTags":
		s.listTags(w, params)
	case "CreateSubnetGroup":
		s.createSubnetGroup(w, params)
	case "DescribeSubnetGroups":
		s.describeSubnetGroups(w, params)
	case "DeleteSubnetGroup":
		s.deleteSubnetGroup(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createCluster(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "ClusterName")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterValueException", "ClusterName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.clusters[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ClusterAlreadyExistsFault", "Cluster "+name+" already exists", http.StatusBadRequest)
		return
	}

	arn := fmt.Sprintf("arn:aws:dax:us-east-1:%s:cache/%s", h.DefaultAccountID, name)
	replicationFactor := h.GetInt(params, "ReplicationFactor", 1)

	c := &cluster{
		name:              name,
		arn:               arn,
		status:            "creating",
		nodeType:          h.GetString(params, "NodeType"),
		replicationFactor: replicationFactor,
		iamRoleArn:        h.GetString(params, "IamRoleArn"),
		description:       h.GetString(params, "Description"),
	}
	s.clusters[name] = c
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Cluster": clusterResp(c),
	})
}

func (s *Service) describeClusters(w http.ResponseWriter, params map[string]interface{}) {
	names := getStringSlice(params, "ClusterNames")

	s.mu.RLock()
	var list []map[string]interface{}
	if len(names) > 0 {
		for _, n := range names {
			if c, ok := s.clusters[n]; ok {
				list = append(list, clusterResp(c))
			}
		}
	} else {
		for _, c := range s.clusters {
			list = append(list, clusterResp(c))
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Clusters": list,
	})
}

func (s *Service) deleteCluster(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "ClusterName")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterValueException", "ClusterName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	c, exists := s.clusters[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ClusterNotFoundFault", "Cluster "+name+" not found", http.StatusBadRequest)
		return
	}
	c.status = "deleting"
	resp := clusterResp(c)
	delete(s.clusters, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Cluster": resp,
	})
}

func (s *Service) listTags(w http.ResponseWriter, params map[string]interface{}) {
	resourceName := h.GetString(params, "ResourceName")
	if resourceName == "" {
		h.WriteJSONError(w, "InvalidParameterValueException", "ResourceName is required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	tags := s.tags[resourceName]
	s.mu.RUnlock()

	if tags == nil {
		tags = []map[string]string{}
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Tags": tags,
	})
}

func (s *Service) createSubnetGroup(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "SubnetGroupName")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterValueException", "SubnetGroupName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.subnetGroups[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "SubnetGroupAlreadyExistsFault", "Subnet group "+name+" already exists", http.StatusBadRequest)
		return
	}

	sg := &subnetGroup{
		name:        name,
		description: h.GetString(params, "Description"),
		subnetIDs:   getStringSlice(params, "SubnetIds"),
	}
	s.subnetGroups[name] = sg
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"SubnetGroup": subnetGroupResp(sg),
	})
}

func (s *Service) describeSubnetGroups(w http.ResponseWriter, params map[string]interface{}) {
	names := getStringSlice(params, "SubnetGroupNames")

	s.mu.RLock()
	var list []map[string]interface{}
	if len(names) > 0 {
		for _, n := range names {
			if sg, ok := s.subnetGroups[n]; ok {
				list = append(list, subnetGroupResp(sg))
			}
		}
	} else {
		for _, sg := range s.subnetGroups {
			list = append(list, subnetGroupResp(sg))
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"SubnetGroups": list,
	})
}

func (s *Service) deleteSubnetGroup(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "SubnetGroupName")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterValueException", "SubnetGroupName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	_, exists := s.subnetGroups[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "SubnetGroupNotFoundFault", "Subnet group "+name+" not found", http.StatusBadRequest)
		return
	}
	delete(s.subnetGroups, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func clusterResp(c *cluster) map[string]interface{} {
	return map[string]interface{}{
		"ClusterName":       c.name,
		"ClusterArn":        c.arn,
		"Status":            c.status,
		"NodeType":          c.nodeType,
		"TotalNodes":        c.replicationFactor,
		"ActiveNodes":       0,
		"Description":       c.description,
		"ReplicationFactor": c.replicationFactor,
		"IamRoleArn":        c.iamRoleArn,
	}
}

func subnetGroupResp(sg *subnetGroup) map[string]interface{} {
	var subnets []map[string]interface{}
	for _, id := range sg.subnetIDs {
		subnets = append(subnets, map[string]interface{}{
			"SubnetIdentifier": id,
		})
	}
	if subnets == nil {
		subnets = []map[string]interface{}{}
	}
	return map[string]interface{}{
		"SubnetGroupName": sg.name,
		"Description":     sg.description,
		"Subnets":         subnets,
	}
}

func getStringSlice(params map[string]interface{}, key string) []string {
	v, ok := params[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

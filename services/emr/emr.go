// Package emr provides a mock implementation of AWS EMR (Elastic MapReduce).
//
// Supported actions:
//   - RunJobFlow
//   - DescribeCluster
//   - ListClusters
//   - TerminateJobFlows
//   - AddJobFlowSteps
//   - ListSteps
package emr

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

// Service implements the EMR mock.
type Service struct {
	mu       sync.RWMutex
	clusters map[string]*cluster
}

type cluster struct {
	id            string
	name          string
	releaseLabel  string
	status        string
	instanceType  string
	instanceCount int
	applications  []string
	steps         []*step
	created       time.Time
}

type step struct {
	id              string
	name            string
	actionOnFailure string
	status          string
	created         time.Time
}

// New creates a new EMR mock service.
func New() *Service {
	return &Service{
		clusters: make(map[string]*cluster),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "elasticmapreduce" }

// Handler returns the HTTP handler for EMR requests.
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
	case "RunJobFlow":
		s.runJobFlow(w, params)
	case "DescribeCluster":
		s.describeCluster(w, params)
	case "ListClusters":
		s.listClusters(w, params)
	case "TerminateJobFlows":
		s.terminateJobFlows(w, params)
	case "AddJobFlowSteps":
		s.addJobFlowSteps(w, params)
	case "ListSteps":
		s.listSteps(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) runJobFlow(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")
	if name == "" {
		h.WriteJSONError(w, "InvalidRequestException", "Name is required", http.StatusBadRequest)
		return
	}

	releaseLabel := h.GetString(params, "ReleaseLabel")

	masterType := "m5.xlarge"
	slaveType := "m5.xlarge"
	instanceCount := 1
	if inst, ok := params["Instances"].(map[string]interface{}); ok {
		if v := h.GetString(inst, "MasterInstanceType"); v != "" {
			masterType = v
		}
		if v := h.GetString(inst, "SlaveInstanceType"); v != "" {
			slaveType = v
		}
		instanceCount = h.GetInt(inst, "InstanceCount", 1)
	}
	_ = slaveType // stored via masterType for simplicity

	var apps []string
	if appList, ok := params["Applications"].([]interface{}); ok {
		for _, a := range appList {
			if appMap, ok := a.(map[string]interface{}); ok {
				if n := h.GetString(appMap, "Name"); n != "" {
					apps = append(apps, n)
				}
			}
		}
	}

	s.mu.Lock()
	id := "j-" + h.RandomID(13)
	c := &cluster{
		id:            id,
		name:          name,
		releaseLabel:  releaseLabel,
		status:        "RUNNING",
		instanceType:  masterType,
		instanceCount: instanceCount,
		applications:  apps,
		created:       time.Now().UTC(),
	}
	s.clusters[id] = c
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"JobFlowId": id,
	})
}

func (s *Service) describeCluster(w http.ResponseWriter, params map[string]interface{}) {
	clusterID := h.GetString(params, "ClusterId")

	s.mu.RLock()
	c, exists := s.clusters[clusterID]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "InvalidRequestException", "Cluster not found: "+clusterID, http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Cluster": clusterResp(c),
	})
}

func (s *Service) listClusters(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var items []map[string]interface{}
	for _, c := range s.clusters {
		items = append(items, map[string]interface{}{
			"Id":                      c.id,
			"Name":                    c.name,
			"Status":                  statusResp(c),
			"NormalizedInstanceHours": 0,
		})
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i]["Name"].(string) < items[j]["Name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Clusters": items,
	})
}

func (s *Service) terminateJobFlows(w http.ResponseWriter, params map[string]interface{}) {
	var ids []string
	if idList, ok := params["JobFlowIds"].([]interface{}); ok {
		for _, v := range idList {
			if id, ok := v.(string); ok {
				ids = append(ids, id)
			}
		}
	}

	s.mu.Lock()
	for _, id := range ids {
		if c, exists := s.clusters[id]; exists {
			c.status = "TERMINATED"
		}
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) addJobFlowSteps(w http.ResponseWriter, params map[string]interface{}) {
	jobFlowID := h.GetString(params, "JobFlowId")

	s.mu.Lock()
	c, exists := s.clusters[jobFlowID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "InvalidRequestException", "Cluster not found: "+jobFlowID, http.StatusBadRequest)
		return
	}

	var stepIDs []string
	if stepList, ok := params["Steps"].([]interface{}); ok {
		for _, raw := range stepList {
			sm, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			id := "s-" + h.RandomID(13)
			st := &step{
				id:              id,
				name:            h.GetString(sm, "Name"),
				actionOnFailure: h.GetString(sm, "ActionOnFailure"),
				status:          "RUNNING",
				created:         time.Now().UTC(),
			}
			c.steps = append(c.steps, st)
			stepIDs = append(stepIDs, id)
		}
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"StepIds": stepIDs,
	})
}

func (s *Service) listSteps(w http.ResponseWriter, params map[string]interface{}) {
	clusterID := h.GetString(params, "ClusterId")

	s.mu.RLock()
	c, exists := s.clusters[clusterID]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "InvalidRequestException", "Cluster not found: "+clusterID, http.StatusBadRequest)
		return
	}

	var items []map[string]interface{}
	for _, st := range c.steps {
		items = append(items, map[string]interface{}{
			"Id":   st.id,
			"Name": st.name,
			"Status": map[string]interface{}{
				"State": st.status,
				"Timeline": map[string]interface{}{
					"CreationDateTime": float64(st.created.Unix()),
				},
			},
			"ActionOnFailure": st.actionOnFailure,
		})
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Steps": items,
	})
}

func clusterResp(c *cluster) map[string]interface{} {
	resp := map[string]interface{}{
		"Id":     c.id,
		"Name":   c.name,
		"Status": statusResp(c),
		"Ec2InstanceAttributes": map[string]interface{}{},
	}
	if c.releaseLabel != "" {
		resp["ReleaseLabel"] = c.releaseLabel
	}
	if len(c.applications) > 0 {
		var apps []map[string]interface{}
		for _, name := range c.applications {
			apps = append(apps, map[string]interface{}{"Name": name})
		}
		resp["Applications"] = apps
	}
	return resp
}

func statusResp(c *cluster) map[string]interface{} {
	return map[string]interface{}{
		"State": c.status,
		"Timeline": map[string]interface{}{
			"CreationDateTime": float64(c.created.Unix()),
		},
	}
}

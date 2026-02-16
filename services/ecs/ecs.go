// Package ecs provides a mock implementation of AWS Elastic Container Service.
//
// Supported actions:
//   - CreateCluster
//   - DeleteCluster
//   - DescribeClusters
//   - ListClusters
//   - RegisterTaskDefinition
//   - DeregisterTaskDefinition
//   - ListTaskDefinitions
//   - RunTask
//   - StopTask
//   - ListTasks
//   - DescribeTasks
//   - CreateService
//   - DeleteService
//   - UpdateService
//   - ListServices
//   - DescribeServices
package ecs

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

// Service implements the ECS mock.
type Service struct {
	mu              sync.RWMutex
	clusters        map[string]*cluster
	taskDefs        map[string]*taskDefinition // keyed by family:revision
	taskDefFamilies map[string]int             // family -> latest revision
	tasks           map[string]*task
	services        map[string]*ecsService
	taskCounter     int
}

type cluster struct {
	name   string
	arn    string
	status string
}

type taskDefinition struct {
	family     string
	revision   int
	arn        string
	status     string
	containers []containerDef
}

type containerDef struct {
	name   string
	image  string
	cpu    int
	memory int
}

type task struct {
	arn           string
	taskDefArn    string
	clusterArn    string
	lastStatus    string
	desiredStatus string
	startedAt     time.Time
}

type ecsService struct {
	name         string
	arn          string
	clusterArn   string
	taskDefArn   string
	desiredCount int
	runningCount int
	status       string
}

// New creates a new ECS mock service.
func New() *Service {
	return &Service{
		clusters:        make(map[string]*cluster),
		taskDefs:        make(map[string]*taskDefinition),
		taskDefFamilies: make(map[string]int),
		tasks:           make(map[string]*task),
		services:        make(map[string]*ecsService),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "ecs" }

// Handler returns the HTTP handler for ECS requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clusters = make(map[string]*cluster)
	s.taskDefs = make(map[string]*taskDefinition)
	s.taskDefFamilies = make(map[string]int)
	s.tasks = make(map[string]*task)
	s.services = make(map[string]*ecsService)
	s.taskCounter = 0
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteJSONError(w, "ServerException", "could not read request body", http.StatusInternalServerError)
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
	case "CreateCluster":
		s.createCluster(w, params)
	case "DeleteCluster":
		s.deleteCluster(w, params)
	case "DescribeClusters":
		s.describeClusters(w, params)
	case "ListClusters":
		s.listClusters(w, params)
	case "RegisterTaskDefinition":
		s.registerTaskDefinition(w, params)
	case "DeregisterTaskDefinition":
		s.deregisterTaskDefinition(w, params)
	case "ListTaskDefinitions":
		s.listTaskDefinitions(w, params)
	case "RunTask":
		s.runTask(w, params)
	case "StopTask":
		s.stopTask(w, params)
	case "ListTasks":
		s.listTasks(w, params)
	case "DescribeTasks":
		s.describeTasks(w, params)
	case "CreateService":
		s.createService(w, params)
	case "DeleteService":
		s.deleteService(w, params)
	case "UpdateService":
		s.updateService(w, params)
	case "ListServices":
		s.listServices(w, params)
	case "DescribeServices":
		s.describeServices(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createCluster(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "clusterName")
	if name == "" {
		name = "default"
	}

	s.mu.Lock()
	c := &cluster{
		name:   name,
		arn:    fmt.Sprintf("arn:aws:ecs:us-east-1:%s:cluster/%s", h.DefaultAccountID, name),
		status: "ACTIVE",
	}
	s.clusters[name] = c
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"cluster": clusterResp(c),
	})
}

func (s *Service) deleteCluster(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "cluster")
	name = clusterNameFromArn(name)

	s.mu.Lock()
	c, exists := s.clusters[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ClusterNotFoundException", "Cluster not found.", http.StatusBadRequest)
		return
	}
	c.status = "INACTIVE"
	delete(s.clusters, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"cluster": clusterResp(c),
	})
}

func (s *Service) describeClusters(w http.ResponseWriter, params map[string]interface{}) {
	clusterNames, _ := params["clusters"].([]interface{})

	s.mu.RLock()
	var clusters []map[string]interface{}
	var failures []map[string]interface{}
	for _, cn := range clusterNames {
		name, _ := cn.(string)
		name = clusterNameFromArn(name)
		if c, exists := s.clusters[name]; exists {
			clusters = append(clusters, clusterResp(c))
		} else {
			failures = append(failures, map[string]interface{}{
				"arn":    name,
				"reason": "MISSING",
			})
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"clusters": clusters,
		"failures": failures,
	})
}

func (s *Service) listClusters(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var arns []string
	for _, c := range s.clusters {
		arns = append(arns, c.arn)
	}
	s.mu.RUnlock()

	sort.Strings(arns)
	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"clusterArns": arns,
	})
}

func (s *Service) registerTaskDefinition(w http.ResponseWriter, params map[string]interface{}) {
	family := h.GetString(params, "family")
	if family == "" {
		h.WriteJSONError(w, "ClientException", "family is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.taskDefFamilies[family]++
	revision := s.taskDefFamilies[family]

	var containers []containerDef
	if cds, ok := params["containerDefinitions"].([]interface{}); ok {
		for _, cd := range cds {
			if cdm, ok := cd.(map[string]interface{}); ok {
				containers = append(containers, containerDef{
					name:   h.GetString(cdm, "name"),
					image:  h.GetString(cdm, "image"),
					cpu:    h.GetInt(cdm, "cpu", 256),
					memory: h.GetInt(cdm, "memory", 512),
				})
			}
		}
	}

	key := fmt.Sprintf("%s:%d", family, revision)
	td := &taskDefinition{
		family:     family,
		revision:   revision,
		arn:        fmt.Sprintf("arn:aws:ecs:us-east-1:%s:task-definition/%s:%d", h.DefaultAccountID, family, revision),
		status:     "ACTIVE",
		containers: containers,
	}
	s.taskDefs[key] = td
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"taskDefinition": taskDefResp(td),
	})
}

func (s *Service) deregisterTaskDefinition(w http.ResponseWriter, params map[string]interface{}) {
	tdArn := h.GetString(params, "taskDefinition")

	s.mu.Lock()
	for key, td := range s.taskDefs {
		if td.arn == tdArn || key == tdArn {
			td.status = "INACTIVE"
			delete(s.taskDefs, key)
			s.mu.Unlock()
			h.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"taskDefinition": taskDefResp(td),
			})
			return
		}
	}
	s.mu.Unlock()
	h.WriteJSONError(w, "ClientException", "Task definition not found.", http.StatusBadRequest)
}

func (s *Service) listTaskDefinitions(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var arns []string
	for _, td := range s.taskDefs {
		arns = append(arns, td.arn)
	}
	s.mu.RUnlock()

	sort.Strings(arns)
	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"taskDefinitionArns": arns,
	})
}

func (s *Service) runTask(w http.ResponseWriter, params map[string]interface{}) {
	clusterName := h.GetString(params, "cluster")
	if clusterName == "" {
		clusterName = "default"
	}
	clusterName = clusterNameFromArn(clusterName)
	tdArn := h.GetString(params, "taskDefinition")
	count := h.GetInt(params, "count", 1)

	s.mu.Lock()
	c, exists := s.clusters[clusterName]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ClusterNotFoundException", "Cluster not found.", http.StatusBadRequest)
		return
	}

	var tasks []map[string]interface{}
	for i := 0; i < count; i++ {
		s.taskCounter++
		taskArn := fmt.Sprintf("arn:aws:ecs:us-east-1:%s:task/%s/%s", h.DefaultAccountID, clusterName, h.NewRequestID())
		t := &task{
			arn:           taskArn,
			taskDefArn:    tdArn,
			clusterArn:    c.arn,
			lastStatus:    "RUNNING",
			desiredStatus: "RUNNING",
			startedAt:     time.Now().UTC(),
		}
		s.tasks[taskArn] = t
		tasks = append(tasks, taskResp(t))
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"tasks":    tasks,
		"failures": []interface{}{},
	})
}

func (s *Service) stopTask(w http.ResponseWriter, params map[string]interface{}) {
	taskArn := h.GetString(params, "task")

	s.mu.Lock()
	t, exists := s.tasks[taskArn]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "InvalidParameterException", "Task not found.", http.StatusBadRequest)
		return
	}
	t.lastStatus = "STOPPED"
	t.desiredStatus = "STOPPED"
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"task": taskResp(t),
	})
}

func (s *Service) listTasks(w http.ResponseWriter, params map[string]interface{}) {
	clusterName := h.GetString(params, "cluster")
	clusterName = clusterNameFromArn(clusterName)

	s.mu.RLock()
	var arns []string
	for _, t := range s.tasks {
		if clusterName == "" || strings.Contains(t.clusterArn, clusterName) {
			arns = append(arns, t.arn)
		}
	}
	s.mu.RUnlock()

	sort.Strings(arns)
	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"taskArns": arns,
	})
}

func (s *Service) describeTasks(w http.ResponseWriter, params map[string]interface{}) {
	taskArns, _ := params["tasks"].([]interface{})

	s.mu.RLock()
	var tasks []map[string]interface{}
	for _, ta := range taskArns {
		arn, _ := ta.(string)
		if t, exists := s.tasks[arn]; exists {
			tasks = append(tasks, taskResp(t))
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"tasks":    tasks,
		"failures": []interface{}{},
	})
}

func (s *Service) createService(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "serviceName")
	clusterName := h.GetString(params, "cluster")
	if clusterName == "" {
		clusterName = "default"
	}
	clusterName = clusterNameFromArn(clusterName)
	tdArn := h.GetString(params, "taskDefinition")
	desiredCount := h.GetInt(params, "desiredCount", 1)

	s.mu.Lock()
	c, exists := s.clusters[clusterName]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ClusterNotFoundException", "Cluster not found.", http.StatusBadRequest)
		return
	}

	svc := &ecsService{
		name:         name,
		arn:          fmt.Sprintf("arn:aws:ecs:us-east-1:%s:service/%s/%s", h.DefaultAccountID, clusterName, name),
		clusterArn:   c.arn,
		taskDefArn:   tdArn,
		desiredCount: desiredCount,
		runningCount: desiredCount,
		status:       "ACTIVE",
	}
	s.services[name] = svc
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"service": serviceResp(svc),
	})
}

func (s *Service) deleteService(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "service")

	s.mu.Lock()
	svc, exists := s.services[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ServiceNotFoundException", "Service not found.", http.StatusBadRequest)
		return
	}
	svc.status = "INACTIVE"
	delete(s.services, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"service": serviceResp(svc),
	})
}

func (s *Service) updateService(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "service")

	s.mu.Lock()
	svc, exists := s.services[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ServiceNotFoundException", "Service not found.", http.StatusBadRequest)
		return
	}

	if td := h.GetString(params, "taskDefinition"); td != "" {
		svc.taskDefArn = td
	}
	if dc := h.GetInt(params, "desiredCount", -1); dc >= 0 {
		svc.desiredCount = dc
		svc.runningCount = dc
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"service": serviceResp(svc),
	})
}

func (s *Service) listServices(w http.ResponseWriter, params map[string]interface{}) {
	s.mu.RLock()
	var arns []string
	for _, svc := range s.services {
		arns = append(arns, svc.arn)
	}
	s.mu.RUnlock()

	sort.Strings(arns)
	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"serviceArns": arns,
	})
}

func (s *Service) describeServices(w http.ResponseWriter, params map[string]interface{}) {
	svcNames, _ := params["services"].([]interface{})

	s.mu.RLock()
	var svcs []map[string]interface{}
	for _, sn := range svcNames {
		name, _ := sn.(string)
		if svc, exists := s.services[name]; exists {
			svcs = append(svcs, serviceResp(svc))
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"services": svcs,
		"failures": []interface{}{},
	})
}

// Helper functions.

func clusterNameFromArn(name string) string {
	if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		return parts[len(parts)-1]
	}
	return name
}

func clusterResp(c *cluster) map[string]interface{} {
	return map[string]interface{}{
		"clusterName": c.name,
		"clusterArn":  c.arn,
		"status":      c.status,
	}
}

func taskDefResp(td *taskDefinition) map[string]interface{} {
	var containers []map[string]interface{}
	for _, c := range td.containers {
		containers = append(containers, map[string]interface{}{
			"name":   c.name,
			"image":  c.image,
			"cpu":    c.cpu,
			"memory": c.memory,
		})
	}
	return map[string]interface{}{
		"taskDefinitionArn":    td.arn,
		"family":               td.family,
		"revision":             td.revision,
		"status":               td.status,
		"containerDefinitions": containers,
	}
}

func taskResp(t *task) map[string]interface{} {
	return map[string]interface{}{
		"taskArn":           t.arn,
		"taskDefinitionArn": t.taskDefArn,
		"clusterArn":        t.clusterArn,
		"lastStatus":        t.lastStatus,
		"desiredStatus":     t.desiredStatus,
		"startedAt":         float64(t.startedAt.Unix()),
	}
}

func serviceResp(svc *ecsService) map[string]interface{} {
	return map[string]interface{}{
		"serviceName":    svc.name,
		"serviceArn":     svc.arn,
		"clusterArn":     svc.clusterArn,
		"taskDefinition": svc.taskDefArn,
		"desiredCount":   svc.desiredCount,
		"runningCount":   svc.runningCount,
		"status":         svc.status,
	}
}

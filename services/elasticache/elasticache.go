// Package elasticache provides a mock implementation of AWS ElastiCache.
//
// Supported actions:
//   - CreateCacheCluster
//   - DeleteCacheCluster
//   - DescribeCacheClusters
//   - ModifyCacheCluster
//   - CreateReplicationGroup
//   - DeleteReplicationGroup
//   - DescribeReplicationGroups
package elasticache

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the ElastiCache mock.
type Service struct {
	mu                sync.RWMutex
	clusters          map[string]*cacheCluster
	replicationGroups map[string]*replicationGroup
}

type cacheCluster struct {
	id        string
	arn       string
	status    string
	engine    string
	engineVer string
	nodeType  string
	numNodes  int
	created   time.Time
}

type replicationGroup struct {
	id          string
	arn         string
	description string
	status      string
	nodeType    string
	numClusters int
	created     time.Time
}

// New creates a new ElastiCache mock service.
func New() *Service {
	return &Service{
		clusters:          make(map[string]*cacheCluster),
		replicationGroups: make(map[string]*replicationGroup),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "elasticache" }

// Handler returns the HTTP handler for ElastiCache requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clusters = make(map[string]*cacheCluster)
	s.replicationGroups = make(map[string]*replicationGroup)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("Action")
	if action == "" {
		action = r.FormValue("Action")
	}

	switch action {
	case "CreateCacheCluster":
		s.createCacheCluster(w, r)
	case "DeleteCacheCluster":
		s.deleteCacheCluster(w, r)
	case "DescribeCacheClusters":
		s.describeCacheClusters(w, r)
	case "ModifyCacheCluster":
		s.modifyCacheCluster(w, r)
	case "CreateReplicationGroup":
		s.createReplicationGroup(w, r)
	case "DeleteReplicationGroup":
		s.deleteReplicationGroup(w, r)
	case "DescribeReplicationGroups":
		s.describeReplicationGroups(w, r)
	default:
		h.WriteXMLError(w, "Sender", "InvalidAction", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func getFormVal(r *http.Request, key string) string {
	v := r.URL.Query().Get(key)
	if v == "" {
		v = r.FormValue(key)
	}
	return v
}

func (s *Service) createCacheCluster(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := getFormVal(r, "CacheClusterId")
	if id == "" {
		h.WriteXMLError(w, "Sender", "InvalidParameterValue", "CacheClusterId is required", http.StatusBadRequest)
		return
	}

	engine := getFormVal(r, "Engine")
	if engine == "" {
		engine = "redis"
	}
	engineVer := getFormVal(r, "EngineVersion")
	if engineVer == "" {
		if engine == "redis" {
			engineVer = "7.0"
		} else {
			engineVer = "1.6.22"
		}
	}
	nodeType := getFormVal(r, "CacheNodeType")
	if nodeType == "" {
		nodeType = "cache.t3.micro"
	}

	s.mu.Lock()
	if _, exists := s.clusters[id]; exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "CacheClusterAlreadyExists", "Cache cluster "+id+" already exists", http.StatusBadRequest)
		return
	}

	arn := fmt.Sprintf("arn:aws:elasticache:us-east-1:%s:cluster:%s", h.DefaultAccountID, id)
	cc := &cacheCluster{
		id:        id,
		arn:       arn,
		status:    "available",
		engine:    engine,
		engineVer: engineVer,
		nodeType:  nodeType,
		numNodes:  1,
		created:   time.Now().UTC(),
	}
	s.clusters[id] = cc
	s.mu.Unlock()

	type ccResult struct {
		XMLName      xml.Name `xml:"CreateCacheClusterResult"`
		CacheCluster ccXML    `xml:"CacheCluster"`
	}
	type ccResp struct {
		XMLName xml.Name `xml:"CreateCacheClusterResponse"`
		Result  ccResult `xml:"CreateCacheClusterResult"`
	}
	h.WriteXML(w, http.StatusOK, ccResp{Result: ccResult{CacheCluster: clusterToXML(cc)}})
}

func (s *Service) deleteCacheCluster(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := getFormVal(r, "CacheClusterId")

	s.mu.Lock()
	cc, exists := s.clusters[id]
	if !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "CacheClusterNotFound", "Cache cluster "+id+" not found", http.StatusNotFound)
		return
	}
	cc.status = "deleting"
	resp := clusterToXML(cc)
	delete(s.clusters, id)
	s.mu.Unlock()

	type delResult struct {
		XMLName      xml.Name `xml:"DeleteCacheClusterResult"`
		CacheCluster ccXML    `xml:"CacheCluster"`
	}
	type delResp struct {
		XMLName xml.Name  `xml:"DeleteCacheClusterResponse"`
		Result  delResult `xml:"DeleteCacheClusterResult"`
	}
	h.WriteXML(w, http.StatusOK, delResp{Result: delResult{CacheCluster: resp}})
}

func (s *Service) describeCacheClusters(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := getFormVal(r, "CacheClusterId")

	s.mu.RLock()
	var items []ccXML
	if id != "" {
		if cc, exists := s.clusters[id]; exists {
			items = append(items, clusterToXML(cc))
		}
	} else {
		for _, cc := range s.clusters {
			items = append(items, clusterToXML(cc))
		}
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].CacheClusterId < items[j].CacheClusterId
	})

	type descResult struct {
		XMLName       xml.Name `xml:"DescribeCacheClustersResult"`
		CacheClusters []ccXML  `xml:"CacheClusters>CacheCluster"`
	}
	type descResp struct {
		XMLName xml.Name   `xml:"DescribeCacheClustersResponse"`
		Result  descResult `xml:"DescribeCacheClustersResult"`
	}
	h.WriteXML(w, http.StatusOK, descResp{Result: descResult{CacheClusters: items}})
}

func (s *Service) modifyCacheCluster(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := getFormVal(r, "CacheClusterId")

	s.mu.Lock()
	cc, exists := s.clusters[id]
	if !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "CacheClusterNotFound", "Cache cluster "+id+" not found", http.StatusNotFound)
		return
	}

	if nodeType := getFormVal(r, "CacheNodeType"); nodeType != "" {
		cc.nodeType = nodeType
	}
	if engineVer := getFormVal(r, "EngineVersion"); engineVer != "" {
		cc.engineVer = engineVer
	}
	s.mu.Unlock()

	type modResult struct {
		XMLName      xml.Name `xml:"ModifyCacheClusterResult"`
		CacheCluster ccXML    `xml:"CacheCluster"`
	}
	type modResp struct {
		XMLName xml.Name  `xml:"ModifyCacheClusterResponse"`
		Result  modResult `xml:"ModifyCacheClusterResult"`
	}
	h.WriteXML(w, http.StatusOK, modResp{Result: modResult{CacheCluster: clusterToXML(cc)}})
}

func (s *Service) createReplicationGroup(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := getFormVal(r, "ReplicationGroupId")
	desc := getFormVal(r, "ReplicationGroupDescription")
	if id == "" {
		h.WriteXMLError(w, "Sender", "InvalidParameterValue", "ReplicationGroupId is required", http.StatusBadRequest)
		return
	}

	nodeType := getFormVal(r, "CacheNodeType")
	if nodeType == "" {
		nodeType = "cache.t3.micro"
	}

	s.mu.Lock()
	if _, exists := s.replicationGroups[id]; exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "ReplicationGroupAlreadyExists", "Replication group "+id+" already exists", http.StatusBadRequest)
		return
	}

	arn := fmt.Sprintf("arn:aws:elasticache:us-east-1:%s:replicationgroup:%s", h.DefaultAccountID, id)
	rg := &replicationGroup{
		id:          id,
		arn:         arn,
		description: desc,
		status:      "available",
		nodeType:    nodeType,
		numClusters: 1,
		created:     time.Now().UTC(),
	}
	s.replicationGroups[id] = rg
	s.mu.Unlock()

	type rgResult struct {
		XMLName          xml.Name `xml:"CreateReplicationGroupResult"`
		ReplicationGroup rgXML    `xml:"ReplicationGroup"`
	}
	type rgResp struct {
		XMLName xml.Name `xml:"CreateReplicationGroupResponse"`
		Result  rgResult `xml:"CreateReplicationGroupResult"`
	}
	h.WriteXML(w, http.StatusOK, rgResp{Result: rgResult{ReplicationGroup: rgToXML(rg)}})
}

func (s *Service) deleteReplicationGroup(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := getFormVal(r, "ReplicationGroupId")

	s.mu.Lock()
	rg, exists := s.replicationGroups[id]
	if !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "ReplicationGroupNotFoundFault", "Replication group "+id+" not found", http.StatusNotFound)
		return
	}
	rg.status = "deleting"
	resp := rgToXML(rg)
	delete(s.replicationGroups, id)
	s.mu.Unlock()

	type delResult struct {
		XMLName          xml.Name `xml:"DeleteReplicationGroupResult"`
		ReplicationGroup rgXML    `xml:"ReplicationGroup"`
	}
	type delResp struct {
		XMLName xml.Name  `xml:"DeleteReplicationGroupResponse"`
		Result  delResult `xml:"DeleteReplicationGroupResult"`
	}
	h.WriteXML(w, http.StatusOK, delResp{Result: delResult{ReplicationGroup: resp}})
}

func (s *Service) describeReplicationGroups(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := getFormVal(r, "ReplicationGroupId")

	s.mu.RLock()
	var items []rgXML
	if id != "" {
		if rg, exists := s.replicationGroups[id]; exists {
			items = append(items, rgToXML(rg))
		}
	} else {
		for _, rg := range s.replicationGroups {
			items = append(items, rgToXML(rg))
		}
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].ReplicationGroupId < items[j].ReplicationGroupId
	})

	type descResult struct {
		XMLName           xml.Name `xml:"DescribeReplicationGroupsResult"`
		ReplicationGroups []rgXML  `xml:"ReplicationGroups>ReplicationGroup"`
	}
	type descResp struct {
		XMLName xml.Name   `xml:"DescribeReplicationGroupsResponse"`
		Result  descResult `xml:"DescribeReplicationGroupsResult"`
	}
	h.WriteXML(w, http.StatusOK, descResp{Result: descResult{ReplicationGroups: items}})
}

type ccXML struct {
	CacheClusterId     string `xml:"CacheClusterId"`
	ARN                string `xml:"ARN"`
	CacheClusterStatus string `xml:"CacheClusterStatus"`
	Engine             string `xml:"Engine"`
	EngineVersion      string `xml:"EngineVersion"`
	CacheNodeType      string `xml:"CacheNodeType"`
	NumCacheNodes      int    `xml:"NumCacheNodes"`
}

func clusterToXML(cc *cacheCluster) ccXML {
	return ccXML{
		CacheClusterId:     cc.id,
		ARN:                cc.arn,
		CacheClusterStatus: cc.status,
		Engine:             cc.engine,
		EngineVersion:      cc.engineVer,
		CacheNodeType:      cc.nodeType,
		NumCacheNodes:      cc.numNodes,
	}
}

type rgXML struct {
	ReplicationGroupId string `xml:"ReplicationGroupId"`
	ARN                string `xml:"ARN"`
	Description        string `xml:"Description"`
	Status             string `xml:"Status"`
	CacheNodeType      string `xml:"CacheNodeType"`
	MemberClusters     int    `xml:"MemberClusters"`
}

func rgToXML(rg *replicationGroup) rgXML {
	return rgXML{
		ReplicationGroupId: rg.id,
		ARN:                rg.arn,
		Description:        rg.description,
		Status:             rg.status,
		CacheNodeType:      rg.nodeType,
		MemberClusters:     rg.numClusters,
	}
}

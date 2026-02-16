// Package neptune provides a mock implementation of AWS Neptune.
//
// Supported actions:
//   - CreateDBCluster
//   - DescribeDBClusters
//   - DeleteDBCluster
//   - ModifyDBCluster
//   - CreateDBInstance
//   - DescribeDBInstances
//   - DeleteDBInstance
package neptune

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the Neptune mock.
type Service struct {
	mu        sync.RWMutex
	clusters  map[string]*cluster
	instances map[string]*instance
}

type cluster struct {
	identifier      string
	engine          string
	engineVersion   string
	subnetGroupName string
	status          string
	arn             string
	endpoint        string
	port            int
	created         time.Time
}

type instance struct {
	identifier    string
	instanceClass string
	engine        string
	clusterID     string
	status        string
	arn           string
	endpoint      string
	port          int
	created       time.Time
}

// New creates a new Neptune mock service.
func New() *Service {
	return &Service{
		clusters:  make(map[string]*cluster),
		instances: make(map[string]*instance),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "neptune" }

// Handler returns the HTTP handler for Neptune requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clusters = make(map[string]*cluster)
	s.instances = make(map[string]*instance)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	action := r.FormValue("Action")

	switch action {
	case "CreateDBCluster":
		s.createDBCluster(w, r)
	case "DescribeDBClusters":
		s.describeDBClusters(w, r)
	case "DeleteDBCluster":
		s.deleteDBCluster(w, r)
	case "ModifyDBCluster":
		s.modifyDBCluster(w, r)
	case "CreateDBInstance":
		s.createDBInstance(w, r)
	case "DescribeDBInstances":
		s.describeDBInstances(w, r)
	case "DeleteDBInstance":
		s.deleteDBInstance(w, r)
	default:
		h.WriteXMLError(w, "Sender", "InvalidAction", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

// --- Cluster operations ---

func (s *Service) createDBCluster(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBClusterIdentifier")
	if id == "" {
		h.WriteXMLError(w, "Sender", "InvalidParameterValue", "DBClusterIdentifier is required", http.StatusBadRequest)
		return
	}

	engine := r.FormValue("Engine")
	if engine == "" {
		engine = "neptune"
	}
	engineVersion := r.FormValue("EngineVersion")
	subnetGroupName := r.FormValue("DBSubnetGroupName")

	s.mu.Lock()
	if _, exists := s.clusters[id]; exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "DBClusterAlreadyExistsFault", "DBCluster "+id+" already exists", http.StatusBadRequest)
		return
	}

	arn := fmt.Sprintf("arn:aws:rds:us-east-1:%s:cluster:%s", h.DefaultAccountID, id)
	c := &cluster{
		identifier:      id,
		engine:          engine,
		engineVersion:   engineVersion,
		subnetGroupName: subnetGroupName,
		status:          "available",
		arn:             arn,
		endpoint:        fmt.Sprintf("%s.cluster-xxxxxxxxxxxx.us-east-1.neptune.amazonaws.com", id),
		port:            8182,
		created:         time.Now().UTC(),
	}
	s.clusters[id] = c
	s.mu.Unlock()

	type result struct {
		XMLName   xml.Name   `xml:"CreateDBClusterResult"`
		DBCluster clusterXML `xml:"DBCluster"`
	}
	type resp struct {
		XMLName  xml.Name     `xml:"CreateDBClusterResponse"`
		Result   result       `xml:"CreateDBClusterResult"`
		Metadata responseMeta `xml:"ResponseMetadata"`
	}
	h.WriteXML(w, http.StatusOK, resp{
		Result:   result{DBCluster: clusterToXML(c)},
		Metadata: responseMeta{RequestID: h.NewRequestID()},
	})
}

func (s *Service) describeDBClusters(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBClusterIdentifier")

	s.mu.RLock()
	var items []clusterXML
	if id != "" {
		if c, exists := s.clusters[id]; exists {
			items = append(items, clusterToXML(c))
		}
	} else {
		for _, c := range s.clusters {
			items = append(items, clusterToXML(c))
		}
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].DBClusterIdentifier < items[j].DBClusterIdentifier
	})

	type result struct {
		XMLName    xml.Name     `xml:"DescribeDBClustersResult"`
		DBClusters []clusterXML `xml:"DBClusters>DBCluster"`
	}
	type resp struct {
		XMLName  xml.Name     `xml:"DescribeDBClustersResponse"`
		Result   result       `xml:"DescribeDBClustersResult"`
		Metadata responseMeta `xml:"ResponseMetadata"`
	}
	h.WriteXML(w, http.StatusOK, resp{
		Result:   result{DBClusters: items},
		Metadata: responseMeta{RequestID: h.NewRequestID()},
	})
}

func (s *Service) deleteDBCluster(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBClusterIdentifier")

	s.mu.Lock()
	c, exists := s.clusters[id]
	if !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "DBClusterNotFoundFault", "DBCluster "+id+" not found", http.StatusNotFound)
		return
	}
	c.status = "deleting"
	x := clusterToXML(c)
	delete(s.clusters, id)
	s.mu.Unlock()

	type result struct {
		XMLName   xml.Name   `xml:"DeleteDBClusterResult"`
		DBCluster clusterXML `xml:"DBCluster"`
	}
	type resp struct {
		XMLName  xml.Name     `xml:"DeleteDBClusterResponse"`
		Result   result       `xml:"DeleteDBClusterResult"`
		Metadata responseMeta `xml:"ResponseMetadata"`
	}
	h.WriteXML(w, http.StatusOK, resp{
		Result:   result{DBCluster: x},
		Metadata: responseMeta{RequestID: h.NewRequestID()},
	})
}

func (s *Service) modifyDBCluster(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBClusterIdentifier")

	s.mu.Lock()
	c, exists := s.clusters[id]
	if !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "DBClusterNotFoundFault", "DBCluster "+id+" not found", http.StatusNotFound)
		return
	}

	if v := r.FormValue("EngineVersion"); v != "" {
		c.engineVersion = v
	}
	if v := r.FormValue("DBSubnetGroupName"); v != "" {
		c.subnetGroupName = v
	}
	s.mu.Unlock()

	type result struct {
		XMLName   xml.Name   `xml:"ModifyDBClusterResult"`
		DBCluster clusterXML `xml:"DBCluster"`
	}
	type resp struct {
		XMLName  xml.Name     `xml:"ModifyDBClusterResponse"`
		Result   result       `xml:"ModifyDBClusterResult"`
		Metadata responseMeta `xml:"ResponseMetadata"`
	}
	h.WriteXML(w, http.StatusOK, resp{
		Result:   result{DBCluster: clusterToXML(c)},
		Metadata: responseMeta{RequestID: h.NewRequestID()},
	})
}

// --- Instance operations ---

func (s *Service) createDBInstance(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBInstanceIdentifier")
	if id == "" {
		h.WriteXMLError(w, "Sender", "InvalidParameterValue", "DBInstanceIdentifier is required", http.StatusBadRequest)
		return
	}

	instanceClass := r.FormValue("DBInstanceClass")
	if instanceClass == "" {
		instanceClass = "db.r5.large"
	}
	engine := r.FormValue("Engine")
	if engine == "" {
		engine = "neptune"
	}
	clusterID := r.FormValue("DBClusterIdentifier")

	s.mu.Lock()
	if _, exists := s.instances[id]; exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "DBInstanceAlreadyExistsFault", "DBInstance "+id+" already exists", http.StatusBadRequest)
		return
	}

	arn := fmt.Sprintf("arn:aws:rds:us-east-1:%s:db:%s", h.DefaultAccountID, id)
	inst := &instance{
		identifier:    id,
		instanceClass: instanceClass,
		engine:        engine,
		clusterID:     clusterID,
		status:        "available",
		arn:           arn,
		endpoint:      fmt.Sprintf("%s.xxxxxxxxxxxx.us-east-1.neptune.amazonaws.com", id),
		port:          8182,
		created:       time.Now().UTC(),
	}
	s.instances[id] = inst
	s.mu.Unlock()

	type result struct {
		XMLName    xml.Name    `xml:"CreateDBInstanceResult"`
		DBInstance instanceXML `xml:"DBInstance"`
	}
	type resp struct {
		XMLName  xml.Name     `xml:"CreateDBInstanceResponse"`
		Result   result       `xml:"CreateDBInstanceResult"`
		Metadata responseMeta `xml:"ResponseMetadata"`
	}
	h.WriteXML(w, http.StatusOK, resp{
		Result:   result{DBInstance: instanceToXML(inst)},
		Metadata: responseMeta{RequestID: h.NewRequestID()},
	})
}

func (s *Service) describeDBInstances(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBInstanceIdentifier")

	s.mu.RLock()
	var items []instanceXML
	if id != "" {
		if inst, exists := s.instances[id]; exists {
			items = append(items, instanceToXML(inst))
		}
	} else {
		for _, inst := range s.instances {
			items = append(items, instanceToXML(inst))
		}
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].DBInstanceIdentifier < items[j].DBInstanceIdentifier
	})

	type result struct {
		XMLName     xml.Name      `xml:"DescribeDBInstancesResult"`
		DBInstances []instanceXML `xml:"DBInstances>DBInstance"`
	}
	type resp struct {
		XMLName  xml.Name     `xml:"DescribeDBInstancesResponse"`
		Result   result       `xml:"DescribeDBInstancesResult"`
		Metadata responseMeta `xml:"ResponseMetadata"`
	}
	h.WriteXML(w, http.StatusOK, resp{
		Result:   result{DBInstances: items},
		Metadata: responseMeta{RequestID: h.NewRequestID()},
	})
}

func (s *Service) deleteDBInstance(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBInstanceIdentifier")

	s.mu.Lock()
	inst, exists := s.instances[id]
	if !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "DBInstanceNotFoundFault", "DBInstance "+id+" not found", http.StatusNotFound)
		return
	}
	inst.status = "deleting"
	x := instanceToXML(inst)
	delete(s.instances, id)
	s.mu.Unlock()

	type result struct {
		XMLName    xml.Name    `xml:"DeleteDBInstanceResult"`
		DBInstance instanceXML `xml:"DBInstance"`
	}
	type resp struct {
		XMLName  xml.Name     `xml:"DeleteDBInstanceResponse"`
		Result   result       `xml:"DeleteDBInstanceResult"`
		Metadata responseMeta `xml:"ResponseMetadata"`
	}
	h.WriteXML(w, http.StatusOK, resp{
		Result:   result{DBInstance: x},
		Metadata: responseMeta{RequestID: h.NewRequestID()},
	})
}

// --- XML types ---

type responseMeta struct {
	RequestID string `xml:"RequestId"`
}

type clusterXML struct {
	DBClusterIdentifier string `xml:"DBClusterIdentifier"`
	DBClusterArn        string `xml:"DBClusterArn"`
	Status              string `xml:"Status"`
	Engine              string `xml:"Engine"`
	EngineVersion       string `xml:"EngineVersion"`
	Endpoint            string `xml:"Endpoint"`
	Port                int    `xml:"Port"`
}

func clusterToXML(c *cluster) clusterXML {
	return clusterXML{
		DBClusterIdentifier: c.identifier,
		DBClusterArn:        c.arn,
		Status:              c.status,
		Engine:              c.engine,
		EngineVersion:       c.engineVersion,
		Endpoint:            c.endpoint,
		Port:                c.port,
	}
}

type instanceXML struct {
	DBInstanceIdentifier string `xml:"DBInstanceIdentifier"`
	DBInstanceArn        string `xml:"DBInstanceArn"`
	DBInstanceClass      string `xml:"DBInstanceClass"`
	Engine               string `xml:"Engine"`
	DBClusterIdentifier  string `xml:"DBClusterIdentifier"`
	DBInstanceStatus     string `xml:"DBInstanceStatus"`
	Endpoint             string `xml:"Endpoint"`
	Port                 int    `xml:"Port"`
}

func instanceToXML(inst *instance) instanceXML {
	return instanceXML{
		DBInstanceIdentifier: inst.identifier,
		DBInstanceArn:        inst.arn,
		DBInstanceClass:      inst.instanceClass,
		Engine:               inst.engine,
		DBClusterIdentifier:  inst.clusterID,
		DBInstanceStatus:     inst.status,
		Endpoint:             inst.endpoint,
		Port:                 inst.port,
	}
}

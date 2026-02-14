// Package rds provides a mock implementation of AWS Relational Database Service.
//
// Supported actions:
//   - CreateDBInstance
//   - DeleteDBInstance
//   - DescribeDBInstances
//   - ModifyDBInstance
//   - CreateDBCluster
//   - DeleteDBCluster
//   - DescribeDBClusters
package rds

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the RDS mock.
type Service struct {
	mu        sync.RWMutex
	instances map[string]*dbInstance
	clusters  map[string]*dbCluster
}

type dbInstance struct {
	id               string
	arn              string
	instanceClass    string
	engine           string
	engineVersion    string
	status           string
	masterUsername   string
	allocatedStorage int
	endpoint         string
	port             int
	created          time.Time
}

type dbCluster struct {
	id             string
	arn            string
	engine         string
	engineVersion  string
	status         string
	masterUsername string
	endpoint       string
	readerEndpoint string
	port           int
	created        time.Time
}

// New creates a new RDS mock service.
func New() *Service {
	return &Service{
		instances: make(map[string]*dbInstance),
		clusters:  make(map[string]*dbCluster),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "rds" }

// Handler returns the HTTP handler for RDS requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instances = make(map[string]*dbInstance)
	s.clusters = make(map[string]*dbCluster)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeRDSError(w, "InvalidParameterValue", "could not parse request", http.StatusBadRequest)
		return
	}

	action := r.FormValue("Action")
	switch action {
	case "CreateDBInstance":
		s.createDBInstance(w, r)
	case "DeleteDBInstance":
		s.deleteDBInstance(w, r)
	case "DescribeDBInstances":
		s.describeDBInstances(w, r)
	case "ModifyDBInstance":
		s.modifyDBInstance(w, r)
	case "CreateDBCluster":
		s.createDBCluster(w, r)
	case "DeleteDBCluster":
		s.deleteDBCluster(w, r)
	case "DescribeDBClusters":
		s.describeDBClusters(w, r)
	default:
		writeRDSError(w, "UnsupportedOperation", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createDBInstance(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBInstanceIdentifier")
	if id == "" {
		writeRDSError(w, "InvalidParameterValue", "DBInstanceIdentifier is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.instances[id]; exists {
		s.mu.Unlock()
		writeRDSError(w, "DBInstanceAlreadyExists", "DB instance already exists", http.StatusBadRequest)
		return
	}

	engine := r.FormValue("Engine")
	if engine == "" {
		engine = "mysql"
	}
	engineVersion := r.FormValue("EngineVersion")
	if engineVersion == "" {
		engineVersion = "8.0"
	}
	instanceClass := r.FormValue("DBInstanceClass")
	if instanceClass == "" {
		instanceClass = "db.t3.micro"
	}
	port := 3306
	fmt.Sscanf(r.FormValue("Port"), "%d", &port)
	allocatedStorage := 20
	fmt.Sscanf(r.FormValue("AllocatedStorage"), "%d", &allocatedStorage)

	inst := &dbInstance{
		id:               id,
		arn:              fmt.Sprintf("arn:aws:rds:us-east-1:%s:db:%s", h.DefaultAccountID, id),
		instanceClass:    instanceClass,
		engine:           engine,
		engineVersion:    engineVersion,
		status:           "available",
		masterUsername:   r.FormValue("MasterUsername"),
		allocatedStorage: allocatedStorage,
		endpoint:         fmt.Sprintf("%s.c%s.us-east-1.rds.amazonaws.com", id, h.RandomHex(12)),
		port:             port,
		created:          time.Now().UTC(),
	}
	s.instances[id] = inst
	s.mu.Unlock()

	resp := createDBInstanceResponse{
		Result:    createDBInstanceResult{DBInstance: instanceToXML(inst)},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) deleteDBInstance(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBInstanceIdentifier")

	s.mu.Lock()
	inst, exists := s.instances[id]
	if !exists {
		s.mu.Unlock()
		writeRDSError(w, "DBInstanceNotFound", "DB instance "+id+" not found", http.StatusNotFound)
		return
	}
	inst.status = "deleting"
	delete(s.instances, id)
	s.mu.Unlock()

	resp := deleteDBInstanceResponse{
		Result:    deleteDBInstanceResult{DBInstance: instanceToXML(inst)},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) describeDBInstances(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBInstanceIdentifier")

	s.mu.RLock()
	var members []xmlDBInstance
	if id != "" {
		if inst, exists := s.instances[id]; exists {
			members = append(members, instanceToXML(inst))
		}
	} else {
		for _, inst := range s.instances {
			members = append(members, instanceToXML(inst))
		}
	}
	s.mu.RUnlock()

	sort.Slice(members, func(i, j int) bool { return members[i].Identifier < members[j].Identifier })

	resp := describeDBInstancesResponse{
		Result:    describeDBInstancesResult{DBInstances: members},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) modifyDBInstance(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBInstanceIdentifier")

	s.mu.Lock()
	inst, exists := s.instances[id]
	if !exists {
		s.mu.Unlock()
		writeRDSError(w, "DBInstanceNotFound", "DB instance "+id+" not found", http.StatusNotFound)
		return
	}

	if v := r.FormValue("DBInstanceClass"); v != "" {
		inst.instanceClass = v
	}
	if v := r.FormValue("AllocatedStorage"); v != "" {
		fmt.Sscanf(v, "%d", &inst.allocatedStorage)
	}
	s.mu.Unlock()

	resp := modifyDBInstanceResponse{
		Result:    modifyDBInstanceResult{DBInstance: instanceToXML(inst)},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) createDBCluster(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBClusterIdentifier")
	if id == "" {
		writeRDSError(w, "InvalidParameterValue", "DBClusterIdentifier is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.clusters[id]; exists {
		s.mu.Unlock()
		writeRDSError(w, "DBClusterAlreadyExistsFault", "DB cluster already exists", http.StatusBadRequest)
		return
	}

	engine := r.FormValue("Engine")
	if engine == "" {
		engine = "aurora-mysql"
	}
	port := 3306
	fmt.Sscanf(r.FormValue("Port"), "%d", &port)

	cl := &dbCluster{
		id:             id,
		arn:            fmt.Sprintf("arn:aws:rds:us-east-1:%s:cluster:%s", h.DefaultAccountID, id),
		engine:         engine,
		engineVersion:  r.FormValue("EngineVersion"),
		status:         "available",
		masterUsername: r.FormValue("MasterUsername"),
		endpoint:       fmt.Sprintf("%s.cluster-c%s.us-east-1.rds.amazonaws.com", id, h.RandomHex(12)),
		readerEndpoint: fmt.Sprintf("%s.cluster-ro-c%s.us-east-1.rds.amazonaws.com", id, h.RandomHex(12)),
		port:           port,
		created:        time.Now().UTC(),
	}
	s.clusters[id] = cl
	s.mu.Unlock()

	resp := createDBClusterResponse{
		Result:    createDBClusterResult{DBCluster: clusterToXML(cl)},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) deleteDBCluster(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBClusterIdentifier")

	s.mu.Lock()
	cl, exists := s.clusters[id]
	if !exists {
		s.mu.Unlock()
		writeRDSError(w, "DBClusterNotFoundFault", "DB cluster "+id+" not found", http.StatusNotFound)
		return
	}
	cl.status = "deleting"
	delete(s.clusters, id)
	s.mu.Unlock()

	resp := deleteDBClusterResponse{
		Result:    deleteDBClusterResult{DBCluster: clusterToXML(cl)},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) describeDBClusters(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("DBClusterIdentifier")

	s.mu.RLock()
	var members []xmlDBCluster
	if id != "" {
		if cl, exists := s.clusters[id]; exists {
			members = append(members, clusterToXML(cl))
		}
	} else {
		for _, cl := range s.clusters {
			members = append(members, clusterToXML(cl))
		}
	}
	s.mu.RUnlock()

	sort.Slice(members, func(i, j int) bool { return members[i].Identifier < members[j].Identifier })

	resp := describeDBClustersResponse{
		Result:    describeDBClustersResult{DBClusters: members},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

// XML helpers.

func instanceToXML(inst *dbInstance) xmlDBInstance {
	return xmlDBInstance{
		Identifier:       inst.id,
		Arn:              inst.arn,
		InstanceClass:    inst.instanceClass,
		Engine:           inst.engine,
		EngineVersion:    inst.engineVersion,
		Status:           inst.status,
		MasterUsername:   inst.masterUsername,
		AllocatedStorage: inst.allocatedStorage,
		Endpoint: xmlEndpoint{
			Address: inst.endpoint,
			Port:    inst.port,
		},
	}
}

func clusterToXML(cl *dbCluster) xmlDBCluster {
	return xmlDBCluster{
		Identifier:     cl.id,
		Arn:            cl.arn,
		Engine:         cl.engine,
		EngineVersion:  cl.engineVersion,
		Status:         cl.status,
		MasterUsername: cl.masterUsername,
		Endpoint:       cl.endpoint,
		ReaderEndpoint: cl.readerEndpoint,
		Port:           cl.port,
	}
}

// XML types.

type xmlDBInstance struct {
	Identifier       string      `xml:"DBInstanceIdentifier"`
	Arn              string      `xml:"DBInstanceArn"`
	InstanceClass    string      `xml:"DBInstanceClass"`
	Engine           string      `xml:"Engine"`
	EngineVersion    string      `xml:"EngineVersion"`
	Status           string      `xml:"DBInstanceStatus"`
	MasterUsername   string      `xml:"MasterUsername"`
	AllocatedStorage int         `xml:"AllocatedStorage"`
	Endpoint         xmlEndpoint `xml:"Endpoint"`
}

type xmlEndpoint struct {
	Address string `xml:"Address"`
	Port    int    `xml:"Port"`
}

type xmlDBCluster struct {
	Identifier     string `xml:"DBClusterIdentifier"`
	Arn            string `xml:"DBClusterArn"`
	Engine         string `xml:"Engine"`
	EngineVersion  string `xml:"EngineVersion"`
	Status         string `xml:"Status"`
	MasterUsername string `xml:"MasterUsername"`
	Endpoint       string `xml:"Endpoint"`
	ReaderEndpoint string `xml:"ReaderEndpoint"`
	Port           int    `xml:"Port"`
}

type createDBInstanceResponse struct {
	XMLName   xml.Name               `xml:"CreateDBInstanceResponse"`
	Result    createDBInstanceResult `xml:"CreateDBInstanceResult"`
	RequestID string                 `xml:"ResponseMetadata>RequestId"`
}
type createDBInstanceResult struct {
	DBInstance xmlDBInstance `xml:"DBInstance"`
}

type deleteDBInstanceResponse struct {
	XMLName   xml.Name               `xml:"DeleteDBInstanceResponse"`
	Result    deleteDBInstanceResult `xml:"DeleteDBInstanceResult"`
	RequestID string                 `xml:"ResponseMetadata>RequestId"`
}
type deleteDBInstanceResult struct {
	DBInstance xmlDBInstance `xml:"DBInstance"`
}

type describeDBInstancesResponse struct {
	XMLName   xml.Name                  `xml:"DescribeDBInstancesResponse"`
	Result    describeDBInstancesResult `xml:"DescribeDBInstancesResult"`
	RequestID string                    `xml:"ResponseMetadata>RequestId"`
}
type describeDBInstancesResult struct {
	DBInstances []xmlDBInstance `xml:"DBInstances>DBInstance"`
}

type modifyDBInstanceResponse struct {
	XMLName   xml.Name               `xml:"ModifyDBInstanceResponse"`
	Result    modifyDBInstanceResult `xml:"ModifyDBInstanceResult"`
	RequestID string                 `xml:"ResponseMetadata>RequestId"`
}
type modifyDBInstanceResult struct {
	DBInstance xmlDBInstance `xml:"DBInstance"`
}

type createDBClusterResponse struct {
	XMLName   xml.Name              `xml:"CreateDBClusterResponse"`
	Result    createDBClusterResult `xml:"CreateDBClusterResult"`
	RequestID string                `xml:"ResponseMetadata>RequestId"`
}
type createDBClusterResult struct {
	DBCluster xmlDBCluster `xml:"DBCluster"`
}

type deleteDBClusterResponse struct {
	XMLName   xml.Name              `xml:"DeleteDBClusterResponse"`
	Result    deleteDBClusterResult `xml:"DeleteDBClusterResult"`
	RequestID string                `xml:"ResponseMetadata>RequestId"`
}
type deleteDBClusterResult struct {
	DBCluster xmlDBCluster `xml:"DBCluster"`
}

type describeDBClustersResponse struct {
	XMLName   xml.Name                 `xml:"DescribeDBClustersResponse"`
	Result    describeDBClustersResult `xml:"DescribeDBClustersResult"`
	RequestID string                   `xml:"ResponseMetadata>RequestId"`
}
type describeDBClustersResult struct {
	DBClusters []xmlDBCluster `xml:"DBClusters>DBCluster"`
}

func writeRDSError(w http.ResponseWriter, code, message string, status int) {
	h.WriteXMLError(w, "Sender", code, message, status)
}

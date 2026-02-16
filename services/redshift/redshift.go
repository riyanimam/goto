// Package redshift provides a mock implementation of AWS Redshift.
//
// Supported actions:
//   - CreateCluster
//   - DescribeClusters
//   - DeleteCluster
//   - ModifyCluster
package redshift

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the Redshift mock.
type Service struct {
	mu       sync.RWMutex
	clusters map[string]*cluster
}

type endpoint struct {
	address string
	port    int
}

type cluster struct {
	identifier     string
	nodeType       string
	masterUsername string
	numberOfNodes  int
	status         string
	arn            string
	endpoint       endpoint
	dbName         string
	created        time.Time
}

// New creates a new Redshift mock service.
func New() *Service {
	return &Service{
		clusters: make(map[string]*cluster),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "redshift" }

// Handler returns the HTTP handler for Redshift requests.
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
	r.ParseForm()
	action := r.FormValue("Action")

	switch action {
	case "CreateCluster":
		s.createCluster(w, r)
	case "DescribeClusters":
		s.describeClusters(w, r)
	case "DeleteCluster":
		s.deleteCluster(w, r)
	case "ModifyCluster":
		s.modifyCluster(w, r)
	default:
		h.WriteXMLError(w, "Sender", "InvalidAction", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createCluster(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("ClusterIdentifier")
	if id == "" {
		h.WriteXMLError(w, "Sender", "InvalidParameterValue", "ClusterIdentifier is required", http.StatusBadRequest)
		return
	}

	nodeType := r.FormValue("NodeType")
	if nodeType == "" {
		nodeType = "dc2.large"
	}
	masterUsername := r.FormValue("MasterUsername")
	if masterUsername == "" {
		masterUsername = "awsuser"
	}

	numberOfNodes := 1
	if v := r.FormValue("NumberOfNodes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			numberOfNodes = n
		}
	}

	dbName := r.FormValue("DBName")
	if dbName == "" {
		dbName = "dev"
	}

	s.mu.Lock()
	if _, exists := s.clusters[id]; exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "ClusterAlreadyExists", "Cluster "+id+" already exists", http.StatusBadRequest)
		return
	}

	arn := fmt.Sprintf("arn:aws:redshift:us-east-1:%s:cluster:%s", h.DefaultAccountID, id)
	c := &cluster{
		identifier:     id,
		nodeType:       nodeType,
		masterUsername: masterUsername,
		numberOfNodes:  numberOfNodes,
		status:         "available",
		arn:            arn,
		endpoint: endpoint{
			address: fmt.Sprintf("%s.xxxxxxxxxxxx.us-east-1.redshift.amazonaws.com", id),
			port:    5439,
		},
		dbName:  dbName,
		created: time.Now().UTC(),
	}
	s.clusters[id] = c
	s.mu.Unlock()

	type result struct {
		XMLName xml.Name   `xml:"CreateClusterResult"`
		Cluster clusterXML `xml:"Cluster"`
	}
	type resp struct {
		XMLName  xml.Name     `xml:"CreateClusterResponse"`
		Result   result       `xml:"CreateClusterResult"`
		Metadata responseMeta `xml:"ResponseMetadata"`
	}
	h.WriteXML(w, http.StatusOK, resp{
		Result:   result{Cluster: clusterToXML(c)},
		Metadata: responseMeta{RequestID: h.NewRequestID()},
	})
}

func (s *Service) describeClusters(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("ClusterIdentifier")

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
		return items[i].ClusterIdentifier < items[j].ClusterIdentifier
	})

	type result struct {
		XMLName  xml.Name     `xml:"DescribeClustersResult"`
		Clusters []clusterXML `xml:"Clusters>Cluster"`
	}
	type resp struct {
		XMLName  xml.Name     `xml:"DescribeClustersResponse"`
		Result   result       `xml:"DescribeClustersResult"`
		Metadata responseMeta `xml:"ResponseMetadata"`
	}
	h.WriteXML(w, http.StatusOK, resp{
		Result:   result{Clusters: items},
		Metadata: responseMeta{RequestID: h.NewRequestID()},
	})
}

func (s *Service) deleteCluster(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("ClusterIdentifier")

	s.mu.Lock()
	c, exists := s.clusters[id]
	if !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "ClusterNotFound", "Cluster "+id+" not found", http.StatusNotFound)
		return
	}
	c.status = "deleting"
	x := clusterToXML(c)
	delete(s.clusters, id)
	s.mu.Unlock()

	type result struct {
		XMLName xml.Name   `xml:"DeleteClusterResult"`
		Cluster clusterXML `xml:"Cluster"`
	}
	type resp struct {
		XMLName  xml.Name     `xml:"DeleteClusterResponse"`
		Result   result       `xml:"DeleteClusterResult"`
		Metadata responseMeta `xml:"ResponseMetadata"`
	}
	h.WriteXML(w, http.StatusOK, resp{
		Result:   result{Cluster: x},
		Metadata: responseMeta{RequestID: h.NewRequestID()},
	})
}

func (s *Service) modifyCluster(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("ClusterIdentifier")

	s.mu.Lock()
	c, exists := s.clusters[id]
	if !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "ClusterNotFound", "Cluster "+id+" not found", http.StatusNotFound)
		return
	}

	if nodeType := r.FormValue("NodeType"); nodeType != "" {
		c.nodeType = nodeType
	}
	if v := r.FormValue("NumberOfNodes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.numberOfNodes = n
		}
	}
	s.mu.Unlock()

	type result struct {
		XMLName xml.Name   `xml:"ModifyClusterResult"`
		Cluster clusterXML `xml:"Cluster"`
	}
	type resp struct {
		XMLName  xml.Name     `xml:"ModifyClusterResponse"`
		Result   result       `xml:"ModifyClusterResult"`
		Metadata responseMeta `xml:"ResponseMetadata"`
	}
	h.WriteXML(w, http.StatusOK, resp{
		Result:   result{Cluster: clusterToXML(c)},
		Metadata: responseMeta{RequestID: h.NewRequestID()},
	})
}

type responseMeta struct {
	RequestID string `xml:"RequestId"`
}

type endpointXML struct {
	Address string `xml:"Address"`
	Port    int    `xml:"Port"`
}

type clusterXML struct {
	ClusterIdentifier string      `xml:"ClusterIdentifier"`
	NodeType          string      `xml:"NodeType"`
	MasterUsername    string      `xml:"MasterUsername"`
	NumberOfNodes     int         `xml:"NumberOfNodes"`
	ClusterStatus     string      `xml:"ClusterStatus"`
	ARN               string      `xml:"ARN"`
	Endpoint          endpointXML `xml:"Endpoint"`
	DBName            string      `xml:"DBName"`
}

func clusterToXML(c *cluster) clusterXML {
	return clusterXML{
		ClusterIdentifier: c.identifier,
		NodeType:          c.nodeType,
		MasterUsername:    c.masterUsername,
		NumberOfNodes:     c.numberOfNodes,
		ClusterStatus:     c.status,
		ARN:               c.arn,
		Endpoint: endpointXML{
			Address: c.endpoint.address,
			Port:    c.endpoint.port,
		},
		DBName: c.dbName,
	}
}

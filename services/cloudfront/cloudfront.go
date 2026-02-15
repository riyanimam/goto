// Package cloudfront provides a mock implementation of AWS CloudFront.
//
// Supported actions:
//   - CreateDistribution
//   - GetDistribution
//   - DeleteDistribution
//   - ListDistributions
//   - UpdateDistribution
package cloudfront

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the CloudFront mock.
type Service struct {
	mu            sync.RWMutex
	distributions map[string]*distribution
}

type distribution struct {
	id           string
	arn          string
	domainName   string
	status       string
	enabled      bool
	comment      string
	etag         string
	originDomain string
	originID     string
	created      time.Time
	modified     time.Time
}

// New creates a new CloudFront mock service.
func New() *Service {
	return &Service{
		distributions: make(map[string]*distribution),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "cloudfront" }

// Handler returns the HTTP handler for CloudFront requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.distributions = make(map[string]*distribution)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	switch {
	case path == "/2020-05-31/distribution" && method == http.MethodPost:
		s.createDistribution(w, r)
	case path == "/2020-05-31/distribution" && method == http.MethodGet:
		s.listDistributions(w, r)
	case strings.HasPrefix(path, "/2020-05-31/distribution/") && method == http.MethodGet:
		id := extractDistID(path)
		s.getDistribution(w, r, id)
	case strings.HasPrefix(path, "/2020-05-31/distribution/") && method == http.MethodDelete:
		id := extractDistID(path)
		s.deleteDistribution(w, r, id)
	case strings.HasPrefix(path, "/2020-05-31/distribution/") && method == http.MethodPut:
		id := extractDistID(path)
		s.updateDistribution(w, r, id)
	default:
		h.WriteXMLError(w, "Sender", "InvalidAction", "unsupported operation", http.StatusBadRequest)
	}
}

func extractDistID(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/config"), "/")
	// /2020-05-31/distribution/{id}
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return ""
}

// DistributionConfig represents the XML input for create/update.
type DistributionConfig struct {
	XMLName         xml.Name `xml:"DistributionConfig"`
	CallerReference string   `xml:"CallerReference"`
	Comment         string   `xml:"Comment"`
	Enabled         bool     `xml:"Enabled"`
	Origins         *Origins `xml:"Origins"`
}

// Origins represents the Origins section.
type Origins struct {
	Items []Origin `xml:"Items>Origin"`
}

// Origin represents a single origin.
type Origin struct {
	DomainName string `xml:"DomainName"`
	Id         string `xml:"Id"`
}

func (s *Service) createDistribution(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)

	var cfg DistributionConfig
	if err := xml.Unmarshal(bodyBytes, &cfg); err != nil {
		h.WriteXMLError(w, "Sender", "MalformedXML", "could not parse request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	id := strings.ToUpper(h.RandomID(14))
	arn := fmt.Sprintf("arn:aws:cloudfront::%s:distribution/%s", h.DefaultAccountID, id)
	etag := "E" + h.RandomID(14)
	now := time.Now().UTC()

	var originDomain, originID string
	if cfg.Origins != nil && len(cfg.Origins.Items) > 0 {
		originDomain = cfg.Origins.Items[0].DomainName
		originID = cfg.Origins.Items[0].Id
	}

	dist := &distribution{
		id:           id,
		arn:          arn,
		domainName:   id + ".cloudfront.net",
		status:       "Deployed",
		enabled:      cfg.Enabled,
		comment:      cfg.Comment,
		etag:         etag,
		originDomain: originDomain,
		originID:     originID,
		created:      now,
		modified:     now,
	}
	s.distributions[id] = dist
	s.mu.Unlock()

	w.Header().Set("ETag", etag)
	h.WriteXML(w, http.StatusCreated, distFullResp(dist))
}

func (s *Service) getDistribution(w http.ResponseWriter, _ *http.Request, id string) {
	s.mu.RLock()
	dist, exists := s.distributions[id]
	s.mu.RUnlock()

	if !exists {
		h.WriteXMLError(w, "Sender", "NoSuchDistribution", "Distribution "+id+" not found", http.StatusNotFound)
		return
	}

	w.Header().Set("ETag", dist.etag)
	h.WriteXML(w, http.StatusOK, distFullResp(dist))
}

func (s *Service) deleteDistribution(w http.ResponseWriter, _ *http.Request, id string) {
	s.mu.Lock()
	if _, exists := s.distributions[id]; !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "NoSuchDistribution", "Distribution "+id+" not found", http.StatusNotFound)
		return
	}
	delete(s.distributions, id)
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) listDistributions(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var items []distSummary
	for _, dist := range s.distributions {
		items = append(items, distSummary{
			Id:         dist.id,
			ARN:        dist.arn,
			DomainName: dist.domainName,
			Status:     dist.status,
			Enabled:    dist.enabled,
			Comment:    dist.comment,
		})
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].Id < items[j].Id
	})

	type distributionList struct {
		XMLName  xml.Name      `xml:"DistributionList"`
		Items    []distSummary `xml:"Items>DistributionSummary"`
		Quantity int           `xml:"Quantity"`
	}

	h.WriteXML(w, http.StatusOK, distributionList{
		Items:    items,
		Quantity: len(items),
	})
}

func (s *Service) updateDistribution(w http.ResponseWriter, r *http.Request, id string) {
	bodyBytes, _ := io.ReadAll(r.Body)

	var cfg DistributionConfig
	if err := xml.Unmarshal(bodyBytes, &cfg); err != nil {
		h.WriteXMLError(w, "Sender", "MalformedXML", "could not parse request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	dist, exists := s.distributions[id]
	if !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "NoSuchDistribution", "Distribution "+id+" not found", http.StatusNotFound)
		return
	}

	if cfg.Comment != "" {
		dist.comment = cfg.Comment
	}
	dist.enabled = cfg.Enabled
	dist.modified = time.Now().UTC()
	dist.etag = "E" + h.RandomID(14)

	if cfg.Origins != nil && len(cfg.Origins.Items) > 0 {
		dist.originDomain = cfg.Origins.Items[0].DomainName
		dist.originID = cfg.Origins.Items[0].Id
	}
	s.mu.Unlock()

	w.Header().Set("ETag", dist.etag)
	h.WriteXML(w, http.StatusOK, distFullResp(dist))
}

type distSummary struct {
	XMLName    xml.Name `xml:"DistributionSummary"`
	Id         string   `xml:"Id"`
	ARN        string   `xml:"ARN"`
	DomainName string   `xml:"DomainName"`
	Status     string   `xml:"Status"`
	Enabled    bool     `xml:"Enabled"`
	Comment    string   `xml:"Comment"`
}

type distFullResponse struct {
	XMLName      xml.Name   `xml:"Distribution"`
	Id           string     `xml:"Id"`
	ARN          string     `xml:"ARN"`
	DomainName   string     `xml:"DomainName"`
	Status       string     `xml:"Status"`
	DistConfig   distConfig `xml:"DistributionConfig"`
	LastModified string     `xml:"LastModifiedTime"`
}

type distConfig struct {
	Enabled bool   `xml:"Enabled"`
	Comment string `xml:"Comment"`
	Origins struct {
		Items []struct {
			DomainName string `xml:"DomainName"`
			Id         string `xml:"Id"`
		} `xml:"Items>Origin"`
		Quantity int `xml:"Quantity"`
	} `xml:"Origins"`
}

func distFullResp(dist *distribution) distFullResponse {
	resp := distFullResponse{
		Id:           dist.id,
		ARN:          dist.arn,
		DomainName:   dist.domainName,
		Status:       dist.status,
		LastModified: dist.modified.Format(time.RFC3339),
	}
	resp.DistConfig.Enabled = dist.enabled
	resp.DistConfig.Comment = dist.comment
	resp.DistConfig.Origins.Quantity = 1
	resp.DistConfig.Origins.Items = []struct {
		DomainName string `xml:"DomainName"`
		Id         string `xml:"Id"`
	}{
		{DomainName: dist.originDomain, Id: dist.originID},
	}
	return resp
}

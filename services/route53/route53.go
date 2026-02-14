// Package route53 provides a mock implementation of AWS Route 53.
//
// Supported actions:
//   - CreateHostedZone
//   - GetHostedZone
//   - DeleteHostedZone
//   - ListHostedZones
//   - ChangeResourceRecordSets
//   - ListResourceRecordSets
package route53

import (
	"encoding/json"
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

// Service implements the Route 53 mock.
type Service struct {
	mu          sync.RWMutex
	hostedZones map[string]*hostedZone
	zoneCounter int
}

type hostedZone struct {
	id         string
	name       string
	callerRef  string
	comment    string
	recordSets []*resourceRecordSet
	created    time.Time
}

type resourceRecordSet struct {
	name    string
	rrType  string
	ttl     int
	records []string
}

// New creates a new Route 53 mock service.
func New() *Service {
	return &Service{
		hostedZones: make(map[string]*hostedZone),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "route53" }

// Handler returns the HTTP handler for Route 53 requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hostedZones = make(map[string]*hostedZone)
	s.zoneCounter = 0
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case strings.HasSuffix(path, "/hostedzone") && r.Method == http.MethodGet:
		s.listHostedZones(w, r)
	case strings.HasSuffix(path, "/hostedzone") && r.Method == http.MethodPost:
		s.createHostedZone(w, r)
	case strings.Contains(path, "/hostedzone/") && strings.HasSuffix(path, "/rrset") && r.Method == http.MethodGet:
		id := extractZoneID(path, "/rrset")
		s.listResourceRecordSets(w, r, id)
	case strings.Contains(path, "/hostedzone/") && strings.HasSuffix(path, "/rrset") && r.Method == http.MethodPost:
		id := extractZoneID(path, "/rrset")
		s.changeResourceRecordSets(w, r, id)
	case strings.Contains(path, "/hostedzone/") && r.Method == http.MethodGet:
		id := extractLastSegment(path)
		s.getHostedZone(w, r, id)
	case strings.Contains(path, "/hostedzone/") && r.Method == http.MethodDelete:
		id := extractLastSegment(path)
		s.deleteHostedZone(w, r, id)
	default:
		h.WriteXMLError(w, "Sender", "InvalidAction", "unsupported operation", http.StatusBadRequest)
	}
}

func extractZoneID(path, suffix string) string {
	path = strings.TrimSuffix(path, suffix)
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func extractLastSegment(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func (s *Service) createHostedZone(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var req struct {
		XMLName          xml.Name `xml:"CreateHostedZoneRequest"`
		Name             string   `xml:"Name"`
		CallerReference  string   `xml:"CallerReference"`
		HostedZoneConfig struct {
			Comment string `xml:"Comment"`
		} `xml:"HostedZoneConfig"`
	}
	if err := xml.Unmarshal(bodyBytes, &req); err != nil {
		// Try JSON fallback.
		var jreq map[string]interface{}
		if err2 := json.Unmarshal(bodyBytes, &jreq); err2 == nil {
			req.Name = h.GetString(jreq, "Name")
			req.CallerReference = h.GetString(jreq, "CallerReference")
		}
	}

	if req.Name == "" {
		h.WriteXMLError(w, "Sender", "InvalidInput", "Name is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.zoneCounter++
	zoneID := fmt.Sprintf("Z%s", h.RandomID(13))
	zone := &hostedZone{
		id:        zoneID,
		name:      req.Name,
		callerRef: req.CallerReference,
		comment:   req.HostedZoneConfig.Comment,
		created:   time.Now().UTC(),
		recordSets: []*resourceRecordSet{
			{name: req.Name, rrType: "NS", ttl: 172800, records: []string{"ns-1.awsdns-01.com.", "ns-2.awsdns-02.net."}},
			{name: req.Name, rrType: "SOA", ttl: 900, records: []string{"ns-1.awsdns-01.com. hostmaster.example.com. 1 7200 900 1209600 86400"}},
		},
	}
	s.hostedZones[zoneID] = zone
	s.mu.Unlock()

	resp := createHostedZoneResp{
		HostedZone: zoneToXML(zone),
		ChangeInfo: changeInfo{ID: "/change/" + h.NewRequestID(), Status: "INSYNC", SubmittedAt: zone.created.Format(time.RFC3339)},
		DelegationSet: delegationSet{
			NameServers: []string{"ns-1.awsdns-01.com.", "ns-2.awsdns-02.net."},
		},
	}
	h.WriteXML(w, http.StatusCreated, resp)
}

func (s *Service) getHostedZone(w http.ResponseWriter, _ *http.Request, id string) {
	s.mu.RLock()
	zone, exists := s.hostedZones[id]
	s.mu.RUnlock()

	if !exists {
		h.WriteXMLError(w, "Sender", "NoSuchHostedZone", "No hosted zone found with ID: "+id, http.StatusNotFound)
		return
	}

	resp := getHostedZoneResp{
		HostedZone: zoneToXML(zone),
		DelegationSet: delegationSet{
			NameServers: []string{"ns-1.awsdns-01.com.", "ns-2.awsdns-02.net."},
		},
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) deleteHostedZone(w http.ResponseWriter, _ *http.Request, id string) {
	s.mu.Lock()
	if _, exists := s.hostedZones[id]; !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "NoSuchHostedZone", "No hosted zone found with ID: "+id, http.StatusNotFound)
		return
	}
	delete(s.hostedZones, id)
	s.mu.Unlock()

	resp := deleteHostedZoneResp{
		ChangeInfo: changeInfo{
			ID:          "/change/" + h.NewRequestID(),
			Status:      "INSYNC",
			SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) listHostedZones(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var zones []xmlHostedZone
	for _, z := range s.hostedZones {
		zones = append(zones, zoneToXML(z))
	}
	s.mu.RUnlock()

	sort.Slice(zones, func(i, j int) bool {
		return zones[i].Name < zones[j].Name
	})

	resp := listHostedZonesResp{
		HostedZones: zones,
		IsTruncated: false,
		MaxItems:    "100",
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) changeResourceRecordSets(w http.ResponseWriter, r *http.Request, zoneID string) {
	s.mu.Lock()
	zone, exists := s.hostedZones[zoneID]
	if !exists {
		s.mu.Unlock()
		h.WriteXMLError(w, "Sender", "NoSuchHostedZone", "No hosted zone found with ID: "+zoneID, http.StatusNotFound)
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	var req struct {
		XMLName     xml.Name `xml:"ChangeResourceRecordSetsRequest"`
		ChangeBatch struct {
			Changes []struct {
				Action            string `xml:"Action"`
				ResourceRecordSet struct {
					Name            string `xml:"Name"`
					Type            string `xml:"Type"`
					TTL             int    `xml:"TTL"`
					ResourceRecords struct {
						ResourceRecord []struct {
							Value string `xml:"Value"`
						} `xml:"ResourceRecord"`
					} `xml:"ResourceRecords"`
				} `xml:"ResourceRecordSet"`
			} `xml:"Changes>Change"`
		} `xml:"ChangeBatch"`
	}
	xml.Unmarshal(bodyBytes, &req)

	for _, change := range req.ChangeBatch.Changes {
		rrs := change.ResourceRecordSet
		var records []string
		for _, rr := range rrs.ResourceRecords.ResourceRecord {
			records = append(records, rr.Value)
		}

		switch change.Action {
		case "CREATE", "UPSERT":
			// Remove existing if UPSERT.
			if change.Action == "UPSERT" {
				zone.recordSets = removeRecordSet(zone.recordSets, rrs.Name, rrs.Type)
			}
			zone.recordSets = append(zone.recordSets, &resourceRecordSet{
				name:    rrs.Name,
				rrType:  rrs.Type,
				ttl:     rrs.TTL,
				records: records,
			})
		case "DELETE":
			zone.recordSets = removeRecordSet(zone.recordSets, rrs.Name, rrs.Type)
		}
	}
	s.mu.Unlock()

	resp := changeResourceRecordSetsResp{
		ChangeInfo: changeInfo{
			ID:          "/change/" + h.NewRequestID(),
			Status:      "INSYNC",
			SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) listResourceRecordSets(w http.ResponseWriter, _ *http.Request, zoneID string) {
	s.mu.RLock()
	zone, exists := s.hostedZones[zoneID]
	if !exists {
		s.mu.RUnlock()
		h.WriteXMLError(w, "Sender", "NoSuchHostedZone", "No hosted zone found with ID: "+zoneID, http.StatusNotFound)
		return
	}

	var sets []xmlResourceRecordSet
	for _, rrs := range zone.recordSets {
		var records []xmlResourceRecord
		for _, r := range rrs.records {
			records = append(records, xmlResourceRecord{Value: r})
		}
		sets = append(sets, xmlResourceRecordSet{
			Name:            rrs.name,
			Type:            rrs.rrType,
			TTL:             rrs.ttl,
			ResourceRecords: records,
		})
	}
	s.mu.RUnlock()

	resp := listResourceRecordSetsResp{
		ResourceRecordSets: sets,
		IsTruncated:        false,
		MaxItems:           "100",
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func removeRecordSet(sets []*resourceRecordSet, name, rrType string) []*resourceRecordSet {
	var result []*resourceRecordSet
	for _, rrs := range sets {
		if rrs.name != name || rrs.rrType != rrType {
			result = append(result, rrs)
		}
	}
	return result
}

// XML types.

func zoneToXML(z *hostedZone) xmlHostedZone {
	return xmlHostedZone{
		ID:                     "/hostedzone/" + z.id,
		Name:                   z.name,
		CallerReference:        z.callerRef,
		Config:                 xmlHostedZoneConfig{Comment: z.comment},
		ResourceRecordSetCount: len(z.recordSets),
	}
}

type xmlHostedZone struct {
	ID                     string              `xml:"Id"`
	Name                   string              `xml:"Name"`
	CallerReference        string              `xml:"CallerReference"`
	Config                 xmlHostedZoneConfig `xml:"Config"`
	ResourceRecordSetCount int                 `xml:"ResourceRecordSetCount"`
}

type xmlHostedZoneConfig struct {
	Comment string `xml:"Comment"`
}

type xmlResourceRecordSet struct {
	Name            string              `xml:"Name"`
	Type            string              `xml:"Type"`
	TTL             int                 `xml:"TTL"`
	ResourceRecords []xmlResourceRecord `xml:"ResourceRecords>ResourceRecord"`
}

type xmlResourceRecord struct {
	Value string `xml:"Value"`
}

type changeInfo struct {
	ID          string `xml:"Id"`
	Status      string `xml:"Status"`
	SubmittedAt string `xml:"SubmittedAt"`
}

type delegationSet struct {
	NameServers []string `xml:"NameServers>NameServer"`
}

type createHostedZoneResp struct {
	XMLName       xml.Name      `xml:"CreateHostedZoneResponse"`
	HostedZone    xmlHostedZone `xml:"HostedZone"`
	ChangeInfo    changeInfo    `xml:"ChangeInfo"`
	DelegationSet delegationSet `xml:"DelegationSet"`
}

type getHostedZoneResp struct {
	XMLName       xml.Name      `xml:"GetHostedZoneResponse"`
	HostedZone    xmlHostedZone `xml:"HostedZone"`
	DelegationSet delegationSet `xml:"DelegationSet"`
}

type deleteHostedZoneResp struct {
	XMLName    xml.Name   `xml:"DeleteHostedZoneResponse"`
	ChangeInfo changeInfo `xml:"ChangeInfo"`
}

type listHostedZonesResp struct {
	XMLName     xml.Name        `xml:"ListHostedZonesResponse"`
	HostedZones []xmlHostedZone `xml:"HostedZones>HostedZone"`
	IsTruncated bool            `xml:"IsTruncated"`
	MaxItems    string          `xml:"MaxItems"`
}

type changeResourceRecordSetsResp struct {
	XMLName    xml.Name   `xml:"ChangeResourceRecordSetsResponse"`
	ChangeInfo changeInfo `xml:"ChangeInfo"`
}

type listResourceRecordSetsResp struct {
	XMLName            xml.Name               `xml:"ListResourceRecordSetsResponse"`
	ResourceRecordSets []xmlResourceRecordSet `xml:"ResourceRecordSets>ResourceRecordSet"`
	IsTruncated        bool                   `xml:"IsTruncated"`
	MaxItems           string                 `xml:"MaxItems"`
}

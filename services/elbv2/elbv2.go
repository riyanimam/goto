// Package elbv2 provides a mock implementation of AWS Elastic Load Balancing v2.
//
// Supported actions:
//   - CreateLoadBalancer
//   - DeleteLoadBalancer
//   - DescribeLoadBalancers
//   - CreateTargetGroup
//   - DeleteTargetGroup
//   - DescribeTargetGroups
//   - RegisterTargets
//   - DeregisterTargets
//   - DescribeTargetHealth
//   - CreateListener
//   - DeleteListener
//   - DescribeListeners
package elbv2

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the ELBv2 mock.
type Service struct {
	mu           sync.RWMutex
	lbs          map[string]*loadBalancer
	targetGroups map[string]*targetGroup
	listeners    map[string]*listener
	lbCounter    int
	tgCounter    int
	lnCounter    int
}

type loadBalancer struct {
	name    string
	arn     string
	dnsName string
	scheme  string
	lbType  string
	state   string
	vpcID   string
	created time.Time
}

type targetGroup struct {
	name     string
	arn      string
	protocol string
	port     int
	vpcID    string
	targets  map[string]*targetEntry
}

type targetEntry struct {
	id     string
	port   int
	health string
}

type listener struct {
	arn      string
	lbArn    string
	protocol string
	port     int
}

// New creates a new ELBv2 mock service.
func New() *Service {
	return &Service{
		lbs:          make(map[string]*loadBalancer),
		targetGroups: make(map[string]*targetGroup),
		listeners:    make(map[string]*listener),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "elasticloadbalancing" }

// Handler returns the HTTP handler for ELBv2 requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lbs = make(map[string]*loadBalancer)
	s.targetGroups = make(map[string]*targetGroup)
	s.listeners = make(map[string]*listener)
	s.lbCounter = 0
	s.tgCounter = 0
	s.lnCounter = 0
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeELBError(w, "InvalidInput", "could not parse request", http.StatusBadRequest)
		return
	}

	action := r.FormValue("Action")
	switch action {
	case "CreateLoadBalancer":
		s.createLoadBalancer(w, r)
	case "DeleteLoadBalancer":
		s.deleteLoadBalancer(w, r)
	case "DescribeLoadBalancers":
		s.describeLoadBalancers(w, r)
	case "CreateTargetGroup":
		s.createTargetGroup(w, r)
	case "DeleteTargetGroup":
		s.deleteTargetGroup(w, r)
	case "DescribeTargetGroups":
		s.describeTargetGroups(w, r)
	case "RegisterTargets":
		s.registerTargets(w, r)
	case "DeregisterTargets":
		s.deregisterTargets(w, r)
	case "DescribeTargetHealth":
		s.describeTargetHealth(w, r)
	case "CreateListener":
		s.createListener(w, r)
	case "DeleteListener":
		s.deleteListener(w, r)
	case "DescribeListeners":
		s.describeListeners(w, r)
	default:
		writeELBError(w, "UnsupportedOperation", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createLoadBalancer(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("Name")
	scheme := r.FormValue("Scheme")
	if scheme == "" {
		scheme = "internet-facing"
	}
	lbType := r.FormValue("Type")
	if lbType == "" {
		lbType = "application"
	}

	s.mu.Lock()
	s.lbCounter++
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:%s:loadbalancer/app/%s/%s",
		h.DefaultAccountID, name, h.RandomHex(16))
	lb := &loadBalancer{
		name:    name,
		arn:     arn,
		dnsName: fmt.Sprintf("%s-1234567890.us-east-1.elb.amazonaws.com", name),
		scheme:  scheme,
		lbType:  lbType,
		state:   "active",
		created: time.Now().UTC(),
	}
	s.lbs[arn] = lb
	s.mu.Unlock()

	resp := createLBResponse{
		Result:    createLBResult{LoadBalancers: []xmlLoadBalancer{lbToXML(lb)}},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) deleteLoadBalancer(w http.ResponseWriter, r *http.Request) {
	arn := r.FormValue("LoadBalancerArn")

	s.mu.Lock()
	delete(s.lbs, arn)
	s.mu.Unlock()

	resp := deleteLBResponse{RequestID: h.NewRequestID()}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) describeLoadBalancers(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var lbs []xmlLoadBalancer
	for _, lb := range s.lbs {
		lbs = append(lbs, lbToXML(lb))
	}
	s.mu.RUnlock()

	sort.Slice(lbs, func(i, j int) bool { return lbs[i].Name < lbs[j].Name })

	resp := describeLBsResponse{
		Result:    describeLBsResult{LoadBalancers: lbs},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) createTargetGroup(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("Name")
	protocol := r.FormValue("Protocol")
	if protocol == "" {
		protocol = "HTTP"
	}
	port := 80
	fmt.Sscanf(r.FormValue("Port"), "%d", &port)
	vpcID := r.FormValue("VpcId")

	s.mu.Lock()
	s.tgCounter++
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:%s:targetgroup/%s/%s",
		h.DefaultAccountID, name, h.RandomHex(16))
	tg := &targetGroup{
		name:     name,
		arn:      arn,
		protocol: protocol,
		port:     port,
		vpcID:    vpcID,
		targets:  make(map[string]*targetEntry),
	}
	s.targetGroups[arn] = tg
	s.mu.Unlock()

	resp := createTGResponse{
		Result:    createTGResult{TargetGroups: []xmlTargetGroup{tgToXML(tg)}},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) deleteTargetGroup(w http.ResponseWriter, r *http.Request) {
	arn := r.FormValue("TargetGroupArn")

	s.mu.Lock()
	delete(s.targetGroups, arn)
	s.mu.Unlock()

	resp := deleteTGResponse{RequestID: h.NewRequestID()}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) describeTargetGroups(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var tgs []xmlTargetGroup
	for _, tg := range s.targetGroups {
		tgs = append(tgs, tgToXML(tg))
	}
	s.mu.RUnlock()

	sort.Slice(tgs, func(i, j int) bool { return tgs[i].Name < tgs[j].Name })

	resp := describeTGsResponse{
		Result:    describeTGsResult{TargetGroups: tgs},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) registerTargets(w http.ResponseWriter, r *http.Request) {
	arn := r.FormValue("TargetGroupArn")

	s.mu.Lock()
	tg, exists := s.targetGroups[arn]
	if !exists {
		s.mu.Unlock()
		writeELBError(w, "TargetGroupNotFound", "Target group not found", http.StatusBadRequest)
		return
	}

	for i := 1; ; i++ {
		id := r.FormValue(fmt.Sprintf("Targets.member.%d.Id", i))
		if id == "" {
			break
		}
		port := tg.port
		fmt.Sscanf(r.FormValue(fmt.Sprintf("Targets.member.%d.Port", i)), "%d", &port)
		tg.targets[id] = &targetEntry{id: id, port: port, health: "healthy"}
	}
	s.mu.Unlock()

	resp := registerTargetsResponse{RequestID: h.NewRequestID()}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) deregisterTargets(w http.ResponseWriter, r *http.Request) {
	arn := r.FormValue("TargetGroupArn")

	s.mu.Lock()
	tg, exists := s.targetGroups[arn]
	if !exists {
		s.mu.Unlock()
		writeELBError(w, "TargetGroupNotFound", "Target group not found", http.StatusBadRequest)
		return
	}

	for i := 1; ; i++ {
		id := r.FormValue(fmt.Sprintf("Targets.member.%d.Id", i))
		if id == "" {
			break
		}
		delete(tg.targets, id)
	}
	s.mu.Unlock()

	resp := deregisterTargetsResponse{RequestID: h.NewRequestID()}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) describeTargetHealth(w http.ResponseWriter, r *http.Request) {
	arn := r.FormValue("TargetGroupArn")

	s.mu.RLock()
	tg, exists := s.targetGroups[arn]
	if !exists {
		s.mu.RUnlock()
		writeELBError(w, "TargetGroupNotFound", "Target group not found", http.StatusBadRequest)
		return
	}

	var descs []xmlTargetHealthDescription
	for _, t := range tg.targets {
		descs = append(descs, xmlTargetHealthDescription{
			Target:       xmlTarget{ID: t.id, Port: t.port},
			TargetHealth: xmlTargetHealth{State: t.health},
		})
	}
	s.mu.RUnlock()

	resp := describeTargetHealthResponse{
		Result:    describeTargetHealthResult{TargetHealthDescriptions: descs},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) createListener(w http.ResponseWriter, r *http.Request) {
	lbArn := r.FormValue("LoadBalancerArn")
	protocol := r.FormValue("Protocol")
	if protocol == "" {
		protocol = "HTTP"
	}
	port := 80
	fmt.Sscanf(r.FormValue("Port"), "%d", &port)

	s.mu.Lock()
	s.lnCounter++
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:%s:listener/app/%s/%s",
		h.DefaultAccountID, h.RandomHex(8), h.RandomHex(16))
	ln := &listener{
		arn:      arn,
		lbArn:    lbArn,
		protocol: protocol,
		port:     port,
	}
	s.listeners[arn] = ln
	s.mu.Unlock()

	resp := createListenerResponse{
		Result:    createListenerResult{Listeners: []xmlListener{listenerToXML(ln)}},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) deleteListener(w http.ResponseWriter, r *http.Request) {
	arn := r.FormValue("ListenerArn")

	s.mu.Lock()
	delete(s.listeners, arn)
	s.mu.Unlock()

	resp := deleteListenerResponse{RequestID: h.NewRequestID()}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) describeListeners(w http.ResponseWriter, r *http.Request) {
	lbArn := r.FormValue("LoadBalancerArn")

	s.mu.RLock()
	var lns []xmlListener
	for _, ln := range s.listeners {
		if lbArn == "" || ln.lbArn == lbArn {
			lns = append(lns, listenerToXML(ln))
		}
	}
	s.mu.RUnlock()

	resp := describeListenersResponse{
		Result:    describeListenersResult{Listeners: lns},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

// XML helpers.

func lbToXML(lb *loadBalancer) xmlLoadBalancer {
	return xmlLoadBalancer{
		Arn:     lb.arn,
		Name:    lb.name,
		DNSName: lb.dnsName,
		Scheme:  lb.scheme,
		Type:    lb.lbType,
		State:   xmlLBState{Code: lb.state},
	}
}

func tgToXML(tg *targetGroup) xmlTargetGroup {
	return xmlTargetGroup{
		Arn:      tg.arn,
		Name:     tg.name,
		Protocol: tg.protocol,
		Port:     tg.port,
		VpcID:    tg.vpcID,
	}
}

func listenerToXML(ln *listener) xmlListener {
	return xmlListener{
		Arn:      ln.arn,
		LBArn:    ln.lbArn,
		Protocol: ln.protocol,
		Port:     ln.port,
	}
}

// XML types.

type xmlLoadBalancer struct {
	Arn     string     `xml:"LoadBalancerArn"`
	Name    string     `xml:"LoadBalancerName"`
	DNSName string     `xml:"DNSName"`
	Scheme  string     `xml:"Scheme"`
	Type    string     `xml:"Type"`
	State   xmlLBState `xml:"State"`
}

type xmlLBState struct {
	Code string `xml:"Code"`
}

type xmlTargetGroup struct {
	Arn      string `xml:"TargetGroupArn"`
	Name     string `xml:"TargetGroupName"`
	Protocol string `xml:"Protocol"`
	Port     int    `xml:"Port"`
	VpcID    string `xml:"VpcId"`
}

type xmlListener struct {
	Arn      string `xml:"ListenerArn"`
	LBArn    string `xml:"LoadBalancerArn"`
	Protocol string `xml:"Protocol"`
	Port     int    `xml:"Port"`
}

type xmlTarget struct {
	ID   string `xml:"Id"`
	Port int    `xml:"Port"`
}

type xmlTargetHealth struct {
	State string `xml:"State"`
}

type xmlTargetHealthDescription struct {
	Target       xmlTarget       `xml:"Target"`
	TargetHealth xmlTargetHealth `xml:"TargetHealth"`
}

type createLBResponse struct {
	XMLName   xml.Name       `xml:"CreateLoadBalancerResponse"`
	Result    createLBResult `xml:"CreateLoadBalancerResult"`
	RequestID string         `xml:"ResponseMetadata>RequestId"`
}
type createLBResult struct {
	LoadBalancers []xmlLoadBalancer `xml:"LoadBalancers>member"`
}

type deleteLBResponse struct {
	XMLName   xml.Name `xml:"DeleteLoadBalancerResponse"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type describeLBsResponse struct {
	XMLName   xml.Name          `xml:"DescribeLoadBalancersResponse"`
	Result    describeLBsResult `xml:"DescribeLoadBalancersResult"`
	RequestID string            `xml:"ResponseMetadata>RequestId"`
}
type describeLBsResult struct {
	LoadBalancers []xmlLoadBalancer `xml:"LoadBalancers>member"`
}

type createTGResponse struct {
	XMLName   xml.Name       `xml:"CreateTargetGroupResponse"`
	Result    createTGResult `xml:"CreateTargetGroupResult"`
	RequestID string         `xml:"ResponseMetadata>RequestId"`
}
type createTGResult struct {
	TargetGroups []xmlTargetGroup `xml:"TargetGroups>member"`
}

type deleteTGResponse struct {
	XMLName   xml.Name `xml:"DeleteTargetGroupResponse"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type describeTGsResponse struct {
	XMLName   xml.Name          `xml:"DescribeTargetGroupsResponse"`
	Result    describeTGsResult `xml:"DescribeTargetGroupsResult"`
	RequestID string            `xml:"ResponseMetadata>RequestId"`
}
type describeTGsResult struct {
	TargetGroups []xmlTargetGroup `xml:"TargetGroups>member"`
}

type registerTargetsResponse struct {
	XMLName   xml.Name `xml:"RegisterTargetsResponse"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type deregisterTargetsResponse struct {
	XMLName   xml.Name `xml:"DeregisterTargetsResponse"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type describeTargetHealthResponse struct {
	XMLName   xml.Name                   `xml:"DescribeTargetHealthResponse"`
	Result    describeTargetHealthResult `xml:"DescribeTargetHealthResult"`
	RequestID string                     `xml:"ResponseMetadata>RequestId"`
}
type describeTargetHealthResult struct {
	TargetHealthDescriptions []xmlTargetHealthDescription `xml:"TargetHealthDescriptions>member"`
}

type createListenerResponse struct {
	XMLName   xml.Name             `xml:"CreateListenerResponse"`
	Result    createListenerResult `xml:"CreateListenerResult"`
	RequestID string               `xml:"ResponseMetadata>RequestId"`
}
type createListenerResult struct {
	Listeners []xmlListener `xml:"Listeners>member"`
}

type deleteListenerResponse struct {
	XMLName   xml.Name `xml:"DeleteListenerResponse"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type describeListenersResponse struct {
	XMLName   xml.Name                `xml:"DescribeListenersResponse"`
	Result    describeListenersResult `xml:"DescribeListenersResult"`
	RequestID string                  `xml:"ResponseMetadata>RequestId"`
}
type describeListenersResult struct {
	Listeners []xmlListener `xml:"Listeners>member"`
}

func writeELBError(w http.ResponseWriter, code, message string, status int) {
	h.WriteXMLError(w, "Sender", code, message, status)
}

// Package ec2 provides a mock implementation of AWS Elastic Compute Cloud.
//
// Supported actions:
//   - RunInstances
//   - DescribeInstances
//   - TerminateInstances
//   - CreateVpc
//   - DescribeVpcs
//   - DeleteVpc
//   - CreateSecurityGroup
//   - DescribeSecurityGroups
//   - DeleteSecurityGroup
//   - CreateSubnet
//   - DescribeSubnets
//   - DeleteSubnet
package ec2

import (
	"encoding/xml"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const defaultAccountID = "123456789012"

// Service implements the EC2 mock.
type Service struct {
	mu              sync.RWMutex
	instances       map[string]*instance
	vpcs            map[string]*vpc
	securityGroups  map[string]*securityGroup
	subnets         map[string]*subnet
	instanceCounter int
	vpcCounter      int
	sgCounter       int
	subnetCounter   int
}

type instance struct {
	id           string
	imageID      string
	instanceType string
	state        string
	stateCode    int
	launchTime   time.Time
	subnetID     string
	vpcID        string
	privateIP    string
}

type vpc struct {
	id        string
	cidrBlock string
	state     string
}

type securityGroup struct {
	id          string
	name        string
	description string
	vpcID       string
}

type subnet struct {
	id               string
	vpcID            string
	cidrBlock        string
	availabilityZone string
	state            string
}

// New creates a new EC2 mock service.
func New() *Service {
	return &Service{
		instances:      make(map[string]*instance),
		vpcs:           make(map[string]*vpc),
		securityGroups: make(map[string]*securityGroup),
		subnets:        make(map[string]*subnet),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "ec2" }

// Handler returns the HTTP handler for EC2 requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instances = make(map[string]*instance)
	s.vpcs = make(map[string]*vpc)
	s.securityGroups = make(map[string]*securityGroup)
	s.subnets = make(map[string]*subnet)
	s.instanceCounter = 0
	s.vpcCounter = 0
	s.sgCounter = 0
	s.subnetCounter = 0
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeEC2Error(w, "InvalidRequest", "could not parse request", http.StatusBadRequest)
		return
	}

	action := r.FormValue("Action")
	switch action {
	case "RunInstances":
		s.runInstances(w, r)
	case "DescribeInstances":
		s.describeInstances(w, r)
	case "TerminateInstances":
		s.terminateInstances(w, r)
	case "CreateVpc":
		s.createVpc(w, r)
	case "DescribeVpcs":
		s.describeVpcs(w, r)
	case "DeleteVpc":
		s.deleteVpc(w, r)
	case "CreateSecurityGroup":
		s.createSecurityGroup(w, r)
	case "DescribeSecurityGroups":
		s.describeSecurityGroups(w, r)
	case "DeleteSecurityGroup":
		s.deleteSecurityGroup(w, r)
	case "CreateSubnet":
		s.createSubnet(w, r)
	case "DescribeSubnets":
		s.describeSubnets(w, r)
	case "DeleteSubnet":
		s.deleteSubnet(w, r)
	default:
		writeEC2Error(w, "UnsupportedOperation", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) runInstances(w http.ResponseWriter, r *http.Request) {
	imageID := r.FormValue("ImageId")
	instanceType := r.FormValue("InstanceType")
	if instanceType == "" {
		instanceType = "t2.micro"
	}
	minCount := 1
	fmt.Sscanf(r.FormValue("MinCount"), "%d", &minCount)
	if minCount < 1 {
		minCount = 1
	}

	s.mu.Lock()
	var items []ec2Instance
	for i := 0; i < minCount; i++ {
		s.instanceCounter++
		inst := &instance{
			id:           fmt.Sprintf("i-%017x", s.instanceCounter),
			imageID:      imageID,
			instanceType: instanceType,
			state:        "running",
			stateCode:    16,
			launchTime:   time.Now().UTC(),
			privateIP:    fmt.Sprintf("10.0.%d.%d", rand.Intn(255), rand.Intn(255)+1),
		}
		s.instances[inst.id] = inst
		items = append(items, instanceToXML(inst))
	}
	s.mu.Unlock()

	resp := runInstancesResponse{
		RequestID: newRequestID(),
		Instances: items,
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) describeInstances(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var items []ec2Instance
	for _, inst := range s.instances {
		items = append(items, instanceToXML(inst))
	}
	s.mu.RUnlock()

	resp := describeInstancesResponse{
		RequestID: newRequestID(),
		Reservations: []reservation{{
			ReservationID: "r-" + newRequestID()[:8],
			OwnerID:       defaultAccountID,
			Instances:     items,
		}},
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) terminateInstances(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	var changes []instanceStateChange
	for i := 1; ; i++ {
		id := r.FormValue(fmt.Sprintf("InstanceId.%d", i))
		if id == "" {
			break
		}
		if inst, exists := s.instances[id]; exists {
			changes = append(changes, instanceStateChange{
				InstanceID: id,
				PrevState:  instanceState{Code: inst.stateCode, Name: inst.state},
				CurrState:  instanceState{Code: 48, Name: "terminated"},
			})
			inst.state = "terminated"
			inst.stateCode = 48
		}
	}
	s.mu.Unlock()

	resp := terminateInstancesResponse{
		RequestID: newRequestID(),
		Changes:   changes,
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) createVpc(w http.ResponseWriter, r *http.Request) {
	cidr := r.FormValue("CidrBlock")
	if cidr == "" {
		cidr = "10.0.0.0/16"
	}

	s.mu.Lock()
	s.vpcCounter++
	v := &vpc{
		id:        fmt.Sprintf("vpc-%017x", s.vpcCounter),
		cidrBlock: cidr,
		state:     "available",
	}
	s.vpcs[v.id] = v
	s.mu.Unlock()

	resp := createVpcResponse{
		RequestID: newRequestID(),
		Vpc:       vpcToXML(v),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) describeVpcs(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var items []ec2Vpc
	for _, v := range s.vpcs {
		items = append(items, vpcToXML(v))
	}
	s.mu.RUnlock()

	resp := describeVpcsResponse{
		RequestID: newRequestID(),
		Vpcs:      items,
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) deleteVpc(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("VpcId")

	s.mu.Lock()
	delete(s.vpcs, id)
	s.mu.Unlock()

	resp := simpleResponse{RequestID: newRequestID(), Return: true}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) createSecurityGroup(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("GroupName")
	description := r.FormValue("GroupDescription")
	vpcID := r.FormValue("VpcId")

	s.mu.Lock()
	s.sgCounter++
	sg := &securityGroup{
		id:          fmt.Sprintf("sg-%017x", s.sgCounter),
		name:        name,
		description: description,
		vpcID:       vpcID,
	}
	s.securityGroups[sg.id] = sg
	s.mu.Unlock()

	resp := createSecurityGroupResponse{
		RequestID: newRequestID(),
		GroupID:   sg.id,
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) describeSecurityGroups(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var items []ec2SecurityGroup
	for _, sg := range s.securityGroups {
		items = append(items, ec2SecurityGroup{
			GroupID:     sg.id,
			GroupName:   sg.name,
			Description: sg.description,
			VpcID:       sg.vpcID,
			OwnerID:     defaultAccountID,
		})
	}
	s.mu.RUnlock()

	resp := describeSecurityGroupsResponse{
		RequestID:      newRequestID(),
		SecurityGroups: items,
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) deleteSecurityGroup(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("GroupId")

	s.mu.Lock()
	delete(s.securityGroups, id)
	s.mu.Unlock()

	resp := simpleResponse{RequestID: newRequestID(), Return: true}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) createSubnet(w http.ResponseWriter, r *http.Request) {
	vpcID := r.FormValue("VpcId")
	cidr := r.FormValue("CidrBlock")
	az := r.FormValue("AvailabilityZone")
	if az == "" {
		az = "us-east-1a"
	}

	s.mu.Lock()
	s.subnetCounter++
	sn := &subnet{
		id:               fmt.Sprintf("subnet-%017x", s.subnetCounter),
		vpcID:            vpcID,
		cidrBlock:        cidr,
		availabilityZone: az,
		state:            "available",
	}
	s.subnets[sn.id] = sn
	s.mu.Unlock()

	resp := createSubnetResponse{
		RequestID: newRequestID(),
		Subnet:    subnetToXML(sn),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) describeSubnets(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var items []ec2Subnet
	for _, sn := range s.subnets {
		items = append(items, subnetToXML(sn))
	}
	s.mu.RUnlock()

	resp := describeSubnetsResponse{
		RequestID: newRequestID(),
		Subnets:   items,
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) deleteSubnet(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("SubnetId")

	s.mu.Lock()
	delete(s.subnets, id)
	s.mu.Unlock()

	resp := simpleResponse{RequestID: newRequestID(), Return: true}
	writeXML(w, http.StatusOK, resp)
}

// XML helpers.

func instanceToXML(inst *instance) ec2Instance {
	return ec2Instance{
		InstanceID:   inst.id,
		ImageID:      inst.imageID,
		InstanceType: inst.instanceType,
		State:        instanceState{Code: inst.stateCode, Name: inst.state},
		LaunchTime:   inst.launchTime.Format(time.RFC3339),
		PrivateIP:    inst.privateIP,
	}
}

func vpcToXML(v *vpc) ec2Vpc {
	return ec2Vpc{
		VpcID:     v.id,
		CidrBlock: v.cidrBlock,
		State:     v.state,
		OwnerID:   defaultAccountID,
	}
}

func subnetToXML(sn *subnet) ec2Subnet {
	return ec2Subnet{
		SubnetID:         sn.id,
		VpcID:            sn.vpcID,
		CidrBlock:        sn.cidrBlock,
		AvailabilityZone: sn.availabilityZone,
		State:            sn.state,
	}
}

// XML types.

type ec2Instance struct {
	InstanceID   string        `xml:"instanceId"`
	ImageID      string        `xml:"imageId"`
	InstanceType string        `xml:"instanceType"`
	State        instanceState `xml:"instanceState"`
	LaunchTime   string        `xml:"launchTime"`
	PrivateIP    string        `xml:"privateIpAddress"`
}

type instanceState struct {
	Code int    `xml:"code"`
	Name string `xml:"name"`
}

type ec2Vpc struct {
	VpcID     string `xml:"vpcId"`
	CidrBlock string `xml:"cidrBlock"`
	State     string `xml:"state"`
	OwnerID   string `xml:"ownerId"`
}

type ec2SecurityGroup struct {
	GroupID     string `xml:"groupId"`
	GroupName   string `xml:"groupName"`
	Description string `xml:"groupDescription"`
	VpcID       string `xml:"vpcId"`
	OwnerID     string `xml:"ownerId"`
}

type ec2Subnet struct {
	SubnetID         string `xml:"subnetId"`
	VpcID            string `xml:"vpcId"`
	CidrBlock        string `xml:"cidrBlock"`
	AvailabilityZone string `xml:"availabilityZone"`
	State            string `xml:"state"`
}

type instanceStateChange struct {
	InstanceID string        `xml:"instanceId"`
	PrevState  instanceState `xml:"previousState"`
	CurrState  instanceState `xml:"currentState"`
}

type reservation struct {
	ReservationID string        `xml:"reservationId"`
	OwnerID       string        `xml:"ownerId"`
	Instances     []ec2Instance `xml:"instancesSet>item"`
}

type runInstancesResponse struct {
	XMLName   xml.Name      `xml:"RunInstancesResponse"`
	RequestID string        `xml:"requestId"`
	Instances []ec2Instance `xml:"instancesSet>item"`
}

type describeInstancesResponse struct {
	XMLName      xml.Name      `xml:"DescribeInstancesResponse"`
	RequestID    string        `xml:"requestId"`
	Reservations []reservation `xml:"reservationSet>item"`
}

type terminateInstancesResponse struct {
	XMLName   xml.Name              `xml:"TerminateInstancesResponse"`
	RequestID string                `xml:"requestId"`
	Changes   []instanceStateChange `xml:"instancesSet>item"`
}

type createVpcResponse struct {
	XMLName   xml.Name `xml:"CreateVpcResponse"`
	RequestID string   `xml:"requestId"`
	Vpc       ec2Vpc   `xml:"vpc"`
}

type describeVpcsResponse struct {
	XMLName   xml.Name `xml:"DescribeVpcsResponse"`
	RequestID string   `xml:"requestId"`
	Vpcs      []ec2Vpc `xml:"vpcSet>item"`
}

type createSecurityGroupResponse struct {
	XMLName   xml.Name `xml:"CreateSecurityGroupResponse"`
	RequestID string   `xml:"requestId"`
	GroupID   string   `xml:"groupId"`
}

type describeSecurityGroupsResponse struct {
	XMLName        xml.Name           `xml:"DescribeSecurityGroupsResponse"`
	RequestID      string             `xml:"requestId"`
	SecurityGroups []ec2SecurityGroup `xml:"securityGroupInfo>item"`
}

type createSubnetResponse struct {
	XMLName   xml.Name  `xml:"CreateSubnetResponse"`
	RequestID string    `xml:"requestId"`
	Subnet    ec2Subnet `xml:"subnet"`
}

type describeSubnetsResponse struct {
	XMLName   xml.Name    `xml:"DescribeSubnetsResponse"`
	RequestID string      `xml:"requestId"`
	Subnets   []ec2Subnet `xml:"subnetSet>item"`
}

type simpleResponse struct {
	XMLName   xml.Name `xml:"DeleteVpcResponse"`
	RequestID string   `xml:"requestId"`
	Return    bool     `xml:"return"`
}

type ec2ErrorResponse struct {
	XMLName   xml.Name `xml:"Response"`
	Errors    []ec2Err `xml:"Errors>Error"`
	RequestID string   `xml:"RequestID"`
}

type ec2Err struct {
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func writeXML(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(v)
}

func writeEC2Error(w http.ResponseWriter, code, message string, status int) {
	resp := ec2ErrorResponse{
		Errors:    []ec2Err{{Code: code, Message: message}},
		RequestID: newRequestID(),
	}
	writeXML(w, status, resp)
}

func newRequestID() string {
	const chars = "abcdef0123456789"
	b := make([]byte, 36)
	sections := []int{8, 4, 4, 4, 12}
	pos := 0
	for i, l := range sections {
		if i > 0 {
			b[pos] = '-'
			pos++
		}
		for j := 0; j < l; j++ {
			b[pos] = chars[rand.Intn(len(chars))]
			pos++
		}
	}
	return string(b[:pos])
}

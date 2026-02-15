// Package autoscaling provides a mock implementation of AWS Auto Scaling.
//
// Supported actions:
//   - CreateAutoScalingGroup
//   - DescribeAutoScalingGroups
//   - DeleteAutoScalingGroup
//   - UpdateAutoScalingGroup
//   - CreateLaunchConfiguration
//   - DescribeLaunchConfigurations
//   - DeleteLaunchConfiguration
//   - SetDesiredCapacity
package autoscaling

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the Auto Scaling mock.
type Service struct {
	mu            sync.RWMutex
	groups        map[string]*autoScalingGroup
	launchConfigs map[string]*launchConfiguration
}

type autoScalingGroup struct {
	name                    string
	arn                     string
	minSize                 int
	maxSize                 int
	desiredCapacity         int
	launchConfigurationName string
}

type launchConfiguration struct {
	name         string
	arn          string
	imageID      string
	instanceType string
}

// New creates a new Auto Scaling mock service.
func New() *Service {
	return &Service{
		groups:        make(map[string]*autoScalingGroup),
		launchConfigs: make(map[string]*launchConfiguration),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "autoscaling" }

// Handler returns the HTTP handler for Auto Scaling requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groups = make(map[string]*autoScalingGroup)
	s.launchConfigs = make(map[string]*launchConfiguration)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeASError(w, "ValidationError", "could not parse request", http.StatusBadRequest)
		return
	}

	action := r.FormValue("Action")
	switch action {
	case "CreateAutoScalingGroup":
		s.createAutoScalingGroup(w, r)
	case "DescribeAutoScalingGroups":
		s.describeAutoScalingGroups(w, r)
	case "DeleteAutoScalingGroup":
		s.deleteAutoScalingGroup(w, r)
	case "UpdateAutoScalingGroup":
		s.updateAutoScalingGroup(w, r)
	case "CreateLaunchConfiguration":
		s.createLaunchConfiguration(w, r)
	case "DescribeLaunchConfigurations":
		s.describeLaunchConfigurations(w, r)
	case "DeleteLaunchConfiguration":
		s.deleteLaunchConfiguration(w, r)
	case "SetDesiredCapacity":
		s.setDesiredCapacity(w, r)
	default:
		writeASError(w, "ValidationError", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createAutoScalingGroup(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("AutoScalingGroupName")
	if name == "" {
		writeASError(w, "ValidationError", "AutoScalingGroupName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.groups[name]; exists {
		s.mu.Unlock()
		writeASError(w, "AlreadyExists", "AutoScalingGroup ["+name+"] already exists", http.StatusBadRequest)
		return
	}

	minSize := parseIntParam(r, "MinSize", 0)
	maxSize := parseIntParam(r, "MaxSize", 0)
	desiredCapacity := parseIntParam(r, "DesiredCapacity", minSize)

	g := &autoScalingGroup{
		name:                    name,
		arn:                     fmt.Sprintf("arn:aws:autoscaling:us-east-1:%s:autoScalingGroup:%s:autoScalingGroupName/%s", h.DefaultAccountID, h.RandomHex(16), name),
		minSize:                 minSize,
		maxSize:                 maxSize,
		desiredCapacity:         desiredCapacity,
		launchConfigurationName: r.FormValue("LaunchConfigurationName"),
	}
	s.groups[name] = g
	s.mu.Unlock()

	resp := createAutoScalingGroupResponse{RequestID: h.NewRequestID()}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) describeAutoScalingGroups(w http.ResponseWriter, r *http.Request) {
	// Collect filter names from query parameters.
	var filterNames []string
	for i := 1; ; i++ {
		name := r.FormValue(fmt.Sprintf("AutoScalingGroupNames.member.%d", i))
		if name == "" {
			break
		}
		filterNames = append(filterNames, name)
	}

	s.mu.RLock()
	var groups []xmlAutoScalingGroup
	for _, g := range s.groups {
		if len(filterNames) > 0 && !contains(filterNames, g.name) {
			continue
		}
		groups = append(groups, groupToXML(g))
	}
	s.mu.RUnlock()

	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })

	resp := describeAutoScalingGroupsResponse{
		Result:    describeAutoScalingGroupsResult{AutoScalingGroups: groups},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) deleteAutoScalingGroup(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("AutoScalingGroupName")
	if name == "" {
		writeASError(w, "ValidationError", "AutoScalingGroupName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.groups[name]; !exists {
		s.mu.Unlock()
		writeASError(w, "ValidationError", "AutoScalingGroup ["+name+"] not found", http.StatusBadRequest)
		return
	}
	delete(s.groups, name)
	s.mu.Unlock()

	resp := deleteAutoScalingGroupResponse{RequestID: h.NewRequestID()}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) updateAutoScalingGroup(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("AutoScalingGroupName")
	if name == "" {
		writeASError(w, "ValidationError", "AutoScalingGroupName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	g, exists := s.groups[name]
	if !exists {
		s.mu.Unlock()
		writeASError(w, "ValidationError", "AutoScalingGroup ["+name+"] not found", http.StatusBadRequest)
		return
	}

	if v := r.FormValue("MinSize"); v != "" {
		g.minSize = parseIntParam(r, "MinSize", g.minSize)
	}
	if v := r.FormValue("MaxSize"); v != "" {
		g.maxSize = parseIntParam(r, "MaxSize", g.maxSize)
	}
	if v := r.FormValue("DesiredCapacity"); v != "" {
		g.desiredCapacity = parseIntParam(r, "DesiredCapacity", g.desiredCapacity)
	}
	s.mu.Unlock()

	resp := updateAutoScalingGroupResponse{RequestID: h.NewRequestID()}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) createLaunchConfiguration(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("LaunchConfigurationName")
	if name == "" {
		writeASError(w, "ValidationError", "LaunchConfigurationName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.launchConfigs[name]; exists {
		s.mu.Unlock()
		writeASError(w, "AlreadyExists", "LaunchConfiguration ["+name+"] already exists", http.StatusBadRequest)
		return
	}

	lc := &launchConfiguration{
		name:         name,
		arn:          fmt.Sprintf("arn:aws:autoscaling:us-east-1:%s:launchConfiguration:%s:launchConfigurationName/%s", h.DefaultAccountID, h.RandomHex(16), name),
		imageID:      r.FormValue("ImageId"),
		instanceType: r.FormValue("InstanceType"),
	}
	s.launchConfigs[name] = lc
	s.mu.Unlock()

	resp := createLaunchConfigurationResponse{RequestID: h.NewRequestID()}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) describeLaunchConfigurations(w http.ResponseWriter, r *http.Request) {
	var filterNames []string
	for i := 1; ; i++ {
		name := r.FormValue(fmt.Sprintf("LaunchConfigurationNames.member.%d", i))
		if name == "" {
			break
		}
		filterNames = append(filterNames, name)
	}

	s.mu.RLock()
	var configs []xmlLaunchConfiguration
	for _, lc := range s.launchConfigs {
		if len(filterNames) > 0 && !contains(filterNames, lc.name) {
			continue
		}
		configs = append(configs, launchConfigToXML(lc))
	}
	s.mu.RUnlock()

	sort.Slice(configs, func(i, j int) bool { return configs[i].Name < configs[j].Name })

	resp := describeLaunchConfigurationsResponse{
		Result:    describeLaunchConfigurationsResult{LaunchConfigurations: configs},
		RequestID: h.NewRequestID(),
	}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) deleteLaunchConfiguration(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("LaunchConfigurationName")
	if name == "" {
		writeASError(w, "ValidationError", "LaunchConfigurationName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.launchConfigs[name]; !exists {
		s.mu.Unlock()
		writeASError(w, "ValidationError", "LaunchConfiguration ["+name+"] not found", http.StatusBadRequest)
		return
	}
	delete(s.launchConfigs, name)
	s.mu.Unlock()

	resp := deleteLaunchConfigurationResponse{RequestID: h.NewRequestID()}
	h.WriteXML(w, http.StatusOK, resp)
}

func (s *Service) setDesiredCapacity(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("AutoScalingGroupName")
	if name == "" {
		writeASError(w, "ValidationError", "AutoScalingGroupName is required", http.StatusBadRequest)
		return
	}

	capacityStr := r.FormValue("DesiredCapacity")
	if capacityStr == "" {
		writeASError(w, "ValidationError", "DesiredCapacity is required", http.StatusBadRequest)
		return
	}
	capacity, err := strconv.Atoi(capacityStr)
	if err != nil {
		writeASError(w, "ValidationError", "DesiredCapacity must be a number", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	g, exists := s.groups[name]
	if !exists {
		s.mu.Unlock()
		writeASError(w, "ValidationError", "AutoScalingGroup ["+name+"] not found", http.StatusBadRequest)
		return
	}
	g.desiredCapacity = capacity
	s.mu.Unlock()

	resp := setDesiredCapacityResponse{RequestID: h.NewRequestID()}
	h.WriteXML(w, http.StatusOK, resp)
}

// Helpers.

func parseIntParam(r *http.Request, key string, defaultVal int) int {
	v := r.FormValue(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func groupToXML(g *autoScalingGroup) xmlAutoScalingGroup {
	return xmlAutoScalingGroup{
		Name:                    g.name,
		Arn:                     g.arn,
		MinSize:                 g.minSize,
		MaxSize:                 g.maxSize,
		DesiredCapacity:         g.desiredCapacity,
		LaunchConfigurationName: g.launchConfigurationName,
	}
}

func launchConfigToXML(lc *launchConfiguration) xmlLaunchConfiguration {
	return xmlLaunchConfiguration{
		Name:         lc.name,
		Arn:          lc.arn,
		ImageID:      lc.imageID,
		InstanceType: lc.instanceType,
	}
}

// XML types.

type xmlAutoScalingGroup struct {
	Name                    string `xml:"AutoScalingGroupName"`
	Arn                     string `xml:"AutoScalingGroupARN"`
	MinSize                 int    `xml:"MinSize"`
	MaxSize                 int    `xml:"MaxSize"`
	DesiredCapacity         int    `xml:"DesiredCapacity"`
	LaunchConfigurationName string `xml:"LaunchConfigurationName"`
}

type xmlLaunchConfiguration struct {
	Name         string `xml:"LaunchConfigurationName"`
	Arn          string `xml:"LaunchConfigurationARN"`
	ImageID      string `xml:"ImageId"`
	InstanceType string `xml:"InstanceType"`
}

type createAutoScalingGroupResponse struct {
	XMLName   xml.Name `xml:"CreateAutoScalingGroupResponse"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type describeAutoScalingGroupsResponse struct {
	XMLName   xml.Name                        `xml:"DescribeAutoScalingGroupsResponse"`
	Result    describeAutoScalingGroupsResult `xml:"DescribeAutoScalingGroupsResult"`
	RequestID string                          `xml:"ResponseMetadata>RequestId"`
}
type describeAutoScalingGroupsResult struct {
	AutoScalingGroups []xmlAutoScalingGroup `xml:"AutoScalingGroups>member"`
}

type deleteAutoScalingGroupResponse struct {
	XMLName   xml.Name `xml:"DeleteAutoScalingGroupResponse"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type updateAutoScalingGroupResponse struct {
	XMLName   xml.Name `xml:"UpdateAutoScalingGroupResponse"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type createLaunchConfigurationResponse struct {
	XMLName   xml.Name `xml:"CreateLaunchConfigurationResponse"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type describeLaunchConfigurationsResponse struct {
	XMLName   xml.Name                           `xml:"DescribeLaunchConfigurationsResponse"`
	Result    describeLaunchConfigurationsResult `xml:"DescribeLaunchConfigurationsResult"`
	RequestID string                             `xml:"ResponseMetadata>RequestId"`
}
type describeLaunchConfigurationsResult struct {
	LaunchConfigurations []xmlLaunchConfiguration `xml:"LaunchConfigurations>member"`
}

type deleteLaunchConfigurationResponse struct {
	XMLName   xml.Name `xml:"DeleteLaunchConfigurationResponse"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type setDesiredCapacityResponse struct {
	XMLName   xml.Name `xml:"SetDesiredCapacityResponse"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

func writeASError(w http.ResponseWriter, code, message string, status int) {
	h.WriteXMLError(w, "Sender", code, message, status)
}

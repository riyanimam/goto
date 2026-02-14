// Package cloudformation provides a mock implementation of AWS CloudFormation.
//
// Supported actions:
//   - CreateStack
//   - DeleteStack
//   - DescribeStacks
//   - ListStacks
//   - UpdateStack
package cloudformation

import (
	"encoding/xml"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"
)

const defaultAccountID = "123456789012"

// Service implements the CloudFormation mock.
type Service struct {
	mu     sync.RWMutex
	stacks map[string]*stack // keyed by stack name
}

type stack struct {
	name         string
	id           string
	arn          string
	status       string
	templateBody string
	description  string
	created      time.Time
	updated      time.Time
	parameters   map[string]string
}

// New creates a new CloudFormation mock service.
func New() *Service {
	return &Service{
		stacks: make(map[string]*stack),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "cloudformation" }

// Handler returns the HTTP handler for CloudFormation requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all stacks.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stacks = make(map[string]*stack)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeCFError(w, "ValidationError", "could not parse request", http.StatusBadRequest)
		return
	}

	action := r.FormValue("Action")
	switch action {
	case "CreateStack":
		s.createStack(w, r)
	case "DeleteStack":
		s.deleteStack(w, r)
	case "DescribeStacks":
		s.describeStacks(w, r)
	case "ListStacks":
		s.listStacks(w, r)
	case "UpdateStack":
		s.updateStack(w, r)
	default:
		writeCFError(w, "ValidationError", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createStack(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("StackName")
	if name == "" {
		writeCFError(w, "ValidationError", "StackName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.stacks[name]; exists {
		s.mu.Unlock()
		writeCFError(w, "AlreadyExistsException", "Stack ["+name+"] already exists", http.StatusBadRequest)
		return
	}

	stackID := newRequestID()
	now := time.Now().UTC()
	st := &stack{
		name:         name,
		id:           stackID,
		arn:          fmt.Sprintf("arn:aws:cloudformation:us-east-1:%s:stack/%s/%s", defaultAccountID, name, stackID),
		status:       "CREATE_COMPLETE",
		templateBody: r.FormValue("TemplateBody"),
		description:  r.FormValue("Description"),
		created:      now,
		updated:      now,
		parameters:   make(map[string]string),
	}

	// Parse parameters.
	for i := 1; ; i++ {
		key := r.FormValue(fmt.Sprintf("Parameters.member.%d.ParameterKey", i))
		if key == "" {
			break
		}
		value := r.FormValue(fmt.Sprintf("Parameters.member.%d.ParameterValue", i))
		st.parameters[key] = value
	}

	s.stacks[name] = st
	s.mu.Unlock()

	resp := createStackResponse{
		Result:    createStackResult{StackId: st.arn},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) deleteStack(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("StackName")

	s.mu.Lock()
	st, exists := s.stacks[name]
	if exists {
		st.status = "DELETE_COMPLETE"
		delete(s.stacks, name)
	}
	s.mu.Unlock()

	resp := deleteStackResponse{RequestID: newRequestID()}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) describeStacks(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("StackName")

	s.mu.RLock()
	var members []cfStack
	if name != "" {
		if st, exists := s.stacks[name]; exists {
			members = append(members, stackToXML(st))
		}
	} else {
		for _, st := range s.stacks {
			members = append(members, stackToXML(st))
		}
	}
	s.mu.RUnlock()

	sort.Slice(members, func(i, j int) bool {
		return members[i].StackName < members[j].StackName
	})

	resp := describeStacksResponse{
		Result:    describeStacksResult{Stacks: members},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) listStacks(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var summaries []cfStackSummary
	for _, st := range s.stacks {
		summaries = append(summaries, cfStackSummary{
			StackName:   st.name,
			StackId:     st.arn,
			StackStatus: st.status,
			CreationTime: st.created.Format(time.RFC3339),
		})
	}
	s.mu.RUnlock()

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].StackName < summaries[j].StackName
	})

	resp := listStacksResponse{
		Result:    listStacksResult{StackSummaries: summaries},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) updateStack(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("StackName")

	s.mu.Lock()
	st, exists := s.stacks[name]
	if !exists {
		s.mu.Unlock()
		writeCFError(w, "ValidationError", "Stack ["+name+"] does not exist", http.StatusBadRequest)
		return
	}

	if body := r.FormValue("TemplateBody"); body != "" {
		st.templateBody = body
	}
	st.status = "UPDATE_COMPLETE"
	st.updated = time.Now().UTC()
	arn := st.arn
	s.mu.Unlock()

	resp := updateStackResponse{
		Result:    updateStackResult{StackId: arn},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func stackToXML(st *stack) cfStack {
	var params []cfParameter
	for k, v := range st.parameters {
		params = append(params, cfParameter{
			ParameterKey:   k,
			ParameterValue: v,
		})
	}
	return cfStack{
		StackName:    st.name,
		StackId:      st.arn,
		StackStatus:  st.status,
		Description:  st.description,
		CreationTime: st.created.Format(time.RFC3339),
		Parameters:   params,
	}
}

// XML response types.

type cfStack struct {
	StackName    string        `xml:"StackName"`
	StackId      string        `xml:"StackId"`
	StackStatus  string        `xml:"StackStatus"`
	Description  string        `xml:"Description"`
	CreationTime string        `xml:"CreationTime"`
	Parameters   []cfParameter `xml:"Parameters>member"`
}

type cfParameter struct {
	ParameterKey   string `xml:"ParameterKey"`
	ParameterValue string `xml:"ParameterValue"`
}

type cfStackSummary struct {
	StackName    string `xml:"StackName"`
	StackId      string `xml:"StackId"`
	StackStatus  string `xml:"StackStatus"`
	CreationTime string `xml:"CreationTime"`
}

type createStackResponse struct {
	XMLName   xml.Name          `xml:"CreateStackResponse"`
	XMLNS     string            `xml:"xmlns,attr"`
	Result    createStackResult `xml:"CreateStackResult"`
	RequestID string            `xml:"ResponseMetadata>RequestId"`
}
type createStackResult struct{ StackId string `xml:"StackId"` }

type deleteStackResponse struct {
	XMLName   xml.Name `xml:"DeleteStackResponse"`
	XMLNS     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type describeStacksResponse struct {
	XMLName   xml.Name             `xml:"DescribeStacksResponse"`
	XMLNS     string               `xml:"xmlns,attr"`
	Result    describeStacksResult `xml:"DescribeStacksResult"`
	RequestID string               `xml:"ResponseMetadata>RequestId"`
}
type describeStacksResult struct {
	Stacks []cfStack `xml:"Stacks>member"`
}

type listStacksResponse struct {
	XMLName   xml.Name         `xml:"ListStacksResponse"`
	XMLNS     string           `xml:"xmlns,attr"`
	Result    listStacksResult `xml:"ListStacksResult"`
	RequestID string           `xml:"ResponseMetadata>RequestId"`
}
type listStacksResult struct {
	StackSummaries []cfStackSummary `xml:"StackSummaries>member"`
}

type updateStackResponse struct {
	XMLName   xml.Name          `xml:"UpdateStackResponse"`
	XMLNS     string            `xml:"xmlns,attr"`
	Result    updateStackResult `xml:"UpdateStackResult"`
	RequestID string            `xml:"ResponseMetadata>RequestId"`
}
type updateStackResult struct{ StackId string `xml:"StackId"` }

type cfErrorResponse struct {
	XMLName   xml.Name `xml:"ErrorResponse"`
	Error     cfError  `xml:"Error"`
	RequestID string   `xml:"RequestId"`
}

type cfError struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func writeXML(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(v)
}

func writeCFError(w http.ResponseWriter, code, message string, status int) {
	resp := cfErrorResponse{
		Error: cfError{
			Type:    "Sender",
			Code:    code,
			Message: message,
		},
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

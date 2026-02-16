// Package ssoadmin provides a mock implementation of AWS SSO Admin.
//
// Supported actions:
//   - CreatePermissionSet
//   - DescribePermissionSet
//   - DeletePermissionSet
//   - ListPermissionSets
//   - CreateAccountAssignment
//   - ListAccountAssignments
package ssoadmin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the SSO Admin mock.
type Service struct {
	mu          sync.RWMutex
	permSets    map[string]*permissionSet
	assignments map[string]*accountAssignment
}

type permissionSet struct {
	arn             string
	name            string
	description     string
	sessionDuration string
	instanceArn     string
	created         time.Time
}

type accountAssignment struct {
	instanceArn      string
	targetId         string
	targetType       string
	permissionSetArn string
	principalType    string
	principalId      string
}

// New creates a new SSO Admin mock service.
func New() *Service {
	return &Service{
		permSets:    make(map[string]*permissionSet),
		assignments: make(map[string]*accountAssignment),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "sso" }

// Handler returns the HTTP handler for SSO Admin requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.permSets = make(map[string]*permissionSet)
	s.assignments = make(map[string]*accountAssignment)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteJSONError(w, "InternalServerError", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			h.WriteJSONError(w, "SerializationException", "could not parse request body", http.StatusBadRequest)
			return
		}
	}
	if params == nil {
		params = make(map[string]interface{})
	}

	action := ""
	if target != "" {
		parts := strings.SplitN(target, ".", 2)
		if len(parts) == 2 {
			action = parts[1]
		}
	}

	switch action {
	case "CreatePermissionSet":
		s.createPermissionSet(w, params)
	case "DescribePermissionSet":
		s.describePermissionSet(w, params)
	case "DeletePermissionSet":
		s.deletePermissionSet(w, params)
	case "ListPermissionSets":
		s.listPermissionSets(w, params)
	case "CreateAccountAssignment":
		s.createAccountAssignment(w, params)
	case "ListAccountAssignments":
		s.listAccountAssignments(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createPermissionSet(w http.ResponseWriter, params map[string]interface{}) {
	instanceArn := h.GetString(params, "InstanceArn")
	name := h.GetString(params, "Name")
	description := h.GetString(params, "Description")
	sessionDuration := h.GetString(params, "SessionDuration")

	if instanceArn == "" {
		h.WriteJSONError(w, "ValidationException", "InstanceArn is required", http.StatusBadRequest)
		return
	}
	if name == "" {
		h.WriteJSONError(w, "ValidationException", "Name is required", http.StatusBadRequest)
		return
	}

	psID := h.RandomID(36)
	arn := fmt.Sprintf("arn:aws:sso:::permissionSet/%s/%s", h.DefaultAccountID, psID)
	now := time.Now().UTC()

	ps := &permissionSet{
		arn:             arn,
		name:            name,
		description:     description,
		sessionDuration: sessionDuration,
		instanceArn:     instanceArn,
		created:         now,
	}

	s.mu.Lock()
	s.permSets[arn] = ps
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"PermissionSet": permSetResp(ps),
	})
}

func (s *Service) describePermissionSet(w http.ResponseWriter, params map[string]interface{}) {
	permSetArn := h.GetString(params, "PermissionSetArn")

	s.mu.RLock()
	ps, exists := s.permSets[permSetArn]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "PermissionSet "+permSetArn+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"PermissionSet": permSetResp(ps),
	})
}

func (s *Service) deletePermissionSet(w http.ResponseWriter, params map[string]interface{}) {
	permSetArn := h.GetString(params, "PermissionSetArn")

	s.mu.Lock()
	_, exists := s.permSets[permSetArn]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "PermissionSet "+permSetArn+" not found", http.StatusNotFound)
		return
	}
	delete(s.permSets, permSetArn)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listPermissionSets(w http.ResponseWriter, params map[string]interface{}) {
	s.mu.RLock()
	var arns []string
	for arn := range s.permSets {
		arns = append(arns, arn)
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"PermissionSets": arns,
	})
}

func (s *Service) createAccountAssignment(w http.ResponseWriter, params map[string]interface{}) {
	instanceArn := h.GetString(params, "InstanceArn")
	targetId := h.GetString(params, "TargetId")
	targetType := h.GetString(params, "TargetType")
	permSetArn := h.GetString(params, "PermissionSetArn")
	principalType := h.GetString(params, "PrincipalType")
	principalId := h.GetString(params, "PrincipalId")

	if instanceArn == "" || permSetArn == "" || principalId == "" {
		h.WriteJSONError(w, "ValidationException", "InstanceArn, PermissionSetArn, and PrincipalId are required", http.StatusBadRequest)
		return
	}

	key := instanceArn + "+" + permSetArn + "+" + principalId

	aa := &accountAssignment{
		instanceArn:      instanceArn,
		targetId:         targetId,
		targetType:       targetType,
		permissionSetArn: permSetArn,
		principalType:    principalType,
		principalId:      principalId,
	}

	s.mu.Lock()
	s.assignments[key] = aa
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"AccountAssignmentCreationStatus": map[string]interface{}{
			"Status": "SUCCEEDED",
		},
	})
}

func (s *Service) listAccountAssignments(w http.ResponseWriter, params map[string]interface{}) {
	instanceArn := h.GetString(params, "InstanceArn")
	accountId := h.GetString(params, "AccountId")
	permSetArn := h.GetString(params, "PermissionSetArn")

	s.mu.RLock()
	var list []map[string]interface{}
	for _, aa := range s.assignments {
		if aa.instanceArn == instanceArn && aa.permissionSetArn == permSetArn && aa.targetId == accountId {
			list = append(list, map[string]interface{}{
				"AccountId":        aa.targetId,
				"PermissionSetArn": aa.permissionSetArn,
				"PrincipalType":    aa.principalType,
				"PrincipalId":      aa.principalId,
			})
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"AccountAssignments": list,
	})
}

func permSetResp(ps *permissionSet) map[string]interface{} {
	return map[string]interface{}{
		"PermissionSetArn": ps.arn,
		"Name":             ps.name,
		"Description":      ps.description,
		"CreatedDate":      float64(ps.created.Unix()),
		"SessionDuration":  ps.sessionDuration,
	}
}

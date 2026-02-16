// Package iam provides a mock implementation of AWS Identity and Access Management.
//
// Supported actions:
//   - CreateUser
//   - GetUser
//   - DeleteUser
//   - ListUsers
//   - CreateRole
//   - GetRole
//   - DeleteRole
//   - ListRoles
//   - CreatePolicy
//   - GetPolicy
//   - DeletePolicy
//   - ListPolicies
//   - AttachRolePolicy
//   - DetachRolePolicy
package iam

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

// Service implements the IAM mock.
type Service struct {
	mu           sync.RWMutex
	users        map[string]*user
	roles        map[string]*role
	policies     map[string]*policy
	rolePolicies map[string]map[string]bool // roleArn -> set of policyArns
}

type user struct {
	name    string
	arn     string
	userID  string
	path    string
	created time.Time
}

type role struct {
	name                string
	arn                 string
	roleID              string
	path                string
	assumeRolePolicyDoc string
	description         string
	created             time.Time
}

type policy struct {
	name     string
	arn      string
	policyID string
	path     string
	document string
	created  time.Time
}

// New creates a new IAM mock service.
func New() *Service {
	return &Service{
		users:        make(map[string]*user),
		roles:        make(map[string]*role),
		policies:     make(map[string]*policy),
		rolePolicies: make(map[string]map[string]bool),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "iam" }

// Handler returns the HTTP handler for IAM requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = make(map[string]*user)
	s.roles = make(map[string]*role)
	s.policies = make(map[string]*policy)
	s.rolePolicies = make(map[string]map[string]bool)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeIAMError(w, "InvalidInput", "could not parse request", http.StatusBadRequest)
		return
	}

	action := r.FormValue("Action")
	switch action {
	case "CreateUser":
		s.createUser(w, r)
	case "GetUser":
		s.getUser(w, r)
	case "DeleteUser":
		s.deleteUser(w, r)
	case "ListUsers":
		s.listUsers(w, r)
	case "CreateRole":
		s.createRole(w, r)
	case "GetRole":
		s.getRole(w, r)
	case "DeleteRole":
		s.deleteRole(w, r)
	case "ListRoles":
		s.listRoles(w, r)
	case "CreatePolicy":
		s.createPolicy(w, r)
	case "GetPolicy":
		s.getPolicy(w, r)
	case "DeletePolicy":
		s.deletePolicy(w, r)
	case "ListPolicies":
		s.listPolicies(w, r)
	case "AttachRolePolicy":
		s.attachRolePolicy(w, r)
	case "DetachRolePolicy":
		s.detachRolePolicy(w, r)
	default:
		writeIAMError(w, "InvalidAction", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createUser(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("UserName")
	path := r.FormValue("Path")
	if path == "" {
		path = "/"
	}

	s.mu.Lock()
	if _, exists := s.users[name]; exists {
		s.mu.Unlock()
		writeIAMError(w, "EntityAlreadyExists", "User with name "+name+" already exists.", http.StatusConflict)
		return
	}

	u := &user{
		name:    name,
		arn:     fmt.Sprintf("arn:aws:iam::%s:user%s%s", defaultAccountID, path, name),
		userID:  "AIDA" + randomID(16),
		path:    path,
		created: time.Now().UTC(),
	}
	s.users[name] = u
	s.mu.Unlock()

	resp := createUserResponse{
		Result:    createUserResult{User: userXML(u)},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) getUser(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("UserName")

	s.mu.RLock()
	u, exists := s.users[name]
	s.mu.RUnlock()

	if !exists {
		writeIAMError(w, "NoSuchEntity", "The user with name "+name+" cannot be found.", http.StatusNotFound)
		return
	}

	resp := getUserResponse{
		Result:    getUserResult{User: userXML(u)},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) deleteUser(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("UserName")

	s.mu.Lock()
	if _, exists := s.users[name]; !exists {
		s.mu.Unlock()
		writeIAMError(w, "NoSuchEntity", "The user with name "+name+" cannot be found.", http.StatusNotFound)
		return
	}
	delete(s.users, name)
	s.mu.Unlock()

	resp := deleteUserResponse{RequestID: newRequestID()}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) listUsers(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var members []iamUser
	for _, u := range s.users {
		members = append(members, userXML(u))
	}
	s.mu.RUnlock()

	sort.Slice(members, func(i, j int) bool {
		return members[i].UserName < members[j].UserName
	})

	resp := listUsersResponse{
		Result:    listUsersResult{Users: members, IsTruncated: false},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) createRole(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("RoleName")
	path := r.FormValue("Path")
	assumeRolePolicy := r.FormValue("AssumeRolePolicyDocument")
	description := r.FormValue("Description")
	if path == "" {
		path = "/"
	}

	s.mu.Lock()
	if _, exists := s.roles[name]; exists {
		s.mu.Unlock()
		writeIAMError(w, "EntityAlreadyExists", "Role with name "+name+" already exists.", http.StatusConflict)
		return
	}

	rl := &role{
		name:                name,
		arn:                 fmt.Sprintf("arn:aws:iam::%s:role%s%s", defaultAccountID, path, name),
		roleID:              "AROA" + randomID(16),
		path:                path,
		assumeRolePolicyDoc: assumeRolePolicy,
		description:         description,
		created:             time.Now().UTC(),
	}
	s.roles[name] = rl
	s.mu.Unlock()

	resp := createRoleResponse{
		Result:    createRoleResult{Role: roleXML(rl)},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) getRole(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("RoleName")

	s.mu.RLock()
	rl, exists := s.roles[name]
	s.mu.RUnlock()

	if !exists {
		writeIAMError(w, "NoSuchEntity", "The role with name "+name+" cannot be found.", http.StatusNotFound)
		return
	}

	resp := getRoleResponse{
		Result:    getRoleResult{Role: roleXML(rl)},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) deleteRole(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("RoleName")

	s.mu.Lock()
	if _, exists := s.roles[name]; !exists {
		s.mu.Unlock()
		writeIAMError(w, "NoSuchEntity", "The role with name "+name+" cannot be found.", http.StatusNotFound)
		return
	}
	delete(s.roles, name)
	s.mu.Unlock()

	resp := deleteRoleResponse{RequestID: newRequestID()}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) listRoles(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var members []iamRole
	for _, rl := range s.roles {
		members = append(members, roleXML(rl))
	}
	s.mu.RUnlock()

	sort.Slice(members, func(i, j int) bool {
		return members[i].RoleName < members[j].RoleName
	})

	resp := listRolesResponse{
		Result:    listRolesResult{Roles: members, IsTruncated: false},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) createPolicy(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("PolicyName")
	path := r.FormValue("Path")
	document := r.FormValue("PolicyDocument")
	if path == "" {
		path = "/"
	}

	s.mu.Lock()
	arn := fmt.Sprintf("arn:aws:iam::%s:policy%s%s", defaultAccountID, path, name)
	if _, exists := s.policies[arn]; exists {
		s.mu.Unlock()
		writeIAMError(w, "EntityAlreadyExists", "A policy called "+name+" already exists.", http.StatusConflict)
		return
	}

	p := &policy{
		name:     name,
		arn:      arn,
		policyID: "ANPA" + randomID(16),
		path:     path,
		document: document,
		created:  time.Now().UTC(),
	}
	s.policies[arn] = p
	s.mu.Unlock()

	resp := createPolicyResponse{
		Result:    createPolicyResult{Policy: policyXML(p)},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) getPolicy(w http.ResponseWriter, r *http.Request) {
	arn := r.FormValue("PolicyArn")

	s.mu.RLock()
	p, exists := s.policies[arn]
	s.mu.RUnlock()

	if !exists {
		writeIAMError(w, "NoSuchEntity", "Policy "+arn+" does not exist.", http.StatusNotFound)
		return
	}

	resp := getPolicyResponse{
		Result:    getPolicyResult{Policy: policyXML(p)},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) deletePolicy(w http.ResponseWriter, r *http.Request) {
	arn := r.FormValue("PolicyArn")

	s.mu.Lock()
	if _, exists := s.policies[arn]; !exists {
		s.mu.Unlock()
		writeIAMError(w, "NoSuchEntity", "Policy "+arn+" does not exist.", http.StatusNotFound)
		return
	}
	delete(s.policies, arn)
	s.mu.Unlock()

	resp := deletePolicyResponse{RequestID: newRequestID()}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) listPolicies(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var members []iamPolicy
	for _, p := range s.policies {
		members = append(members, policyXML(p))
	}
	s.mu.RUnlock()

	sort.Slice(members, func(i, j int) bool {
		return members[i].PolicyName < members[j].PolicyName
	})

	resp := listPoliciesResponse{
		Result:    listPoliciesResult{Policies: members, IsTruncated: false},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) attachRolePolicy(w http.ResponseWriter, r *http.Request) {
	roleName := r.FormValue("RoleName")
	policyArn := r.FormValue("PolicyArn")

	s.mu.Lock()
	rl, exists := s.roles[roleName]
	if !exists {
		s.mu.Unlock()
		writeIAMError(w, "NoSuchEntity", "The role with name "+roleName+" cannot be found.", http.StatusNotFound)
		return
	}

	if s.rolePolicies[rl.arn] == nil {
		s.rolePolicies[rl.arn] = make(map[string]bool)
	}
	s.rolePolicies[rl.arn][policyArn] = true
	s.mu.Unlock()

	resp := attachRolePolicyResponse{RequestID: newRequestID()}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) detachRolePolicy(w http.ResponseWriter, r *http.Request) {
	roleName := r.FormValue("RoleName")
	policyArn := r.FormValue("PolicyArn")

	s.mu.Lock()
	rl, exists := s.roles[roleName]
	if !exists {
		s.mu.Unlock()
		writeIAMError(w, "NoSuchEntity", "The role with name "+roleName+" cannot be found.", http.StatusNotFound)
		return
	}
	if s.rolePolicies[rl.arn] != nil {
		delete(s.rolePolicies[rl.arn], policyArn)
	}
	s.mu.Unlock()

	resp := detachRolePolicyResponse{RequestID: newRequestID()}
	writeXML(w, http.StatusOK, resp)
}

// XML type helpers.

func userXML(u *user) iamUser {
	return iamUser{
		UserName:   u.name,
		UserId:     u.userID,
		Arn:        u.arn,
		Path:       u.path,
		CreateDate: u.created.Format(time.RFC3339),
	}
}

func roleXML(rl *role) iamRole {
	return iamRole{
		RoleName:                 rl.name,
		RoleId:                   rl.roleID,
		Arn:                      rl.arn,
		Path:                     rl.path,
		AssumeRolePolicyDocument: rl.assumeRolePolicyDoc,
		Description:              rl.description,
		CreateDate:               rl.created.Format(time.RFC3339),
	}
}

func policyXML(p *policy) iamPolicy {
	return iamPolicy{
		PolicyName: p.name,
		PolicyId:   p.policyID,
		Arn:        p.arn,
		Path:       p.path,
		CreateDate: p.created.Format(time.RFC3339),
	}
}

// XML response types.

type iamUser struct {
	UserName   string `xml:"UserName"`
	UserId     string `xml:"UserId"`
	Arn        string `xml:"Arn"`
	Path       string `xml:"Path"`
	CreateDate string `xml:"CreateDate"`
}

type iamRole struct {
	RoleName                 string `xml:"RoleName"`
	RoleId                   string `xml:"RoleId"`
	Arn                      string `xml:"Arn"`
	Path                     string `xml:"Path"`
	AssumeRolePolicyDocument string `xml:"AssumeRolePolicyDocument"`
	Description              string `xml:"Description"`
	CreateDate               string `xml:"CreateDate"`
}

type iamPolicy struct {
	PolicyName string `xml:"PolicyName"`
	PolicyId   string `xml:"PolicyId"`
	Arn        string `xml:"Arn"`
	Path       string `xml:"Path"`
	CreateDate string `xml:"CreateDate"`
}

type createUserResponse struct {
	XMLName   xml.Name         `xml:"CreateUserResponse"`
	XMLNS     string           `xml:"xmlns,attr"`
	Result    createUserResult `xml:"CreateUserResult"`
	RequestID string           `xml:"ResponseMetadata>RequestId"`
}
type createUserResult struct {
	User iamUser `xml:"User"`
}

type getUserResponse struct {
	XMLName   xml.Name      `xml:"GetUserResponse"`
	XMLNS     string        `xml:"xmlns,attr"`
	Result    getUserResult `xml:"GetUserResult"`
	RequestID string        `xml:"ResponseMetadata>RequestId"`
}
type getUserResult struct {
	User iamUser `xml:"User"`
}

type deleteUserResponse struct {
	XMLName   xml.Name `xml:"DeleteUserResponse"`
	XMLNS     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type listUsersResponse struct {
	XMLName   xml.Name        `xml:"ListUsersResponse"`
	XMLNS     string          `xml:"xmlns,attr"`
	Result    listUsersResult `xml:"ListUsersResult"`
	RequestID string          `xml:"ResponseMetadata>RequestId"`
}
type listUsersResult struct {
	Users       []iamUser `xml:"Users>member"`
	IsTruncated bool      `xml:"IsTruncated"`
}

type createRoleResponse struct {
	XMLName   xml.Name         `xml:"CreateRoleResponse"`
	XMLNS     string           `xml:"xmlns,attr"`
	Result    createRoleResult `xml:"CreateRoleResult"`
	RequestID string           `xml:"ResponseMetadata>RequestId"`
}
type createRoleResult struct {
	Role iamRole `xml:"Role"`
}

type getRoleResponse struct {
	XMLName   xml.Name      `xml:"GetRoleResponse"`
	XMLNS     string        `xml:"xmlns,attr"`
	Result    getRoleResult `xml:"GetRoleResult"`
	RequestID string        `xml:"ResponseMetadata>RequestId"`
}
type getRoleResult struct {
	Role iamRole `xml:"Role"`
}

type deleteRoleResponse struct {
	XMLName   xml.Name `xml:"DeleteRoleResponse"`
	XMLNS     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type listRolesResponse struct {
	XMLName   xml.Name        `xml:"ListRolesResponse"`
	XMLNS     string          `xml:"xmlns,attr"`
	Result    listRolesResult `xml:"ListRolesResult"`
	RequestID string          `xml:"ResponseMetadata>RequestId"`
}
type listRolesResult struct {
	Roles       []iamRole `xml:"Roles>member"`
	IsTruncated bool      `xml:"IsTruncated"`
}

type createPolicyResponse struct {
	XMLName   xml.Name           `xml:"CreatePolicyResponse"`
	XMLNS     string             `xml:"xmlns,attr"`
	Result    createPolicyResult `xml:"CreatePolicyResult"`
	RequestID string             `xml:"ResponseMetadata>RequestId"`
}
type createPolicyResult struct {
	Policy iamPolicy `xml:"Policy"`
}

type getPolicyResponse struct {
	XMLName   xml.Name        `xml:"GetPolicyResponse"`
	XMLNS     string          `xml:"xmlns,attr"`
	Result    getPolicyResult `xml:"GetPolicyResult"`
	RequestID string          `xml:"ResponseMetadata>RequestId"`
}
type getPolicyResult struct {
	Policy iamPolicy `xml:"Policy"`
}

type deletePolicyResponse struct {
	XMLName   xml.Name `xml:"DeletePolicyResponse"`
	XMLNS     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type listPoliciesResponse struct {
	XMLName   xml.Name           `xml:"ListPoliciesResponse"`
	XMLNS     string             `xml:"xmlns,attr"`
	Result    listPoliciesResult `xml:"ListPoliciesResult"`
	RequestID string             `xml:"ResponseMetadata>RequestId"`
}
type listPoliciesResult struct {
	Policies    []iamPolicy `xml:"Policies>member"`
	IsTruncated bool        `xml:"IsTruncated"`
}

type attachRolePolicyResponse struct {
	XMLName   xml.Name `xml:"AttachRolePolicyResponse"`
	XMLNS     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type detachRolePolicyResponse struct {
	XMLName   xml.Name `xml:"DetachRolePolicyResponse"`
	XMLNS     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type iamErrorResponse struct {
	XMLName   xml.Name `xml:"ErrorResponse"`
	Error     iamError `xml:"Error"`
	RequestID string   `xml:"RequestId"`
}

type iamError struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

// Helper functions.

func writeXML(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(v)
}

func writeIAMError(w http.ResponseWriter, code, message string, status int) {
	resp := iamErrorResponse{
		Error: iamError{
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

func randomID(n int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

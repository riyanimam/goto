// Package transfer provides a mock implementation of AWS Transfer Family.
//
// Supported actions:
//   - CreateServer
//   - DescribeServer
//   - DeleteServer
//   - ListServers
//   - CreateUser
//   - DescribeUser
//   - DeleteUser
package transfer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the Transfer Family mock.
type Service struct {
	mu      sync.RWMutex
	servers map[string]*server
}

type server struct {
	id                   string
	arn                  string
	endpointType         string
	identityProviderType string
	protocols            []string
	state                string
	users                map[string]*user
}

type user struct {
	userName      string
	serverID      string
	arn           string
	role          string
	homeDirectory string
}

// New creates a new Transfer Family mock service.
func New() *Service {
	return &Service{
		servers: make(map[string]*server),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "transfer" }

// Handler returns the HTTP handler for Transfer Family requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.servers = make(map[string]*server)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteJSONError(w, "InternalFailure", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		json.Unmarshal(bodyBytes, &params)
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
	case "CreateServer":
		s.createServer(w, params)
	case "DescribeServer":
		s.describeServer(w, params)
	case "DeleteServer":
		s.deleteServer(w, params)
	case "ListServers":
		s.listServers(w, params)
	case "CreateUser":
		s.createUser(w, params)
	case "DescribeUser":
		s.describeUser(w, params)
	case "DeleteUser":
		s.deleteUser(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createServer(w http.ResponseWriter, params map[string]interface{}) {
	endpointType := h.GetString(params, "EndpointType")
	if endpointType == "" {
		endpointType = "PUBLIC"
	}

	identityProviderType := h.GetString(params, "IdentityProviderType")
	if identityProviderType == "" {
		identityProviderType = "SERVICE_MANAGED"
	}

	var protocols []string
	if p, ok := params["Protocols"].([]interface{}); ok {
		for _, v := range p {
			if proto, ok := v.(string); ok {
				protocols = append(protocols, proto)
			}
		}
	}
	if len(protocols) == 0 {
		protocols = []string{"SFTP"}
	}

	id := "s-" + h.RandomHex(17)
	arn := fmt.Sprintf("arn:aws:transfer:us-east-1:%s:server/%s", h.DefaultAccountID, id)

	srv := &server{
		id:                   id,
		arn:                  arn,
		endpointType:         endpointType,
		identityProviderType: identityProviderType,
		protocols:            protocols,
		state:                "ONLINE",
		users:                make(map[string]*user),
	}

	s.mu.Lock()
	s.servers[id] = srv
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ServerId": id,
	})
}

func (s *Service) describeServer(w http.ResponseWriter, params map[string]interface{}) {
	serverID := h.GetString(params, "ServerId")

	s.mu.RLock()
	srv, exists := s.servers[serverID]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Server not found: "+serverID, http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Server": serverResp(srv),
	})
}

func (s *Service) deleteServer(w http.ResponseWriter, params map[string]interface{}) {
	serverID := h.GetString(params, "ServerId")

	s.mu.Lock()
	if _, exists := s.servers[serverID]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Server not found: "+serverID, http.StatusBadRequest)
		return
	}
	delete(s.servers, serverID)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listServers(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var list []map[string]interface{}
	for _, srv := range s.servers {
		list = append(list, map[string]interface{}{
			"ServerId":             srv.id,
			"Arn":                  srv.arn,
			"State":               srv.state,
			"EndpointType":        srv.endpointType,
			"IdentityProviderType": srv.identityProviderType,
			"UserCount":           len(srv.users),
		})
	}
	s.mu.RUnlock()

	sort.Slice(list, func(i, j int) bool {
		return list[i]["ServerId"].(string) < list[j]["ServerId"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Servers": list,
	})
}

func (s *Service) createUser(w http.ResponseWriter, params map[string]interface{}) {
	serverID := h.GetString(params, "ServerId")
	userName := h.GetString(params, "UserName")
	role := h.GetString(params, "Role")
	homeDirectory := h.GetString(params, "HomeDirectory")

	if serverID == "" {
		h.WriteJSONError(w, "InvalidParameterException", "ServerId is required", http.StatusBadRequest)
		return
	}
	if userName == "" {
		h.WriteJSONError(w, "InvalidParameterException", "UserName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	srv, exists := s.servers[serverID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Server not found: "+serverID, http.StatusBadRequest)
		return
	}

	if _, exists := srv.users[userName]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceExistsException", "User already exists: "+userName, http.StatusBadRequest)
		return
	}

	arn := fmt.Sprintf("arn:aws:transfer:us-east-1:%s:user/%s/%s", h.DefaultAccountID, serverID, userName)
	u := &user{
		userName:      userName,
		serverID:      serverID,
		arn:           arn,
		role:          role,
		homeDirectory: homeDirectory,
	}
	srv.users[userName] = u
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ServerId": serverID,
		"UserName": userName,
	})
}

func (s *Service) describeUser(w http.ResponseWriter, params map[string]interface{}) {
	serverID := h.GetString(params, "ServerId")
	userName := h.GetString(params, "UserName")

	s.mu.RLock()
	srv, exists := s.servers[serverID]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Server not found: "+serverID, http.StatusBadRequest)
		return
	}

	u, exists := srv.users[userName]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "User not found: "+userName, http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ServerId": serverID,
		"User":     userResp(u),
	})
}

func (s *Service) deleteUser(w http.ResponseWriter, params map[string]interface{}) {
	serverID := h.GetString(params, "ServerId")
	userName := h.GetString(params, "UserName")

	s.mu.Lock()
	srv, exists := s.servers[serverID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Server not found: "+serverID, http.StatusBadRequest)
		return
	}

	if _, exists := srv.users[userName]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "User not found: "+userName, http.StatusBadRequest)
		return
	}
	delete(srv.users, userName)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func serverResp(srv *server) map[string]interface{} {
	return map[string]interface{}{
		"ServerId":             srv.id,
		"Arn":                  srv.arn,
		"State":               srv.state,
		"EndpointType":        srv.endpointType,
		"IdentityProviderType": srv.identityProviderType,
		"Protocols":           srv.protocols,
		"UserCount":           len(srv.users),
	}
}

func userResp(u *user) map[string]interface{} {
	return map[string]interface{}{
		"UserName":      u.userName,
		"ServerId":      u.serverID,
		"Arn":           u.arn,
		"Role":          u.role,
		"HomeDirectory": u.homeDirectory,
	}
}

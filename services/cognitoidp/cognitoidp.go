// Package cognitoidp provides a mock implementation of AWS Cognito Identity Provider.
//
// Supported actions:
//   - CreateUserPool
//   - DescribeUserPool
//   - DeleteUserPool
//   - ListUserPools
//   - CreateUserPoolClient
//   - AdminCreateUser
//   - AdminGetUser
//   - AdminDeleteUser
//   - ListUsers
package cognitoidp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the Cognito Identity Provider mock.
type Service struct {
	mu    sync.RWMutex
	pools map[string]*userPool
}

type userPool struct {
	id       string
	name     string
	arn      string
	status   string
	created  time.Time
	modified time.Time
	clients  map[string]*userPoolClient
	users    map[string]*cognitoUser
}

type userPoolClient struct {
	clientID   string
	clientName string
	poolID     string
}

type cognitoUser struct {
	username   string
	status     string
	enabled    bool
	created    time.Time
	modified   time.Time
	attributes map[string]string
}

// New creates a new Cognito Identity Provider mock service.
func New() *Service {
	return &Service{
		pools: make(map[string]*userPool),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "cognito-idp" }

// Handler returns the HTTP handler for Cognito requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pools = make(map[string]*userPool)
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
	case "CreateUserPool":
		s.createUserPool(w, params)
	case "DescribeUserPool":
		s.describeUserPool(w, params)
	case "DeleteUserPool":
		s.deleteUserPool(w, params)
	case "ListUserPools":
		s.listUserPools(w, params)
	case "CreateUserPoolClient":
		s.createUserPoolClient(w, params)
	case "AdminCreateUser":
		s.adminCreateUser(w, params)
	case "AdminGetUser":
		s.adminGetUser(w, params)
	case "AdminDeleteUser":
		s.adminDeleteUser(w, params)
	case "ListUsers":
		s.listUsers(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createUserPool(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "PoolName")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterException", "PoolName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	id := "us-east-1_" + h.RandomID(9)
	arn := fmt.Sprintf("arn:aws:cognito-idp:us-east-1:%s:userpool/%s", h.DefaultAccountID, id)
	now := time.Now().UTC()
	pool := &userPool{
		id:       id,
		name:     name,
		arn:      arn,
		status:   "Enabled",
		created:  now,
		modified: now,
		clients:  make(map[string]*userPoolClient),
		users:    make(map[string]*cognitoUser),
	}
	s.pools[id] = pool
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"UserPool": poolResp(pool),
	})
}

func (s *Service) describeUserPool(w http.ResponseWriter, params map[string]interface{}) {
	poolID := h.GetString(params, "UserPoolId")

	s.mu.RLock()
	pool, exists := s.pools[poolID]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "User pool "+poolID+" does not exist.", http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"UserPool": poolResp(pool),
	})
}

func (s *Service) deleteUserPool(w http.ResponseWriter, params map[string]interface{}) {
	poolID := h.GetString(params, "UserPoolId")

	s.mu.Lock()
	if _, exists := s.pools[poolID]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "User pool "+poolID+" does not exist.", http.StatusBadRequest)
		return
	}
	delete(s.pools, poolID)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listUserPools(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var pools []map[string]interface{}
	for _, pool := range s.pools {
		pools = append(pools, map[string]interface{}{
			"Id":           pool.id,
			"Name":         pool.name,
			"Status":       pool.status,
			"CreationDate": float64(pool.created.Unix()),
		})
	}
	s.mu.RUnlock()

	sort.Slice(pools, func(i, j int) bool {
		return pools[i]["Name"].(string) < pools[j]["Name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"UserPools": pools,
	})
}

func (s *Service) createUserPoolClient(w http.ResponseWriter, params map[string]interface{}) {
	poolID := h.GetString(params, "UserPoolId")
	clientName := h.GetString(params, "ClientName")

	s.mu.Lock()
	pool, exists := s.pools[poolID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "User pool "+poolID+" does not exist.", http.StatusBadRequest)
		return
	}

	clientID := h.RandomHex(26)
	client := &userPoolClient{
		clientID:   clientID,
		clientName: clientName,
		poolID:     poolID,
	}
	pool.clients[clientID] = client
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"UserPoolClient": map[string]interface{}{
			"ClientId":   clientID,
			"ClientName": clientName,
			"UserPoolId": poolID,
		},
	})
}

func (s *Service) adminCreateUser(w http.ResponseWriter, params map[string]interface{}) {
	poolID := h.GetString(params, "UserPoolId")
	username := h.GetString(params, "Username")

	if username == "" {
		h.WriteJSONError(w, "InvalidParameterException", "Username is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	pool, exists := s.pools[poolID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "User pool "+poolID+" does not exist.", http.StatusBadRequest)
		return
	}

	attrs := make(map[string]string)
	if userAttrs, ok := params["UserAttributes"].([]interface{}); ok {
		for _, a := range userAttrs {
			if attr, ok := a.(map[string]interface{}); ok {
				name := h.GetString(attr, "Name")
				value := h.GetString(attr, "Value")
				if name != "" {
					attrs[name] = value
				}
			}
		}
	}

	now := time.Now().UTC()
	user := &cognitoUser{
		username:   username,
		status:     "FORCE_CHANGE_PASSWORD",
		enabled:    true,
		created:    now,
		modified:   now,
		attributes: attrs,
	}
	pool.users[username] = user
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"User": userResp(user),
	})
}

func (s *Service) adminGetUser(w http.ResponseWriter, params map[string]interface{}) {
	poolID := h.GetString(params, "UserPoolId")
	username := h.GetString(params, "Username")

	s.mu.RLock()
	pool, exists := s.pools[poolID]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "User pool "+poolID+" does not exist.", http.StatusBadRequest)
		return
	}
	user, exists := pool.users[username]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "UserNotFoundException", "User does not exist.", http.StatusBadRequest)
		return
	}

	resp := userResp(user)
	resp["UserPoolId"] = poolID // AdminGetUser also returns UserPoolId.
	h.WriteJSON(w, http.StatusOK, resp)
}

func (s *Service) adminDeleteUser(w http.ResponseWriter, params map[string]interface{}) {
	poolID := h.GetString(params, "UserPoolId")
	username := h.GetString(params, "Username")

	s.mu.Lock()
	pool, exists := s.pools[poolID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "User pool "+poolID+" does not exist.", http.StatusBadRequest)
		return
	}
	if _, exists := pool.users[username]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "UserNotFoundException", "User does not exist.", http.StatusBadRequest)
		return
	}
	delete(pool.users, username)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listUsers(w http.ResponseWriter, params map[string]interface{}) {
	poolID := h.GetString(params, "UserPoolId")

	s.mu.RLock()
	pool, exists := s.pools[poolID]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "User pool "+poolID+" does not exist.", http.StatusBadRequest)
		return
	}

	var users []map[string]interface{}
	for _, user := range pool.users {
		users = append(users, userResp(user))
	}
	s.mu.RUnlock()

	sort.Slice(users, func(i, j int) bool {
		return users[i]["Username"].(string) < users[j]["Username"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Users": users,
	})
}

func poolResp(pool *userPool) map[string]interface{} {
	return map[string]interface{}{
		"Id":               pool.id,
		"Name":             pool.name,
		"Arn":              pool.arn,
		"Status":           pool.status,
		"CreationDate":     float64(pool.created.Unix()),
		"LastModifiedDate": float64(pool.modified.Unix()),
	}
}

func userResp(user *cognitoUser) map[string]interface{} {
	var attrs []map[string]interface{}
	for k, v := range user.attributes {
		attrs = append(attrs, map[string]interface{}{
			"Name":  k,
			"Value": v,
		})
	}
	return map[string]interface{}{
		"Username":             user.username,
		"UserStatus":           user.status,
		"Enabled":              user.enabled,
		"UserCreateDate":       float64(user.created.Unix()),
		"UserLastModifiedDate": float64(user.modified.Unix()),
		"Attributes":           attrs,
	}
}

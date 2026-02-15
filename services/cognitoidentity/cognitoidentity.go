// Package cognitoidentity provides a mock implementation of AWS Cognito Identity.
//
// Supported actions:
//   - CreateIdentityPool
//   - DescribeIdentityPool
//   - DeleteIdentityPool
//   - ListIdentityPools
//   - UpdateIdentityPool
package cognitoidentity

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

// Service implements the Cognito Identity mock.
type Service struct {
	mu    sync.RWMutex
	pools map[string]*identityPool
}

type identityPool struct {
	id                    string
	name                  string
	allowUnauthenticated  bool
	created               time.Time
}

// New creates a new Cognito Identity mock service.
func New() *Service {
	return &Service{
		pools: make(map[string]*identityPool),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "cognito-identity" }

// Handler returns the HTTP handler for Cognito Identity requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pools = make(map[string]*identityPool)
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
	case "CreateIdentityPool":
		s.createIdentityPool(w, params)
	case "DescribeIdentityPool":
		s.describeIdentityPool(w, params)
	case "DeleteIdentityPool":
		s.deleteIdentityPool(w, params)
	case "ListIdentityPools":
		s.listIdentityPools(w, params)
	case "UpdateIdentityPool":
		s.updateIdentityPool(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createIdentityPool(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "IdentityPoolName")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterException", "IdentityPoolName is required", http.StatusBadRequest)
		return
	}

	allowUnauth := h.GetBool(params, "AllowUnauthenticatedIdentities")

	s.mu.Lock()
	id := fmt.Sprintf("us-east-1:%s", h.NewRequestID())
	pool := &identityPool{
		id:                   id,
		name:                 name,
		allowUnauthenticated: allowUnauth,
		created:              time.Now().UTC(),
	}
	s.pools[id] = pool
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, poolResp(pool))
}

func (s *Service) describeIdentityPool(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "IdentityPoolId")

	s.mu.RLock()
	pool, exists := s.pools[id]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Identity pool "+id+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, poolResp(pool))
}

func (s *Service) deleteIdentityPool(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "IdentityPoolId")

	s.mu.Lock()
	if _, exists := s.pools[id]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Identity pool "+id+" not found", http.StatusNotFound)
		return
	}
	delete(s.pools, id)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listIdentityPools(w http.ResponseWriter, params map[string]interface{}) {
	maxResults := h.GetInt(params, "MaxResults", 60)

	s.mu.RLock()
	var pools []map[string]interface{}
	for _, pool := range s.pools {
		pools = append(pools, map[string]interface{}{
			"IdentityPoolId":   pool.id,
			"IdentityPoolName": pool.name,
		})
	}
	s.mu.RUnlock()

	sort.Slice(pools, func(i, j int) bool {
		return pools[i]["IdentityPoolName"].(string) < pools[j]["IdentityPoolName"].(string)
	})

	if maxResults > 0 && len(pools) > maxResults {
		pools = pools[:maxResults]
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"IdentityPools": pools,
	})
}

func (s *Service) updateIdentityPool(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "IdentityPoolId")

	s.mu.Lock()
	pool, exists := s.pools[id]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Identity pool "+id+" not found", http.StatusNotFound)
		return
	}

	if name := h.GetString(params, "IdentityPoolName"); name != "" {
		pool.name = name
	}
	if _, ok := params["AllowUnauthenticatedIdentities"]; ok {
		pool.allowUnauthenticated = h.GetBool(params, "AllowUnauthenticatedIdentities")
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, poolResp(pool))
}

func poolResp(pool *identityPool) map[string]interface{} {
	return map[string]interface{}{
		"IdentityPoolId":                 pool.id,
		"IdentityPoolName":               pool.name,
		"AllowUnauthenticatedIdentities": pool.allowUnauthenticated,
	}
}

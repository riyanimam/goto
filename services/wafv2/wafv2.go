// Package wafv2 provides a mock implementation of AWS WAF v2.
//
// Supported actions:
//   - CreateWebACL
//   - GetWebACL
//   - DeleteWebACL
//   - ListWebACLs
//   - UpdateWebACL
//   - CreateIPSet
//   - GetIPSet
//   - DeleteIPSet
//   - ListIPSets
package wafv2

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

type webACL struct {
	id               string
	name             string
	arn              string
	scope            string
	defaultAction    interface{}
	rules            interface{}
	visibilityConfig interface{}
	lockToken        string
	description      string
}

type ipSet struct {
	id               string
	name             string
	arn              string
	scope            string
	ipAddressVersion string
	addresses        []string
	lockToken        string
	description      string
}

// Service implements the WAFv2 mock.
type Service struct {
	mu      sync.RWMutex
	webACLs map[string]*webACL
	ipSets  map[string]*ipSet
}

// New creates a new WAFv2 mock service.
func New() *Service {
	return &Service{
		webACLs: make(map[string]*webACL),
		ipSets:  make(map[string]*ipSet),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "wafv2" }

// Handler returns the HTTP handler for WAFv2 requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webACLs = make(map[string]*webACL)
	s.ipSets = make(map[string]*ipSet)
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
	case "CreateWebACL":
		s.createWebACL(w, params)
	case "GetWebACL":
		s.getWebACL(w, params)
	case "DeleteWebACL":
		s.deleteWebACL(w, params)
	case "ListWebACLs":
		s.listWebACLs(w, params)
	case "UpdateWebACL":
		s.updateWebACL(w, params)
	case "CreateIPSet":
		s.createIPSet(w, params)
	case "GetIPSet":
		s.getIPSet(w, params)
	case "DeleteIPSet":
		s.deleteIPSet(w, params)
	case "ListIPSets":
		s.listIPSets(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func buildARN(resource, scope, name, id string) string {
	region := "us-east-1"
	if strings.EqualFold(scope, "CLOUDFRONT") {
		region = "global"
	}
	return fmt.Sprintf("arn:aws:wafv2:%s:%s:%s/%s/%s/%s", region, h.DefaultAccountID, strings.ToLower(scope), resource, name, id)
}

func (s *Service) createWebACL(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")
	scope := h.GetString(params, "Scope")
	if name == "" || scope == "" {
		h.WriteJSONError(w, "WAFInvalidParameterException", "Name and Scope are required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, acl := range s.webACLs {
		if acl.name == name && acl.scope == scope {
			h.WriteJSONError(w, "WAFDuplicateItemException", "A WebACL with the same name already exists", http.StatusBadRequest)
			return
		}
	}

	id := h.NewRequestID()
	lockToken := h.RandomHex(36)
	arn := buildARN("webacl", scope, name, id)

	acl := &webACL{
		id:               id,
		name:             name,
		arn:              arn,
		scope:            scope,
		defaultAction:    params["DefaultAction"],
		rules:            params["Rules"],
		visibilityConfig: params["VisibilityConfig"],
		lockToken:        lockToken,
		description:      h.GetString(params, "Description"),
	}
	s.webACLs[id] = acl

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Summary": map[string]interface{}{
			"Id":        id,
			"ARN":       arn,
			"Name":      name,
			"LockToken": lockToken,
		},
	})
}

func (s *Service) getWebACL(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "Id")

	s.mu.RLock()
	acl, exists := s.webACLs[id]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "WAFNonexistentItemException", "WebACL not found", http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"WebACL":    webACLResp(acl),
		"LockToken": acl.lockToken,
	})
}

func (s *Service) deleteWebACL(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "Id")
	lockToken := h.GetString(params, "LockToken")

	s.mu.Lock()
	acl, exists := s.webACLs[id]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "WAFNonexistentItemException", "WebACL not found", http.StatusBadRequest)
		return
	}
	if acl.lockToken != lockToken {
		s.mu.Unlock()
		h.WriteJSONError(w, "WAFOptimisticLockException", "LockToken mismatch", http.StatusBadRequest)
		return
	}
	delete(s.webACLs, id)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listWebACLs(w http.ResponseWriter, params map[string]interface{}) {
	scope := h.GetString(params, "Scope")

	s.mu.RLock()
	var results []map[string]interface{}
	for _, acl := range s.webACLs {
		if scope != "" && acl.scope != scope {
			continue
		}
		results = append(results, map[string]interface{}{
			"Id":        acl.id,
			"ARN":       acl.arn,
			"Name":      acl.name,
			"LockToken": acl.lockToken,
		})
	}
	s.mu.RUnlock()

	sort.Slice(results, func(i, j int) bool {
		return results[i]["Name"].(string) < results[j]["Name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"WebACLs": results,
	})
}

func (s *Service) updateWebACL(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "Id")
	lockToken := h.GetString(params, "LockToken")

	s.mu.Lock()
	acl, exists := s.webACLs[id]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "WAFNonexistentItemException", "WebACL not found", http.StatusBadRequest)
		return
	}
	if acl.lockToken != lockToken {
		s.mu.Unlock()
		h.WriteJSONError(w, "WAFOptimisticLockException", "LockToken mismatch", http.StatusBadRequest)
		return
	}

	if v, ok := params["DefaultAction"]; ok {
		acl.defaultAction = v
	}
	if v, ok := params["Rules"]; ok {
		acl.rules = v
	}
	if v, ok := params["VisibilityConfig"]; ok {
		acl.visibilityConfig = v
	}
	if v := h.GetString(params, "Description"); v != "" {
		acl.description = v
	}

	nextLockToken := h.RandomHex(36)
	acl.lockToken = nextLockToken
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"NextLockToken": nextLockToken,
	})
}

func (s *Service) createIPSet(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")
	scope := h.GetString(params, "Scope")
	ipAddressVersion := h.GetString(params, "IPAddressVersion")
	if name == "" || scope == "" || ipAddressVersion == "" {
		h.WriteJSONError(w, "WAFInvalidParameterException", "Name, Scope, and IPAddressVersion are required", http.StatusBadRequest)
		return
	}

	var addresses []string
	if addrs, ok := params["Addresses"].([]interface{}); ok {
		for _, a := range addrs {
			if s, ok := a.(string); ok {
				addresses = append(addresses, s)
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ip := range s.ipSets {
		if ip.name == name && ip.scope == scope {
			h.WriteJSONError(w, "WAFDuplicateItemException", "An IPSet with the same name already exists", http.StatusBadRequest)
			return
		}
	}

	id := h.NewRequestID()
	lockToken := h.RandomHex(36)
	arn := buildARN("ipset", scope, name, id)

	set := &ipSet{
		id:               id,
		name:             name,
		arn:              arn,
		scope:            scope,
		ipAddressVersion: ipAddressVersion,
		addresses:        addresses,
		lockToken:        lockToken,
		description:      h.GetString(params, "Description"),
	}
	s.ipSets[id] = set

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Summary": map[string]interface{}{
			"Id":        id,
			"ARN":       arn,
			"Name":      name,
			"LockToken": lockToken,
		},
	})
}

func (s *Service) getIPSet(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "Id")

	s.mu.RLock()
	set, exists := s.ipSets[id]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "WAFNonexistentItemException", "IPSet not found", http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"IPSet":     ipSetResp(set),
		"LockToken": set.lockToken,
	})
}

func (s *Service) deleteIPSet(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "Id")
	lockToken := h.GetString(params, "LockToken")

	s.mu.Lock()
	set, exists := s.ipSets[id]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "WAFNonexistentItemException", "IPSet not found", http.StatusBadRequest)
		return
	}
	if set.lockToken != lockToken {
		s.mu.Unlock()
		h.WriteJSONError(w, "WAFOptimisticLockException", "LockToken mismatch", http.StatusBadRequest)
		return
	}
	delete(s.ipSets, id)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listIPSets(w http.ResponseWriter, params map[string]interface{}) {
	scope := h.GetString(params, "Scope")

	s.mu.RLock()
	var results []map[string]interface{}
	for _, set := range s.ipSets {
		if scope != "" && set.scope != scope {
			continue
		}
		results = append(results, map[string]interface{}{
			"Id":        set.id,
			"ARN":       set.arn,
			"Name":      set.name,
			"LockToken": set.lockToken,
		})
	}
	s.mu.RUnlock()

	sort.Slice(results, func(i, j int) bool {
		return results[i]["Name"].(string) < results[j]["Name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"IPSets": results,
	})
}

func webACLResp(acl *webACL) map[string]interface{} {
	resp := map[string]interface{}{
		"Id":               acl.id,
		"ARN":              acl.arn,
		"Name":             acl.name,
		"DefaultAction":    acl.defaultAction,
		"VisibilityConfig": acl.visibilityConfig,
	}
	if acl.description != "" {
		resp["Description"] = acl.description
	}
	if acl.rules != nil {
		resp["Rules"] = acl.rules
	}
	return resp
}

func ipSetResp(set *ipSet) map[string]interface{} {
	resp := map[string]interface{}{
		"Id":               set.id,
		"ARN":              set.arn,
		"Name":             set.name,
		"IPAddressVersion": set.ipAddressVersion,
		"Addresses":        set.addresses,
	}
	if set.description != "" {
		resp["Description"] = set.description
	}
	return resp
}

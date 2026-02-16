// Package applicationautoscaling provides a mock implementation of AWS Application Auto Scaling.
//
// Supported actions:
//   - RegisterScalableTarget
//   - DescribeScalableTargets
//   - DeregisterScalableTarget
//   - PutScalingPolicy
//   - DescribeScalingPolicies
//   - DeleteScalingPolicy
package applicationautoscaling

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

// Service implements the Application Auto Scaling mock.
type Service struct {
	mu       sync.RWMutex
	targets  map[string]*scalableTarget
	policies map[string]*scalingPolicy
}

type scalableTarget struct {
	serviceNamespace  string
	resourceID        string
	scalableDimension string
	minCapacity       int
	maxCapacity       int
	roleARN           string
	created           time.Time
}

type scalingPolicy struct {
	policyName        string
	policyARN         string
	serviceNamespace  string
	resourceID        string
	scalableDimension string
	policyType        string
	created           time.Time
}

// New creates a new Application Auto Scaling mock service.
func New() *Service {
	return &Service{
		targets:  make(map[string]*scalableTarget),
		policies: make(map[string]*scalingPolicy),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "application-autoscaling" }

// Handler returns the HTTP handler for Application Auto Scaling requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.targets = make(map[string]*scalableTarget)
	s.policies = make(map[string]*scalingPolicy)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteJSONError(w, "InternalError", "could not read request body", http.StatusInternalServerError)
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
	case "RegisterScalableTarget":
		s.registerScalableTarget(w, params)
	case "DescribeScalableTargets":
		s.describeScalableTargets(w, params)
	case "DeregisterScalableTarget":
		s.deregisterScalableTarget(w, params)
	case "PutScalingPolicy":
		s.putScalingPolicy(w, params)
	case "DescribeScalingPolicies":
		s.describeScalingPolicies(w, params)
	case "DeleteScalingPolicy":
		s.deleteScalingPolicy(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func targetKey(namespace, resourceID, dimension string) string {
	return namespace + "|" + resourceID + "|" + dimension
}

func policyKey(policyName, namespace, resourceID, dimension string) string {
	return policyName + "|" + namespace + "|" + resourceID + "|" + dimension
}

func (s *Service) registerScalableTarget(w http.ResponseWriter, params map[string]interface{}) {
	namespace := h.GetString(params, "ServiceNamespace")
	resourceID := h.GetString(params, "ResourceId")
	dimension := h.GetString(params, "ScalableDimension")

	if namespace == "" || resourceID == "" || dimension == "" {
		h.WriteJSONError(w, "ValidationException", "ServiceNamespace, ResourceId, and ScalableDimension are required", http.StatusBadRequest)
		return
	}

	minCap := h.GetInt(params, "MinCapacity", 0)
	maxCap := h.GetInt(params, "MaxCapacity", 0)
	roleARN := h.GetString(params, "RoleARN")

	key := targetKey(namespace, resourceID, dimension)

	s.mu.Lock()
	existing, exists := s.targets[key]
	if exists {
		// Update existing target
		if _, ok := params["MinCapacity"]; ok {
			existing.minCapacity = minCap
		}
		if _, ok := params["MaxCapacity"]; ok {
			existing.maxCapacity = maxCap
		}
		if roleARN != "" {
			existing.roleARN = roleARN
		}
		s.mu.Unlock()
	} else {
		s.targets[key] = &scalableTarget{
			serviceNamespace:  namespace,
			resourceID:        resourceID,
			scalableDimension: dimension,
			minCapacity:       minCap,
			maxCapacity:       maxCap,
			roleARN:           roleARN,
			created:           time.Now().UTC(),
		}
		s.mu.Unlock()
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) describeScalableTargets(w http.ResponseWriter, params map[string]interface{}) {
	namespace := h.GetString(params, "ServiceNamespace")
	if namespace == "" {
		h.WriteJSONError(w, "ValidationException", "ServiceNamespace is required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	var list []map[string]interface{}
	for _, t := range s.targets {
		if t.serviceNamespace == namespace {
			list = append(list, targetResp(t))
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ScalableTargets": list,
	})
}

func (s *Service) deregisterScalableTarget(w http.ResponseWriter, params map[string]interface{}) {
	namespace := h.GetString(params, "ServiceNamespace")
	resourceID := h.GetString(params, "ResourceId")
	dimension := h.GetString(params, "ScalableDimension")

	if namespace == "" || resourceID == "" || dimension == "" {
		h.WriteJSONError(w, "ValidationException", "ServiceNamespace, ResourceId, and ScalableDimension are required", http.StatusBadRequest)
		return
	}

	key := targetKey(namespace, resourceID, dimension)

	s.mu.Lock()
	if _, exists := s.targets[key]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ObjectNotFoundException", "No scalable target found for service namespace: "+namespace+", resource ID: "+resourceID+", scalable dimension: "+dimension, http.StatusBadRequest)
		return
	}
	delete(s.targets, key)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) putScalingPolicy(w http.ResponseWriter, params map[string]interface{}) {
	policyName := h.GetString(params, "PolicyName")
	namespace := h.GetString(params, "ServiceNamespace")
	resourceID := h.GetString(params, "ResourceId")
	dimension := h.GetString(params, "ScalableDimension")
	policyType := h.GetString(params, "PolicyType")

	if policyName == "" || namespace == "" || resourceID == "" || dimension == "" {
		h.WriteJSONError(w, "ValidationException", "PolicyName, ServiceNamespace, ResourceId, and ScalableDimension are required", http.StatusBadRequest)
		return
	}

	if policyType == "" {
		policyType = "TargetTrackingScaling"
	}

	key := policyKey(policyName, namespace, resourceID, dimension)
	policyARN := fmt.Sprintf("arn:aws:autoscaling:us-east-1:%s:scalingPolicy:%s:resource/%s/%s:policyName/%s",
		h.DefaultAccountID, h.RandomHex(36), namespace, resourceID, policyName)

	s.mu.Lock()
	existing, exists := s.policies[key]
	if exists {
		// Update existing policy, keep existing ARN
		existing.policyType = policyType
		policyARN = existing.policyARN
		s.mu.Unlock()
	} else {
		s.policies[key] = &scalingPolicy{
			policyName:        policyName,
			policyARN:         policyARN,
			serviceNamespace:  namespace,
			resourceID:        resourceID,
			scalableDimension: dimension,
			policyType:        policyType,
			created:           time.Now().UTC(),
		}
		s.mu.Unlock()
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"PolicyARN": policyARN,
	})
}

func (s *Service) describeScalingPolicies(w http.ResponseWriter, params map[string]interface{}) {
	namespace := h.GetString(params, "ServiceNamespace")
	if namespace == "" {
		h.WriteJSONError(w, "ValidationException", "ServiceNamespace is required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	var list []map[string]interface{}
	for _, p := range s.policies {
		if p.serviceNamespace == namespace {
			list = append(list, policyResp(p))
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ScalingPolicies": list,
	})
}

func (s *Service) deleteScalingPolicy(w http.ResponseWriter, params map[string]interface{}) {
	policyName := h.GetString(params, "PolicyName")
	namespace := h.GetString(params, "ServiceNamespace")
	resourceID := h.GetString(params, "ResourceId")
	dimension := h.GetString(params, "ScalableDimension")

	if policyName == "" || namespace == "" || resourceID == "" || dimension == "" {
		h.WriteJSONError(w, "ValidationException", "PolicyName, ServiceNamespace, ResourceId, and ScalableDimension are required", http.StatusBadRequest)
		return
	}

	key := policyKey(policyName, namespace, resourceID, dimension)

	s.mu.Lock()
	if _, exists := s.policies[key]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ObjectNotFoundException", "No scaling policy found for policy name: "+policyName, http.StatusBadRequest)
		return
	}
	delete(s.policies, key)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func targetResp(t *scalableTarget) map[string]interface{} {
	return map[string]interface{}{
		"ServiceNamespace":  t.serviceNamespace,
		"ResourceId":        t.resourceID,
		"ScalableDimension": t.scalableDimension,
		"MinCapacity":       t.minCapacity,
		"MaxCapacity":       t.maxCapacity,
		"RoleARN":           t.roleARN,
		"CreationTime":      float64(t.created.Unix()),
	}
}

func policyResp(p *scalingPolicy) map[string]interface{} {
	return map[string]interface{}{
		"PolicyName":        p.policyName,
		"PolicyARN":         p.policyARN,
		"ServiceNamespace":  p.serviceNamespace,
		"ResourceId":        p.resourceID,
		"ScalableDimension": p.scalableDimension,
		"PolicyType":        p.policyType,
		"CreationTime":      float64(p.created.Unix()),
	}
}

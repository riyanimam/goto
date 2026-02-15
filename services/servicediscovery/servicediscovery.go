// Package servicediscovery provides a mock implementation of AWS Cloud Map (Service Discovery).
//
// Supported actions:
//   - CreatePrivateDnsNamespace
//   - CreateService
//   - GetService
//   - DeleteService
//   - ListServices
//   - RegisterInstance
//   - DeregisterInstance
//   - ListInstances
package servicediscovery

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

// Service implements the Cloud Map mock.
type Service struct {
	mu         sync.RWMutex
	namespaces map[string]*namespace
	services   map[string]*service
	instances  map[string][]*instance // keyed by service ID
}

type namespace struct {
	id          string
	name        string
	arn         string
	nsType      string
	vpc         string
	description string
}

type service struct {
	id          string
	name        string
	arn         string
	namespaceID string
	dnsConfig   interface{}
}

type instance struct {
	id         string
	serviceID  string
	attributes map[string]interface{}
}

// New creates a new Cloud Map mock service.
func New() *Service {
	return &Service{
		namespaces: make(map[string]*namespace),
		services:   make(map[string]*service),
		instances:  make(map[string][]*instance),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "servicediscovery" }

// Handler returns the HTTP handler for Cloud Map requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.namespaces = make(map[string]*namespace)
	s.services = make(map[string]*service)
	s.instances = make(map[string][]*instance)
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
	case "CreatePrivateDnsNamespace":
		s.createPrivateDnsNamespace(w, params)
	case "CreateService":
		s.createService(w, params)
	case "GetService":
		s.getService(w, params)
	case "DeleteService":
		s.deleteService(w, params)
	case "ListServices":
		s.listServices(w, params)
	case "RegisterInstance":
		s.registerInstance(w, params)
	case "DeregisterInstance":
		s.deregisterInstance(w, params)
	case "ListInstances":
		s.listInstances(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createPrivateDnsNamespace(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")
	if name == "" {
		h.WriteJSONError(w, "InvalidInput", "Name is required", http.StatusBadRequest)
		return
	}

	vpc := h.GetString(params, "Vpc")
	description := h.GetString(params, "Description")

	s.mu.Lock()
	id := "ns-" + h.RandomHex(16)
	arn := fmt.Sprintf("arn:aws:servicediscovery:us-east-1:%s:namespace/%s", h.DefaultAccountID, id)
	s.namespaces[id] = &namespace{
		id:          id,
		name:        name,
		arn:         arn,
		nsType:      "DNS_PRIVATE",
		vpc:         vpc,
		description: description,
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"OperationId": h.NewRequestID(),
	})
}

func (s *Service) createService(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")
	if name == "" {
		h.WriteJSONError(w, "InvalidInput", "Name is required", http.StatusBadRequest)
		return
	}

	namespaceID := h.GetString(params, "NamespaceId")
	dnsConfig := params["DnsConfig"]

	s.mu.Lock()
	id := "srv-" + h.RandomHex(16)
	arn := fmt.Sprintf("arn:aws:servicediscovery:us-east-1:%s:service/%s", h.DefaultAccountID, id)
	svc := &service{
		id:          id,
		name:        name,
		arn:         arn,
		namespaceID: namespaceID,
		dnsConfig:   dnsConfig,
	}
	s.services[id] = svc
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Service": svcResp(svc),
	})
}

func (s *Service) getService(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "Id")

	s.mu.RLock()
	svc, exists := s.services[id]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ServiceNotFound", "Service not found: "+id, http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Service": svcResp(svc),
	})
}

func (s *Service) deleteService(w http.ResponseWriter, params map[string]interface{}) {
	id := h.GetString(params, "Id")

	s.mu.Lock()
	if _, exists := s.services[id]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ServiceNotFound", "Service not found: "+id, http.StatusBadRequest)
		return
	}
	delete(s.services, id)
	delete(s.instances, id)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listServices(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var list []map[string]interface{}
	for _, svc := range s.services {
		list = append(list, svcResp(svc))
	}
	s.mu.RUnlock()

	sort.Slice(list, func(i, j int) bool {
		return list[i]["Name"].(string) < list[j]["Name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Services": list,
	})
}

func (s *Service) registerInstance(w http.ResponseWriter, params map[string]interface{}) {
	serviceID := h.GetString(params, "ServiceId")
	instanceID := h.GetString(params, "InstanceId")
	if serviceID == "" || instanceID == "" {
		h.WriteJSONError(w, "InvalidInput", "ServiceId and InstanceId are required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.services[serviceID]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ServiceNotFound", "Service not found: "+serviceID, http.StatusBadRequest)
		return
	}

	attrs, _ := params["Attributes"].(map[string]interface{})

	// Update existing instance or create new one.
	found := false
	for _, inst := range s.instances[serviceID] {
		if inst.id == instanceID {
			inst.attributes = attrs
			found = true
			break
		}
	}
	if !found {
		s.instances[serviceID] = append(s.instances[serviceID], &instance{
			id:         instanceID,
			serviceID:  serviceID,
			attributes: attrs,
		})
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"OperationId": h.NewRequestID(),
	})
}

func (s *Service) deregisterInstance(w http.ResponseWriter, params map[string]interface{}) {
	serviceID := h.GetString(params, "ServiceId")
	instanceID := h.GetString(params, "InstanceId")
	if serviceID == "" || instanceID == "" {
		h.WriteJSONError(w, "InvalidInput", "ServiceId and InstanceId are required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	insts := s.instances[serviceID]
	for i, inst := range insts {
		if inst.id == instanceID {
			s.instances[serviceID] = append(insts[:i], insts[i+1:]...)
			s.mu.Unlock()
			h.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"OperationId": h.NewRequestID(),
			})
			return
		}
	}
	s.mu.Unlock()

	h.WriteJSONError(w, "InstanceNotFound", "Instance not found: "+instanceID, http.StatusBadRequest)
}

func (s *Service) listInstances(w http.ResponseWriter, params map[string]interface{}) {
	serviceID := h.GetString(params, "ServiceId")

	s.mu.RLock()
	if _, exists := s.services[serviceID]; !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "ServiceNotFound", "Service not found: "+serviceID, http.StatusBadRequest)
		return
	}

	var list []map[string]interface{}
	for _, inst := range s.instances[serviceID] {
		entry := map[string]interface{}{
			"Id": inst.id,
		}
		if inst.attributes != nil {
			entry["Attributes"] = inst.attributes
		}
		list = append(list, entry)
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Instances": list,
	})
}

func svcResp(svc *service) map[string]interface{} {
	resp := map[string]interface{}{
		"Id":          svc.id,
		"Name":        svc.name,
		"Arn":         svc.arn,
		"NamespaceId": svc.namespaceID,
	}
	if svc.dnsConfig != nil {
		resp["DnsConfig"] = svc.dnsConfig
	}
	return resp
}

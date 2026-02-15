// Package configservice provides a mock implementation of AWS Config.
//
// Supported actions:
//   - PutConfigRule
//   - DescribeConfigRules
//   - DeleteConfigRule
//   - PutConfigurationRecorder
//   - DescribeConfigurationRecorders
//   - PutDeliveryChannel
package configservice

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

// Service implements the AWS Config mock.
type Service struct {
	mu        sync.RWMutex
	rules     map[string]*configRule
	recorders map[string]*configurationRecorder
	channels  map[string]*deliveryChannel
}

type configRule struct {
	name        string
	arn         string
	ruleID      string
	source      map[string]interface{}
	state       string
	description string
}

type configurationRecorder struct {
	name           string
	roleARN        string
	recordingGroup map[string]interface{}
}

type deliveryChannel struct {
	name         string
	s3BucketName string
	snsTopicARN  string
}

// New creates a new AWS Config mock service.
func New() *Service {
	return &Service{
		rules:     make(map[string]*configRule),
		recorders: make(map[string]*configurationRecorder),
		channels:  make(map[string]*deliveryChannel),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "config" }

// Handler returns the HTTP handler for Config requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = make(map[string]*configRule)
	s.recorders = make(map[string]*configurationRecorder)
	s.channels = make(map[string]*deliveryChannel)
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
	case "PutConfigRule":
		s.putConfigRule(w, params)
	case "DescribeConfigRules":
		s.describeConfigRules(w, params)
	case "DeleteConfigRule":
		s.deleteConfigRule(w, params)
	case "PutConfigurationRecorder":
		s.putConfigurationRecorder(w, params)
	case "DescribeConfigurationRecorders":
		s.describeConfigurationRecorders(w, params)
	case "PutDeliveryChannel":
		s.putDeliveryChannel(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) putConfigRule(w http.ResponseWriter, params map[string]interface{}) {
	ruleObj, ok := params["ConfigRule"].(map[string]interface{})
	if !ok {
		h.WriteJSONError(w, "InvalidParameterValueException", "ConfigRule is required", http.StatusBadRequest)
		return
	}

	name := h.GetString(ruleObj, "ConfigRuleName")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterValueException", "ConfigRuleName is required", http.StatusBadRequest)
		return
	}

	var source map[string]interface{}
	if src, ok := ruleObj["Source"].(map[string]interface{}); ok {
		source = src
	}

	description := h.GetString(ruleObj, "Description")

	s.mu.Lock()
	rule, exists := s.rules[name]
	if !exists {
		rule = &configRule{
			name:   name,
			arn:    fmt.Sprintf("arn:aws:config:us-east-1:%s:config-rule/config-rule-%s", h.DefaultAccountID, h.RandomHex(6)),
			ruleID: fmt.Sprintf("config-rule-%s", h.RandomHex(6)),
			state:  "ACTIVE",
		}
		s.rules[name] = rule
	}
	if source != nil {
		rule.source = source
	}
	if description != "" {
		rule.description = description
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ConfigRuleArn": rule.arn,
	})
}

func (s *Service) describeConfigRules(w http.ResponseWriter, params map[string]interface{}) {
	var nameFilter []string
	if names, ok := params["ConfigRuleNames"].([]interface{}); ok {
		for _, n := range names {
			if name, ok := n.(string); ok {
				nameFilter = append(nameFilter, name)
			}
		}
	}

	s.mu.RLock()
	var rules []map[string]interface{}
	for _, rule := range s.rules {
		if len(nameFilter) > 0 && !contains(nameFilter, rule.name) {
			continue
		}
		rules = append(rules, ruleResp(rule))
	}
	s.mu.RUnlock()

	sort.Slice(rules, func(i, j int) bool {
		return rules[i]["ConfigRuleName"].(string) < rules[j]["ConfigRuleName"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ConfigRules": rules,
	})
}

func (s *Service) deleteConfigRule(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "ConfigRuleName")
	if name == "" {
		h.WriteJSONError(w, "InvalidParameterValueException", "ConfigRuleName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.rules[name]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "NoSuchConfigRuleException", "Config rule not found: "+name, http.StatusBadRequest)
		return
	}
	delete(s.rules, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) putConfigurationRecorder(w http.ResponseWriter, params map[string]interface{}) {
	recObj, ok := params["ConfigurationRecorder"].(map[string]interface{})
	if !ok {
		h.WriteJSONError(w, "InvalidParameterValueException", "ConfigurationRecorder is required", http.StatusBadRequest)
		return
	}

	name := h.GetString(recObj, "name")
	if name == "" {
		name = "default"
	}

	roleARN := h.GetString(recObj, "roleARN")

	var recordingGroup map[string]interface{}
	if rg, ok := recObj["recordingGroup"].(map[string]interface{}); ok {
		recordingGroup = rg
	}

	s.mu.Lock()
	rec, exists := s.recorders[name]
	if !exists {
		rec = &configurationRecorder{name: name}
		s.recorders[name] = rec
	}
	if roleARN != "" {
		rec.roleARN = roleARN
	}
	if recordingGroup != nil {
		rec.recordingGroup = recordingGroup
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) describeConfigurationRecorders(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var recorders []map[string]interface{}
	for _, rec := range s.recorders {
		r := map[string]interface{}{
			"name":    rec.name,
			"roleARN": rec.roleARN,
		}
		if rec.recordingGroup != nil {
			r["recordingGroup"] = rec.recordingGroup
		}
		recorders = append(recorders, r)
	}
	s.mu.RUnlock()

	sort.Slice(recorders, func(i, j int) bool {
		return recorders[i]["name"].(string) < recorders[j]["name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ConfigurationRecorders": recorders,
	})
}

func (s *Service) putDeliveryChannel(w http.ResponseWriter, params map[string]interface{}) {
	chObj, ok := params["DeliveryChannel"].(map[string]interface{})
	if !ok {
		h.WriteJSONError(w, "InvalidParameterValueException", "DeliveryChannel is required", http.StatusBadRequest)
		return
	}

	name := h.GetString(chObj, "name")
	if name == "" {
		name = "default"
	}

	s3Bucket := h.GetString(chObj, "s3BucketName")
	snsTopic := h.GetString(chObj, "snsTopicARN")

	s.mu.Lock()
	ch, exists := s.channels[name]
	if !exists {
		ch = &deliveryChannel{name: name}
		s.channels[name] = ch
	}
	if s3Bucket != "" {
		ch.s3BucketName = s3Bucket
	}
	if snsTopic != "" {
		ch.snsTopicARN = snsTopic
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func ruleResp(rule *configRule) map[string]interface{} {
	resp := map[string]interface{}{
		"ConfigRuleName": rule.name,
		"ConfigRuleArn":  rule.arn,
		"ConfigRuleId":   rule.ruleID,
		"ConfigRuleState": rule.state,
	}
	if rule.source != nil {
		resp["Source"] = rule.source
	}
	if rule.description != "" {
		resp["Description"] = rule.description
	}
	return resp
}

func contains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}

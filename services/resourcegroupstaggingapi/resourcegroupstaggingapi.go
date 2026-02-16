// Package resourcegroupstaggingapi provides a mock implementation of AWS Resource Groups Tagging API.
//
// Supported actions:
//   - TagResources
//   - UntagResources
//   - GetResources
//   - GetTagKeys
//   - GetTagValues
package resourcegroupstaggingapi

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the Resource Groups Tagging API mock.
type Service struct {
	mu   sync.RWMutex
	tags map[string]map[string]string // ARN -> tag key -> tag value
}

// New creates a new Resource Groups Tagging API mock service.
func New() *Service {
	return &Service{
		tags: make(map[string]map[string]string),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "tagging" }

// Handler returns the HTTP handler for Resource Groups Tagging API requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tags = make(map[string]map[string]string)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")
	parts := strings.SplitN(target, ".", 2)
	if len(parts) != 2 {
		h.WriteJSONError(w, "InvalidAction", "missing or invalid X-Amz-Target header", http.StatusBadRequest)
		return
	}
	action := parts[1]

	switch action {
	case "TagResources":
		s.tagResources(w, r)
	case "UntagResources":
		s.untagResources(w, r)
	case "GetResources":
		s.getResources(w, r)
	case "GetTagKeys":
		s.getTagKeys(w)
	case "GetTagValues":
		s.getTagValues(w, r)
	default:
		h.WriteJSONError(w, "InvalidAction", "unsupported action: "+action, http.StatusBadRequest)
	}
}

func (s *Service) tagResources(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	arns := toStringSlice(params["ResourceARNList"])
	tagsInput := toStringMap(params["Tags"])

	s.mu.Lock()
	for _, arn := range arns {
		if s.tags[arn] == nil {
			s.tags[arn] = make(map[string]string)
		}
		for k, v := range tagsInput {
			s.tags[arn][k] = v
		}
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"FailedResourcesMap": map[string]interface{}{},
	})
}

func (s *Service) untagResources(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	arns := toStringSlice(params["ResourceARNList"])
	tagKeys := toStringSlice(params["TagKeys"])

	s.mu.Lock()
	for _, arn := range arns {
		if m := s.tags[arn]; m != nil {
			for _, k := range tagKeys {
				delete(m, k)
			}
			if len(m) == 0 {
				delete(s.tags, arn)
			}
		}
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"FailedResourcesMap": map[string]interface{}{},
	})
}

func (s *Service) getResources(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	var filters []tagFilter
	if raw, ok := params["TagFilters"].([]interface{}); ok {
		for _, f := range raw {
			if fm, ok := f.(map[string]interface{}); ok {
				tf := tagFilter{Key: h.GetString(fm, "Key")}
				tf.Values = toStringSlice(fm["Values"])
				filters = append(filters, tf)
			}
		}
	}

	s.mu.RLock()
	var list []map[string]interface{}
	for arn, tagsMap := range s.tags {
		if !matchFilters(tagsMap, filters) {
			continue
		}
		var tagList []map[string]string
		for k, v := range tagsMap {
			tagList = append(tagList, map[string]string{"Key": k, "Value": v})
		}
		sort.Slice(tagList, func(i, j int) bool {
			return tagList[i]["Key"] < tagList[j]["Key"]
		})
		list = append(list, map[string]interface{}{
			"ResourceARN": arn,
			"Tags":        tagList,
		})
	}
	s.mu.RUnlock()

	sort.Slice(list, func(i, j int) bool {
		return list[i]["ResourceARN"].(string) < list[j]["ResourceARN"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"PaginationToken":        "",
		"ResourceTagMappingList": list,
	})
}

func (s *Service) getTagKeys(w http.ResponseWriter) {
	s.mu.RLock()
	keySet := make(map[string]struct{})
	for _, tagsMap := range s.tags {
		for k := range tagsMap {
			keySet[k] = struct{}{}
		}
	}
	s.mu.RUnlock()

	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"PaginationToken": "",
		"TagKeys":         keys,
	})
}

func (s *Service) getTagValues(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	key := h.GetString(params, "Key")

	s.mu.RLock()
	valueSet := make(map[string]struct{})
	for _, tagsMap := range s.tags {
		if v, ok := tagsMap[key]; ok {
			valueSet[v] = struct{}{}
		}
	}
	s.mu.RUnlock()

	values := make([]string, 0, len(valueSet))
	for v := range valueSet {
		values = append(values, v)
	}
	sort.Strings(values)

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"PaginationToken": "",
		"TagValues":       values,
	})
}

type tagFilter struct {
	Key    string
	Values []string
}

func matchFilters(tags map[string]string, filters []tagFilter) bool {
	for _, f := range filters {
		v, ok := tags[f.Key]
		if !ok {
			return false
		}
		if len(f.Values) > 0 {
			found := false
			for _, fv := range f.Values {
				if v == fv {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

func toStringSlice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func toStringMap(v interface{}) map[string]string {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, val := range m {
		if s, ok := val.(string); ok {
			out[k] = s
		}
	}
	return out
}

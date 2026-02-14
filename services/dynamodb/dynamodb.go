// Package dynamodb provides a mock implementation of AWS DynamoDB.
//
// Supported actions:
//   - CreateTable
//   - DeleteTable
//   - DescribeTable
//   - ListTables
//   - PutItem
//   - GetItem
//   - DeleteItem
//   - Query
//   - Scan
package dynamodb

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultAccountID = "123456789012"

// Service implements the DynamoDB mock.
type Service struct {
	mu     sync.RWMutex
	tables map[string]*table
}

type table struct {
	name             string
	arn              string
	status           string
	keySchema        []keySchemaElement
	attributeDefs    []attributeDefinition
	created          time.Time
	itemCount        int64
	billingMode      string
	provisionedRead  int64
	provisionedWrite int64
	items            []map[string]interface{}
	mu               sync.Mutex
}

type keySchemaElement struct {
	AttributeName string `json:"AttributeName"`
	KeyType       string `json:"KeyType"`
}

type attributeDefinition struct {
	AttributeName string `json:"AttributeName"`
	AttributeType string `json:"AttributeType"`
}

// New creates a new DynamoDB mock service.
func New() *Service {
	return &Service{
		tables: make(map[string]*table),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "dynamodb" }

// Handler returns the HTTP handler for DynamoDB requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all tables and items.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tables = make(map[string]*table)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, "InternalServerError", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			writeJSONError(w, "SerializationException", "could not parse request body", http.StatusBadRequest)
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
	case "CreateTable":
		s.createTable(w, params)
	case "DeleteTable":
		s.deleteTable(w, params)
	case "DescribeTable":
		s.describeTable(w, params)
	case "ListTables":
		s.listTables(w, params)
	case "PutItem":
		s.putItem(w, params)
	case "GetItem":
		s.getItem(w, params)
	case "DeleteItem":
		s.deleteItem(w, params)
	case "Query":
		s.query(w, params)
	case "Scan":
		s.scan(w, params)
	default:
		writeJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createTable(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "TableName")
	if name == "" {
		writeJSONError(w, "ValidationException", "TableName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.tables[name]; exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceInUseException", "Table already exists: "+name, http.StatusBadRequest)
		return
	}

	t := &table{
		name:    name,
		arn:     fmt.Sprintf("arn:aws:dynamodb:us-east-1:%s:table/%s", defaultAccountID, name),
		status:  "ACTIVE",
		created: time.Now().UTC(),
	}

	// Parse KeySchema.
	if ks, ok := params["KeySchema"].([]interface{}); ok {
		for _, elem := range ks {
			if m, ok := elem.(map[string]interface{}); ok {
				t.keySchema = append(t.keySchema, keySchemaElement{
					AttributeName: getString(m, "AttributeName"),
					KeyType:       getString(m, "KeyType"),
				})
			}
		}
	}

	// Parse AttributeDefinitions.
	if ad, ok := params["AttributeDefinitions"].([]interface{}); ok {
		for _, elem := range ad {
			if m, ok := elem.(map[string]interface{}); ok {
				t.attributeDefs = append(t.attributeDefs, attributeDefinition{
					AttributeName: getString(m, "AttributeName"),
					AttributeType: getString(m, "AttributeType"),
				})
			}
		}
	}

	// Parse BillingMode.
	t.billingMode = getString(params, "BillingMode")
	if t.billingMode == "" {
		t.billingMode = "PROVISIONED"
	}

	// Parse ProvisionedThroughput.
	if pt, ok := params["ProvisionedThroughput"].(map[string]interface{}); ok {
		t.provisionedRead = getInt64(pt, "ReadCapacityUnits", 5)
		t.provisionedWrite = getInt64(pt, "WriteCapacityUnits", 5)
	} else if t.billingMode == "PROVISIONED" {
		t.provisionedRead = 5
		t.provisionedWrite = 5
	}

	s.tables[name] = t
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"TableDescription": s.tableDescription(t),
	})
}

func (s *Service) deleteTable(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "TableName")

	s.mu.Lock()
	t, exists := s.tables[name]
	if !exists {
		s.mu.Unlock()
		writeJSONError(w, "ResourceNotFoundException", "Requested resource not found: Table: "+name+" not found", http.StatusBadRequest)
		return
	}
	delete(s.tables, name)
	s.mu.Unlock()

	desc := s.tableDescription(t)
	desc["TableStatus"] = "DELETING"
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"TableDescription": desc,
	})
}

func (s *Service) describeTable(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "TableName")

	s.mu.RLock()
	t, exists := s.tables[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Requested resource not found: Table: "+name+" not found", http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Table": s.tableDescription(t),
	})
}

func (s *Service) listTables(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var names []string
	for name := range s.tables {
		names = append(names, name)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"TableNames": names,
	})
}

func (s *Service) putItem(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "TableName")

	s.mu.RLock()
	t, exists := s.tables[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Requested resource not found: Table: "+name+" not found", http.StatusBadRequest)
		return
	}

	item, ok := params["Item"].(map[string]interface{})
	if !ok {
		writeJSONError(w, "ValidationException", "Item is required", http.StatusBadRequest)
		return
	}

	t.mu.Lock()
	// Check if item with same key exists and replace it.
	keyAttrs := s.getKeyAttributes(t)
	replaced := false
	for i, existing := range t.items {
		if itemKeysMatch(existing, item, keyAttrs) {
			t.items[i] = item
			replaced = true
			break
		}
	}
	if !replaced {
		t.items = append(t.items, item)
		t.itemCount++
	}
	t.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) getItem(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "TableName")

	s.mu.RLock()
	t, exists := s.tables[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Requested resource not found: Table: "+name+" not found", http.StatusBadRequest)
		return
	}

	key, ok := params["Key"].(map[string]interface{})
	if !ok {
		writeJSONError(w, "ValidationException", "Key is required", http.StatusBadRequest)
		return
	}

	keyAttrs := s.getKeyAttributes(t)

	t.mu.Lock()
	var found map[string]interface{}
	for _, item := range t.items {
		if itemKeysMatch(item, key, keyAttrs) {
			found = item
			break
		}
	}
	t.mu.Unlock()

	resp := map[string]interface{}{}
	if found != nil {
		resp["Item"] = found
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) deleteItem(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "TableName")

	s.mu.RLock()
	t, exists := s.tables[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Requested resource not found: Table: "+name+" not found", http.StatusBadRequest)
		return
	}

	key, ok := params["Key"].(map[string]interface{})
	if !ok {
		writeJSONError(w, "ValidationException", "Key is required", http.StatusBadRequest)
		return
	}

	keyAttrs := s.getKeyAttributes(t)

	t.mu.Lock()
	for i, item := range t.items {
		if itemKeysMatch(item, key, keyAttrs) {
			t.items = append(t.items[:i], t.items[i+1:]...)
			t.itemCount--
			break
		}
	}
	t.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) query(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "TableName")

	s.mu.RLock()
	t, exists := s.tables[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Requested resource not found: Table: "+name+" not found", http.StatusBadRequest)
		return
	}

	// Simple implementation: return items matching the KeyConditionExpression values.
	expressionValues, _ := params["ExpressionAttributeValues"].(map[string]interface{})

	t.mu.Lock()
	var items []interface{}
	if expressionValues != nil && len(t.keySchema) > 0 {
		partitionKeyName := t.keySchema[0].AttributeName
		// Try to find the partition key value from expression attribute values.
		for _, val := range expressionValues {
			for _, item := range t.items {
				if itemAttrVal, ok := item[partitionKeyName]; ok {
					if attrValuesEqual(itemAttrVal, val) {
						items = append(items, item)
					}
				}
			}
			break // Only use the first expression value for partition key matching.
		}
	} else {
		for _, item := range t.items {
			items = append(items, item)
		}
	}
	t.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Items":            items,
		"Count":            len(items),
		"ScannedCount":     len(items),
		"ConsumedCapacity": nil,
	})
}

func (s *Service) scan(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "TableName")

	s.mu.RLock()
	t, exists := s.tables[name]
	s.mu.RUnlock()

	if !exists {
		writeJSONError(w, "ResourceNotFoundException", "Requested resource not found: Table: "+name+" not found", http.StatusBadRequest)
		return
	}

	t.mu.Lock()
	var items []interface{}
	for _, item := range t.items {
		items = append(items, item)
	}
	t.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Items":            items,
		"Count":            len(items),
		"ScannedCount":     len(items),
		"ConsumedCapacity": nil,
	})
}

func (s *Service) tableDescription(t *table) map[string]interface{} {
	t.mu.Lock()
	itemCount := t.itemCount
	t.mu.Unlock()

	desc := map[string]interface{}{
		"TableName":            t.name,
		"TableArn":             t.arn,
		"TableStatus":          t.status,
		"CreationDateTime":     float64(t.created.Unix()),
		"ItemCount":            itemCount,
		"TableSizeBytes":       0,
		"BillingModeSummary":   map[string]interface{}{"BillingMode": t.billingMode},
		"KeySchema":            t.keySchema,
		"AttributeDefinitions": t.attributeDefs,
	}

	if t.billingMode == "PROVISIONED" {
		desc["ProvisionedThroughput"] = map[string]interface{}{
			"ReadCapacityUnits":  t.provisionedRead,
			"WriteCapacityUnits": t.provisionedWrite,
			"NumberOfDecreasesToday": 0,
		}
	}

	return desc
}

func (s *Service) getKeyAttributes(t *table) []string {
	var keys []string
	for _, ks := range t.keySchema {
		keys = append(keys, ks.AttributeName)
	}
	return keys
}

// itemKeysMatch checks if two DynamoDB items have the same key attribute values.
func itemKeysMatch(item, key map[string]interface{}, keyAttrs []string) bool {
	for _, attr := range keyAttrs {
		itemVal, ok1 := item[attr]
		keyVal, ok2 := key[attr]
		if !ok1 || !ok2 {
			return false
		}
		if !attrValuesEqual(itemVal, keyVal) {
			return false
		}
	}
	return true
}

// attrValuesEqual compares two DynamoDB attribute values in their map representation.
func attrValuesEqual(a, b interface{}) bool {
	aMap, aOk := a.(map[string]interface{})
	bMap, bOk := b.(map[string]interface{})
	if !aOk || !bOk {
		return false
	}
	// Compare the typed value (e.g., {"S": "val"} == {"S": "val"}).
	for k, av := range aMap {
		if bv, ok := bMap[k]; ok {
			if fmt.Sprintf("%v", av) == fmt.Sprintf("%v", bv) {
				return true
			}
		}
	}
	return false
}

// Helper functions.

func getString(params map[string]interface{}, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt64(params map[string]interface{}, key string, defaultVal int64) int64 {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int:
			return int64(n)
		case int64:
			return n
		}
	}
	return defaultVal
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"__type":  code,
		"message": message,
	})
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

// Package mockhelpers provides shared helper functions for mock AWS services.
//
// This package reduces duplication across service implementations by providing
// common utilities for request ID generation, JSON/XML response writing, and
// parameter extraction.
package mockhelpers

import (
	"encoding/json"
	"encoding/xml"
	"math/rand"
	"net/http"
)

// NewRequestID generates a random UUID-like request ID string.
func NewRequestID() string {
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

// RandomID generates a random uppercase alphanumeric string of length n.
func RandomID(n int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// RandomHex generates a random hexadecimal string of length n.
func RandomHex(n int) string {
	const chars = "abcdef0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// GetString extracts a string value from a params map.
func GetString(params map[string]interface{}, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetInt extracts an integer value from a params map with a default.
func GetInt(params map[string]interface{}, key string, defaultVal int) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}

// GetBool extracts a boolean value from a params map.
func GetBool(params map[string]interface{}, key string) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteJSONError writes a JSON error response with the given code, message, and HTTP status.
func WriteJSONError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"__type":  code,
		"message": message,
	})
}

// WriteXML writes an XML response with the given status code.
func WriteXML(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(v)
}

// WriteXMLError writes a standard AWS XML error response.
func WriteXMLError(w http.ResponseWriter, errType, code, message string, status int) {
	type xmlError struct {
		Type    string `xml:"Type"`
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	}
	type xmlErrorResponse struct {
		XMLName   xml.Name `xml:"ErrorResponse"`
		Error     xmlError `xml:"Error"`
		RequestID string   `xml:"RequestId"`
	}
	resp := xmlErrorResponse{
		Error: xmlError{
			Type:    errType,
			Code:    code,
			Message: message,
		},
		RequestID: NewRequestID(),
	}
	WriteXML(w, status, resp)
}

// DefaultAccountID is the mock AWS account ID used by all services.
const DefaultAccountID = "123456789012"

// Package kms provides a mock implementation of AWS Key Management Service.
//
// Supported actions:
//   - CreateKey
//   - DescribeKey
//   - ListKeys
//   - Encrypt
//   - Decrypt
//   - GenerateDataKey
//   - CreateAlias
//   - ListAliases
//   - DeleteAlias
//   - ScheduleKeyDeletion
package kms

import (
	"encoding/base64"
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

// Service implements the KMS mock.
type Service struct {
	mu      sync.RWMutex
	keys    map[string]*key    // keyed by key ID
	aliases map[string]*alias  // keyed by alias name
}

type key struct {
	id          string
	arn         string
	description string
	state       string
	created     time.Time
	keyUsage    string
	keySpec     string
	deletionDate *time.Time
}

type alias struct {
	name     string
	arn      string
	targetID string
}

// New creates a new KMS mock service.
func New() *Service {
	return &Service{
		keys:    make(map[string]*key),
		aliases: make(map[string]*alias),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "kms" }

// Handler returns the HTTP handler for KMS requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all keys and aliases.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys = make(map[string]*key)
	s.aliases = make(map[string]*alias)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, "KMSInternalException", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			writeJSONError(w, "ValidationException", "could not parse request body", http.StatusBadRequest)
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
	case "CreateKey":
		s.createKey(w, params)
	case "DescribeKey":
		s.describeKey(w, params)
	case "ListKeys":
		s.listKeys(w, params)
	case "Encrypt":
		s.encrypt(w, params)
	case "Decrypt":
		s.decrypt(w, params)
	case "GenerateDataKey":
		s.generateDataKey(w, params)
	case "CreateAlias":
		s.createAlias(w, params)
	case "ListAliases":
		s.listAliases(w, params)
	case "DeleteAlias":
		s.deleteAlias(w, params)
	case "ScheduleKeyDeletion":
		s.scheduleKeyDeletion(w, params)
	default:
		writeJSONError(w, "UnsupportedOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createKey(w http.ResponseWriter, params map[string]interface{}) {
	description := getString(params, "Description")
	keyUsage := getString(params, "KeyUsage")
	if keyUsage == "" {
		keyUsage = "ENCRYPT_DECRYPT"
	}
	keySpec := getString(params, "KeySpec")
	if keySpec == "" {
		keySpec = "SYMMETRIC_DEFAULT"
	}

	s.mu.Lock()
	id := newKeyID()
	k := &key{
		id:          id,
		arn:         fmt.Sprintf("arn:aws:kms:us-east-1:%s:key/%s", defaultAccountID, id),
		description: description,
		state:       "Enabled",
		created:     time.Now().UTC(),
		keyUsage:    keyUsage,
		keySpec:     keySpec,
	}
	s.keys[id] = k
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"KeyMetadata": keyMetadata(k),
	})
}

func (s *Service) describeKey(w http.ResponseWriter, params map[string]interface{}) {
	keyID := getString(params, "KeyId")

	s.mu.RLock()
	k := s.findKey(keyID)
	s.mu.RUnlock()

	if k == nil {
		writeJSONError(w, "NotFoundException", "Key '"+keyID+"' does not exist", http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"KeyMetadata": keyMetadata(k),
	})
}

func (s *Service) listKeys(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var keyList []map[string]interface{}
	for _, k := range s.keys {
		keyList = append(keyList, map[string]interface{}{
			"KeyId":  k.id,
			"KeyArn": k.arn,
		})
	}
	s.mu.RUnlock()

	sort.Slice(keyList, func(i, j int) bool {
		return keyList[i]["KeyId"].(string) < keyList[j]["KeyId"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Keys":      keyList,
		"Truncated": false,
	})
}

func (s *Service) encrypt(w http.ResponseWriter, params map[string]interface{}) {
	keyID := getString(params, "KeyId")
	plaintextB64 := getString(params, "Plaintext")

	s.mu.RLock()
	k := s.findKey(keyID)
	s.mu.RUnlock()

	if k == nil {
		writeJSONError(w, "NotFoundException", "Key '"+keyID+"' does not exist", http.StatusBadRequest)
		return
	}

	// Simple mock: "encrypt" by prepending key ID to plaintext.
	plaintext, _ := base64.StdEncoding.DecodeString(plaintextB64)
	ciphertext := append([]byte(k.id+":"), plaintext...)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"CiphertextBlob":    base64.StdEncoding.EncodeToString(ciphertext),
		"KeyId":             k.arn,
		"EncryptionAlgorithm": "SYMMETRIC_DEFAULT",
	})
}

func (s *Service) decrypt(w http.ResponseWriter, params map[string]interface{}) {
	ciphertextB64 := getString(params, "CiphertextBlob")

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		writeJSONError(w, "InvalidCiphertextException", "Invalid ciphertext", http.StatusBadRequest)
		return
	}

	// Extract key ID and plaintext from our mock format.
	parts := strings.SplitN(string(ciphertext), ":", 2)
	if len(parts) != 2 {
		writeJSONError(w, "InvalidCiphertextException", "Invalid ciphertext format", http.StatusBadRequest)
		return
	}

	keyID := parts[0]
	plaintext := []byte(parts[1])

	s.mu.RLock()
	k := s.findKey(keyID)
	s.mu.RUnlock()

	if k == nil {
		writeJSONError(w, "NotFoundException", "Key not found", http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Plaintext":           base64.StdEncoding.EncodeToString(plaintext),
		"KeyId":               k.arn,
		"EncryptionAlgorithm": "SYMMETRIC_DEFAULT",
	})
}

func (s *Service) generateDataKey(w http.ResponseWriter, params map[string]interface{}) {
	keyID := getString(params, "KeyId")

	s.mu.RLock()
	k := s.findKey(keyID)
	s.mu.RUnlock()

	if k == nil {
		writeJSONError(w, "NotFoundException", "Key '"+keyID+"' does not exist", http.StatusBadRequest)
		return
	}

	// Generate a random 32-byte data key.
	dataKey := make([]byte, 32)
	rand.Read(dataKey)

	// "Encrypt" the data key.
	ciphertext := append([]byte(k.id+":"), dataKey...)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Plaintext":      base64.StdEncoding.EncodeToString(dataKey),
		"CiphertextBlob": base64.StdEncoding.EncodeToString(ciphertext),
		"KeyId":          k.arn,
	})
}

func (s *Service) createAlias(w http.ResponseWriter, params map[string]interface{}) {
	aliasName := getString(params, "AliasName")
	targetKeyID := getString(params, "TargetKeyId")

	s.mu.Lock()
	s.aliases[aliasName] = &alias{
		name:     aliasName,
		arn:      fmt.Sprintf("arn:aws:kms:us-east-1:%s:%s", defaultAccountID, aliasName),
		targetID: targetKeyID,
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listAliases(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var aliasList []map[string]interface{}
	for _, a := range s.aliases {
		aliasList = append(aliasList, map[string]interface{}{
			"AliasName":   a.name,
			"AliasArn":    a.arn,
			"TargetKeyId": a.targetID,
		})
	}
	s.mu.RUnlock()

	sort.Slice(aliasList, func(i, j int) bool {
		return aliasList[i]["AliasName"].(string) < aliasList[j]["AliasName"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"Aliases":   aliasList,
		"Truncated": false,
	})
}

func (s *Service) deleteAlias(w http.ResponseWriter, params map[string]interface{}) {
	aliasName := getString(params, "AliasName")

	s.mu.Lock()
	delete(s.aliases, aliasName)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) scheduleKeyDeletion(w http.ResponseWriter, params map[string]interface{}) {
	keyID := getString(params, "KeyId")

	s.mu.Lock()
	k := s.findKey(keyID)
	if k == nil {
		s.mu.Unlock()
		writeJSONError(w, "NotFoundException", "Key '"+keyID+"' does not exist", http.StatusBadRequest)
		return
	}

	deletionDate := time.Now().UTC().Add(30 * 24 * time.Hour)
	k.state = "PendingDeletion"
	k.deletionDate = &deletionDate
	arn := k.arn
	kid := k.id
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"KeyId":        kid,
		"DeletionDate": float64(deletionDate.Unix()),
		"KeyState":     "PendingDeletion",
		"KeyArn":       arn,
	})
}

// findKey looks up a key by ID, ARN, or alias. Caller must hold s.mu.
func (s *Service) findKey(keyID string) *key {
	// Direct ID lookup.
	if k, ok := s.keys[keyID]; ok {
		return k
	}
	// ARN lookup.
	for _, k := range s.keys {
		if k.arn == keyID {
			return k
		}
	}
	// Alias lookup.
	if a, ok := s.aliases[keyID]; ok {
		if k, ok := s.keys[a.targetID]; ok {
			return k
		}
	}
	return nil
}

func keyMetadata(k *key) map[string]interface{} {
	meta := map[string]interface{}{
		"KeyId":               k.id,
		"Arn":                 k.arn,
		"Description":         k.description,
		"KeyState":            k.state,
		"CreationDate":        float64(k.created.Unix()),
		"Enabled":             k.state == "Enabled",
		"KeyUsage":            k.keyUsage,
		"KeySpec":             k.keySpec,
		"KeyManager":          "CUSTOMER",
		"CustomerMasterKeySpec": k.keySpec,
	}
	if k.deletionDate != nil {
		meta["DeletionDate"] = float64(k.deletionDate.Unix())
	}
	return meta
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

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"__type":  code,
		"message": message,
	})
}

func newKeyID() string {
	return newRequestID()
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

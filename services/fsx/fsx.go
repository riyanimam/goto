// Package fsx provides a mock implementation of AWS FSx.
//
// Supported actions:
//   - CreateFileSystem
//   - DescribeFileSystems
//   - DeleteFileSystem
//   - UpdateFileSystem
//   - TagResource
package fsx

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

// Service implements the FSx mock.
type Service struct {
	mu          sync.RWMutex
	fileSystems map[string]*fileSystem
}

type fileSystem struct {
	id              string
	fileSystemType  string
	storageCapacity int
	storageType     string
	lifecycle       string
	creationTime    time.Time
	arn             string
	subnetIDs       []string
	tags            []map[string]interface{}
}

// New creates a new FSx mock service.
func New() *Service {
	return &Service{
		fileSystems: make(map[string]*fileSystem),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "fsx" }

// Handler returns the HTTP handler for FSx requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fileSystems = make(map[string]*fileSystem)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.WriteJSONError(w, "InternalServerError", "could not read request body", http.StatusInternalServerError)
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
	case "CreateFileSystem":
		s.createFileSystem(w, params)
	case "DescribeFileSystems":
		s.describeFileSystems(w, params)
	case "DeleteFileSystem":
		s.deleteFileSystem(w, params)
	case "UpdateFileSystem":
		s.updateFileSystem(w, params)
	case "TagResource":
		s.tagResource(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createFileSystem(w http.ResponseWriter, params map[string]interface{}) {
	fsType := h.GetString(params, "FileSystemType")
	storageCapacity := h.GetInt(params, "StorageCapacity", 0)
	storageType := h.GetString(params, "StorageType")

	var subnetIDs []string
	if raw, ok := params["SubnetIds"].([]interface{}); ok {
		for _, v := range raw {
			if str, ok := v.(string); ok {
				subnetIDs = append(subnetIDs, str)
			}
		}
	}

	var tags []map[string]interface{}
	if raw, ok := params["Tags"].([]interface{}); ok {
		for _, v := range raw {
			if m, ok := v.(map[string]interface{}); ok {
				tags = append(tags, m)
			}
		}
	}

	fsID := "fs-" + h.RandomHex(17)
	arn := fmt.Sprintf("arn:aws:fsx:us-east-1:%s:file-system/%s", h.DefaultAccountID, fsID)
	now := time.Now().UTC()

	fs := &fileSystem{
		id:              fsID,
		fileSystemType:  fsType,
		storageCapacity: storageCapacity,
		storageType:     storageType,
		lifecycle:       "CREATING",
		creationTime:    now,
		arn:             arn,
		subnetIDs:       subnetIDs,
		tags:            tags,
	}

	s.mu.Lock()
	s.fileSystems[fsID] = fs
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"FileSystem": fsResp(fs),
	})
}

func (s *Service) describeFileSystems(w http.ResponseWriter, params map[string]interface{}) {
	var filterIDs []string
	if raw, ok := params["FileSystemIds"].([]interface{}); ok {
		for _, v := range raw {
			if str, ok := v.(string); ok {
				filterIDs = append(filterIDs, str)
			}
		}
	}

	s.mu.RLock()
	var list []map[string]interface{}
	if len(filterIDs) > 0 {
		for _, id := range filterIDs {
			if fs, ok := s.fileSystems[id]; ok {
				list = append(list, fsResp(fs))
			}
		}
	} else {
		for _, fs := range s.fileSystems {
			list = append(list, fsResp(fs))
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"FileSystems": list,
	})
}

func (s *Service) deleteFileSystem(w http.ResponseWriter, params map[string]interface{}) {
	fsID := h.GetString(params, "FileSystemId")
	if fsID == "" {
		h.WriteJSONError(w, "BadRequest", "FileSystemId is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	_, exists := s.fileSystems[fsID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "FileSystemNotFound", fmt.Sprintf("File system %q not found", fsID), http.StatusNotFound)
		return
	}
	delete(s.fileSystems, fsID)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"FileSystemId": fsID,
		"Lifecycle":    "DELETING",
	})
}

func (s *Service) updateFileSystem(w http.ResponseWriter, params map[string]interface{}) {
	fsID := h.GetString(params, "FileSystemId")
	if fsID == "" {
		h.WriteJSONError(w, "BadRequest", "FileSystemId is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	fs, exists := s.fileSystems[fsID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "FileSystemNotFound", fmt.Sprintf("File system %q not found", fsID), http.StatusNotFound)
		return
	}

	if _, ok := params["StorageCapacity"]; ok {
		fs.storageCapacity = h.GetInt(params, "StorageCapacity", 0)
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"FileSystem": fsResp(fs),
	})
}

func (s *Service) tagResource(w http.ResponseWriter, params map[string]interface{}) {
	resourceARN := h.GetString(params, "ResourceARN")
	if resourceARN == "" {
		h.WriteJSONError(w, "BadRequest", "ResourceARN is required", http.StatusBadRequest)
		return
	}

	var newTags []map[string]interface{}
	if raw, ok := params["Tags"].([]interface{}); ok {
		for _, v := range raw {
			if m, ok := v.(map[string]interface{}); ok {
				newTags = append(newTags, m)
			}
		}
	}

	// Find the file system by ARN and append tags.
	s.mu.Lock()
	for _, fs := range s.fileSystems {
		if fs.arn == resourceARN {
			fs.tags = append(fs.tags, newTags...)
			break
		}
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func fsResp(fs *fileSystem) map[string]interface{} {
	return map[string]interface{}{
		"FileSystemId":    fs.id,
		"FileSystemType":  fs.fileSystemType,
		"StorageCapacity": fs.storageCapacity,
		"StorageType":     fs.storageType,
		"Lifecycle":       fs.lifecycle,
		"CreationTime":    float64(fs.creationTime.Unix()),
		"ResourceARN":     fs.arn,
		"Tags":            fs.tags,
		"SubnetIds":       fs.subnetIDs,
	}
}

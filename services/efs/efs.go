// Package efs provides a mock implementation of AWS Elastic File System.
//
// Supported actions:
//   - CreateFileSystem
//   - DescribeFileSystems
//   - DeleteFileSystem
//   - CreateMountTarget
//   - DescribeMountTargets
//   - DeleteMountTarget
package efs

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

// Service implements the EFS mock.
type Service struct {
	mu           sync.RWMutex
	fileSystems  map[string]*fileSystem
	mountTargets map[string]*mountTarget
}

type fileSystem struct {
	id              string
	creationToken   string
	performanceMode string
	encrypted       bool
	lifeCycleState  string
	sizeInBytes     int64
	created         time.Time
}

type mountTarget struct {
	id             string
	fileSystemId   string
	subnetId       string
	ipAddress      string
	lifeCycleState string
}

// New creates a new EFS mock service.
func New() *Service {
	return &Service{
		fileSystems:  make(map[string]*fileSystem),
		mountTargets: make(map[string]*mountTarget),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "elasticfilesystem" }

// Handler returns the HTTP handler for EFS requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fileSystems = make(map[string]*fileSystem)
	s.mountTargets = make(map[string]*mountTarget)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	switch {
	// DeleteFileSystem: DELETE /2015-02-01/file-systems/{fsId}
	case strings.HasPrefix(path, "/2015-02-01/file-systems/") && method == http.MethodDelete:
		s.deleteFileSystem(w, r, path)

	// CreateFileSystem: POST /2015-02-01/file-systems
	case path == "/2015-02-01/file-systems" && method == http.MethodPost:
		s.createFileSystem(w, r)

	// DescribeFileSystems: GET /2015-02-01/file-systems
	case path == "/2015-02-01/file-systems" && method == http.MethodGet:
		s.describeFileSystems(w, r)

	// DeleteMountTarget: DELETE /2015-02-01/mount-targets/{mtId}
	case strings.HasPrefix(path, "/2015-02-01/mount-targets/") && method == http.MethodDelete:
		s.deleteMountTarget(w, r, path)

	// CreateMountTarget: POST /2015-02-01/mount-targets
	case path == "/2015-02-01/mount-targets" && method == http.MethodPost:
		s.createMountTarget(w, r)

	// DescribeMountTargets: GET /2015-02-01/mount-targets
	case path == "/2015-02-01/mount-targets" && method == http.MethodGet:
		s.describeMountTargets(w, r)

	default:
		h.WriteJSONError(w, "NotFoundException", "unsupported operation", http.StatusNotFound)
	}
}

func (s *Service) createFileSystem(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	creationToken := h.GetString(params, "CreationToken")
	if creationToken == "" {
		h.WriteJSONError(w, "BadRequest", "CreationToken is required", http.StatusBadRequest)
		return
	}

	performanceMode := h.GetString(params, "PerformanceMode")
	if performanceMode == "" {
		performanceMode = "generalPurpose"
	}

	encrypted := h.GetBool(params, "Encrypted")

	s.mu.Lock()
	// Check for duplicate creation token.
	for _, fs := range s.fileSystems {
		if fs.creationToken == creationToken {
			s.mu.Unlock()
			h.WriteJSONError(w, "FileSystemAlreadyExists", "File system with CreationToken "+creationToken+" already exists", http.StatusConflict)
			return
		}
	}

	id := fmt.Sprintf("fs-%s", h.RandomHex(17))
	now := time.Now().UTC()

	fs := &fileSystem{
		id:              id,
		creationToken:   creationToken,
		performanceMode: performanceMode,
		encrypted:       encrypted,
		lifeCycleState:  "available",
		sizeInBytes:     0,
		created:         now,
	}
	s.fileSystems[id] = fs
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusCreated, fileSystemResp(fs))
}

func (s *Service) describeFileSystems(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var systems []map[string]interface{}
	for _, fs := range s.fileSystems {
		systems = append(systems, fileSystemResp(fs))
	}
	s.mu.RUnlock()

	if systems == nil {
		systems = []map[string]interface{}{}
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"FileSystems": systems,
	})
}

func (s *Service) deleteFileSystem(w http.ResponseWriter, _ *http.Request, path string) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 3 {
		h.WriteJSONError(w, "BadRequest", "invalid path", http.StatusBadRequest)
		return
	}
	fsId := parts[2]

	s.mu.Lock()
	if _, exists := s.fileSystems[fsId]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "FileSystemNotFound", "File system "+fsId+" not found", http.StatusNotFound)
		return
	}
	delete(s.fileSystems, fsId)
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) createMountTarget(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	fileSystemId := h.GetString(params, "FileSystemId")
	if fileSystemId == "" {
		h.WriteJSONError(w, "BadRequest", "FileSystemId is required", http.StatusBadRequest)
		return
	}

	subnetId := h.GetString(params, "SubnetId")
	if subnetId == "" {
		h.WriteJSONError(w, "BadRequest", "SubnetId is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.fileSystems[fileSystemId]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "FileSystemNotFound", "File system "+fileSystemId+" not found", http.StatusNotFound)
		return
	}

	id := fmt.Sprintf("fsmt-%s", h.RandomHex(17))
	ipAddress := fmt.Sprintf("10.0.%d.%d", len(s.mountTargets)%256, (len(s.mountTargets)+1)%256)

	mt := &mountTarget{
		id:             id,
		fileSystemId:   fileSystemId,
		subnetId:       subnetId,
		ipAddress:      ipAddress,
		lifeCycleState: "available",
	}
	s.mountTargets[id] = mt
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, mountTargetResp(mt))
}

func (s *Service) describeMountTargets(w http.ResponseWriter, r *http.Request) {
	fileSystemId := r.URL.Query().Get("FileSystemId")

	s.mu.RLock()
	var targets []map[string]interface{}
	for _, mt := range s.mountTargets {
		if fileSystemId == "" || mt.fileSystemId == fileSystemId {
			targets = append(targets, mountTargetResp(mt))
		}
	}
	s.mu.RUnlock()

	if targets == nil {
		targets = []map[string]interface{}{}
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"MountTargets": targets,
	})
}

func (s *Service) deleteMountTarget(w http.ResponseWriter, _ *http.Request, path string) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 3 {
		h.WriteJSONError(w, "BadRequest", "invalid path", http.StatusBadRequest)
		return
	}
	mtId := parts[2]

	s.mu.Lock()
	if _, exists := s.mountTargets[mtId]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "MountTargetNotFound", "Mount target "+mtId+" not found", http.StatusNotFound)
		return
	}
	delete(s.mountTargets, mtId)
	s.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func fileSystemResp(fs *fileSystem) map[string]interface{} {
	return map[string]interface{}{
		"FileSystemId":    fs.id,
		"CreationToken":   fs.creationToken,
		"PerformanceMode": fs.performanceMode,
		"Encrypted":       fs.encrypted,
		"LifeCycleState":  fs.lifeCycleState,
		"SizeInBytes": map[string]interface{}{
			"Value": fs.sizeInBytes,
		},
		"CreationTime": float64(fs.created.Unix()),
	}
}

func mountTargetResp(mt *mountTarget) map[string]interface{} {
	return map[string]interface{}{
		"MountTargetId":  mt.id,
		"FileSystemId":   mt.fileSystemId,
		"SubnetId":       mt.subnetId,
		"IpAddress":      mt.ipAddress,
		"LifeCycleState": mt.lifeCycleState,
	}
}

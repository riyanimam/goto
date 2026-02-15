// Package backup provides a mock implementation of AWS Backup.
//
// Supported actions:
//   - CreateBackupVault
//   - DeleteBackupVault
//   - ListBackupVaults
//   - DescribeBackupVault
//   - CreateBackupPlan
//   - GetBackupPlan
//   - DeleteBackupPlan
//   - ListBackupPlans
package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the Backup mock.
type Service struct {
	mu     sync.RWMutex
	vaults map[string]*backupVault
	plans  map[string]*backupPlan
}

type backupVault struct {
	name                   string
	arn                    string
	created                time.Time
	numberOfRecoveryPoints int64
	tags                   map[string]string
}

type backupPlan struct {
	id        string
	name      string
	arn       string
	versionId string
	rules     interface{}
	created   time.Time
}

// New creates a new Backup mock service.
func New() *Service {
	return &Service{
		vaults: make(map[string]*backupVault),
		plans:  make(map[string]*backupPlan),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "backup" }

// Handler returns the HTTP handler for Backup requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vaults = make(map[string]*backupVault)
	s.plans = make(map[string]*backupPlan)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	switch {
	// Backup vaults: /backup-vaults/{name}
	case len(parts) == 2 && parts[0] == "backup-vaults" && parts[1] != "" && method == http.MethodPut:
		s.createBackupVault(w, r, parts[1])
	case len(parts) == 2 && parts[0] == "backup-vaults" && parts[1] != "" && method == http.MethodDelete:
		s.deleteBackupVault(w, parts[1])
	case len(parts) == 2 && parts[0] == "backup-vaults" && parts[1] != "" && method == http.MethodGet:
		s.describeBackupVault(w, parts[1])

	// List backup vaults: /backup-vaults/
	case len(parts) >= 1 && parts[0] == "backup-vaults" && (len(parts) == 1 || parts[1] == "") && method == http.MethodGet:
		s.listBackupVaults(w)

	// Backup plans: /backup/plans/{planId}
	case len(parts) == 3 && parts[0] == "backup" && parts[1] == "plans" && parts[2] != "" && method == http.MethodGet:
		s.getBackupPlan(w, parts[2])
	case len(parts) == 3 && parts[0] == "backup" && parts[1] == "plans" && parts[2] != "" && method == http.MethodDelete:
		s.deleteBackupPlan(w, parts[2])

	// Create/List backup plans: /backup/plans/
	case len(parts) >= 2 && parts[0] == "backup" && parts[1] == "plans" && (len(parts) == 2 || parts[2] == "") && method == http.MethodPut:
		s.createBackupPlan(w, r)
	case len(parts) >= 2 && parts[0] == "backup" && parts[1] == "plans" && (len(parts) == 2 || parts[2] == "") && method == http.MethodGet:
		s.listBackupPlans(w)

	default:
		h.WriteJSONError(w, "NotFoundException", "unsupported operation", http.StatusNotFound)
	}
}

func (s *Service) createBackupVault(w http.ResponseWriter, r *http.Request, name string) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	tags := make(map[string]string)
	if t, ok := params["BackupVaultTags"].(map[string]interface{}); ok {
		for k, v := range t {
			if str, ok := v.(string); ok {
				tags[k] = str
			}
		}
	}

	s.mu.Lock()
	if _, exists := s.vaults[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "AlreadyExistsException", "Backup vault "+name+" already exists", http.StatusConflict)
		return
	}

	arn := fmt.Sprintf("arn:aws:backup:us-east-1:%s:backup-vault:%s", h.DefaultAccountID, name)
	now := time.Now().UTC()

	v := &backupVault{
		name:                   name,
		arn:                    arn,
		created:                now,
		numberOfRecoveryPoints: 0,
		tags:                   tags,
	}
	s.vaults[name] = v
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, vaultResp(v))
}

func (s *Service) deleteBackupVault(w http.ResponseWriter, name string) {
	s.mu.Lock()
	_, exists := s.vaults[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Backup vault "+name+" not found", http.StatusNotFound)
		return
	}
	delete(s.vaults, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) describeBackupVault(w http.ResponseWriter, name string) {
	s.mu.RLock()
	v, exists := s.vaults[name]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Backup vault "+name+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, vaultResp(v))
}

func (s *Service) listBackupVaults(w http.ResponseWriter) {
	s.mu.RLock()
	var list []map[string]interface{}
	for _, v := range s.vaults {
		list = append(list, vaultResp(v))
	}
	s.mu.RUnlock()

	sort.Slice(list, func(i, j int) bool {
		return list[i]["BackupVaultName"].(string) < list[j]["BackupVaultName"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"BackupVaultList": list,
	})
}

func (s *Service) createBackupPlan(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	bp, _ := params["BackupPlan"].(map[string]interface{})
	planName := ""
	var rules interface{}
	if bp != nil {
		planName, _ = bp["BackupPlanName"].(string)
		rules = bp["Rules"]
	}

	if planName == "" {
		h.WriteJSONError(w, "InvalidParameterValueException", "BackupPlanName is required", http.StatusBadRequest)
		return
	}

	planID := h.RandomID(36)
	versionID := h.RandomID(36)
	arn := fmt.Sprintf("arn:aws:backup:us-east-1:%s:backup-plan:%s", h.DefaultAccountID, planID)
	now := time.Now().UTC()

	p := &backupPlan{
		id:        planID,
		name:      planName,
		arn:       arn,
		versionId: versionID,
		rules:     rules,
		created:   now,
	}

	s.mu.Lock()
	s.plans[planID] = p
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"BackupPlanId":  planID,
		"BackupPlanArn": arn,
		"VersionId":     versionID,
		"CreationDate":  float64(now.Unix()),
	})
}

func (s *Service) getBackupPlan(w http.ResponseWriter, planID string) {
	s.mu.RLock()
	p, exists := s.plans[planID]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Backup plan "+planID+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, planResp(p))
}

func (s *Service) deleteBackupPlan(w http.ResponseWriter, planID string) {
	s.mu.Lock()
	p, exists := s.plans[planID]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Backup plan "+planID+" not found", http.StatusNotFound)
		return
	}
	resp := map[string]interface{}{
		"BackupPlanId":  p.id,
		"BackupPlanArn": p.arn,
		"VersionId":     p.versionId,
		"DeletionDate":  float64(time.Now().UTC().Unix()),
	}
	delete(s.plans, planID)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, resp)
}

func (s *Service) listBackupPlans(w http.ResponseWriter) {
	s.mu.RLock()
	var list []map[string]interface{}
	for _, p := range s.plans {
		list = append(list, planResp(p))
	}
	s.mu.RUnlock()

	sort.Slice(list, func(i, j int) bool {
		return list[i]["BackupPlanName"].(string) < list[j]["BackupPlanName"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"BackupPlansList": list,
	})
}

func vaultResp(v *backupVault) map[string]interface{} {
	return map[string]interface{}{
		"BackupVaultName":        v.name,
		"BackupVaultArn":         v.arn,
		"CreationDate":           float64(v.created.Unix()),
		"NumberOfRecoveryPoints": v.numberOfRecoveryPoints,
		"BackupVaultTags":        v.tags,
	}
}

func planResp(p *backupPlan) map[string]interface{} {
	return map[string]interface{}{
		"BackupPlanId":  p.id,
		"BackupPlanName": p.name,
		"BackupPlanArn": p.arn,
		"VersionId":     p.versionId,
		"CreationDate":  float64(p.created.Unix()),
		"BackupPlan": map[string]interface{}{
			"BackupPlanName": p.name,
			"Rules":          p.rules,
		},
	}
}

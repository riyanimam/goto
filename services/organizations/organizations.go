// Package organizations provides a mock implementation of AWS Organizations.
//
// Supported actions:
//   - CreateOrganization
//   - DescribeOrganization
//   - ListAccounts
//   - CreateAccount
//   - DescribeAccount
//   - CreateOrganizationalUnit
//   - ListOrganizationalUnitsForParent
package organizations

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

// Service implements the Organizations mock.
type Service struct {
	mu       sync.RWMutex
	org      *organization
	accounts map[string]*account
	ous      map[string]*organizationalUnit
	rootID   string
}

type organization struct {
	id                 string
	arn                string
	masterAccountID    string
	masterAccountEmail string
	featureSet         string
}

type account struct {
	id              string
	name            string
	email           string
	arn             string
	status          string
	joinedMethod    string
	joinedTimestamp time.Time
}

type organizationalUnit struct {
	id       string
	name     string
	arn      string
	parentID string
}

// New creates a new Organizations mock service.
func New() *Service {
	return &Service{
		accounts: make(map[string]*account),
		ous:      make(map[string]*organizationalUnit),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "organizations" }

// Handler returns the HTTP handler for Organizations requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.org = nil
	s.accounts = make(map[string]*account)
	s.ous = make(map[string]*organizationalUnit)
	s.rootID = ""
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
	case "CreateOrganization":
		s.createOrganization(w, params)
	case "DescribeOrganization":
		s.describeOrganization(w)
	case "ListAccounts":
		s.listAccounts(w)
	case "CreateAccount":
		s.createAccount(w, params)
	case "DescribeAccount":
		s.describeAccount(w, params)
	case "CreateOrganizationalUnit":
		s.createOrganizationalUnit(w, params)
	case "ListOrganizationalUnitsForParent":
		s.listOrganizationalUnitsForParent(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createOrganization(w http.ResponseWriter, params map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.org != nil {
		h.WriteJSONError(w, "AlreadyInOrganizationException", "The AWS account is already a member of an organization", http.StatusConflict)
		return
	}

	featureSet := h.GetString(params, "FeatureSet")
	if featureSet == "" {
		featureSet = "ALL"
	}

	orgID := "o-" + h.RandomID(10)
	masterAccountID := h.DefaultAccountID
	masterEmail := "master@example.com"
	rootID := "r-" + h.RandomID(4)

	s.org = &organization{
		id:                 orgID,
		arn:                fmt.Sprintf("arn:aws:organizations::%s:organization/%s", masterAccountID, orgID),
		masterAccountID:    masterAccountID,
		masterAccountEmail: masterEmail,
		featureSet:         featureSet,
	}

	s.rootID = rootID

	// Create the root OU.
	s.ous[rootID] = &organizationalUnit{
		id:       rootID,
		name:     "Root",
		arn:      fmt.Sprintf("arn:aws:organizations::%s:root/%s/%s", masterAccountID, orgID, rootID),
		parentID: "",
	}

	// Create the master account.
	s.accounts[masterAccountID] = &account{
		id:              masterAccountID,
		name:            "Master Account",
		email:           masterEmail,
		arn:             fmt.Sprintf("arn:aws:organizations::%s:account/%s/%s", masterAccountID, orgID, masterAccountID),
		status:          "ACTIVE",
		joinedMethod:    "CREATED",
		joinedTimestamp: time.Now().UTC(),
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Organization": orgResp(s.org),
	})
}

func (s *Service) describeOrganization(w http.ResponseWriter) {
	s.mu.RLock()
	org := s.org
	s.mu.RUnlock()

	if org == nil {
		h.WriteJSONError(w, "AWSOrganizationsNotInUseException", "Your account is not a member of an organization", http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Organization": orgResp(org),
	})
}

func (s *Service) listAccounts(w http.ResponseWriter) {
	s.mu.RLock()
	if s.org == nil {
		s.mu.RUnlock()
		h.WriteJSONError(w, "AWSOrganizationsNotInUseException", "Your account is not a member of an organization", http.StatusBadRequest)
		return
	}

	var list []map[string]interface{}
	for _, a := range s.accounts {
		list = append(list, acctResp(a))
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Accounts": list,
	})
}

func (s *Service) createAccount(w http.ResponseWriter, params map[string]interface{}) {
	accountName := h.GetString(params, "AccountName")
	email := h.GetString(params, "Email")
	if accountName == "" || email == "" {
		h.WriteJSONError(w, "InvalidInputException", "AccountName and Email are required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if s.org == nil {
		s.mu.Unlock()
		h.WriteJSONError(w, "AWSOrganizationsNotInUseException", "Your account is not a member of an organization", http.StatusBadRequest)
		return
	}

	acctID := h.RandomID(12)
	a := &account{
		id:              acctID,
		name:            accountName,
		email:           email,
		arn:             fmt.Sprintf("arn:aws:organizations::%s:account/%s/%s", h.DefaultAccountID, s.org.id, acctID),
		status:          "ACTIVE",
		joinedMethod:    "CREATED",
		joinedTimestamp: time.Now().UTC(),
	}
	s.accounts[acctID] = a
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"CreateAccountStatus": map[string]interface{}{
			"Id":          h.NewRequestID(),
			"AccountId":   acctID,
			"AccountName": accountName,
			"State":       "SUCCEEDED",
		},
	})
}

func (s *Service) describeAccount(w http.ResponseWriter, params map[string]interface{}) {
	accountID := h.GetString(params, "AccountId")

	s.mu.RLock()
	if s.org == nil {
		s.mu.RUnlock()
		h.WriteJSONError(w, "AWSOrganizationsNotInUseException", "Your account is not a member of an organization", http.StatusBadRequest)
		return
	}
	a, exists := s.accounts[accountID]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "AccountNotFoundException", "Account not found: "+accountID, http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Account": acctResp(a),
	})
}

func (s *Service) createOrganizationalUnit(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")
	parentID := h.GetString(params, "ParentId")
	if name == "" || parentID == "" {
		h.WriteJSONError(w, "InvalidInputException", "Name and ParentId are required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if s.org == nil {
		s.mu.Unlock()
		h.WriteJSONError(w, "AWSOrganizationsNotInUseException", "Your account is not a member of an organization", http.StatusBadRequest)
		return
	}

	if _, exists := s.ous[parentID]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ParentNotFoundException", "Parent not found: "+parentID, http.StatusBadRequest)
		return
	}

	ouID := "ou-" + h.RandomID(10)
	ou := &organizationalUnit{
		id:       ouID,
		name:     name,
		arn:      fmt.Sprintf("arn:aws:organizations::%s:ou/%s/%s", h.DefaultAccountID, s.org.id, ouID),
		parentID: parentID,
	}
	s.ous[ouID] = ou
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"OrganizationalUnit": ouResp(ou),
	})
}

func (s *Service) listOrganizationalUnitsForParent(w http.ResponseWriter, params map[string]interface{}) {
	parentID := h.GetString(params, "ParentId")

	s.mu.RLock()
	if s.org == nil {
		s.mu.RUnlock()
		h.WriteJSONError(w, "AWSOrganizationsNotInUseException", "Your account is not a member of an organization", http.StatusBadRequest)
		return
	}

	if _, exists := s.ous[parentID]; !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "ParentNotFoundException", "Parent not found: "+parentID, http.StatusBadRequest)
		return
	}

	var list []map[string]interface{}
	for _, ou := range s.ous {
		if ou.parentID == parentID {
			list = append(list, ouResp(ou))
		}
	}
	s.mu.RUnlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"OrganizationalUnits": list,
	})
}

func orgResp(o *organization) map[string]interface{} {
	return map[string]interface{}{
		"Id":                 o.id,
		"Arn":                o.arn,
		"MasterAccountId":    o.masterAccountID,
		"MasterAccountEmail": o.masterAccountEmail,
		"FeatureSet":         o.featureSet,
	}
}

func acctResp(a *account) map[string]interface{} {
	return map[string]interface{}{
		"Id":              a.id,
		"Name":            a.name,
		"Email":           a.email,
		"Arn":             a.arn,
		"Status":          a.status,
		"JoinedMethod":    a.joinedMethod,
		"JoinedTimestamp": float64(a.joinedTimestamp.Unix()),
	}
}

func ouResp(ou *organizationalUnit) map[string]interface{} {
	return map[string]interface{}{
		"Id":   ou.id,
		"Name": ou.name,
		"Arn":  ou.arn,
	}
}

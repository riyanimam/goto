// Package ses provides a mock implementation of AWS Simple Email Service (SES v2).
//
// Supported actions:
//   - CreateEmailIdentity (VerifyEmailIdentity)
//   - GetEmailIdentity
//   - ListEmailIdentities
//   - SendEmail
//   - DeleteEmailIdentity
package ses

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

// Service implements the SES mock.
type Service struct {
	mu         sync.RWMutex
	identities map[string]*emailIdentity
	sentEmails []*sentEmail
}

type emailIdentity struct {
	identity     string
	identityType string // EMAIL_ADDRESS or DOMAIN
	verified     bool
	created      time.Time
}

type sentEmail struct {
	messageID string
	from      string
	to        []string
	subject   string
	body      string
	sentAt    time.Time
}

// New creates a new SES mock service.
func New() *Service {
	return &Service{
		identities: make(map[string]*emailIdentity),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "ses" }

// Handler returns the HTTP handler for SES requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.identities = make(map[string]*emailIdentity)
	s.sentEmails = nil
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case strings.HasSuffix(path, "/v2/email/identities") && r.Method == http.MethodGet:
		s.listEmailIdentities(w, r)
	case strings.HasSuffix(path, "/v2/email/identities") && r.Method == http.MethodPost:
		s.createEmailIdentity(w, r)
	case strings.Contains(path, "/v2/email/identities/") && r.Method == http.MethodGet:
		identity := extractLastSegment(path)
		s.getEmailIdentity(w, r, identity)
	case strings.Contains(path, "/v2/email/identities/") && r.Method == http.MethodDelete:
		identity := extractLastSegment(path)
		s.deleteEmailIdentity(w, r, identity)
	case strings.HasSuffix(path, "/v2/email/outbound-emails") && r.Method == http.MethodPost:
		s.sendEmail(w, r)
	default:
		h.WriteJSONError(w, "NotFoundException", "unsupported operation", http.StatusBadRequest)
	}
}

func extractLastSegment(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func (s *Service) createEmailIdentity(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	identity := h.GetString(params, "EmailIdentity")
	if identity == "" {
		h.WriteJSONError(w, "BadRequestException", "EmailIdentity is required", http.StatusBadRequest)
		return
	}

	identityType := "EMAIL_ADDRESS"
	if !strings.Contains(identity, "@") {
		identityType = "DOMAIN"
	}

	s.mu.Lock()
	s.identities[identity] = &emailIdentity{
		identity:     identity,
		identityType: identityType,
		verified:     true, // Auto-verify in mock.
		created:      time.Now().UTC(),
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"IdentityType":             identityType,
		"VerifiedForSendingStatus": true,
		"DkimAttributes": map[string]interface{}{
			"SigningEnabled":          true,
			"Status":                  "SUCCESS",
			"SigningAttributesOrigin": "AWS_SES",
		},
	})
}

func (s *Service) getEmailIdentity(w http.ResponseWriter, _ *http.Request, identity string) {
	s.mu.RLock()
	id, exists := s.identities[identity]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "NotFoundException", "Identity "+identity+" does not exist.", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"IdentityType":             id.identityType,
		"VerifiedForSendingStatus": id.verified,
		"FeedbackForwardingStatus": true,
	})
}

func (s *Service) deleteEmailIdentity(w http.ResponseWriter, _ *http.Request, identity string) {
	s.mu.Lock()
	delete(s.identities, identity)
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

func (s *Service) listEmailIdentities(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var identities []map[string]interface{}
	for _, id := range s.identities {
		identities = append(identities, map[string]interface{}{
			"IdentityType":   id.identityType,
			"IdentityName":   id.identity,
			"SendingEnabled": id.verified,
		})
	}
	s.mu.RUnlock()

	sort.Slice(identities, func(i, j int) bool {
		return identities[i]["IdentityName"].(string) < identities[j]["IdentityName"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"EmailIdentities": identities,
	})
}

func (s *Service) sendEmail(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	var params map[string]interface{}
	json.Unmarshal(bodyBytes, &params)

	from := ""
	if fromAddr := h.GetString(params, "FromEmailAddress"); fromAddr != "" {
		from = fromAddr
	}

	var to []string
	if dest, ok := params["Destination"].(map[string]interface{}); ok {
		if toAddrs, ok := dest["ToAddresses"].([]interface{}); ok {
			for _, addr := range toAddrs {
				if s, ok := addr.(string); ok {
					to = append(to, s)
				}
			}
		}
	}

	subject := ""
	body := ""
	if content, ok := params["Content"].(map[string]interface{}); ok {
		if simple, ok := content["Simple"].(map[string]interface{}); ok {
			if subj, ok := simple["Subject"].(map[string]interface{}); ok {
				subject = h.GetString(subj, "Data")
			}
			if b, ok := simple["Body"].(map[string]interface{}); ok {
				if text, ok := b["Text"].(map[string]interface{}); ok {
					body = h.GetString(text, "Data")
				}
			}
		}
	}

	messageID := fmt.Sprintf("%s@email.amazonses.com", h.NewRequestID())

	s.mu.Lock()
	s.sentEmails = append(s.sentEmails, &sentEmail{
		messageID: messageID,
		from:      from,
		to:        to,
		subject:   subject,
		body:      body,
		sentAt:    time.Now().UTC(),
	})
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"MessageId": messageID,
	})
}

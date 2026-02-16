// Package acm provides a mock implementation of AWS Certificate Manager.
//
// Supported actions:
//   - RequestCertificate
//   - DescribeCertificate
//   - ListCertificates
//   - DeleteCertificate
package acm

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

// Service implements the ACM mock.
type Service struct {
	mu    sync.RWMutex
	certs map[string]*certificate
}

type certificate struct {
	arn              string
	domainName       string
	subjectAltNames  []string
	status           string
	certType         string
	validationMethod string
	created          time.Time
}

// New creates a new ACM mock service.
func New() *Service {
	return &Service{
		certs: make(map[string]*certificate),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "acm" }

// Handler returns the HTTP handler for ACM requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.certs = make(map[string]*certificate)
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
	case "RequestCertificate":
		s.requestCertificate(w, params)
	case "DescribeCertificate":
		s.describeCertificate(w, params)
	case "ListCertificates":
		s.listCertificates(w, params)
	case "DeleteCertificate":
		s.deleteCertificate(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) requestCertificate(w http.ResponseWriter, params map[string]interface{}) {
	domainName := h.GetString(params, "DomainName")
	if domainName == "" {
		h.WriteJSONError(w, "InvalidParameterException", "DomainName is required", http.StatusBadRequest)
		return
	}

	validationMethod := h.GetString(params, "ValidationMethod")
	if validationMethod == "" {
		validationMethod = "DNS"
	}

	var altNames []string
	if sans, ok := params["SubjectAlternativeNames"].([]interface{}); ok {
		for _, san := range sans {
			if name, ok := san.(string); ok {
				altNames = append(altNames, name)
			}
		}
	}

	s.mu.Lock()
	arn := fmt.Sprintf("arn:aws:acm:us-east-1:%s:certificate/%s", h.DefaultAccountID, h.NewRequestID())
	cert := &certificate{
		arn:              arn,
		domainName:       domainName,
		subjectAltNames:  altNames,
		status:           "ISSUED",
		certType:         "AMAZON_ISSUED",
		validationMethod: validationMethod,
		created:          time.Now().UTC(),
	}
	s.certs[arn] = cert
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"CertificateArn": arn,
	})
}

func (s *Service) describeCertificate(w http.ResponseWriter, params map[string]interface{}) {
	arn := h.GetString(params, "CertificateArn")

	s.mu.RLock()
	cert, exists := s.certs[arn]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "ResourceNotFoundException", "Certificate not found: "+arn, http.StatusBadRequest)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Certificate": certResp(cert),
	})
}

func (s *Service) listCertificates(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var summaries []map[string]interface{}
	for _, cert := range s.certs {
		summaries = append(summaries, map[string]interface{}{
			"CertificateArn": cert.arn,
			"DomainName":     cert.domainName,
			"Status":         cert.status,
			"Type":           cert.certType,
		})
	}
	s.mu.RUnlock()

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i]["DomainName"].(string) < summaries[j]["DomainName"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"CertificateSummaryList": summaries,
	})
}

func (s *Service) deleteCertificate(w http.ResponseWriter, params map[string]interface{}) {
	arn := h.GetString(params, "CertificateArn")

	s.mu.Lock()
	if _, exists := s.certs[arn]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "ResourceNotFoundException", "Certificate not found: "+arn, http.StatusBadRequest)
		return
	}
	delete(s.certs, arn)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func certResp(cert *certificate) map[string]interface{} {
	resp := map[string]interface{}{
		"CertificateArn":          cert.arn,
		"DomainName":              cert.domainName,
		"Status":                  cert.status,
		"Type":                    cert.certType,
		"DomainValidationOptions": []interface{}{},
		"CreatedAt":               float64(cert.created.Unix()),
	}
	if len(cert.subjectAltNames) > 0 {
		resp["SubjectAlternativeNames"] = cert.subjectAltNames
	}
	return resp
}

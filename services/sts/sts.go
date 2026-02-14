// Package sts provides a mock implementation of AWS Security Token Service.
//
// Supported actions:
//   - GetCallerIdentity
//   - AssumeRole
//   - GetSessionToken
package sts

import (
	"encoding/xml"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const defaultAccountID = "123456789012"

// Service implements the STS mock.
type Service struct {
	mu        sync.RWMutex
	accountID string
}

// New creates a new STS mock service.
func New() *Service {
	return &Service{
		accountID: defaultAccountID,
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "sts" }

// Handler returns the HTTP handler for STS requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountID = defaultAccountID
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeSTSError(w, "InvalidParameterValue", "could not parse request", http.StatusBadRequest)
		return
	}

	action := r.FormValue("Action")
	switch action {
	case "GetCallerIdentity":
		s.getCallerIdentity(w, r)
	case "AssumeRole":
		s.assumeRole(w, r)
	case "GetSessionToken":
		s.getSessionToken(w, r)
	default:
		writeSTSError(w, "InvalidAction", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) getCallerIdentity(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	accountID := s.accountID
	s.mu.RUnlock()

	resp := getCallerIdentityResponse{
		Result: getCallerIdentityResult{
			Arn:     fmt.Sprintf("arn:aws:iam::%s:user/moto", accountID),
			UserID:  "AKIAIOSFODNN7EXAMPLE",
			Account: accountID,
		},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) assumeRole(w http.ResponseWriter, r *http.Request) {
	roleArn := r.FormValue("RoleArn")
	sessionName := r.FormValue("RoleSessionName")
	durationStr := r.FormValue("DurationSeconds")

	if roleArn == "" {
		writeSTSError(w, "MalformedInput", "RoleArn is required", http.StatusBadRequest)
		return
	}
	if sessionName == "" {
		sessionName = "session"
	}

	duration := 3600
	if durationStr != "" {
		fmt.Sscanf(durationStr, "%d", &duration)
	}

	s.mu.RLock()
	accountID := s.accountID
	s.mu.RUnlock()

	now := time.Now().UTC()
	expiration := now.Add(time.Duration(duration) * time.Second)

	resp := assumeRoleResponse{
		Result: assumeRoleResult{
			Credentials: stsCredentials{
				AccessKeyID:     "ASIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SessionToken:    "FwoGZXIvYXdzEBY" + newRequestID(),
				Expiration:      expiration.Format(time.RFC3339),
			},
			AssumedRoleUser: assumedRoleUser{
				AssumedRoleID: "AROAIOSFODNN7EXAMPLE:" + sessionName,
				Arn:           fmt.Sprintf("arn:aws:sts::%s:assumed-role/%s/%s", accountID, roleArn, sessionName),
			},
		},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) getSessionToken(w http.ResponseWriter, r *http.Request) {
	durationStr := r.FormValue("DurationSeconds")

	duration := 43200
	if durationStr != "" {
		fmt.Sscanf(durationStr, "%d", &duration)
	}

	now := time.Now().UTC()
	expiration := now.Add(time.Duration(duration) * time.Second)

	resp := getSessionTokenResponse{
		Result: getSessionTokenResult{
			Credentials: stsCredentials{
				AccessKeyID:     "ASIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SessionToken:    "FwoGZXIvYXdzEBY" + newRequestID(),
				Expiration:      expiration.Format(time.RFC3339),
			},
		},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

// XML response types.

type getCallerIdentityResponse struct {
	XMLName   xml.Name                `xml:"GetCallerIdentityResponse"`
	XMLNS     string                  `xml:"xmlns,attr"`
	Result    getCallerIdentityResult `xml:"GetCallerIdentityResult"`
	RequestID string                  `xml:"ResponseMetadata>RequestId"`
}

type getCallerIdentityResult struct {
	Arn     string `xml:"Arn"`
	UserID  string `xml:"UserId"`
	Account string `xml:"Account"`
}

type assumeRoleResponse struct {
	XMLName   xml.Name         `xml:"AssumeRoleResponse"`
	XMLNS     string           `xml:"xmlns,attr"`
	Result    assumeRoleResult `xml:"AssumeRoleResult"`
	RequestID string           `xml:"ResponseMetadata>RequestId"`
}

type assumeRoleResult struct {
	Credentials     stsCredentials  `xml:"Credentials"`
	AssumedRoleUser assumedRoleUser `xml:"AssumedRoleUser"`
}

type stsCredentials struct {
	AccessKeyID     string `xml:"AccessKeyId"`
	SecretAccessKey string `xml:"SecretAccessKey"`
	SessionToken    string `xml:"SessionToken"`
	Expiration      string `xml:"Expiration"`
}

type assumedRoleUser struct {
	AssumedRoleID string `xml:"AssumedRoleId"`
	Arn           string `xml:"Arn"`
}

type getSessionTokenResponse struct {
	XMLName   xml.Name              `xml:"GetSessionTokenResponse"`
	XMLNS     string                `xml:"xmlns,attr"`
	Result    getSessionTokenResult `xml:"GetSessionTokenResult"`
	RequestID string                `xml:"ResponseMetadata>RequestId"`
}

type getSessionTokenResult struct {
	Credentials stsCredentials `xml:"Credentials"`
}

type stsErrorResponse struct {
	XMLName   xml.Name `xml:"ErrorResponse"`
	Error     stsError `xml:"Error"`
	RequestID string   `xml:"RequestId"`
}

type stsError struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func writeXML(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(v)
}

func writeSTSError(w http.ResponseWriter, code, message string, status int) {
	resp := stsErrorResponse{
		Error: stsError{
			Type:    "Sender",
			Code:    code,
			Message: message,
		},
		RequestID: newRequestID(),
	}
	writeXML(w, status, resp)
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

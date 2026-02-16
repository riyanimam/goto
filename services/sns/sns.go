// Package sns provides a mock implementation of AWS Simple Notification Service.
//
// Supported actions:
//   - CreateTopic
//   - DeleteTopic
//   - ListTopics
//   - Subscribe
//   - Unsubscribe
//   - ListSubscriptions
//   - Publish
package sns

import (
	"encoding/xml"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"sync"
)

const defaultAccountID = "123456789012"

// Service implements the SNS mock.
type Service struct {
	mu            sync.RWMutex
	topics        map[string]*topic        // keyed by ARN
	subscriptions map[string]*subscription // keyed by subscription ARN
}

type topic struct {
	arn  string
	name string
}

type subscription struct {
	arn      string
	topicArn string
	protocol string
	endpoint string
}

// New creates a new SNS mock service.
func New() *Service {
	return &Service{
		topics:        make(map[string]*topic),
		subscriptions: make(map[string]*subscription),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "sns" }

// Handler returns the HTTP handler for SNS requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all topics and subscriptions.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.topics = make(map[string]*topic)
	s.subscriptions = make(map[string]*subscription)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeSNSError(w, "InvalidParameterValue", "could not parse request", http.StatusBadRequest)
		return
	}

	action := r.FormValue("Action")
	switch action {
	case "CreateTopic":
		s.createTopic(w, r)
	case "DeleteTopic":
		s.deleteTopic(w, r)
	case "ListTopics":
		s.listTopics(w, r)
	case "Subscribe":
		s.subscribe(w, r)
	case "Unsubscribe":
		s.unsubscribe(w, r)
	case "ListSubscriptions":
		s.listSubscriptions(w, r)
	case "Publish":
		s.publish(w, r)
	default:
		writeSNSError(w, "InvalidAction", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createTopic(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("Name")
	if name == "" {
		writeSNSError(w, "InvalidParameter", "Name is required", http.StatusBadRequest)
		return
	}

	arn := fmt.Sprintf("arn:aws:sns:us-east-1:%s:%s", defaultAccountID, name)

	s.mu.Lock()
	// CreateTopic is idempotent - return existing ARN if topic exists.
	if t, exists := s.topics[arn]; exists {
		s.mu.Unlock()
		resp := createTopicResponse{
			Result:    createTopicResult{TopicArn: t.arn},
			RequestID: newRequestID(),
		}
		writeXML(w, http.StatusOK, resp)
		return
	}

	s.topics[arn] = &topic{
		arn:  arn,
		name: name,
	}
	s.mu.Unlock()

	resp := createTopicResponse{
		Result:    createTopicResult{TopicArn: arn},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) deleteTopic(w http.ResponseWriter, r *http.Request) {
	arn := r.FormValue("TopicArn")

	s.mu.Lock()
	delete(s.topics, arn)
	// Remove subscriptions for this topic.
	for subArn, sub := range s.subscriptions {
		if sub.topicArn == arn {
			delete(s.subscriptions, subArn)
		}
	}
	s.mu.Unlock()

	resp := deleteTopicResponse{
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) listTopics(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var members []topicMember
	for _, t := range s.topics {
		members = append(members, topicMember{TopicArn: t.arn})
	}
	s.mu.RUnlock()

	sort.Slice(members, func(i, j int) bool {
		return members[i].TopicArn < members[j].TopicArn
	})

	resp := listTopicsResponse{
		Result:    listTopicsResult{Topics: members},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) subscribe(w http.ResponseWriter, r *http.Request) {
	topicArn := r.FormValue("TopicArn")
	protocol := r.FormValue("Protocol")
	endpoint := r.FormValue("Endpoint")

	s.mu.Lock()
	if _, exists := s.topics[topicArn]; !exists {
		s.mu.Unlock()
		writeSNSError(w, "NotFound", "Topic does not exist", http.StatusNotFound)
		return
	}

	subArn := fmt.Sprintf("%s:%s", topicArn, newRequestID())
	sub := &subscription{
		arn:      subArn,
		topicArn: topicArn,
		protocol: protocol,
		endpoint: endpoint,
	}
	s.subscriptions[subArn] = sub
	s.mu.Unlock()

	resp := subscribeResponse{
		Result:    subscribeResult{SubscriptionArn: subArn},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) unsubscribe(w http.ResponseWriter, r *http.Request) {
	subArn := r.FormValue("SubscriptionArn")

	s.mu.Lock()
	delete(s.subscriptions, subArn)
	s.mu.Unlock()

	resp := unsubscribeResponse{
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) listSubscriptions(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	var members []subscriptionMember
	for _, sub := range s.subscriptions {
		members = append(members, subscriptionMember{
			SubscriptionArn: sub.arn,
			TopicArn:        sub.topicArn,
			Protocol:        sub.protocol,
			Endpoint:        sub.endpoint,
			Owner:           defaultAccountID,
		})
	}
	s.mu.RUnlock()

	sort.Slice(members, func(i, j int) bool {
		return members[i].SubscriptionArn < members[j].SubscriptionArn
	})

	resp := listSubscriptionsResponse{
		Result:    listSubscriptionsResult{Subscriptions: members},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) publish(w http.ResponseWriter, r *http.Request) {
	topicArn := r.FormValue("TopicArn")
	_ = r.FormValue("Message") // Accept the message but we don't need to store it.

	s.mu.RLock()
	_, exists := s.topics[topicArn]
	s.mu.RUnlock()

	if !exists {
		writeSNSError(w, "NotFound", "Topic does not exist", http.StatusNotFound)
		return
	}

	msgID := newRequestID()
	resp := publishResponse{
		Result:    publishResult{MessageId: msgID},
		RequestID: newRequestID(),
	}
	writeXML(w, http.StatusOK, resp)
}

// XML response types.

type createTopicResponse struct {
	XMLName   xml.Name          `xml:"CreateTopicResponse"`
	XMLNS     string            `xml:"xmlns,attr"`
	Result    createTopicResult `xml:"CreateTopicResult"`
	RequestID string            `xml:"ResponseMetadata>RequestId"`
}

type createTopicResult struct {
	TopicArn string `xml:"TopicArn"`
}

type deleteTopicResponse struct {
	XMLName   xml.Name `xml:"DeleteTopicResponse"`
	XMLNS     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type listTopicsResponse struct {
	XMLName   xml.Name         `xml:"ListTopicsResponse"`
	XMLNS     string           `xml:"xmlns,attr"`
	Result    listTopicsResult `xml:"ListTopicsResult"`
	RequestID string           `xml:"ResponseMetadata>RequestId"`
}

type listTopicsResult struct {
	Topics []topicMember `xml:"Topics>member"`
}

type topicMember struct {
	TopicArn string `xml:"TopicArn"`
}

type subscribeResponse struct {
	XMLName   xml.Name        `xml:"SubscribeResponse"`
	XMLNS     string          `xml:"xmlns,attr"`
	Result    subscribeResult `xml:"SubscribeResult"`
	RequestID string          `xml:"ResponseMetadata>RequestId"`
}

type subscribeResult struct {
	SubscriptionArn string `xml:"SubscriptionArn"`
}

type unsubscribeResponse struct {
	XMLName   xml.Name `xml:"UnsubscribeResponse"`
	XMLNS     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"ResponseMetadata>RequestId"`
}

type listSubscriptionsResponse struct {
	XMLName   xml.Name                `xml:"ListSubscriptionsResponse"`
	XMLNS     string                  `xml:"xmlns,attr"`
	Result    listSubscriptionsResult `xml:"ListSubscriptionsResult"`
	RequestID string                  `xml:"ResponseMetadata>RequestId"`
}

type listSubscriptionsResult struct {
	Subscriptions []subscriptionMember `xml:"Subscriptions>member"`
}

type subscriptionMember struct {
	SubscriptionArn string `xml:"SubscriptionArn"`
	TopicArn        string `xml:"TopicArn"`
	Protocol        string `xml:"Protocol"`
	Endpoint        string `xml:"Endpoint"`
	Owner           string `xml:"Owner"`
}

type publishResponse struct {
	XMLName   xml.Name      `xml:"PublishResponse"`
	XMLNS     string        `xml:"xmlns,attr"`
	Result    publishResult `xml:"PublishResult"`
	RequestID string        `xml:"ResponseMetadata>RequestId"`
}

type publishResult struct {
	MessageId string `xml:"MessageId"`
}

type snsErrorResponse struct {
	XMLName   xml.Name `xml:"ErrorResponse"`
	Error     snsError `xml:"Error"`
	RequestID string   `xml:"RequestId"`
}

type snsError struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

// Helper functions.

func writeXML(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(v)
}

func writeSNSError(w http.ResponseWriter, code, message string, status int) {
	resp := snsErrorResponse{
		Error: snsError{
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

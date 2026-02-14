// Package sqs provides a mock implementation of AWS Simple Queue Service.
//
// Supported actions:
//   - CreateQueue
//   - DeleteQueue
//   - ListQueues
//   - GetQueueUrl
//   - GetQueueAttributes
//   - SendMessage
//   - ReceiveMessage
//   - DeleteMessage
//   - PurgeQueue
//   - SetQueueAttributes
package sqs

import (
"crypto/md5"
"encoding/hex"
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

// Service implements the SQS mock.
type Service struct {
mu     sync.RWMutex
queues map[string]*queue // keyed by queue URL
}

type queue struct {
name       string
url        string
arn        string
attributes map[string]string
messages   []*message
mu         sync.Mutex
created    time.Time
}

type message struct {
id            string
body          string
md5           string
receiptHandle string
sentTimestamp string
visible       bool
}

// New creates a new SQS mock service.
func New() *Service {
return &Service{
queues: make(map[string]*queue),
}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "sqs" }

// Handler returns the HTTP handler for SQS requests.
func (s *Service) Handler() http.Handler {
return http.HandlerFunc(s.handle)
}

// Reset clears all queues and messages.
func (s *Service) Reset() {
s.mu.Lock()
defer s.mu.Unlock()
s.queues = make(map[string]*queue)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
// AWS SDK v2 SQS uses the JSON protocol with X-Amz-Target header.
target := r.Header.Get("X-Amz-Target")

// Read request body.
bodyBytes, err := io.ReadAll(r.Body)
if err != nil {
writeJSONError(w, "InternalError", "could not read request body", http.StatusInternalServerError)
return
}

// Parse the JSON body into a generic map.
var params map[string]interface{}
if len(bodyBytes) > 0 {
if err := json.Unmarshal(bodyBytes, &params); err != nil {
writeJSONError(w, "InvalidParameterValue", "could not parse request body", http.StatusBadRequest)
return
}
}
if params == nil {
params = make(map[string]interface{})
}

// Route based on X-Amz-Target header.
action := ""
if target != "" {
parts := strings.SplitN(target, ".", 2)
if len(parts) == 2 {
action = parts[1]
}
}

switch action {
case "CreateQueue":
s.createQueue(w, params)
case "DeleteQueue":
s.deleteQueue(w, params)
case "ListQueues":
s.listQueues(w, params)
case "GetQueueUrl":
s.getQueueURL(w, params)
case "GetQueueAttributes":
s.getQueueAttributes(w, params)
case "SetQueueAttributes":
s.setQueueAttributes(w, params)
case "SendMessage":
s.sendMessage(w, params)
case "ReceiveMessage":
s.receiveMessage(w, params)
case "DeleteMessage":
s.deleteMessage(w, params)
case "PurgeQueue":
s.purgeQueue(w, params)
default:
writeJSONError(w, "InvalidAction", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
}
}

func (s *Service) createQueue(w http.ResponseWriter, params map[string]interface{}) {
name := getString(params, "QueueName")
if name == "" {
writeJSONError(w, "MissingParameter", "QueueName is required", http.StatusBadRequest)
return
}

queueURL := fmt.Sprintf("http://localhost/%s/%s", defaultAccountID, name)

s.mu.Lock()
// Check if queue with same name already exists.
for _, q := range s.queues {
if q.name == name {
s.mu.Unlock()
writeJSON(w, http.StatusOK, map[string]interface{}{
"QueueUrl": q.url,
})
return
}
}

q := &queue{
name:    name,
url:     queueURL,
arn:     fmt.Sprintf("arn:aws:sqs:us-east-1:%s:%s", defaultAccountID, name),
created: time.Now().UTC(),
attributes: map[string]string{
"QueueArn":                                fmt.Sprintf("arn:aws:sqs:us-east-1:%s:%s", defaultAccountID, name),
"ApproximateNumberOfMessages":              "0",
"ApproximateNumberOfMessagesDelayed":       "0",
"ApproximateNumberOfMessagesNotVisible":    "0",
"CreatedTimestamp":                         fmt.Sprintf("%d", time.Now().Unix()),
"LastModifiedTimestamp":                    fmt.Sprintf("%d", time.Now().Unix()),
"VisibilityTimeout":                        "30",
"MaximumMessageSize":                       "262144",
"MessageRetentionPeriod":                   "345600",
"DelaySeconds":                             "0",
"ReceiveMessageWaitTimeSeconds":            "0",
},
}
s.queues[queueURL] = q
s.mu.Unlock()

// Apply any attribute overrides from the request.
if attrs, ok := params["Attributes"].(map[string]interface{}); ok {
q.mu.Lock()
for k, v := range attrs {
if sv, ok := v.(string); ok {
q.attributes[k] = sv
}
}
q.mu.Unlock()
}

writeJSON(w, http.StatusOK, map[string]interface{}{
"QueueUrl": queueURL,
})
}

func (s *Service) deleteQueue(w http.ResponseWriter, params map[string]interface{}) {
queueURL := getString(params, "QueueUrl")

s.mu.Lock()
delete(s.queues, queueURL)
s.mu.Unlock()

writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listQueues(w http.ResponseWriter, params map[string]interface{}) {
prefix := getString(params, "QueueNamePrefix")

s.mu.RLock()
var urls []string
for _, q := range s.queues {
if prefix == "" || strings.HasPrefix(q.name, prefix) {
urls = append(urls, q.url)
}
}
s.mu.RUnlock()

sort.Strings(urls)

writeJSON(w, http.StatusOK, map[string]interface{}{
"QueueUrls": urls,
})
}

func (s *Service) getQueueURL(w http.ResponseWriter, params map[string]interface{}) {
name := getString(params, "QueueName")

s.mu.RLock()
for _, q := range s.queues {
if q.name == name {
s.mu.RUnlock()
writeJSON(w, http.StatusOK, map[string]interface{}{
"QueueUrl": q.url,
})
return
}
}
s.mu.RUnlock()

writeJSONError(w, "AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist.", http.StatusBadRequest)
}

func (s *Service) getQueueAttributes(w http.ResponseWriter, params map[string]interface{}) {
queueURL := getString(params, "QueueUrl")

s.mu.RLock()
q, exists := s.queues[queueURL]
s.mu.RUnlock()

if !exists {
writeJSONError(w, "AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist.", http.StatusBadRequest)
return
}

// Collect requested attribute names.
requestAll := false
requestedNames := make(map[string]bool)
if attrNames, ok := params["AttributeNames"].([]interface{}); ok {
for _, n := range attrNames {
if ns, ok := n.(string); ok {
if ns == "All" {
requestAll = true
break
}
requestedNames[ns] = true
}
}
} else {
requestAll = true
}

q.mu.Lock()
q.attributes["ApproximateNumberOfMessages"] = fmt.Sprintf("%d", countVisible(q))
attrs := make(map[string]string)
for k, v := range q.attributes {
if requestAll || requestedNames[k] {
attrs[k] = v
}
}
q.mu.Unlock()

writeJSON(w, http.StatusOK, map[string]interface{}{
"Attributes": attrs,
})
}

func (s *Service) setQueueAttributes(w http.ResponseWriter, params map[string]interface{}) {
queueURL := getString(params, "QueueUrl")

s.mu.RLock()
q, exists := s.queues[queueURL]
s.mu.RUnlock()

if !exists {
writeJSONError(w, "AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist.", http.StatusBadRequest)
return
}

if attrs, ok := params["Attributes"].(map[string]interface{}); ok {
q.mu.Lock()
for k, v := range attrs {
if sv, ok := v.(string); ok {
q.attributes[k] = sv
}
}
q.mu.Unlock()
}

writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) sendMessage(w http.ResponseWriter, params map[string]interface{}) {
queueURL := getString(params, "QueueUrl")
body := getString(params, "MessageBody")

s.mu.RLock()
q, exists := s.queues[queueURL]
s.mu.RUnlock()

if !exists {
writeJSONError(w, "AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist.", http.StatusBadRequest)
return
}

hash := md5.Sum([]byte(body))
md5Hex := hex.EncodeToString(hash[:])

msg := &message{
id:            newMessageID(),
body:          body,
md5:           md5Hex,
receiptHandle: newMessageID() + newMessageID(),
sentTimestamp: fmt.Sprintf("%d", time.Now().UnixMilli()),
visible:       true,
}

q.mu.Lock()
q.messages = append(q.messages, msg)
q.mu.Unlock()

writeJSON(w, http.StatusOK, map[string]interface{}{
"MessageId":        msg.id,
"MD5OfMessageBody": md5Hex,
})
}

func (s *Service) receiveMessage(w http.ResponseWriter, params map[string]interface{}) {
queueURL := getString(params, "QueueUrl")
maxMessages := getInt(params, "MaxNumberOfMessages", 1)
if maxMessages > 10 {
maxMessages = 10
}

s.mu.RLock()
q, exists := s.queues[queueURL]
s.mu.RUnlock()

if !exists {
writeJSONError(w, "AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist.", http.StatusBadRequest)
return
}

q.mu.Lock()
var received []map[string]interface{}
count := 0
for _, msg := range q.messages {
if count >= maxMessages {
break
}
if msg.visible {
msg.visible = false
received = append(received, map[string]interface{}{
"MessageId":     msg.id,
"ReceiptHandle": msg.receiptHandle,
"Body":          msg.body,
"MD5OfBody":     msg.md5,
})
count++
}
}
q.mu.Unlock()

writeJSON(w, http.StatusOK, map[string]interface{}{
"Messages": received,
})
}

func (s *Service) deleteMessage(w http.ResponseWriter, params map[string]interface{}) {
queueURL := getString(params, "QueueUrl")
receiptHandle := getString(params, "ReceiptHandle")

s.mu.RLock()
q, exists := s.queues[queueURL]
s.mu.RUnlock()

if !exists {
writeJSONError(w, "AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist.", http.StatusBadRequest)
return
}

q.mu.Lock()
for i, msg := range q.messages {
if msg.receiptHandle == receiptHandle {
q.messages = append(q.messages[:i], q.messages[i+1:]...)
break
}
}
q.mu.Unlock()

writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) purgeQueue(w http.ResponseWriter, params map[string]interface{}) {
queueURL := getString(params, "QueueUrl")

s.mu.RLock()
q, exists := s.queues[queueURL]
s.mu.RUnlock()

if !exists {
writeJSONError(w, "AWS.SimpleQueueService.NonExistentQueue", "The specified queue does not exist.", http.StatusBadRequest)
return
}

q.mu.Lock()
q.messages = nil
q.mu.Unlock()

writeJSON(w, http.StatusOK, map[string]interface{}{})
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

func getInt(params map[string]interface{}, key string, defaultVal int) int {
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

func countVisible(q *queue) int {
count := 0
for _, msg := range q.messages {
if msg.visible {
count++
}
}
return count
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
w.Header().Set("Content-Type", "application/x-amz-json-1.0")
w.WriteHeader(status)
json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, code, message string, status int) {
w.Header().Set("Content-Type", "application/x-amz-json-1.0")
w.WriteHeader(status)
json.NewEncoder(w).Encode(map[string]interface{}{
"__type":  code,
"message": message,
})
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

func newMessageID() string {
return newRequestID()
}

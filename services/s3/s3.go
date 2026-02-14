// Package s3 provides a mock implementation of AWS Simple Storage Service.
//
// Supported operations:
//   - CreateBucket
//   - DeleteBucket
//   - ListBuckets
//   - HeadBucket
//   - PutObject
//   - GetObject
//   - HeadObject
//   - DeleteObject
//   - ListObjectsV2
//   - CopyObject
package s3

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Service implements the S3 mock.
type Service struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
}

type bucket struct {
	name      string
	region    string
	created   time.Time
	objects   map[string]*object
	objectsMu sync.RWMutex
}

type object struct {
	key          string
	data         []byte
	contentType  string
	etag         string
	lastModified time.Time
	metadata     map[string]string
}

// New creates a new S3 mock service.
func New() *Service {
	return &Service{
		buckets: make(map[string]*bucket),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "s3" }

// Handler returns the HTTP handler for S3 requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all buckets and objects.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buckets = make(map[string]*bucket)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	// Parse bucket and key from the path.
	// Path format: /bucket or /bucket/key/parts
	path := strings.TrimPrefix(r.URL.Path, "/")
	bucketName, key := parsePath(path)

	switch {
	case bucketName == "" && r.Method == http.MethodGet:
		s.listBuckets(w, r)
	case key == "" && r.Method == http.MethodPut:
		s.createBucket(w, r, bucketName)
	case key == "" && r.Method == http.MethodDelete:
		s.deleteBucket(w, r, bucketName)
	case key == "" && r.Method == http.MethodHead:
		s.headBucket(w, r, bucketName)
	case key == "" && r.Method == http.MethodGet:
		s.listObjects(w, r, bucketName)
	case key != "" && r.Method == http.MethodPut:
		if r.Header.Get("X-Amz-Copy-Source") != "" {
			s.copyObject(w, r, bucketName, key)
		} else {
			s.putObject(w, r, bucketName, key)
		}
	case key != "" && r.Method == http.MethodGet:
		s.getObject(w, r, bucketName, key)
	case key != "" && r.Method == http.MethodHead:
		s.headObject(w, r, bucketName, key)
	case key != "" && r.Method == http.MethodDelete:
		s.deleteObject(w, r, bucketName, key)
	default:
		writeS3Error(w, "MethodNotAllowed", "The specified method is not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Service) listBuckets(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var bucketList []listBucketEntry
	for _, b := range s.buckets {
		bucketList = append(bucketList, listBucketEntry{
			Name:         b.name,
			CreationDate: b.created.Format(time.RFC3339),
		})
	}
	sort.Slice(bucketList, func(i, j int) bool {
		return bucketList[i].Name < bucketList[j].Name
	})

	resp := listAllMyBucketsResult{
		Owner: owner{
			ID:          "75aa57f09aa0c8caeab4f8c24e99d10f8e7faeebf76c078efc7c6caea54ba06a",
			DisplayName: "webfile",
		},
		Buckets: bucketList,
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) createBucket(w http.ResponseWriter, _ *http.Request, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.buckets[name]; exists {
		writeS3Error(w, "BucketAlreadyOwnedByYou", "Your previous request to create the named bucket succeeded and you already own it.", http.StatusConflict)
		return
	}

	s.buckets[name] = &bucket{
		name:    name,
		region:  "us-east-1",
		created: time.Now().UTC(),
		objects: make(map[string]*object),
	}

	w.Header().Set("Location", "/"+name)
	w.WriteHeader(http.StatusOK)
}

func (s *Service) deleteBucket(w http.ResponseWriter, _ *http.Request, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, exists := s.buckets[name]
	if !exists {
		writeS3Error(w, "NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)
		return
	}

	b.objectsMu.RLock()
	count := len(b.objects)
	b.objectsMu.RUnlock()

	if count > 0 {
		writeS3Error(w, "BucketNotEmpty", "The bucket you tried to delete is not empty", http.StatusConflict)
		return
	}

	delete(s.buckets, name)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) headBucket(w http.ResponseWriter, _ *http.Request, name string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.buckets[name]; !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("X-Amz-Bucket-Region", "us-east-1")
	w.WriteHeader(http.StatusOK)
}

func (s *Service) listObjects(w http.ResponseWriter, r *http.Request, bucketName string) {
	s.mu.RLock()
	b, exists := s.buckets[bucketName]
	s.mu.RUnlock()

	if !exists {
		writeS3Error(w, "NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)
		return
	}

	prefix := r.URL.Query().Get("prefix")
	delimiter := r.URL.Query().Get("delimiter")
	maxKeysStr := r.URL.Query().Get("max-keys")
	maxKeys := 1000
	if maxKeysStr != "" {
		fmt.Sscanf(maxKeysStr, "%d", &maxKeys)
	}

	b.objectsMu.RLock()
	var contents []listObjectEntry
	commonPrefixes := make(map[string]bool)
	for _, obj := range b.objects {
		if prefix != "" && !strings.HasPrefix(obj.key, prefix) {
			continue
		}

		if delimiter != "" {
			rest := strings.TrimPrefix(obj.key, prefix)
			idx := strings.Index(rest, delimiter)
			if idx >= 0 {
				commonPrefixes[prefix+rest[:idx+len(delimiter)]] = true
				continue
			}
		}

		contents = append(contents, listObjectEntry{
			Key:          obj.key,
			LastModified: obj.lastModified.Format(time.RFC3339),
			ETag:         obj.etag,
			Size:         len(obj.data),
			StorageClass: "STANDARD",
		})
	}
	b.objectsMu.RUnlock()

	sort.Slice(contents, func(i, j int) bool {
		return contents[i].Key < contents[j].Key
	})

	if len(contents) > maxKeys {
		contents = contents[:maxKeys]
	}

	var prefixEntries []commonPrefix
	for p := range commonPrefixes {
		prefixEntries = append(prefixEntries, commonPrefix{Prefix: p})
	}
	sort.Slice(prefixEntries, func(i, j int) bool {
		return prefixEntries[i].Prefix < prefixEntries[j].Prefix
	})

	resp := listBucketResult{
		XMLNS:          "http://s3.amazonaws.com/doc/2006-03-01/",
		Name:           bucketName,
		Prefix:         prefix,
		Delimiter:      delimiter,
		MaxKeys:        maxKeys,
		KeyCount:       len(contents),
		IsTruncated:    false,
		Contents:       contents,
		CommonPrefixes: prefixEntries,
	}
	writeXML(w, http.StatusOK, resp)
}

func (s *Service) putObject(w http.ResponseWriter, r *http.Request, bucketName, key string) {
	s.mu.RLock()
	b, exists := s.buckets[bucketName]
	s.mu.RUnlock()

	if !exists {
		writeS3Error(w, "NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeS3Error(w, "InternalError", "could not read request body", http.StatusInternalServerError)
		return
	}

	hash := md5.Sum(data)
	etag := `"` + hex.EncodeToString(hash[:]) + `"`

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "binary/octet-stream"
	}

	// Collect user metadata (X-Amz-Meta-* headers).
	metadata := make(map[string]string)
	for name, values := range r.Header {
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "x-amz-meta-") {
			metaKey := strings.TrimPrefix(lower, "x-amz-meta-")
			metadata[metaKey] = values[0]
		}
	}

	obj := &object{
		key:          key,
		data:         data,
		contentType:  contentType,
		etag:         etag,
		lastModified: time.Now().UTC(),
		metadata:     metadata,
	}

	b.objectsMu.Lock()
	b.objects[key] = obj
	b.objectsMu.Unlock()

	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
}

func (s *Service) getObject(w http.ResponseWriter, _ *http.Request, bucketName, key string) {
	s.mu.RLock()
	b, exists := s.buckets[bucketName]
	s.mu.RUnlock()

	if !exists {
		writeS3Error(w, "NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)
		return
	}

	b.objectsMu.RLock()
	obj, exists := b.objects[key]
	b.objectsMu.RUnlock()

	if !exists {
		writeS3Error(w, "NoSuchKey", "The specified key does not exist.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", obj.contentType)
	w.Header().Set("ETag", obj.etag)
	w.Header().Set("Last-Modified", obj.lastModified.Format(http.TimeFormat))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(obj.data)))
	for k, v := range obj.metadata {
		w.Header().Set("X-Amz-Meta-"+k, v)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(obj.data)
}

func (s *Service) headObject(w http.ResponseWriter, _ *http.Request, bucketName, key string) {
	s.mu.RLock()
	b, exists := s.buckets[bucketName]
	s.mu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	b.objectsMu.RLock()
	obj, exists := b.objects[key]
	b.objectsMu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", obj.contentType)
	w.Header().Set("ETag", obj.etag)
	w.Header().Set("Last-Modified", obj.lastModified.Format(http.TimeFormat))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(obj.data)))
	for k, v := range obj.metadata {
		w.Header().Set("X-Amz-Meta-"+k, v)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Service) deleteObject(w http.ResponseWriter, _ *http.Request, bucketName, key string) {
	s.mu.RLock()
	b, exists := s.buckets[bucketName]
	s.mu.RUnlock()

	if !exists {
		writeS3Error(w, "NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)
		return
	}

	b.objectsMu.Lock()
	delete(b.objects, key)
	b.objectsMu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) copyObject(w http.ResponseWriter, r *http.Request, destBucket, destKey string) {
	source := r.Header.Get("X-Amz-Copy-Source")
	source = strings.TrimPrefix(source, "/")
	parts := strings.SplitN(source, "/", 2)
	if len(parts) != 2 {
		writeS3Error(w, "InvalidArgument", "invalid copy source", http.StatusBadRequest)
		return
	}
	srcBucket, srcKey := parts[0], parts[1]

	s.mu.RLock()
	sb, exists := s.buckets[srcBucket]
	if !exists {
		s.mu.RUnlock()
		writeS3Error(w, "NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)
		return
	}
	db, exists := s.buckets[destBucket]
	if !exists {
		s.mu.RUnlock()
		writeS3Error(w, "NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)
		return
	}
	s.mu.RUnlock()

	sb.objectsMu.RLock()
	srcObj, exists := sb.objects[srcKey]
	if !exists {
		sb.objectsMu.RUnlock()
		writeS3Error(w, "NoSuchKey", "The specified key does not exist.", http.StatusNotFound)
		return
	}
	// Copy data while holding the lock.
	dataCopy := make([]byte, len(srcObj.data))
	copy(dataCopy, srcObj.data)
	contentType := srcObj.contentType
	metadata := make(map[string]string)
	for k, v := range srcObj.metadata {
		metadata[k] = v
	}
	sb.objectsMu.RUnlock()

	hash := md5.Sum(dataCopy)
	etag := `"` + hex.EncodeToString(hash[:]) + `"`
	now := time.Now().UTC()

	newObj := &object{
		key:          destKey,
		data:         dataCopy,
		contentType:  contentType,
		etag:         etag,
		lastModified: now,
		metadata:     metadata,
	}

	db.objectsMu.Lock()
	db.objects[destKey] = newObj
	db.objectsMu.Unlock()

	resp := copyObjectResult{
		ETag:         etag,
		LastModified: now.Format(time.RFC3339),
	}
	writeXML(w, http.StatusOK, resp)
}

// XML types.

type listAllMyBucketsResult struct {
	XMLName xml.Name          `xml:"ListAllMyBucketsResult"`
	XMLNS   string            `xml:"xmlns,attr"`
	Owner   owner             `xml:"Owner"`
	Buckets []listBucketEntry `xml:"Buckets>Bucket"`
}

type owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type listBucketEntry struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

type listBucketResult struct {
	XMLName        xml.Name          `xml:"ListBucketResult"`
	XMLNS          string            `xml:"xmlns,attr"`
	Name           string            `xml:"Name"`
	Prefix         string            `xml:"Prefix"`
	Delimiter      string            `xml:"Delimiter,omitempty"`
	MaxKeys        int               `xml:"MaxKeys"`
	KeyCount       int               `xml:"KeyCount"`
	IsTruncated    bool              `xml:"IsTruncated"`
	Contents       []listObjectEntry `xml:"Contents"`
	CommonPrefixes []commonPrefix    `xml:"CommonPrefixes,omitempty"`
}

type listObjectEntry struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int    `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

type commonPrefix struct {
	Prefix string `xml:"Prefix"`
}

type copyObjectResult struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	ETag         string   `xml:"ETag"`
	LastModified string   `xml:"LastModified"`
}

type s3ErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource"`
	RequestID string   `xml:"RequestId"`
}

// Helper functions.

func parsePath(path string) (bucket, key string) {
	if path == "" {
		return "", ""
	}
	idx := strings.IndexByte(path, '/')
	if idx < 0 {
		return path, ""
	}
	return path[:idx], path[idx+1:]
}

func writeXML(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(v)
}

func writeS3Error(w http.ResponseWriter, code, message string, status int) {
	resp := s3ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: "mock-request-id",
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	xml.NewEncoder(&buf).Encode(resp)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

// Package ecr provides a mock implementation of AWS Elastic Container Registry.
//
// Supported actions:
//   - CreateRepository
//   - DeleteRepository
//   - DescribeRepositories
//   - ListImages
//   - PutImage
//   - BatchGetImage
//   - GetAuthorizationToken
package ecr

import (
	"encoding/base64"
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

// Service implements the ECR mock.
type Service struct {
	mu    sync.RWMutex
	repos map[string]*repository // keyed by repo name
}

type repository struct {
	name       string
	arn        string
	uri        string
	registryID string
	created    time.Time
	images     []*image
}

type image struct {
	tag      string
	digest   string
	manifest string
	pushed   time.Time
}

// New creates a new ECR mock service.
func New() *Service {
	return &Service{
		repos: make(map[string]*repository),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "api.ecr" }

// Handler returns the HTTP handler for ECR requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all repositories and images.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repos = make(map[string]*repository)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, "ServerException", "could not read request body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &params); err != nil {
			writeJSONError(w, "InvalidParameterException", "could not parse request body", http.StatusBadRequest)
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
	case "CreateRepository":
		s.createRepository(w, params)
	case "DeleteRepository":
		s.deleteRepository(w, params)
	case "DescribeRepositories":
		s.describeRepositories(w, params)
	case "ListImages":
		s.listImages(w, params)
	case "PutImage":
		s.putImage(w, params)
	case "BatchGetImage":
		s.batchGetImage(w, params)
	case "GetAuthorizationToken":
		s.getAuthorizationToken(w, params)
	default:
		writeJSONError(w, "UnsupportedCommandException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createRepository(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "repositoryName")
	if name == "" {
		writeJSONError(w, "InvalidParameterException", "repositoryName is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.repos[name]; exists {
		s.mu.Unlock()
		writeJSONError(w, "RepositoryAlreadyExistsException", "The repository with name '"+name+"' already exists", http.StatusBadRequest)
		return
	}

	repo := &repository{
		name:       name,
		arn:        fmt.Sprintf("arn:aws:ecr:us-east-1:%s:repository/%s", defaultAccountID, name),
		uri:        fmt.Sprintf("%s.dkr.ecr.us-east-1.amazonaws.com/%s", defaultAccountID, name),
		registryID: defaultAccountID,
		created:    time.Now().UTC(),
	}
	s.repos[name] = repo
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"repository": repoResponse(repo),
	})
}

func (s *Service) deleteRepository(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "repositoryName")

	s.mu.Lock()
	repo, exists := s.repos[name]
	if !exists {
		s.mu.Unlock()
		writeJSONError(w, "RepositoryNotFoundException", "The repository with name '"+name+"' does not exist", http.StatusBadRequest)
		return
	}
	delete(s.repos, name)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"repository": repoResponse(repo),
	})
}

func (s *Service) describeRepositories(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var repos []map[string]interface{}
	for _, repo := range s.repos {
		repos = append(repos, repoResponse(repo))
	}
	s.mu.RUnlock()

	sort.Slice(repos, func(i, j int) bool {
		return repos[i]["repositoryName"].(string) < repos[j]["repositoryName"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"repositories": repos,
	})
}

func (s *Service) listImages(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "repositoryName")

	s.mu.RLock()
	repo, exists := s.repos[name]
	if !exists {
		s.mu.RUnlock()
		writeJSONError(w, "RepositoryNotFoundException", "The repository with name '"+name+"' does not exist", http.StatusBadRequest)
		return
	}

	var imageIDs []map[string]interface{}
	for _, img := range repo.images {
		id := map[string]interface{}{
			"imageDigest": img.digest,
		}
		if img.tag != "" {
			id["imageTag"] = img.tag
		}
		imageIDs = append(imageIDs, id)
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"imageIds": imageIDs,
	})
}

func (s *Service) putImage(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "repositoryName")
	tag := getString(params, "imageTag")
	manifest := getString(params, "imageManifest")

	s.mu.Lock()
	repo, exists := s.repos[name]
	if !exists {
		s.mu.Unlock()
		writeJSONError(w, "RepositoryNotFoundException", "The repository with name '"+name+"' does not exist", http.StatusBadRequest)
		return
	}

	digest := fmt.Sprintf("sha256:%s", randomHex(64))
	img := &image{
		tag:      tag,
		digest:   digest,
		manifest: manifest,
		pushed:   time.Now().UTC(),
	}
	repo.images = append(repo.images, img)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"image": map[string]interface{}{
			"registryId":     defaultAccountID,
			"repositoryName": name,
			"imageId": map[string]interface{}{
				"imageDigest": digest,
				"imageTag":    tag,
			},
			"imageManifest": manifest,
		},
	})
}

func (s *Service) batchGetImage(w http.ResponseWriter, params map[string]interface{}) {
	name := getString(params, "repositoryName")

	s.mu.RLock()
	repo, exists := s.repos[name]
	if !exists {
		s.mu.RUnlock()
		writeJSONError(w, "RepositoryNotFoundException", "The repository with name '"+name+"' does not exist", http.StatusBadRequest)
		return
	}

	requestedIDs, _ := params["imageIds"].([]interface{})
	var images []map[string]interface{}
	var failures []map[string]interface{}

	for _, reqID := range requestedIDs {
		idMap, ok := reqID.(map[string]interface{})
		if !ok {
			continue
		}
		reqTag := getString(idMap, "imageTag")
		reqDigest := getString(idMap, "imageDigest")

		found := false
		for _, img := range repo.images {
			if (reqTag != "" && img.tag == reqTag) || (reqDigest != "" && img.digest == reqDigest) {
				images = append(images, map[string]interface{}{
					"registryId":     defaultAccountID,
					"repositoryName": name,
					"imageId": map[string]interface{}{
						"imageDigest": img.digest,
						"imageTag":    img.tag,
					},
					"imageManifest": img.manifest,
				})
				found = true
				break
			}
		}
		if !found {
			failures = append(failures, map[string]interface{}{
				"imageId":     idMap,
				"failureCode": "ImageNotFound",
				"failureReason": "Requested image not found",
			})
		}
	}
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"images":   images,
		"failures": failures,
	})
}

func (s *Service) getAuthorizationToken(w http.ResponseWriter, _ map[string]interface{}) {
	token := base64.StdEncoding.EncodeToString([]byte("AWS:" + newRequestID()))
	expiry := time.Now().UTC().Add(12 * time.Hour)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"authorizationData": []map[string]interface{}{
			{
				"authorizationToken": token,
				"expiresAt":          float64(expiry.Unix()),
				"proxyEndpoint":      fmt.Sprintf("https://%s.dkr.ecr.us-east-1.amazonaws.com", defaultAccountID),
			},
		},
	})
}

func repoResponse(repo *repository) map[string]interface{} {
	return map[string]interface{}{
		"repositoryName": repo.name,
		"repositoryArn":  repo.arn,
		"repositoryUri":  repo.uri,
		"registryId":     repo.registryID,
		"createdAt":      float64(repo.created.Unix()),
	}
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

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
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

func randomHex(n int) string {
	const chars = "abcdef0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

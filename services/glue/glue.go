// Package glue provides a mock implementation of AWS Glue.
//
// Supported actions:
//   - CreateDatabase
//   - GetDatabase
//   - DeleteDatabase
//   - GetDatabases
//   - CreateTable
//   - GetTable
//   - DeleteTable
//   - GetTables
//   - CreateCrawler
//   - GetCrawler
//   - DeleteCrawler
//   - StartCrawler
//   - ListCrawlers
package glue

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

// Service implements the Glue mock.
type Service struct {
	mu        sync.RWMutex
	databases map[string]*glueDatabase
	crawlers  map[string]*glueCrawler
}

type glueDatabase struct {
	name        string
	description string
	locationURI string
	created     time.Time
	tables      map[string]*glueTable
}

type glueTable struct {
	name        string
	dbName      string
	description string
	tableType   string
	location    string
	columns     []column
	created     time.Time
	modified    time.Time
}

type column struct {
	name    string
	colType string
	comment string
}

type glueCrawler struct {
	name    string
	role    string
	dbName  string
	targets string
	state   string
	created time.Time
}

// New creates a new Glue mock service.
func New() *Service {
	return &Service{
		databases: make(map[string]*glueDatabase),
		crawlers:  make(map[string]*glueCrawler),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "glue" }

// Handler returns the HTTP handler for Glue requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.databases = make(map[string]*glueDatabase)
	s.crawlers = make(map[string]*glueCrawler)
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
	case "CreateDatabase":
		s.createDatabase(w, params)
	case "GetDatabase":
		s.getDatabase(w, params)
	case "DeleteDatabase":
		s.deleteDatabase(w, params)
	case "GetDatabases":
		s.getDatabases(w, params)
	case "CreateTable":
		s.createTable(w, params)
	case "GetTable":
		s.getTable(w, params)
	case "DeleteTable":
		s.deleteTable(w, params)
	case "GetTables":
		s.getTables(w, params)
	case "CreateCrawler":
		s.createCrawler(w, params)
	case "GetCrawler":
		s.getCrawler(w, params)
	case "DeleteCrawler":
		s.deleteCrawler(w, params)
	case "StartCrawler":
		s.startCrawler(w, params)
	case "ListCrawlers":
		s.listCrawlers(w, params)
	default:
		h.WriteJSONError(w, "UnknownOperationException", fmt.Sprintf("action %q is not supported", action), http.StatusBadRequest)
	}
}

func (s *Service) createDatabase(w http.ResponseWriter, params map[string]interface{}) {
	var name, desc, loc string
	if dbInput, ok := params["DatabaseInput"].(map[string]interface{}); ok {
		name = h.GetString(dbInput, "Name")
		desc = h.GetString(dbInput, "Description")
		loc = h.GetString(dbInput, "LocationUri")
	}
	if name == "" {
		h.WriteJSONError(w, "InvalidInputException", "Database name is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.databases[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "AlreadyExistsException", "Database "+name+" already exists", http.StatusConflict)
		return
	}
	s.databases[name] = &glueDatabase{
		name:        name,
		description: desc,
		locationURI: loc,
		created:     time.Now().UTC(),
		tables:      make(map[string]*glueTable),
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) getDatabase(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")

	s.mu.RLock()
	db, exists := s.databases[name]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "EntityNotFoundException", "Database "+name+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Database": dbResp(db),
	})
}

func (s *Service) deleteDatabase(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")

	s.mu.Lock()
	if _, exists := s.databases[name]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "EntityNotFoundException", "Database "+name+" not found", http.StatusNotFound)
		return
	}
	delete(s.databases, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) getDatabases(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var dbs []map[string]interface{}
	for _, db := range s.databases {
		dbs = append(dbs, dbResp(db))
	}
	s.mu.RUnlock()

	sort.Slice(dbs, func(i, j int) bool {
		return dbs[i]["Name"].(string) < dbs[j]["Name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"DatabaseList": dbs,
	})
}

func (s *Service) createTable(w http.ResponseWriter, params map[string]interface{}) {
	dbName := h.GetString(params, "DatabaseName")

	var tableName, desc, tableType, location string
	var cols []column
	if tableInput, ok := params["TableInput"].(map[string]interface{}); ok {
		tableName = h.GetString(tableInput, "Name")
		desc = h.GetString(tableInput, "Description")
		tableType = h.GetString(tableInput, "TableType")

		if sd, ok := tableInput["StorageDescriptor"].(map[string]interface{}); ok {
			location = h.GetString(sd, "Location")
			if columns, ok := sd["Columns"].([]interface{}); ok {
				for _, c := range columns {
					if cm, ok := c.(map[string]interface{}); ok {
						cols = append(cols, column{
							name:    h.GetString(cm, "Name"),
							colType: h.GetString(cm, "Type"),
							comment: h.GetString(cm, "Comment"),
						})
					}
				}
			}
		}
	}

	if tableName == "" {
		h.WriteJSONError(w, "InvalidInputException", "Table name is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	db, exists := s.databases[dbName]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "EntityNotFoundException", "Database "+dbName+" not found", http.StatusNotFound)
		return
	}

	if _, exists := db.tables[tableName]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "AlreadyExistsException", "Table "+tableName+" already exists", http.StatusConflict)
		return
	}

	now := time.Now().UTC()
	db.tables[tableName] = &glueTable{
		name:        tableName,
		dbName:      dbName,
		description: desc,
		tableType:   tableType,
		location:    location,
		columns:     cols,
		created:     now,
		modified:    now,
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) getTable(w http.ResponseWriter, params map[string]interface{}) {
	dbName := h.GetString(params, "DatabaseName")
	tableName := h.GetString(params, "Name")

	s.mu.RLock()
	db, exists := s.databases[dbName]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "EntityNotFoundException", "Database "+dbName+" not found", http.StatusNotFound)
		return
	}
	table, exists := db.tables[tableName]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "EntityNotFoundException", "Table "+tableName+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Table": tableResp(table),
	})
}

func (s *Service) deleteTable(w http.ResponseWriter, params map[string]interface{}) {
	dbName := h.GetString(params, "DatabaseName")
	tableName := h.GetString(params, "Name")

	s.mu.Lock()
	db, exists := s.databases[dbName]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "EntityNotFoundException", "Database "+dbName+" not found", http.StatusNotFound)
		return
	}
	if _, exists := db.tables[tableName]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "EntityNotFoundException", "Table "+tableName+" not found", http.StatusNotFound)
		return
	}
	delete(db.tables, tableName)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) getTables(w http.ResponseWriter, params map[string]interface{}) {
	dbName := h.GetString(params, "DatabaseName")

	s.mu.RLock()
	db, exists := s.databases[dbName]
	if !exists {
		s.mu.RUnlock()
		h.WriteJSONError(w, "EntityNotFoundException", "Database "+dbName+" not found", http.StatusNotFound)
		return
	}

	var tables []map[string]interface{}
	for _, table := range db.tables {
		tables = append(tables, tableResp(table))
	}
	s.mu.RUnlock()

	sort.Slice(tables, func(i, j int) bool {
		return tables[i]["Name"].(string) < tables[j]["Name"].(string)
	})

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"TableList": tables,
	})
}

func (s *Service) createCrawler(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")
	if name == "" {
		h.WriteJSONError(w, "InvalidInputException", "Name is required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	if _, exists := s.crawlers[name]; exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "AlreadyExistsException", "Crawler "+name+" already exists", http.StatusConflict)
		return
	}
	s.crawlers[name] = &glueCrawler{
		name:    name,
		role:    h.GetString(params, "Role"),
		dbName:  h.GetString(params, "DatabaseName"),
		state:   "READY",
		created: time.Now().UTC(),
	}
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) getCrawler(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")

	s.mu.RLock()
	crawler, exists := s.crawlers[name]
	s.mu.RUnlock()

	if !exists {
		h.WriteJSONError(w, "EntityNotFoundException", "Crawler "+name+" not found", http.StatusNotFound)
		return
	}

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"Crawler": crawlerResp(crawler),
	})
}

func (s *Service) deleteCrawler(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")

	s.mu.Lock()
	if _, exists := s.crawlers[name]; !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "EntityNotFoundException", "Crawler "+name+" not found", http.StatusNotFound)
		return
	}
	delete(s.crawlers, name)
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) startCrawler(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "Name")

	s.mu.Lock()
	crawler, exists := s.crawlers[name]
	if !exists {
		s.mu.Unlock()
		h.WriteJSONError(w, "EntityNotFoundException", "Crawler "+name+" not found", http.StatusNotFound)
		return
	}
	crawler.state = "RUNNING"
	s.mu.Unlock()

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) listCrawlers(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var names []string
	for name := range s.crawlers {
		names = append(names, name)
	}
	s.mu.RUnlock()

	sort.Strings(names)

	h.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"CrawlerNames": names,
	})
}

func dbResp(db *glueDatabase) map[string]interface{} {
	return map[string]interface{}{
		"Name":        db.name,
		"Description": db.description,
		"LocationUri": db.locationURI,
		"CreateTime":  float64(db.created.Unix()),
	}
}

func tableResp(t *glueTable) map[string]interface{} {
	var cols []map[string]interface{}
	for _, c := range t.columns {
		cols = append(cols, map[string]interface{}{
			"Name":    c.name,
			"Type":    c.colType,
			"Comment": c.comment,
		})
	}
	return map[string]interface{}{
		"Name":         t.name,
		"DatabaseName": t.dbName,
		"Description":  t.description,
		"TableType":    t.tableType,
		"CreateTime":   float64(t.created.Unix()),
		"UpdateTime":   float64(t.modified.Unix()),
		"StorageDescriptor": map[string]interface{}{
			"Location": t.location,
			"Columns":  cols,
		},
	}
}

func crawlerResp(c *glueCrawler) map[string]interface{} {
	return map[string]interface{}{
		"Name":         c.name,
		"Role":         c.role,
		"DatabaseName": c.dbName,
		"State":        c.state,
		"CreationTime": float64(c.created.Unix()),
	}
}

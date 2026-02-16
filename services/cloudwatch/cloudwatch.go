// Package cloudwatch provides a mock implementation of AWS CloudWatch (metrics).
//
// Supported actions:
//   - PutMetricData
//   - GetMetricData
//   - ListMetrics
//   - PutMetricAlarm
//   - DescribeAlarms
//   - DeleteAlarms
package cloudwatch

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fxamacker/cbor/v2"
	h "github.com/riyanimam/goto/internal/mockhelpers"
)

// Service implements the CloudWatch metrics mock.
type Service struct {
	mu      sync.RWMutex
	metrics []*metricDatum
	alarms  map[string]*alarm
}

type metricDatum struct {
	namespace  string
	metricName string
	value      float64
	unit       string
	timestamp  time.Time
	dimensions map[string]string
}

type alarm struct {
	name               string
	arn                string
	namespace          string
	metricName         string
	comparisonOperator string
	threshold          float64
	period             int
	evaluationPeriods  int
	statistic          string
	state              string
	stateReason        string
}

// New creates a new CloudWatch mock service.
func New() *Service {
	return &Service{
		alarms: make(map[string]*alarm),
	}
}

// Name returns the service identifier.
func (s *Service) Name() string { return "monitoring" }

// Handler returns the HTTP handler for CloudWatch requests.
func (s *Service) Handler() http.Handler {
	return http.HandlerFunc(s.handle)
}

// Reset clears all state.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics = nil
	s.alarms = make(map[string]*alarm)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	// Extract operation from the URL path.
	// Path format: /service/GraniteServiceVersion20100801/operation/{OperationName}
	path := r.URL.Path
	operation := ""
	if idx := strings.LastIndex(path, "/operation/"); idx >= 0 {
		operation = path[idx+len("/operation/"):]
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeCBORError(w, "InternalError", "could not read body", http.StatusInternalServerError)
		return
	}

	var params map[string]interface{}
	if len(bodyBytes) > 0 {
		cbor.Unmarshal(bodyBytes, &params)
	}
	if params == nil {
		params = make(map[string]interface{})
	}

	switch operation {
	case "PutMetricData":
		s.putMetricData(w, params)
	case "GetMetricData":
		s.getMetricData(w, params)
	case "ListMetrics":
		s.listMetrics(w, params)
	case "PutMetricAlarm":
		s.putMetricAlarm(w, params)
	case "DescribeAlarms":
		s.describeAlarms(w, params)
	case "DeleteAlarms":
		s.deleteAlarms(w, params)
	default:
		writeCBORError(w, "UnsupportedOperation", fmt.Sprintf("action %q is not supported", operation), http.StatusBadRequest)
	}
}

func (s *Service) putMetricData(w http.ResponseWriter, params map[string]interface{}) {
	namespace := h.GetString(params, "Namespace")

	s.mu.Lock()
	if metricData, ok := params["MetricData"].([]interface{}); ok {
		for _, md := range metricData {
			if mdm, ok := md.(map[interface{}]interface{}); ok {
				metricName := ""
				if v, ok := mdm["MetricName"]; ok {
					metricName = fmt.Sprintf("%v", v)
				}
				var value float64
				if v, ok := mdm["Value"]; ok {
					switch n := v.(type) {
					case float64:
						value = n
					case float32:
						value = float64(n)
					}
				}
				unit := "None"
				if v, ok := mdm["Unit"]; ok {
					unit = fmt.Sprintf("%v", v)
				}
				s.metrics = append(s.metrics, &metricDatum{
					namespace:  namespace,
					metricName: metricName,
					value:      value,
					unit:       unit,
					timestamp:  time.Now().UTC(),
				})
			}
		}
	}
	s.mu.Unlock()

	writeCBOR(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) getMetricData(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var results []map[string]interface{}
	for _, m := range s.metrics {
		results = append(results, map[string]interface{}{
			"Id":     fmt.Sprintf("m_%s_%s", m.namespace, m.metricName),
			"Label":  m.metricName,
			"Values": []float64{m.value},
		})
	}
	s.mu.RUnlock()

	writeCBOR(w, http.StatusOK, map[string]interface{}{
		"MetricDataResults": results,
	})
}

func (s *Service) listMetrics(w http.ResponseWriter, params map[string]interface{}) {
	namespace := h.GetString(params, "Namespace")

	s.mu.RLock()
	seen := make(map[string]bool)
	var metrics []map[string]interface{}
	for _, m := range s.metrics {
		if namespace != "" && m.namespace != namespace {
			continue
		}
		key := m.namespace + "/" + m.metricName
		if seen[key] {
			continue
		}
		seen[key] = true

		metrics = append(metrics, map[string]interface{}{
			"Namespace":  m.namespace,
			"MetricName": m.metricName,
		})
	}
	s.mu.RUnlock()

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i]["MetricName"].(string) < metrics[j]["MetricName"].(string)
	})

	writeCBOR(w, http.StatusOK, map[string]interface{}{
		"Metrics": metrics,
	})
}

func (s *Service) putMetricAlarm(w http.ResponseWriter, params map[string]interface{}) {
	name := h.GetString(params, "AlarmName")
	if name == "" {
		writeCBORError(w, "InvalidParameterValue", "AlarmName is required", http.StatusBadRequest)
		return
	}

	var threshold float64
	if v, ok := params["Threshold"]; ok {
		switch n := v.(type) {
		case float64:
			threshold = n
		case float32:
			threshold = float64(n)
		}
	}
	period := h.GetInt(params, "Period", 300)
	evalPeriods := h.GetInt(params, "EvaluationPeriods", 1)

	s.mu.Lock()
	a := &alarm{
		name:               name,
		arn:                fmt.Sprintf("arn:aws:cloudwatch:us-east-1:%s:alarm:%s", h.DefaultAccountID, name),
		namespace:          h.GetString(params, "Namespace"),
		metricName:         h.GetString(params, "MetricName"),
		comparisonOperator: h.GetString(params, "ComparisonOperator"),
		threshold:          threshold,
		period:             period,
		evaluationPeriods:  evalPeriods,
		statistic:          h.GetString(params, "Statistic"),
		state:              "OK",
		stateReason:        "Threshold Crossing: 0 datapoints were OK",
	}
	s.alarms[name] = a
	s.mu.Unlock()

	writeCBOR(w, http.StatusOK, map[string]interface{}{})
}

func (s *Service) describeAlarms(w http.ResponseWriter, _ map[string]interface{}) {
	s.mu.RLock()
	var alarmList []map[string]interface{}
	for _, a := range s.alarms {
		alarmList = append(alarmList, alarmToMap(a))
	}
	s.mu.RUnlock()

	sort.Slice(alarmList, func(i, j int) bool {
		return alarmList[i]["AlarmName"].(string) < alarmList[j]["AlarmName"].(string)
	})

	writeCBOR(w, http.StatusOK, map[string]interface{}{
		"MetricAlarms": alarmList,
	})
}

func (s *Service) deleteAlarms(w http.ResponseWriter, params map[string]interface{}) {
	s.mu.Lock()
	if names, ok := params["AlarmNames"].([]interface{}); ok {
		for _, n := range names {
			if name, ok := n.(string); ok {
				delete(s.alarms, name)
			}
		}
	}
	s.mu.Unlock()

	writeCBOR(w, http.StatusOK, map[string]interface{}{})
}

func alarmToMap(a *alarm) map[string]interface{} {
	return map[string]interface{}{
		"AlarmName":          a.name,
		"AlarmArn":           a.arn,
		"Namespace":          a.namespace,
		"MetricName":         a.metricName,
		"ComparisonOperator": a.comparisonOperator,
		"Threshold":          a.threshold,
		"Period":             a.period,
		"EvaluationPeriods":  a.evaluationPeriods,
		"Statistic":          a.statistic,
		"StateValue":         a.state,
		"StateReason":        a.stateReason,
	}
}

func writeCBOR(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/cbor")
	w.Header().Set("smithy-protocol", "rpc-v2-cbor")
	w.WriteHeader(status)
	data, err := cbor.Marshal(v)
	if err != nil {
		return
	}
	w.Write(data)
}

func writeCBORError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/cbor")
	w.Header().Set("smithy-protocol", "rpc-v2-cbor")
	w.WriteHeader(status)
	data, _ := cbor.Marshal(map[string]interface{}{
		"__type":  code,
		"message": message,
	})
	w.Write(data)
}

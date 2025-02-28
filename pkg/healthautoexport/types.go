package healthautoexport

// API document: https://github.com/Lybron/health-auto-export/wiki/API-Export---JSON-Format

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

const (
	SleepAnalysisName = "sleep_analysis"
)

var (
	// TimeFormat is the default time format to use for output.
	TimeFormat = TimeFormats[0]

	// lastUsedTimeFormat is a global default time format for the package.
	// Attempts to memoize the last used format to speed up unmarshaling.
	// See ParseTime.
	lastUsedTimeFormat = TimeFormat

	// TimeFormats contains all known time formats to parse timestamp by.
	TimeFormats = []string{
		// Using 24-Hour Time
		"2006-01-02 15:04:05 -0700",
		// In case General > Date & Time > 24-Hour Time is set to false
		"2006-01-02 3:04:05 PM -0700",
		"2006-01-02 3:04:05 pm -0700",
		// In case of newer iOS versions which introduce narrow non-breaking space characters into time format
		"2006-01-02 3:04:05\xe2\x80\xafPM -0700",
		"2006-01-02 3:04:05\xe2\x80\xafpm -0700",
	}
)

type Payload struct {
	Data *PayloadData `json:"data,omitempty"`
}

type PayloadData struct {
	Metrics  []*Metric  `json:"metrics,omitempty"`
	Workouts []*Workout `json:"workouts,omitempty"`
}

// Metric defines a single measurement with units, as well as time-series data points.
type Metric struct {
	Name                    string                     `json:"name"`
	Units                   Units                      `json:"units"`
	Datapoints              []*Datapoint               `json:"-"`
	SleepAnalyses           []*SleepAnalysis           `json:"-"`
	AggregatedSleepAnalyses []*AggregatedSleepAnalysis `json:"-"`
}

// metricCopy avoids reflection stack overflow by creating type alias of Metric.
// https://stackoverflow.com/a/43178272/2037090
type metricCopy Metric

// jsonMetric is used to support JSON marshal/unmarshalling of Metric
type jsonMetric struct {
	*metricCopy
	Data json.RawMessage `json:"data"`
}

// SleepAnalysis defines a period during sleep of various types (Value).
// It is only valid for non-aggregate sleep analysis data ("Aggregate Sleep Data" is disabled)
type SleepAnalysis struct {
	StartDate *Time  `json:"startDate"`
	EndDate   *Time  `json:"endDate"`
	Qty       Qty    `json:"qty,omitempty"`
	Source    string `json:"source"`
	Value     string `json:"value"`
}

// AggregatedSleepAnalysis defines an aggregated period of an entire night of sleep.
// It is only valid for aggregate sleep analysis data ("Aggregate Sleep Data" is enabled)
type AggregatedSleepAnalysis struct {
	// we don't parse "date" which doesn't seem to have useful interesting information

	// Start time of sleep.
	SleepStart *Time `json:"sleepStart"`

	// End time of sleep.
	SleepEnd *Time `json:"sleepEnd"`

	// InBed duration in hours.
	InBed Qty `json:"inBed"`

	// Asleep duration in hours.
	Asleep Qty `json:"asleep"`

	// Awake duration in hours.
	// Only available from HAE v6.6.2 onwards.
	Awake Qty `json:"awake,omitempty"`

	// Core sleep duration in hours.
	// Only available from HAE v6.6.2 onwards.
	Core Qty `json:"core,omitempty"`

	// Deep sleep duration in hours.
	// Only available from HAE v6.6.2 onwards.
	Deep Qty `json:"deep,omitempty"`

	// REM sleep duration in hours.
	// Only available from HAE v6.6.2 onwards.
	REM Qty `json:"rem,omitempty"`

	// Start time of inBed phase.
	// Only available prior to HAE v6.6.2.
	InBedStart *Time `json:"inBedStart,omitempty"`

	// End time of inBed phase.
	// Only available prior to HAE v6.6.2.
	InBedEnd *Time `json:"inBedEnd,omitempty"`

	// Data source of sleep data.
	// Multiple source names will be joined together with a pipe (|).
	// Only available from HAE v6.6.2 onwards.
	Source string `json:"source,omitempty"`

	// Data source of sleep phase.
	// Only available prior to HAE v6.6.2.
	SleepSource string `json:"sleepSource,omitempty"`

	// Data source of inBed phase.
	// Only available prior to HAE v6.6.2.
	InBedSource string `json:"inBedSource,omitempty"`
}

func (m *Metric) GetUnits() Units {
	return m.Units
}

// Workout defines a single recorded Workout.
type Workout struct {
	Name  string `json:"name"`
	Start *Time  `json:"start"`
	End   *Time  `json:"end"`

	// Route data
	Route []*RouteDatapoint `json:"route,omitempty"`

	// Heart rate data.
	HeartRateData     []*DatapointWithUnit `json:"heartRateData,omitempty"`
	HeartRateRecovery []*DatapointWithUnit `json:"heartRateRecovery,omitempty"`

	// Elevation data.
	Elevation *Elevation `json:"elevation,omitempty"`

	// Other workout fields.
	Fields WorkoutFields `json:"-"`
}

// WorkoutFields is a map of generic QtyWithUnit fields in a Workout.
type WorkoutFields []Field

// workoutCopy avoids reflection stack overflow by creating type alias of Workout.
// https://stackoverflow.com/a/43178272/2037090
type workoutCopy Workout

func (m *Metric) UnmarshalJSON(bytes []byte) error {
	intermediate := jsonMetric{
		metricCopy: (*metricCopy)(m),
	}
	if err := jsoniter.Unmarshal(bytes, &intermediate); err != nil {
		return err
	}
	switch m.Name {
	case SleepAnalysisName:
		// Try to unmarshal as sleep_analysis on best-effort basis.
		if m.unmarshalSleepAnalysis(intermediate.Data) {
			break
		}
		fallthrough
	default:
		if intermediate.Data == nil {
			return nil
		}
		var d []*Datapoint
		if err := jsoniter.Unmarshal(intermediate.Data, &d); err != nil {
			return err
		}
		m.Datapoints = d
	}

	return nil
}

func (m *Metric) unmarshalSleepAnalysis(data []byte) bool {
	// Try to unmarshal as SleepAnalysis first.
	var sa []*SleepAnalysis
	if err := jsoniter.Unmarshal(data, &sa); err == nil {
		// Non-aggregated sleep_analysis should always have StartDate and EndDate set.
		if len(sa) > 0 && !sa[0].StartDate.IsZero() && !sa[0].EndDate.IsZero() {
			m.SleepAnalyses = sa
			return true
		}
	}

	// Try to unmarshal as AggregatedSleepAnalysis.
	var agg []*AggregatedSleepAnalysis
	if err := jsoniter.Unmarshal(data, &agg); err == nil {
		// Aggregated sleep_analysis should always have SleepStart and SleepEnd set.
		if len(agg) > 0 && !agg[0].SleepStart.IsZero() && !agg[0].SleepEnd.IsZero() {
			m.AggregatedSleepAnalyses = agg
			return true
		}
	}

	return false
}

func (m *Metric) MarshalJSON() ([]byte, error) {
	intermediate := jsonMetric{
		metricCopy: (*metricCopy)(m),
	}
	var data interface{}
	switch m.Name {
	case SleepAnalysisName:
		if len(m.SleepAnalyses) > 0 {
			data = m.SleepAnalyses
			break
		}
		if len(m.AggregatedSleepAnalyses) > 0 {
			data = m.AggregatedSleepAnalyses
			break
		}
		// allow badly parsed sleep analysis to be handled as a Datapoint
		fallthrough
	default:
		data = m.Datapoints
	}
	bytes, err := jsoniter.Marshal(data)
	if err != nil {
		return nil, err
	}
	intermediate.Data = bytes
	return jsoniter.Marshal(intermediate)
}

func (w *Workout) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})
	for _, field := range w.Fields {
		result[field.Key] = field.Value
	}

	// Marshal and unmarshal remaining fields onto the same map
	outerBytes, err := jsoniter.Marshal((*workoutCopy)(w))
	if err != nil {
		return nil, err
	}
	if err := jsoniter.Unmarshal(outerBytes, &result); err != nil {
		return nil, err
	}

	// Marshal result back
	return jsoniter.Marshal(result)
}

// UnmarshalJSON implements a custom json.Unmarshaler for Workout.
// This is necessary to unmarshal arbitrary Fields that may match QtyWithUnit.
func (w *Workout) UnmarshalJSON(bytes []byte) error {
	// First pass: Unmarshal into struct.
	if err := jsoniter.Unmarshal(bytes, (*workoutCopy)(w)); err != nil {
		return err
	}

	// Second pass: Unmarshal into generic map.
	fields := make(map[string]interface{})
	if err := jsoniter.Unmarshal(bytes, &fields); err != nil {
		return err
	}

	// Use mapstructure to decode any matching field into Fields.
	w.Fields = make(WorkoutFields, 0, 10)
	for k, value := range fields {
		var result QtyWithUnit
		dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName:     "json",
			Result:      &result,
			ErrorUnused: true, // required to prevent partial match
		})
		if err != nil {
			return err
		}
		if err := dec.Decode(value); err == nil {
			w.Fields = append(w.Fields, Field{
				Key:   k,
				Value: &result,
			})
		}
	}
	sort.Slice(w.Fields, func(i, j int) bool {
		return w.Fields[i].Key < w.Fields[j].Key
	})

	return nil
}

// Qty is used to define an arbitrary quantity.
type Qty float64

// Units is used to define a unit of measurement.
type Units string

type Field struct {
	Key   string
	Value *QtyWithUnit
}

// QtyWithUnit combines a Qty with Units of measurement.
type QtyWithUnit struct {
	Qty   Qty   `json:"qty"`
	Units Units `json:"units"`
}

func (q QtyWithUnit) GetUnits() Units {
	return q.Units
}

// Datapoint is a point-in-time value of a metric.
type Datapoint struct {
	Date *Time `json:"date"`

	// Qty may not be specified for some types of metrics.
	Qty Qty `json:"qty,omitempty"`

	// Other fields.
	Fields DatapointFields `json:"-"`
}

// DatapointFields is a map of fields with an arbitrary type in a single Datapoint.
type DatapointFields map[string]interface{}

// datapointCopy avoids reflection stack overflow by creating type alias of Datapoint.
// https://stackoverflow.com/a/43178272/2037090
type datapointCopy Datapoint

func (w *Datapoint) MarshalJSON() ([]byte, error) {
	// Marshal and unmarshal Fields into a generic map
	result := make(map[string]interface{})
	bytes, err := jsoniter.Marshal(w.Fields)
	if err != nil {
		return nil, err
	}
	if err := jsoniter.Unmarshal(bytes, &result); err != nil {
		return nil, err
	}

	// Marshal and unmarshal remaining fields onto the same map
	outerBytes, err := jsoniter.Marshal((*datapointCopy)(w))
	if err != nil {
		return nil, err
	}
	if err := jsoniter.Unmarshal(outerBytes, &result); err != nil {
		return nil, err
	}

	// Marshal result back
	return jsoniter.Marshal(result)
}

// UnmarshalJSON implements a custom json.Unmarshaler for Datapoint.
// This is necessary to unmarshal arbitrary DatapointFields.
func (w *Datapoint) UnmarshalJSON(bytes []byte) error {
	// First pass: Unmarshal into struct.
	if err := jsoniter.Unmarshal(bytes, (*datapointCopy)(w)); err != nil {
		return err
	}

	// Second pass: Unmarshal into generic map, dropping qty and date.
	fields := make(map[string]interface{})
	if err := jsoniter.Unmarshal(bytes, &fields); err != nil {
		return err
	}

	t := reflect.TypeOf(*w)
	// remove already parsed fields in Datapoint struct, leaving only unknown fields
	for i := 0; i < t.NumField(); i++ {
		jsonTag := t.Field(i).Tag.Get("json")
		delete(fields, strings.Split(jsonTag, ",")[0])
	}

	// Try to unmarshal special types, otherwise fallback to normal json.Unmarshaler.
	w.Fields = make(DatapointFields)
	for k, v := range fields {
		result := v
		switch value := v.(type) {
		case string:
			var t Time
			if err := jsoniter.Unmarshal([]byte(`"`+value+`"`), &t); err == nil {
				result = &t
				break
			}
		}
		w.Fields[k] = result
	}

	return nil
}

// DatapointWithUnit is a point-in-time value of a QtyWithUnit.
type DatapointWithUnit struct {
	Date *Time `json:"date"`
	QtyWithUnit
}

// RouteDatapoint is a point-in-time location in 3D coordinates.
type RouteDatapoint struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Altitude  float64 `json:"altitude"`
	Timestamp *Time   `json:"timestamp"`
}

// Elevation is a specify QtyWithUnit that specifies Ascent and Descent values.
// It is only used for the Elevation field.
type Elevation struct {
	Units   Units `json:"units"`
	Ascent  Qty   `json:"ascent"`
	Descent Qty   `json:"descent"`
}

func (e Elevation) GetUnits() Units {
	return e.Units
}

// Time is a custom time type for this package.
type Time struct {
	time.Time
}

func NewTime(t time.Time) Time {
	return Time{Time: t}
}

// ParseTime attempts to parses the input string in the time format expected by health auto export.
//
// Since iOS may export timestamps using different time formats depending on the Region setting,
// we need to handle all possible time formats that may be available.
//
// As an optimization, the last known good format will be cached to speed up subsequent calls to ParseTime.
func ParseTime(s string) (Time, error) {
	var multiErr error
	// Attempt to unmarshal using lastUsedTimeFormat first.
	if t, err := parseTime(lastUsedTimeFormat, s); err == nil {
		return t, nil
	}
	// Otherwise, try all formats.
	for _, timeFormat := range TimeFormats {
		parsed, err := parseTime(timeFormat, s)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
			continue
		}
		// Cache the last used format to speed up subsequent parses.
		lastUsedTimeFormat = timeFormat
		return parsed, nil
	}
	return Time{}, fmt.Errorf(`failed to parse time "%v" across all known formats: %w`, s, multiErr)
}

// parseTime attempts to parse the time string with the given time format.
func parseTime(timeFormat string, s string) (Time, error) {
	parsed, err := time.Parse(timeFormat, s)
	if err != nil {
		return Time{}, errors.Wrapf(err, `cannot parse with format: "%v"`, timeFormat)
	}
	return NewTime(parsed), nil
}

// String returns a RFC3339 formatted timestamp string.
func (t Time) String() string {
	return t.Format(time.RFC3339)
}

func (t *Time) IsZero() bool {
	if t == nil {
		return true
	}
	return t.Time.IsZero()
}

// MarshalJSON implements the json.Marshaler interface.
// This implementation overrides the default time.Time json.Marshaler
// implementation, and marshals using TimeFormat.
func (t Time) MarshalJSON() ([]byte, error) {
	// Propagate marshal errors forward first.
	if ret, err := t.Time.MarshalJSON(); err != nil {
		return ret, err
	}

	// Otherwise, marshal using TimeFormat.
	b := make([]byte, 0, len(TimeFormat)+2)
	b = append(b, '"')
	b = t.AppendFormat(b, TimeFormat)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// This implementation overrides the default time.Time json.Unmarshaler
// implementation, and parses timestamps using TimeFormat.
func (t *Time) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return t.Time.UnmarshalJSON(data)
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseTime(s)
	if err != nil {
		return err
	}
	t.Time = parsed.Time
	return nil
}

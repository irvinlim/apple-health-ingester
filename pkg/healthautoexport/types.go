package healthautoexport

// API document: https://github.com/Lybron/health-auto-export/wiki/API-Export---JSON-Format

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/mapstructure"
)

const (
	// TimeFormat is the format to parse time.Time in this package.
	TimeFormat        = "2006-01-02 15:04:05 -0700"
	SleepAnalysisName = "sleep_analysis"
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
	Asleep      Qty    `json:"asleep"`
	SleepSource string `json:"sleepSource"`
	SleepStart  *Time  `json:"sleepStart"`
	SleepEnd    *Time  `json:"sleepEnd"`
	InBed       Qty    `json:"inBed"`
	InBedSource string `json:"inBedSource"`
	InBedStart  *Time  `json:"inBedStart"`
	InBedEnd    *Time  `json:"inBedEnd"`
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
		var sa []*SleepAnalysis
		if err := jsoniter.Unmarshal(intermediate.Data, &sa); err != nil {
			return err
		}
		// only process as SleepAnalysis if first item parses to non-empty SleepAnalysis
		if len(sa) > 0 && *sa[0] != (SleepAnalysis{}) {
			m.SleepAnalyses = sa
			return nil
		}
		var agg []*AggregatedSleepAnalysis
		if err := jsoniter.Unmarshal(intermediate.Data, &agg); err != nil {
			return err
		}
		// only process as AggregatedSleepAnalysis if first item parses to non-empty AggregatedSleepAnalysis
		if len(agg) > 0 && *agg[0] != (AggregatedSleepAnalysis{}) {
			m.AggregatedSleepAnalyses = agg
			return nil
		}
		// if neither type of sleep analysis parsed something, parse as a Datapoint.
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
		return nil
	}
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

func NewTime(t time.Time) *Time {
	return &Time{Time: t}
}

// String returns a RFC3339 formatted timestamp string.
func (t *Time) String() string {
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
	var err error
	t.Time, err = time.Parse(`"`+TimeFormat+`"`, string(data))
	return err
}

package healthautoexport

// API document: https://github.com/Lybron/health-auto-export/wiki/API-Export---JSON-Format

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/mapstructure"
)

const (
	// TimeFormat is the format to parse time.Time in this package.
	TimeFormat = "2006-01-02 15:04:05 -0700"
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
	Name          string           `json:"name"`
	Units         Units            `json:"units"`
	Data          []*Datapoint     `json:"data"`
	SleepAnalyses []*SleepAnalysis `json:"-"`
}

type SleepAnalysis struct {
	// Start/EndDate only defined for sleep_analysis if "Aggregate Sleep Data" is disabled
	StartDate *Time  `json:"startDate"`
	EndDate   *Time  `json:"endDate"`
	Qty       Qty    `json:"qty,omitempty"`
	Source    string `json:"source"`
	Value     string `json:"value"`
}

// metricCopy avoids reflection stack overflow by creating type alias of Metric.
// https://stackoverflow.com/a/43178272/2037090
type metricCopy Metric

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
type WorkoutFields map[string]*QtyWithUnit

// workoutCopy avoids reflection stack overflow by creating type alias of Workout.
// https://stackoverflow.com/a/43178272/2037090
type workoutCopy Workout

func (m *Metric) UnmarshalJSON(bytes []byte) error {
	type MetricInternal struct {
		Name  string          `json:"name"`
		Units Units           `json:"units"`
		Data  json.RawMessage `json:"data"`
	}
	mi := MetricInternal{}
	if err := jsoniter.Unmarshal(bytes, &mi); err != nil {
		return err
	}
	m.Name = mi.Name
	m.Units = mi.Units
	if mi.Name == "sleep_analysis" {
		var sa []*SleepAnalysis
		if err := jsoniter.Unmarshal(mi.Data, &sa); err != nil {
			return err
		}
		if len(sa) > 0 && sa[0].Value != "" {
			m.SleepAnalyses = sa
			return nil
		}
	}
	if mi.Data == nil {
		return nil
	}
	// if name not sleep_analysis, or is an aggregated sleep analysis, do normal Datapoint
	var d []*Datapoint
	if err := jsoniter.Unmarshal(mi.Data, &d); err != nil {
		return err
	}
	m.Data = d

	return nil
}

func (m *Metric) MarshalJSON() ([]byte, error) {
	if len(m.SleepAnalyses) == 0 {
		return jsoniter.Marshal((*metricCopy)(m))
	}
	sleepDatapoints := make([]*Datapoint, len(m.SleepAnalyses))
	for i, s := range m.SleepAnalyses {
		sleepDatapoints[i] = &Datapoint{
			Qty: s.Qty,
			Fields: map[string]interface{}{
				"startDate": s.StartDate,
				"endDate":   s.EndDate,
				"value":     s.Value,
				"source":    s.Source,
			},
		}
	}
	mCopy := Metric{
		Name:  m.Name,
		Units: m.Units,
		Data:  sleepDatapoints,
	}
	return jsoniter.Marshal((metricCopy)(mCopy))

}

func (w *Workout) MarshalJSON() ([]byte, error) {
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
	w.Fields = make(WorkoutFields)
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
			w.Fields[k] = &result
		}
	}

	return nil
}

// Qty is used to define an arbitrary quantity.
type Qty float64

// Units is used to define a unit of measurement.
type Units string

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

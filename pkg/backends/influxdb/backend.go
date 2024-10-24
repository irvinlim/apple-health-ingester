package influxdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	lp "github.com/influxdata/line-protocol"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/irvinlim/apple-health-ingester/pkg/backends"
	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
	utiltime "github.com/irvinlim/apple-health-ingester/pkg/util/time"
)

const (
	MeasurementSleepAnalysisDetailed   = "sleep_analysis_detailed"
	MeasurementSleepAnalysisAggregated = "sleep_analysis_aggregated"
	MeasurementSleepPhases             = "sleep_phases"
)

// Backend InfluxDB is used to store ingested metrics into InfluxDB. All metrics
// will be stored as single Points (i.e. time-series data).
type Backend struct {
	ctx        context.Context
	client     Client
	staticTags []lp.Tag
}

var _ backends.Backend = &Backend{}

func NewBackend(client Client) (backends.Backend, error) {
	backend := &Backend{
		ctx:        context.TODO(),
		client:     client,
		staticTags: make([]lp.Tag, len(staticTags)),
	}

	// Prepare static tags.
	for i, tag := range staticTags {
		tokens := strings.SplitN(tag, "=", 2)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid static tag %v", tag)
		}
		backend.staticTags[i] = lp.Tag{
			Key:   tokens[0],
			Value: tokens[1],
		}
	}

	return backend, nil
}

func (b *Backend) Name() string {
	return "InfluxDB"
}

func (b *Backend) Write(payload *healthautoexport.Payload, targetName string) error {
	// Properly handle nil data.
	if payload == nil || payload.Data == nil {
		log.WithFields(log.Fields{
			"backend": b.Name(),
			"target":  targetName,
		}).Warn("empty payload data received, skipping")
		return nil
	}

	// Write metrics.
	if len(payload.Data.Metrics) > 0 {
		if err := b.writeMetrics(payload.Data.Metrics, targetName); err != nil {
			return errors.Wrapf(err, "write metrics error")
		}
	}

	// Write workouts.
	if len(payload.Data.Workouts) > 0 {
		if err := b.writeWorkouts(payload.Data.Workouts, targetName); err != nil {
			return errors.Wrapf(err, "write workouts error")
		}
	}

	return nil
}

func (b *Backend) writeMetrics(metrics []*healthautoexport.Metric, targetName string) error {
	logger := log.WithFields(log.Fields{
		"backend":     b.Name(),
		"target":      targetName,
		"num_metrics": len(metrics),
	})

	startTime := time.Now()
	logger.Info("start writing all metrics")

	tags := []lp.Tag{
		{Key: "target_name", Value: targetName},
	}
	tags = append(tags, b.staticTags...)

	var info timeseriesInfo
	for _, metric := range metrics {
		points, metricInfo := b.processMetricPoints(metric, tags)
		if len(points) > 0 {
			logger := logger.WithFields(log.Fields{
				"metric_name": metric.Name,
				"count":       len(points),
				"time_range":  utiltime.FormatTimeRange(metricInfo.StartTime, metricInfo.EndTime, time.RFC3339),
			})
			startTime := time.Now()
			logger.Debug("writing metric points")
			if err := b.client.WriteMetrics(b.ctx, points...); err != nil {
				return errors.Wrapf(err, "write error for %v", metric.Name)
			}

			// Process info before moving on.
			info.Count += metricInfo.Count
			info.StartTime = utiltime.MinTimeNonZero(info.StartTime, metricInfo.StartTime)
			info.EndTime = utiltime.MaxTime(info.EndTime, metricInfo.EndTime)

			logger.WithField("elapsed", time.Since(startTime)).Debug("write metric points success")
		}
	}

	logger.WithFields(log.Fields{
		"points":     info.Count,
		"time_range": utiltime.FormatTimeRange(info.StartTime, info.EndTime, time.RFC3339),
		"elapsed":    time.Since(startTime),
	}).Info("write all metrics success")

	return nil
}

type timeseriesInfo struct {
	// Earliest timestamp that is collected.
	StartTime time.Time `json:"start_time"`
	// Latest timestamp that is collected.
	EndTime time.Time `json:"end_time"`
	// Total number of points.
	Count int `json:"count"`
}

func (b *Backend) processMetricPoints(metric *healthautoexport.Metric, tags []lp.Tag) ([]*write.Point, timeseriesInfo) {
	var info timeseriesInfo

	points := make([]*write.Point, 0, len(metric.Datapoints))
	datapointMeasurement := GetUnitizedMeasurementName(metric.Name, metric)
	for _, datum := range metric.Datapoints {
		point := write.NewPointWithMeasurement(datapointMeasurement)
		addTagsToPoint(point, tags)
		// Add qty if set
		if datum.Qty != 0 {
			point.AddField("qty", float64(datum.Qty))
		}
		// Add additional fields
		for name, value := range datum.Fields {
			point.AddField(name, value)
		}
		// Skip if there are no fields to write
		if len(point.FieldList()) == 0 {
			continue
		}
		point.SetTime(datum.Date.Time)

		// Process info before moving on.
		info.StartTime = utiltime.MinTimeNonZero(info.StartTime, datum.Date.Time)
		info.EndTime = utiltime.MaxTime(info.EndTime, datum.Date.Time)
		info.Count++

		points = append(points, point)
	}

	// Add points for detailed sleep analysis.
	for _, s := range metric.SleepAnalyses {
		points = append(points, makeSleepPoint(MeasurementSleepAnalysisDetailed, s.Source, s.Value, 1, nil, s.StartDate, tags))
		// end point has state = 0 (off)
		points = append(points, makeSleepPoint(MeasurementSleepAnalysisDetailed, s.Source, s.Value, 0, &s.Qty, s.EndDate, tags))
	}

	// Add points for aggregated sleep analysis.
	for _, a := range metric.AggregatedSleepAnalyses {
		// Support old aggregated sleep analysis format prior to HAE v6.6.2.
		if a.SleepSource != "" {
			points = append(points, makeSleepPoint(MeasurementSleepAnalysisAggregated, a.SleepSource, "asleep", 1, nil, a.SleepStart, tags))
			points = append(points, makeSleepPoint(MeasurementSleepAnalysisAggregated, a.SleepSource, "asleep", 0, &a.Asleep, a.SleepEnd, tags))
		}
		if a.InBedSource != "" {
			points = append(points, makeSleepPoint(MeasurementSleepAnalysisAggregated, a.InBedSource, "inBed", 1, nil, a.InBedStart, tags))
			points = append(points, makeSleepPoint(MeasurementSleepAnalysisAggregated, a.InBedSource, "inBed", 0, &a.InBed, a.InBedEnd, tags))
		}

		// Support sleep phase data from HAE v6.6.2 onwards.
		// All points for sleep phase will use the SleepEnd time.
		if a.Source != "" {
			points = append(points, makeSleepPhasePoint(MeasurementSleepPhases, a.Source, "awake", a.Awake, a.SleepEnd, tags))
			points = append(points, makeSleepPhasePoint(MeasurementSleepPhases, a.Source, "asleep", a.Asleep, a.SleepEnd, tags))
			points = append(points, makeSleepPhasePoint(MeasurementSleepPhases, a.Source, "inBed", a.InBed, a.SleepEnd, tags))
			points = append(points, makeSleepPhasePoint(MeasurementSleepPhases, a.Source, "core", a.Core, a.SleepEnd, tags))
			points = append(points, makeSleepPhasePoint(MeasurementSleepPhases, a.Source, "deep", a.Deep, a.SleepEnd, tags))
			points = append(points, makeSleepPhasePoint(MeasurementSleepPhases, a.Source, "rem", a.REM, a.SleepEnd, tags))
		}
	}

	return points, info
}

func makeSleepPoint(measurement string, source string, value string,
	state uint8, qty *healthautoexport.Qty, t *healthautoexport.Time, tags []lp.Tag) *write.Point {
	point := write.NewPointWithMeasurement(measurement)
	addTagsToPoint(point, tags)
	point.AddTag("source", source)
	point.AddTag("value", value)
	if qty != nil {
		point.AddField("qty", float64(*qty))
	}
	point.AddField("state", state)
	point.SetTime(t.Time)
	return point
}

func makeSleepPhasePoint(
	measurement string,
	source string,
	value string,
	qty healthautoexport.Qty,
	t *healthautoexport.Time,
	tags []lp.Tag,
) *write.Point {
	point := write.NewPointWithMeasurement(measurement)
	addTagsToPoint(point, tags)
	point.AddTag("source", source)
	point.AddTag("value", value)
	point.AddField("qty", float64(qty))
	point.SetTime(t.Time)
	return point
}

func (b *Backend) writeWorkouts(workouts []*healthautoexport.Workout, targetName string) error {
	logger := log.WithFields(log.Fields{
		"backend":      b.Name(),
		"target":       targetName,
		"num_workouts": len(workouts),
	})

	var count int
	startTime := time.Now()
	logger.Info("start writing all workouts")

	for _, workout := range workouts {
		points := make([]*write.Point, 0)

		// Create tags
		tags := []lp.Tag{
			{Key: "target_name", Value: targetName},
			{Key: "workout_name", Value: workout.Name},
		}
		tags = append(tags, b.staticTags...)

		// Create aggregate workout point
		point, err := b.createWorkoutAggregatePoint(workout)
		if err != nil {
			return errors.Wrapf(err, "conversion error for workout %+v", workout)
		}
		if point != nil {
			addTagsToPoint(point, tags)
			points = append(points, point)
		}

		// Create during-workout datapoints
		points = append(points, b.createRoutePoints("route", tags, workout.Route)...)
		points = append(points, b.createWorkoutPoints("heart_rate_data", tags, workout.HeartRateData)...)
		points = append(points, b.createWorkoutPoints("heart_rate_recovery", tags, workout.HeartRateRecovery)...)

		if len(points) > 0 {
			logger := logger.WithFields(log.Fields{
				"measurement": "workout",
				"workout":     workout.Name,
				"count":       len(points),
			})
			startTime := time.Now()
			count += len(points)
			logger.Debug("writing workout points")
			if err := b.client.WriteWorkouts(b.ctx, points...); err != nil {
				return errors.Wrapf(err, "write error for workout")
			}
			logger.WithField("elapsed", time.Since(startTime)).Debug("write workout points success")
		}
	}

	logger.WithFields(log.Fields{
		"points":  count,
		"elapsed": time.Since(startTime),
	}).Info("write all workouts success")

	return nil
}

func (b *Backend) createWorkoutAggregatePoint(workout *healthautoexport.Workout) (*write.Point, error) {
	// Skip if the workout has no start time (probably invalid)
	if workout.Start.IsZero() {
		return nil, errors.New("workout has no start time")
	}
	point := write.NewPointWithMeasurement("workout")
	// Compute fields from workout
	workoutFields := CreateWorkoutStatistics(workout)
	// Add elevation fields
	if workout.Elevation != nil {
		workoutFields = append(workoutFields, healthautoexport.Field{
			Key: "elevation_ascent",
			Value: &healthautoexport.QtyWithUnit{
				Qty:   workout.Elevation.Ascent,
				Units: workout.Elevation.Units,
			},
		}, healthautoexport.Field{
			Key: "elevation_descent",
			Value: &healthautoexport.QtyWithUnit{
				Qty:   workout.Elevation.Descent,
				Units: workout.Elevation.Units,
			},
		})
	}
	// Add other WorkoutFields
	workoutFields = append(workoutFields, workout.Fields...)
	// Convert to InfluxDB field format
	for _, field := range workoutFields {
		fieldName := GetUnitizedMeasurementName(field.Key, field.Value)
		point.AddField(fieldName, float64(field.Value.Qty))
	}
	// Skip if there are no fields to write
	if len(point.FieldList()) == 0 {
		return nil, nil
	}
	point.SetTime(workout.Start.Time)
	return point, nil
}

func (b *Backend) createWorkoutPoints(
	name string, tags []lp.Tag, data []*healthautoexport.DatapointWithUnit,
) []*write.Point {
	points := make([]*write.Point, 0, len(data))
	for _, datum := range data {
		point := write.NewPointWithMeasurement(GetUnitizedMeasurementName(name, datum))
		addTagsToPoint(point, tags)
		point.AddField("qty", float64(datum.Qty))
		point.SetTime(datum.Date.Time)
		points = append(points, point)
	}
	return points
}

func (b *Backend) createRoutePoints(
	name string, tags []lp.Tag, data []*healthautoexport.RouteDatapoint,
) []*write.Point {
	points := make([]*write.Point, 0, len(data))
	for _, datum := range data {
		point := write.NewPointWithMeasurement(name)
		addTagsToPoint(point, tags)
		point.AddField("lat", datum.Lat)
		point.AddField("lon", datum.Lon)
		point.AddField("altitude", datum.Altitude)
		point.SetTime(datum.Timestamp.Time)
		points = append(points, point)
	}
	return points
}

func addTagsToPoint(point *write.Point, tags []lp.Tag) {
	for _, tag := range tags {
		if tag.Value != "" {
			point.AddTag(tag.Key, tag.Value)
		}
	}
}

type WithUnits interface {
	GetUnits() healthautoexport.Units
}

// GetUnitizedMeasurementName returns the measurement name.
// It will add a suffix for the unit to the measurement name.
func GetUnitizedMeasurementName(name string, metric WithUnits) string {
	return name + "_" + string(metric.GetUnits())
}

package influxdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/irvinlim/apple-health-ingester/pkg/backends"
	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

// Backend InfluxDB is used to store ingested metrics into InfluxDB. All metrics
// will be stored as single Points (i.e. time-series data). Workouts will be
// stored also as time-series data after additional post-processing. Due to the
// complex nature of Workout data, not all information can be properly encoded
// as time-series data.
type Backend struct {
	client         influxdb2.Client
	additionalTags map[string]string
}

var _ backends.Backend = &Backend{}

func NewBackend() (backends.Backend, error) {
	client, err := NewInfluxDBClient()
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create influxdb client")
	}

	backend := &Backend{
		client:         client,
		additionalTags: make(map[string]string),
	}

	// Prepare static tags.
	for _, tag := range staticTags {
		tokens := strings.SplitN(tag, "=", 2)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid static tag %v", tag)
		}
		backend.additionalTags[tokens[0]] = tokens[1]
	}

	return backend, nil
}

func (b *Backend) Name() string {
	return "InfluxDB"
}

func (b *Backend) Write(payload *healthautoexport.Payload, targetName string) error {
	// Write metrics.
	if err := b.writeMetrics(payload.Data.Metrics, targetName); err != nil {
		return errors.Wrapf(err, "write metrics error")
	}

	// Write workouts.
	if err := b.writeWorkouts(payload.Data.Workouts, targetName); err != nil {
		return errors.Wrapf(err, "write workouts error")
	}

	return nil
}

func (b *Backend) writeMetrics(metrics []*healthautoexport.Metric, targetName string) error {
	api := b.client.WriteAPIBlocking(orgName, metricsBucketName)
	for _, metric := range metrics {
		measurementName := GetUnitizedMeasurementName(metric.Name, metric)
		points := b.getMetricPoints(measurementName, metric, targetName)
		if len(points) > 0 {
			logger := log.WithFields(log.Fields{
				"backend":     b.Name(),
				"target":      targetName,
				"measurement": measurementName,
				"count":       len(points),
			})

			startTime := time.Now()
			logger.Info("writing metric points")
			if err := api.WritePoint(context.Background(), points...); err != nil {
				return errors.Wrapf(err, "write error for %v", measurementName)
			}
			logger.WithField("elapsed", startTime).Info("write metric points success")
		}
	}
	return nil
}

func (b *Backend) getMetricPoints(
	measurement string, metric *healthautoexport.Metric, targetName string,
) []*write.Point {
	points := make([]*write.Point, 0, len(metric.Data))
	tags := b.MakeTags(targetName)

	for _, datum := range metric.Data {
		fields := make(map[string]interface{})

		// Add qty if set
		if datum.Qty != 0 {
			fields["qty"] = float64(datum.Qty)
		}

		// Add additional fields
		for name, value := range datum.Fields {
			fields[name] = value
		}

		// Skip if there are no fields to write
		if len(fields) == 0 {
			continue
		}

		point := write.NewPoint(measurement, tags, fields, datum.Date.Time)
		points = append(points, point)
	}

	return points
}

func (b *Backend) writeWorkouts(workouts []*healthautoexport.Workout, targetName string) error {
	api := b.client.WriteAPIBlocking(orgName, workoutsBucketName)

	for _, workout := range workouts {
		points := make([]*write.Point, 0)

		// Create tags
		tags := b.MakeTags(targetName)
		tags["workout_name"] = workout.Name

		// Create aggregate workout point
		point, err := b.createWorkoutAggregatePoint(workout, tags)
		if err != nil {
			return errors.Wrapf(err, "conversion error for workout %+v", workout)
		}
		if point != nil {
			points = append(points, point)
		}

		// Create during-workout datapoints
		points = append(points, b.createWorkoutPoints("heart_rate_data", tags, workout.HeartRateData)...)
		points = append(points, b.createWorkoutPoints("heart_rate_recovery", tags, workout.HeartRateRecovery)...)

		if len(points) > 0 {
			logger := log.WithFields(log.Fields{
				"backend":     b.Name(),
				"target":      targetName,
				"measurement": "workout",
				"workout":     workout.Name,
				"count":       len(points),
			})

			startTime := time.Now()
			logger.Info("writing workout points")
			if err := api.WritePoint(context.Background(), points...); err != nil {
				return errors.Wrapf(err, "write error for workout")
			}
			logger.WithField("elapsed", time.Since(startTime)).Info("write workout points success")
		}
	}

	return nil
}

func (b *Backend) createWorkoutAggregatePoint(workout *healthautoexport.Workout, tags map[string]string) (*write.Point, error) {
	// Skip if the workout has no start time (probably invalid)
	if workout.Start.IsZero() {
		return nil, errors.New("workout has no start time")
	}

	// Compute fields from workout
	workoutFields := CreateWorkoutStatistics(workout)

	// Add elevation fields
	if workout.Elevation != nil {
		workoutFields["elevation_ascent"] = &healthautoexport.QtyWithUnit{
			Qty:   workout.Elevation.Ascent,
			Units: workout.Elevation.Units,
		}
		workoutFields["elevation_descent"] = &healthautoexport.QtyWithUnit{
			Qty:   workout.Elevation.Descent,
			Units: workout.Elevation.Units,
		}
	}

	// Add other WorkoutFields
	for name, field := range workout.Fields {
		workoutFields[name] = field
	}

	// Convert to InfluxDB field format
	fields := MakeInfluxFieldsFromWorkoutFields(workoutFields)

	// Skip if there are no fields to write
	if len(fields) == 0 {
		return nil, nil
	}

	point := write.NewPoint("workout", tags, fields, workout.Start.Time)
	return point, nil
}

func (b *Backend) createWorkoutPoints(
	name string, tags map[string]string, data []*healthautoexport.DatapointWithUnit,
) []*write.Point {
	points := make([]*write.Point, 0, len(data))
	for _, datum := range data {
		measurement := GetUnitizedMeasurementName(name, datum)
		fields := map[string]interface{}{
			"qty": float64(datum.Qty),
		}
		point := write.NewPoint(measurement, tags, fields, datum.Date.Time)
		points = append(points, point)
	}
	return points
}

// MakeTags returns a map of tags that can be safely modified.
// Accepts a targetName, which if not empty, will be added to the map.
func (b *Backend) MakeTags(targetName string) map[string]string {
	tags := make(map[string]string, len(b.additionalTags))
	for k, v := range b.additionalTags {
		tags[k] = v
	}
	if targetName != "" {
		tags["target_name"] = targetName
	}
	return tags
}

type WithUnits interface {
	GetUnits() healthautoexport.Units
}

// GetUnitizedMeasurementName returns the measurement name.
// It will add a suffix for the unit to the measurement name.
func GetUnitizedMeasurementName(name string, metric WithUnits) string {
	return name + "_" + string(metric.GetUnits())
}

// MakeInfluxFieldsFromWorkoutFields converts WorkoutFields to InfluxDB fields.
func MakeInfluxFieldsFromWorkoutFields(fields healthautoexport.WorkoutFields) map[string]interface{} {
	result := make(map[string]interface{}, len(fields))
	for name, field := range fields {
		fieldName := GetUnitizedMeasurementName(name, field)
		result[fieldName] = float64(field.Qty)
	}
	return result
}

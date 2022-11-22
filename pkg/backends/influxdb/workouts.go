package influxdb

import (
	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

// CreateWorkoutStatistics returns additional fields for a Workout.
func CreateWorkoutStatistics(workout *healthautoexport.Workout) healthautoexport.WorkoutFields {
	fields := make(healthautoexport.WorkoutFields, 0, 10)

	// Compute duration of the workout.
	if !workout.End.IsZero() && !workout.Start.IsZero() {
		fields = append(fields, healthautoexport.Field{
			Key: "duration",
			Value: &healthautoexport.QtyWithUnit{
				Qty:   healthautoexport.Qty(workout.End.Sub(workout.Start.Time).Minutes()),
				Units: "min",
			},
		})
	}

	return fields
}

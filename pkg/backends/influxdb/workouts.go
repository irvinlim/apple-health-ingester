package influxdb

import (
	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

// CreateWorkoutStatistics returns additional fields for a Workout.
func CreateWorkoutStatistics(workout *healthautoexport.Workout) healthautoexport.WorkoutFields {
	fields := make(healthautoexport.WorkoutFields)

	// Compute duration of the workout.
	if !workout.End.IsZero() && !workout.Start.IsZero() {
		fields["duration"] = &healthautoexport.QtyWithUnit{
			Qty:   healthautoexport.Qty(workout.End.Sub(workout.Start.Time).Minutes()),
			Units: "min",
		}
	}

	return fields
}

package fixtures

import (
	"time"

	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

var (
	// PayloadWithMetrics is an example Payload with metrics of standard types.
	PayloadWithMetrics = &healthautoexport.Payload{
		Data: &healthautoexport.PayloadData{
			Metrics: []*healthautoexport.Metric{
				MetricActiveEnergy,
				MetricBasalBodyTemperatureNoData,
			},
		},
	}

	// MetricActiveEnergy is an example Metric for active energy.
	MetricActiveEnergy = &healthautoexport.Metric{
		Name:  "active_energy",
		Units: "kJ",
		Data: []*healthautoexport.Datapoint{
			{
				Qty:  0.76856774374845116,
				Date: mktime("2021-12-24 00:04:00 +0800"),
			},
			{
				Qty:  0.377848256251549,
				Date: mktime("2021-12-24 00:05:00 +0800"),
			},
		},
	}

	// MetricBasalBodyTemperatureNoData is an example Metric for basal body temperature with no data.
	MetricBasalBodyTemperatureNoData = &healthautoexport.Metric{
		Name:  "basal_body_temperature",
		Units: "degC",
	}

	// PayloadMetricsSleepAnalysis is an example Payload with sleep analysis metrics.
	PayloadMetricsSleepAnalysis = &healthautoexport.Payload{
		Data: &healthautoexport.PayloadData{
			Metrics: []*healthautoexport.Metric{
				{
					Name:  "sleep_analysis",
					Units: "hr",
					Data: []*healthautoexport.Datapoint{
						{
							Date: mktime("2021-12-18 09:03:36 +0800"),
							Fields: healthautoexport.DatapointFields{
								"asleep":      6.108333333333333,
								"sleepStart":  mktime("2021-12-18 02:21:06 +0800"),
								"sleepEnd":    mktime("2021-12-18 08:57:06 +0800"),
								"sleepSource": "Irvin’s Apple Watch",
								"inBed":       6.809728874299261,
								"inBedStart":  mktime("2021-12-18 02:12:50 +0800"),
								"inBedEnd":    mktime("2021-12-18 09:04:45 +0800"),
								"inBedSource": "iPhone",
							},
						},
					},
				},
			},
		},
	}

	// PayloadWithWorkouts is an example Payload with workouts.
	PayloadWithWorkouts = &healthautoexport.Payload{
		Data: &healthautoexport.PayloadData{
			Workouts: []*healthautoexport.Workout{
				{
					Name:  "Walking",
					Start: mktime("2021-12-24 08:02:43 +0800"),
					End:   mktime("2021-12-24 08:21:53 +0800"),
					Route: []*healthautoexport.RouteDatapoint{
						{
							Lat:       38.8951,
							Lon:       -77.0364,
							Altitude:  8.0276222229003906,
							Timestamp: mktime("2021-12-24 08:04:45 +0800"),
						},
					},
					HeartRateData: []*healthautoexport.DatapointWithUnit{
						{
							Date: mktime("2021-12-24 08:02:47 +0800"),
							QtyWithUnit: healthautoexport.QtyWithUnit{
								Qty:   108,
								Units: "bpm",
							},
						},
					},
					Elevation: &healthautoexport.Elevation{
						Units:   "m",
						Ascent:  16.359999999999999,
						Descent: 0,
					},
					Fields: healthautoexport.WorkoutFields{
						"stepCount": &healthautoexport.QtyWithUnit{
							Qty:   908,
							Units: "steps",
						},
						"activeEnergy": &healthautoexport.QtyWithUnit{
							Qty:   226.21122641832523,
							Units: "kJ",
						},
					},
				},
			},
		},
	}
)

func mktime(ts string) *healthautoexport.Time {
	t, err := time.Parse(healthautoexport.TimeFormat, ts)
	if err != nil {
		panic(err)
	}
	return healthautoexport.NewTime(t)
}
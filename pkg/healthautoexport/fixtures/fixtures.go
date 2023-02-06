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
		Datapoints: []*healthautoexport.Datapoint{
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
					AggregatedSleepAnalyses: []*healthautoexport.AggregatedSleepAnalysis{
						{
							Asleep:      6.108333333333333,
							SleepStart:  mktime("2021-12-18 02:21:06 +0800"),
							SleepEnd:    mktime("2021-12-18 08:57:06 +0800"),
							SleepSource: "Irvin’s Apple Watch",
							InBed:       6.809728874299261,
							InBedStart:  mktime("2021-12-18 02:12:50 +0800"),
							InBedEnd:    mktime("2021-12-18 09:04:45 +0800"),
							InBedSource: "iPhone",
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
						{
							Key: "activeEnergy",
							Value: &healthautoexport.QtyWithUnit{
								Qty:   226.21122641832523,
								Units: "kJ",
							},
						},
						{
							Key: "stepCount",
							Value: &healthautoexport.QtyWithUnit{
								Qty:   908,
								Units: "steps",
							},
						},
					},
				},
			},
		},
	}

	PayloadMetricsSleepAnalysisNonAggregated = &healthautoexport.Payload{
		Data: &healthautoexport.PayloadData{
			Metrics: []*healthautoexport.Metric{
				{
					Name:  "sleep_analysis",
					Units: "hr",
					SleepAnalyses: []*healthautoexport.SleepAnalysis{
						{
							StartDate: mktime("2021-12-18 02:21:06 +0800"),
							EndDate:   mktime("2021-12-18 08:57:06 +0800"),
							Qty:       6.108333333333333,
							Source:    "Irvin's Apple Watch",
							Value:     "Core",
						},
					},
				},
			},
		},
	}

	PayloadMetricsSleepPhases = &healthautoexport.Payload{
		Data: &healthautoexport.PayloadData{
			Metrics: []*healthautoexport.Metric{
				{
					Name:  "sleep_analysis",
					Units: "hr",
					AggregatedSleepAnalyses: []*healthautoexport.AggregatedSleepAnalysis{
						{
							Asleep:     0,
							SleepStart: mktime("2023-01-31 00:23:47 +0800"),
							SleepEnd:   mktime("2023-01-31 08:39:12 +0800"),
							Source:     "Irvin's iPhone|Irvin’s Apple Watch",
							InBed:      8.1450109350019027,
							Core:       3.2999999999999994,
							Awake:      0.11666666666666667,
							Deep:       0.85833333333333339,
							REM:        1.2583333333333333,
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

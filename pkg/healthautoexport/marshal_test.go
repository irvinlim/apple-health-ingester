package healthautoexport_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	jsoniter "github.com/json-iterator/go"

	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

var (
	cmpOptions = []cmp.Option{
		cmpopts.EquateEmpty(),
	}

	payloadWithMetrics = &healthautoexport.Payload{
		Data: &healthautoexport.PayloadData{
			Metrics: []*healthautoexport.Metric{
				{
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
				},
				{
					Name:  "basal_body_temperature",
					Units: "degC",
				},
			},
		},
	}

	payloadMetricsSleepAnalysis = &healthautoexport.Payload{
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

	payloadWithWorkouts = &healthautoexport.Payload{
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

func TestMarshalToString(t *testing.T) {
	tests := []struct {
		name    string
		payload *healthautoexport.Payload
		want    string
		wantErr bool
	}{
		{
			name:    "marshal metrics",
			payload: payloadWithMetrics,
			want:    `{"data":{"metrics":[{"name":"active_energy","units":"kJ","data":[{"qty":0.7685677437484512,"date":"2021-12-24 00:04:00 +0800"},{"qty":0.377848256251549,"date":"2021-12-24 00:05:00 +0800"}]},{"name":"basal_body_temperature","units":"degC","data":null}]}}`,
		},
		{
			name:    "marshal workouts",
			payload: payloadWithWorkouts,
			want:    `{"data":{"workouts":[{"name":"Walking","start":"2021-12-24 08:02:43 +0800","end":"2021-12-24 08:21:53 +0800","heartRateData":[{"qty":108,"date":"2021-12-24 08:02:47 +0800","units":"bpm"}],"elevation":{"units":"m","ascent":16.36,"descent":0},"stepCount":{"qty":908,"units":"steps"},"activeEnergy":{"qty":226.21122641832523,"units":"kJ"},"route":[{"lat":38.8951,"lon":-77.0364,"altitude":8.02762222290039,"timestamp":"2021-12-24 08:04:45 +0800"}],"heartRateRecovery":null}]}}`,
		},
		{
			name:    "marshal sleep analysis custom datapoint fields",
			payload: payloadMetricsSleepAnalysis,
			want: `{
  "data": {
    "metrics": [
      {
        "name": "sleep_analysis",
        "units": "hr",
        "data": [
          {
            "date": "2021-12-18 09:03:36 +0800",
            "asleep": 6.108333333333333,
            "sleepStart": "2021-12-18 02:21:06 +0800",
            "sleepEnd": "2021-12-18 08:57:06 +0800",
            "sleepSource": "Irvin’s Apple Watch",
            "inBed": 6.809728874299261,
            "inBedStart": "2021-12-18 02:12:50 +0800",
            "inBedEnd": "2021-12-18 09:04:45 +0800",
            "inBedSource": "iPhone"
          }
        ]
      }
    ]
  }
}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := healthautoexport.MarshalToString(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalToString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// We don't compare the strings directly due to flakiness of tests.
			// Instead, we unmarshal it back to Payload.
			var result healthautoexport.Payload
			if err := jsoniter.Unmarshal([]byte(got), &result); err != nil {
				t.Errorf("unmarshal error = %v", err)
				return
			}
			if !cmp.Equal(tt.payload, &result, cmpOptions...) {
				t.Errorf("Unmarshal() not equal\ndiff = %v", cmp.Diff(tt.payload, &result, cmpOptions...))
				return
			}
		})
	}
}

func TestUnmarshalFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *healthautoexport.Payload
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid json",
			input:   `{"data": {}`,
			wantErr: true,
		},
		{
			name:  "empty data",
			input: `{"data": {}}`,
			want: &healthautoexport.Payload{
				Data: &healthautoexport.PayloadData{},
			},
		},
		{
			name: "unmarshal metrics",
			want: payloadWithMetrics,
			input: `{
  "data" : {
    "metrics" : [
      {
        "name" : "active_energy",
        "units" : "kJ",
        "data" : [
          {
            "date" : "2021-12-24 00:04:00 +0800",
            "qty" : 0.76856774374845116
          },
          {
            "date" : "2021-12-24 00:05:00 +0800",
            "qty" : 0.377848256251549
          }
        ]
      },
      {
        "data" : [

        ],
        "name" : "basal_body_temperature",
        "units" : "degC"
      }
    ]
  },
  "workouts" : [
  ]
}`,
		},
		{
			name: "unmarshal workouts",
			want: payloadWithWorkouts,
			input: `{
  "data": {
    "workouts" : [
      {
        "stepCount" : {
          "qty" : 908,
          "units" : "steps"
        },
        "name" : "Walking",
        "activeEnergy" : {
          "qty" : 226.21122641832523,
          "units" : "kJ"
        },
        "elevation" : {
          "descent" : 0,
          "ascent" : 16.359999999999999,
          "units" : "m"
        },
        "end" : "2021-12-24 08:21:53 +0800",
        "heartRateData" : [
          {
            "units" : "bpm",
            "qty" : 108,
            "date" : "2021-12-24 08:02:47 +0800"
          }
        ],
        "route" : [
          {
            "altitude" : 8.0276222229003906,
            "lon" : -77.0364,
            "timestamp" : "2021-12-24 08:04:45 +0800",
            "lat" : 38.8951
          }
        ],
        "start" : "2021-12-24 08:02:43 +0800"
      }
    ]
  }
}`,
		},
		{
			name: "unmarshal sleep analysis custom datapoint fields",
			want: payloadMetricsSleepAnalysis,
			input: `{
  "data": {
    "metrics": [
      {
        "name": "sleep_analysis",
        "units": "hr",
        "data": [
          {
            "date": "2021-12-18 09:03:36 +0800",
            "asleep": 6.108333333333333,
            "sleepStart": "2021-12-18 02:21:06 +0800",
            "sleepEnd": "2021-12-18 08:57:06 +0800",
            "sleepSource": "Irvin’s Apple Watch",
            "inBed": 6.809728874299261,
            "inBedStart": "2021-12-18 02:12:50 +0800",
            "inBedEnd": "2021-12-18 09:04:45 +0800",
            "inBedSource": "iPhone"
          }
        ]
      }
    ]
  }
}`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Unmarshal the payload
			got, err := healthautoexport.UnmarshalFromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(tt.want, got, cmpOptions...) {
				t.Errorf("UnmarshalFromString() not equal\ndiff = %v", cmp.Diff(tt.want, got, cmpOptions...))
				return
			}
		})
	}
}

func mktime(ts string) *healthautoexport.Time {
	t, err := time.Parse(healthautoexport.TimeFormat, ts)
	if err != nil {
		panic(err)
	}
	return healthautoexport.NewTime(t)
}

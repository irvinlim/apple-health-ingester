package influxdb_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/stretchr/testify/assert"

	"github.com/irvinlim/apple-health-ingester/pkg/backends"
	"github.com/irvinlim/apple-health-ingester/pkg/backends/influxdb"
	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport/fixtures"
)

const (
	targetName = "test"
)

func TestBackend(t *testing.T) {
	tests := []struct {
		name         string
		payload      *healthautoexport.Payload
		wantMetrics  []string
		wantWorkouts []string
	}{
		{
			name: "write active energy metrics",
			payload: &healthautoexport.Payload{
				Data: &healthautoexport.PayloadData{
					Metrics: []*healthautoexport.Metric{fixtures.MetricActiveEnergy},
				},
			}, wantMetrics: []string{
				"active_energy_kJ,target_name=test qty=0.7685677437484512 1640275440000000000",
				"active_energy_kJ,target_name=test qty=0.377848256251549 1640275500000000000",
			},
		},
		{
			name: "write empty metrics",
			payload: &healthautoexport.Payload{
				Data: &healthautoexport.PayloadData{
					Metrics: []*healthautoexport.Metric{fixtures.MetricBasalBodyTemperatureNoData},
				},
			}, wantMetrics: []string{},
		},
		{
			name:    "write aggregated sleep analysis metrics",
			payload: fixtures.PayloadMetricsSleepAnalysis,
			wantMetrics: []string{
				`sleep_analysis_aggregated,target_name=test,source=Irvin’s\ Apple\ Watch,value=asleep state=1u 1639765266000000000`,
				`sleep_analysis_aggregated,target_name=test,source=Irvin’s\ Apple\ Watch,value=asleep qty=6.108333333333333,state=0u 1639789026000000000`,
				`sleep_analysis_aggregated,target_name=test,source=iPhone,value=inBed state=1u 1639764770000000000`,
				`sleep_analysis_aggregated,target_name=test,source=iPhone,value=inBed qty=6.809728874299261,state=0u 1639789485000000000`,
			},
		},
		{
			name:    "write non aggregated sleep analysis metrics",
			payload: fixtures.PayloadMetricsSleepAnalysisNonAggregated,
			wantMetrics: []string{
				`sleep_analysis_detailed,target_name=test,source=Irvin's\ Apple\ Watch,value=Core state=1u 1639765266000000000`,
				`sleep_analysis_detailed,target_name=test,source=Irvin's\ Apple\ Watch,value=Core qty=6.108333333333333,state=0u 1639789026000000000`,
			},
		},
		{
			name:    "write workouts",
			payload: fixtures.PayloadWithWorkouts,
			wantWorkouts: []string{
				`workout,target_name=test,workout_name=Walking activeEnergy_kJ=226.21122641832523,duration_min=19.166666666666668,elevation_ascent_m=16.36,elevation_descent_m=0,stepCount_steps=908 1640304163000000000`,
				`route,target_name=test,workout_name=Walking altitude=8.02762222290039,lat=38.8951,lon=-77.0364 1640304285000000000`,
				`heart_rate_data_bpm,target_name=test,workout_name=Walking qty=108 1640304167000000000`,
				`workout,target_name=test,workout_name=Walking activeEnergy_kJ=226.21122641832523,duration_min=19.166666666666668,elevation_ascent_m=16.36,elevation_descent_m=0,stepCount_steps=908 1640304163000000000`,
				`route,target_name=test,workout_name=Walking altitude=8.02762222290039,lat=38.8951,lon=-77.0364 1640304285000000000`,
				`heart_rate_data_bpm,target_name=test,workout_name=Walking qty=108 1640304167000000000`,
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			test := NewBackendTest(t)
			test.AssertWriteMetrics(t, tt.payload, tt.wantMetrics)
			test.AssertWriteWorkouts(t, tt.payload, tt.wantWorkouts)
			test.client.Reset()
		})
	}
}

type BackendTest struct {
	backend backends.Backend
	client  *influxdb.MockClient
}

func NewBackendTest(t *testing.T) *BackendTest {
	client := influxdb.NewMockClient()
	backend, err := influxdb.NewBackend(client)
	if err != nil {
		t.Fatalf("init backend failed: %v", err)
	}
	return &BackendTest{backend: backend, client: client}
}

func (b *BackendTest) AssertWriteMetrics(t *testing.T, payload *healthautoexport.Payload, expected []string) {
	assert.NoError(t, b.backend.Write(payload, targetName), "backend write error")
	b.assertPoints(t, expected, b.client.ReadMetrics())
}

func (b *BackendTest) AssertWriteWorkouts(t *testing.T, payload *healthautoexport.Payload, expected []string) {
	assert.NoError(t, b.backend.Write(payload, targetName), "backend write error")
	b.assertPoints(t, expected, b.client.ReadWorkouts())
}

func (b *BackendTest) assertPoints(t *testing.T, expectedLines []string, actual []*write.Point) {
	if len(expectedLines) != len(actual) {
		assert.Equal(t, len(expectedLines), len(actual), fmt.Sprintf("points are not equal length: %v vs %v", expectedLines, actual))
		return
	}
	for i := 0; i < len(expectedLines); i++ {
		b.assertPoint(t, expectedLines[i], actual[i], fmt.Sprintf("point %v is not equal", i))
	}
}

func (b *BackendTest) assertPoint(t *testing.T, expectedLine string, actual *write.Point, msgAndArgs ...interface{}) {
	actualLine := write.PointToLineProtocol(actual, time.Nanosecond)
	actualLine = strings.TrimSuffix(actualLine, "\n")
	assert.Equalf(t, expectedLine, actualLine, "%v: diff = %v", fmt.Sprint(msgAndArgs...), cmp.Diff(expectedLine, actualLine))
}

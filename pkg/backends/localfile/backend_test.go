package localfile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

func TestBackend_MergesIncrementalWrites(t *testing.T) {
	originalMetricsPath := metricsPath
	metricsPath = t.TempDir()
	t.Cleanup(func() {
		metricsPath = originalMetricsPath
	})

	backend, err := NewBackend()
	require.NoError(t, err)

	firstWrite := newPayload(
		newDatapoint(t, 1, "2021-12-24 00:04:00 +0800"),
		newDatapoint(t, 2, "2021-12-24 00:05:00 +0800"),
	)
	secondWrite := newPayload(
		newDatapoint(t, 3, "2021-12-24 00:06:00 +0800"),
	)

	require.NoError(t, backend.Write(firstWrite, "test"))
	require.NoError(t, backend.Write(secondWrite, "test"))

	metricFile, err := backend.loadMetricFile(metricsPath + "/test_swimming_distance_m.json")
	require.NoError(t, err)
	require.Len(t, metricFile.Data, 3)
	assert.Equal(t, healthautoexport.Qty(1), metricFile.Data[0].Qty)
	assert.Equal(t, healthautoexport.Qty(2), metricFile.Data[1].Qty)
	assert.Equal(t, healthautoexport.Qty(3), metricFile.Data[2].Qty)
}

func newPayload(datapoints ...*healthautoexport.Datapoint) *healthautoexport.Payload {
	return &healthautoexport.Payload{
		Data: &healthautoexport.PayloadData{
			Metrics: []*healthautoexport.Metric{
				{
					Name:       "swimming_distance",
					Units:      "m",
					Datapoints: datapoints,
				},
			},
		},
	}
}

func newDatapoint(t *testing.T, qty healthautoexport.Qty, timestamp string) *healthautoexport.Datapoint {
	t.Helper()
	parsed, err := time.Parse(healthautoexport.TimeFormat, timestamp)
	require.NoError(t, err)
	date := healthautoexport.NewTime(parsed)
	return &healthautoexport.Datapoint{Qty: qty, Date: &date}
}

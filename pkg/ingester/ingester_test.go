package ingester_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/irvinlim/apple-health-ingester/pkg/backends/noop"
	"github.com/irvinlim/apple-health-ingester/pkg/ingester"
)

const (
	processingDelay = time.Millisecond * 10
	payload         = `{"data":{"metrics":[{"name":"active_energy","units":"kJ","data":[{"qty":0.7685677437484512,"date":"2021-12-24 00:04:00 +0800"},{"qty":0.377848256251549,"date":"2021-12-24 00:05:00 +0800"}]},{"name":"basal_body_temperature","units":"degC","data":null}]}}`
)

func TestIngester(t *testing.T) {
	ingest := ingester.NewIngester()
	backend := noop.NewBackend()
	assert.NoError(t, ingest.AddBackend(backend))

	// Cannot ingest before start
	assert.Error(t, ingest.IngestFromString("{}", backend.Name(), ""))
	ingest.Start()

	// No named backend
	assert.Error(t, ingest.IngestFromString("{}", "invalid", ""))

	var expectedWrites int

	// Ingest proper payload
	assert.NoError(t, ingest.IngestFromString(payload, backend.Name(), ""))
	time.Sleep(processingDelay)
	expectedWrites++
	assert.Equal(t, expectedWrites, len(backend.Writes))

	// Backend has error, writes should not increase
	backend.ShouldError = true
	assert.NoError(t, ingest.IngestFromString(payload, backend.Name(), ""))
	time.Sleep(processingDelay)
	assert.Equal(t, expectedWrites, len(backend.Writes))

	// Let error recover after a few seconds
	done := make(chan struct{})
	go func() {
		time.Sleep(time.Second * 3)
		backend.ShouldError = false
		close(done)
	}()

	// It should be successful within a set time limit
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	ticker := time.NewTicker(time.Millisecond * 200)
	defer ticker.Stop()

tickerLoop:
	for {
		select {
		case <-ctx.Done():
			assert.Fail(t, "timeout exceeded before write was successful")
			return
		case <-ticker.C:
			// Finally successful
			if len(backend.Writes) == expectedWrites+1 {
				expectedWrites++
				break tickerLoop
			}
		}
	}

	// Ingest many payloads before shutdown
	backend.ShouldError = false
	for i := 0; i < 50; i++ {
		assert.NoError(t, ingest.IngestFromString(payload, backend.Name(), ""))
		expectedWrites++
	}
	ingest.Shutdown()
	assert.Equal(t, expectedWrites, len(backend.Writes))
}

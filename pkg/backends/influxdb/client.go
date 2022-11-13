package influxdb

import (
	"context"
	"sync"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"

	apierrors "github.com/irvinlim/apple-health-ingester/pkg/errors"
)

// Client knows how to write to an InfluxDB database.
type Client interface {
	WriteMetrics(ctx context.Context, point ...*write.Point) error
	WriteWorkouts(ctx context.Context, point ...*write.Point) error
}

// clientImpl is the real implementation of Client.
type clientImpl struct {
	client             influxdb2.Client
	orgName            string
	metricsBucketName  string
	workoutsBucketName string
}

var _ Client = (*clientImpl)(nil)

// NewClient returns a real influxdb Client initialized from flags.
func NewClient() (Client, error) {
	client, err := NewInfluxDBClient()
	if err != nil {
		return nil, err
	}
	impl := &clientImpl{
		client:             client,
		orgName:            orgName,
		metricsBucketName:  metricsBucketName,
		workoutsBucketName: workoutsBucketName,
	}
	return impl, nil
}

func (c *clientImpl) WriteMetrics(ctx context.Context, point ...*write.Point) error {
	if err := c.client.WriteAPIBlocking(c.orgName, c.metricsBucketName).WritePoint(ctx, point...); err != nil {
		return apierrors.WrapRetryableWrite(err)
	}
	return nil
}

func (c *clientImpl) WriteWorkouts(ctx context.Context, point ...*write.Point) error {
	if err := c.client.WriteAPIBlocking(c.orgName, c.workoutsBucketName).WritePoint(ctx, point...); err != nil {
		return apierrors.WrapRetryableWrite(err)
	}
	return nil
}

// MockClient is a mock implementation of Client.
type MockClient struct {
	buckets map[string][]*write.Point
	mu      sync.RWMutex
}

var _ Client = (*MockClient)(nil)

func NewMockClient() *MockClient {
	return &MockClient{
		buckets: make(map[string][]*write.Point),
	}
}

func (m *MockClient) WriteMetrics(_ context.Context, point ...*write.Point) error {
	return m.writePoints("metrics", point...)
}

func (m *MockClient) WriteWorkouts(_ context.Context, point ...*write.Point) error {
	return m.writePoints("workouts", point...)
}

func (m *MockClient) writePoints(bucket string, point ...*write.Point) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.buckets[bucket]; !ok {
		m.buckets[bucket] = make([]*write.Point, 0, len(point))
	}
	m.buckets[bucket] = append(m.buckets[bucket], point...)
	return nil
}

func (m *MockClient) ReadMetrics() []*write.Point {
	return m.readPoints("metrics")
}

func (m *MockClient) ReadWorkouts() []*write.Point {
	return m.readPoints("workouts")
}

func (m *MockClient) readPoints(bucket string) []*write.Point {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.buckets[bucket]
}

func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buckets = nil
}

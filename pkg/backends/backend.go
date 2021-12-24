package backends

import (
	"k8s.io/client-go/util/workqueue"

	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

var (
	defaultRateLimiter = workqueue.DefaultItemBasedRateLimiter()
)

// Backend is implemented by downstream ingester backend implementations.
type Backend interface {
	Name() string
	Write(payload *healthautoexport.Payload, targetName string) error
}

// BackendQueue is a type that composes a Backend and a workqueue.
type BackendQueue struct {
	Backend
	Queue workqueue.RateLimitingInterface
}

func NewBackendWithQueue(backend Backend) *BackendQueue {
	return &BackendQueue{
		Backend: backend,
		Queue:   workqueue.NewNamedRateLimitingQueue(defaultRateLimiter, backend.Name()),
	}
}

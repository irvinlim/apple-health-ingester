package ingester

import (
	"bytes"
	"fmt"
	"io"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/irvinlim/apple-health-ingester/pkg/backends"
	apierrors "github.com/irvinlim/apple-health-ingester/pkg/errors"
	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

// Ingester is a generic ingester for Health Auto Export data.
type Ingester struct {
	backends    map[string]*backends.BackendQueue
	started     bool
	backendsMtx sync.RWMutex
	quit        *sync.WaitGroup
}

func NewIngester() *Ingester {
	return &Ingester{
		backends: make(map[string]*backends.BackendQueue),
		quit:     &sync.WaitGroup{},
	}
}

func (i *Ingester) AddBackend(backend backends.Backend) error {
	if i.started {
		return errors.New("cannot add backend when already started")
	}
	i.backendsMtx.Lock()
	defer i.backendsMtx.Unlock()
	i.backends[backend.Name()] = backends.NewBackendWithQueue(backend)
	i.quit.Add(1)
	return nil
}

func (i *Ingester) ListBackends() []backends.Backend {
	i.backendsMtx.RLock()
	defer i.backendsMtx.RUnlock()
	bs := make([]backends.Backend, 0, len(i.backends))
	for _, backend := range i.backends {
		bs = append(bs, backend.Backend)
	}
	return bs
}

// Start the ingester to perform background work asynchronously.
func (i *Ingester) Start() {
	for _, backend := range i.backends {
		backend := backend
		go i.processQueue(backend)
	}
	i.started = true
}

// Shutdown begins graceful quit of the ingester, and blocks until all
// background ingestion work has been successfully completed.
func (i *Ingester) Shutdown() {
	i.backendsMtx.Lock()
	defer i.backendsMtx.Unlock()

	// Shutdown all queues.
	var drained sync.WaitGroup
	drained.Add(len(i.backends))
	for _, backend := range i.backends {
		go func(backend *backends.BackendQueue) {
			defer drained.Done()
			backend.Queue.ShutDownWithDrain()
		}(backend)
	}

	// Wait for all queues to be drained and finish processing.
	drained.Wait()

	// Block until all queues have terminated.
	i.quit.Wait()
}

// Ingest ingests the payload from io.Reader into the named backend.
// All processing is done asynchronously.
func (i *Ingester) Ingest(r io.Reader, name string, target string) error {
	if !i.started {
		return errors.New("ingester is not yet started")
	}

	i.backendsMtx.RLock()
	defer i.backendsMtx.RUnlock()

	backend, ok := i.backends[name]
	if !ok {
		return fmt.Errorf("invalid backend %v", name)
	}

	payload, err := healthautoexport.Unmarshal(r)
	if err != nil {
		return errors.Wrapf(err, "unmarshal error")
	}

	// Augment with target name.
	payloadWithTarget := &PayloadWithTarget{
		Payload:    payload,
		TargetName: target,
	}

	backend.Queue.Add(payloadWithTarget)
	return nil
}

// IngestFromString ingests the payload from a string into the named backend.
// All processing is done asynchronously.
func (i *Ingester) IngestFromString(s string, name, target string) error {
	return i.Ingest(bytes.NewBufferString(s), name, target)
}

// processQueue will process items from the workqueue, writing into the backend
// one at a time. If a write error is encountered, the write will be retried
// indefinitely with a backoff. Items are also not guaranteed to be processed in
// order due to the above behaviour.
func (i *Ingester) processQueue(backend *backends.BackendQueue) {
	defer i.quit.Done()

	logger := log.WithField("backend", backend.Name())
	for {
		item, shutdown := backend.Queue.Get()
		if shutdown {
			return
		}

		startTime := time.Now()
		err := i.processWriteItem(item, backend)
		logger = logger.WithField("elapsed", time.Since(startTime))

		if err != nil {
			if apierrors.IsRetryableWrite(err) {
				backend.Queue.AddRateLimited(item)
				logger = logger.WithField("retries", backend.Queue.NumRequeues(item))
			}

			logger.WithError(err).Error("write data error")
		} else {
			logger.Info("write data success")
		}

		backend.Queue.Done(item)
	}
}

func (i *Ingester) processWriteItem(item interface{}, backend backends.Backend) (err error) {
	payload, ok := item.(*PayloadWithTarget)
	if !ok {
		return fmt.Errorf("cannot convert to *Payload")
	}

	// Handle panics in backend implementations.
	defer func() {
		if r := recover(); r != nil {
			log.WithFields(log.Fields{
				"backend": backend.Name(),
				"payload": payload,
			}).Error("recovered from panic in backend:\n" + string(debug.Stack()))
			err = errors.New("recovered from panic")
		}
	}()

	if err := backend.Write(payload.Payload, payload.TargetName); err != nil {
		return errors.Wrapf(err, "cannot write payload to database")
	}

	return nil
}

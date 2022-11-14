package noop

import (
	"github.com/irvinlim/apple-health-ingester/pkg/backends"
	apierrors "github.com/irvinlim/apple-health-ingester/pkg/errors"
	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

type Backend struct {
	Writes      []*healthautoexport.Payload
	ShouldError bool
	ShouldPanic bool
}

var _ backends.Backend = &Backend{}

func NewBackend() *Backend {
	return &Backend{}
}

func (b *Backend) Name() string {
	return "Noop"
}

func (b *Backend) Write(payload *healthautoexport.Payload, _ string) error {
	if b.ShouldPanic {
		panic("backend panic during write")
	}
	if b.ShouldError {
		return apierrors.NewRetryableWriteError()
	}
	b.Writes = append(b.Writes, payload)
	return nil
}

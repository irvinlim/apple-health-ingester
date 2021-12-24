package noop

import (
	"errors"

	"github.com/irvinlim/apple-health-ingester/pkg/backends"
	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

type Backend struct {
	Writes      []*healthautoexport.Payload
	ShouldError bool
}

var _ backends.Backend = &Backend{}

func NewBackend() *Backend {
	return &Backend{}
}

func (b *Backend) Name() string {
	return "Noop"
}

func (b *Backend) Write(payload *healthautoexport.Payload, _ string) error {
	if b.ShouldError {
		return errors.New("noop error")
	}
	b.Writes = append(b.Writes, payload)
	return nil
}

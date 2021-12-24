package ingester

import (
	"github.com/irvinlim/apple-health-ingester/pkg/healthautoexport"
)

// PayloadWithTarget augments a target name to the payload, which helps to identify a
// person/device that the data originated from. This is useful when handling
// data from multiple users' devices.
type PayloadWithTarget struct {
	*healthautoexport.Payload
	TargetName string
}

package test_helpers

import (
	"encoding/json"
	"time"

	"code.cloudfoundry.org/gorouter/metrics/reporter"
	"code.cloudfoundry.org/gorouter/route"
	"code.cloudfoundry.org/gorouter/stats"
)

type NullVarz struct{}

func (_ NullVarz) MarshalJSON() ([]byte, error)                               { return json.Marshal(nil) }
func (_ NullVarz) ActiveApps() *stats.ActiveApps                              { return stats.NewActiveApps() }
func (_ NullVarz) CaptureBadRequest()                                         {}
func (_ NullVarz) CaptureBadGateway()                                         {}
func (_ NullVarz) CaptureRoutingRequest(b *route.Endpoint)                    {}
func (_ NullVarz) CaptureRoutingResponse(*route.Endpoint, int, time.Duration) {}
func (_ NullVarz) CaptureRegistryMessage(msg reporter.ComponentTagged)        {}

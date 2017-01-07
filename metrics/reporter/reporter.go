package reporter

import (
	"time"

	"code.cloudfoundry.org/gorouter/route"
)

//go:generate counterfeiter -o fakes/fake_reporter.go . ProxyReporter
type ProxyReporter interface {
	CaptureBadRequest()
	CaptureBadGateway()
	CaptureRoutingRequest(b *route.Endpoint)
	CaptureRoutingResponse(b *route.Endpoint, statusCode int, d time.Duration)
}

type ComponentTagged interface {
	Component() string
}

//go:generate counterfeiter -o fakes/fake_registry_reporter.go . RouteRegistryReporter
type RouteRegistryReporter interface {
	CaptureRouteStats(totalRoutes int, msSinceLastUpdate uint64)
	CaptureLookupTime(t time.Duration)
	CaptureRegistryMessage(msg ComponentTagged)
}

package reporter

import (
	"net/http"
	"time"

	"code.cloudfoundry.org/gorouter/route"
)

//go:generate counterfeiter -o fakes/fake_reporter.go . ProxyReporter
type ProxyReporter interface {
	CaptureBadRequest(req *http.Request)
	CaptureBadGateway(req *http.Request)
	CaptureRoutingRequest(b *route.Endpoint, req *http.Request)
	CaptureRoutingResponse(b *route.Endpoint, res *http.Response, t time.Time, d time.Duration)
	CaptureRouteServiceResponse(b *route.Endpoint, res *http.Response, t time.Time, d time.Duration)
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

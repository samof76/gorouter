package metrics

import (
	"net/http"
	"time"

	"code.cloudfoundry.org/gorouter/metrics/reporter"
	"code.cloudfoundry.org/gorouter/route"
)

type CompositeReporter struct {
	first  reporter.ProxyReporter
	second reporter.ProxyReporter
}

func NewCompositeReporter(first, second reporter.ProxyReporter) reporter.ProxyReporter {
	return &CompositeReporter{
		first:  first,
		second: second,
	}
}

func (c *CompositeReporter) CaptureBadRequest(req *http.Request) {
	c.first.CaptureBadRequest(req)
	c.second.CaptureBadRequest(req)
}

func (c *CompositeReporter) CaptureBadGateway(req *http.Request) {
	c.first.CaptureBadGateway(req)
	c.second.CaptureBadGateway(req)
}

func (c *CompositeReporter) CaptureRoutingRequest(b *route.Endpoint, req *http.Request) {
	c.first.CaptureRoutingRequest(b, req)
	c.second.CaptureRoutingRequest(b, req)
}

func (c *CompositeReporter) CaptureRouteServiceResponse(b *route.Endpoint, res *http.Response, t time.Time, d time.Duration) {
	c.first.CaptureRouteServiceResponse(b, res, t, d)
	c.second.CaptureRouteServiceResponse(b, res, t, d)
}

func (c *CompositeReporter) CaptureRoutingResponse(b *route.Endpoint, res *http.Response, t time.Time, d time.Duration) {
	c.first.CaptureRoutingResponse(b, res, t, d)
	c.second.CaptureRoutingResponse(b, res, t, d)
}

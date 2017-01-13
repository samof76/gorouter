package proxy_test

import (
	"bytes"
	"crypto/tls"
	"net/http/httptest"
	"time"

	fakelogger "code.cloudfoundry.org/gorouter/access_log/fakes"
	"code.cloudfoundry.org/gorouter/logger"
	"code.cloudfoundry.org/gorouter/metrics/reporter/fakes"
	"code.cloudfoundry.org/gorouter/proxy"
	"code.cloudfoundry.org/gorouter/proxy/test_helpers"
	"code.cloudfoundry.org/gorouter/registry"
	"code.cloudfoundry.org/gorouter/route"
	"code.cloudfoundry.org/gorouter/test_util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Proxy Unit tests", func() {
	var (
		proxyObj         proxy.Proxy
		fakeAccessLogger *fakelogger.FakeAccessLogger
		logger           logger.Logger
	)

	Context("ServeHTTP", func() {
		BeforeEach(func() {
			tlsConfig := &tls.Config{
				CipherSuites:       conf.CipherSuites,
				InsecureSkipVerify: conf.SkipSSLValidation,
			}

			fakeAccessLogger = &fakelogger.FakeAccessLogger{}

			logger = test_util.NewTestZapLogger("test")
			r = registry.NewRouteRegistry(logger, conf, new(fakes.FakeRouteRegistryReporter))

			proxyObj = proxy.NewProxy(proxy.ProxyArgs{
				EndpointTimeout:      conf.EndpointTimeout,
				Ip:                   conf.Ip,
				TraceKey:             conf.TraceKey,
				Registry:             r,
				Reporter:             test_helpers.NullVarz{},
				Logger:               logger,
				AccessLogger:         fakeAccessLogger,
				SecureCookies:        conf.SecureCookies,
				TLSConfig:            tlsConfig,
				RouteServiceEnabled:  conf.RouteServiceEnabled,
				RouteServiceTimeout:  conf.RouteServiceTimeout,
				Crypto:               crypto,
				CryptoPrev:           cryptoPrev,
				HealthCheckUserAgent: "HTTP-Monitor/1.1",
			})

			r.Register(route.Uri("some-app"), &route.Endpoint{})
		})

		Context("when backend fails to respond", func() {
			It("logs the error and associated endpoint", func() {
				body := []byte("some body")
				req := test_util.NewRequest("GET", "some-app", "/", bytes.NewReader(body))
				resp := httptest.NewRecorder()

				proxyObj.ServeHTTP(resp, req)

				Eventually(logger).Should(Say("route-endpoint"))
				Eventually(logger).Should(Say("error"))
			})
		})

		Context("Log response time", func() {
			It("logs response time for HTTP connections", func() {
				body := []byte("some body")
				req := test_util.NewRequest("GET", "some-app", "/", bytes.NewReader(body))
				resp := httptest.NewRecorder()

				proxyObj.ServeHTTP(resp, req)
				Expect(fakeAccessLogger.LogCallCount()).To(Equal(1))
				Expect(fakeAccessLogger.LogArgsForCall(0).FinishedAt).NotTo(Equal(time.Time{}))
			})

			It("logs response time for TCP connections", func() {
				req := test_util.NewRequest("UPGRADE", "some-app", "/", nil)
				req.Header.Set("Upgrade", "tcp")
				req.Header.Set("Connection", "upgrade")
				resp := httptest.NewRecorder()

				proxyObj.ServeHTTP(resp, req)
				Expect(fakeAccessLogger.LogCallCount()).To(Equal(1))
				Expect(fakeAccessLogger.LogArgsForCall(0).FinishedAt).NotTo(Equal(time.Time{}))
			})

			It("logs response time for Web Socket connections", func() {
				req := test_util.NewRequest("UPGRADE", "some-app", "/", nil)
				req.Header.Set("Upgrade", "websocket")
				req.Header.Set("Connection", "upgrade")
				resp := httptest.NewRecorder()

				proxyObj.ServeHTTP(resp, req)
				Expect(fakeAccessLogger.LogCallCount()).To(Equal(1))
				Expect(fakeAccessLogger.LogArgsForCall(0).FinishedAt).NotTo(Equal(time.Time{}))
			})
		})
	})
})

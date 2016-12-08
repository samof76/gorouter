package proxy_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	registryfakes "code.cloudfoundry.org/gorouter/registry/fakes"
	"code.cloudfoundry.org/gorouter/route"
	"code.cloudfoundry.org/routing-api/models"

	"code.cloudfoundry.org/gorouter/proxy"
	"code.cloudfoundry.org/gorouter/test_util"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/urfave/negroni"
)

var _ = FDescribe("FastReverseProxy", func() {
	var (
		handler            negroni.Handler
		testServer         *ghttp.Server
		testServerRoute    string
		testServerEndpoint *route.Endpoint
		nextCalled         bool
		resp               *httptest.ResponseRecorder
		reg                *registryfakes.FakeRegistryInterface
		logger             lager.Logger
	)

	nextHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Body != nil {
			_, err := ioutil.ReadAll(req.Body)
			Expect(err).NotTo(HaveOccurred())
		}

		rw.WriteHeader(http.StatusTeapot)

		nextCalled = true
	})

	BeforeEach(func() {
		testServer = ghttp.NewServer()
		resp = httptest.NewRecorder()

		testServerRoute = "foo.com"

		logger = lagertest.NewTestLogger("fastreverseproxy-test")

		// Set up route registry
		reg = new(registryfakes.FakeRegistryInterface)
		pool := route.NewPool(1*time.Second, "")
		host, strPort, err := net.SplitHostPort(testServer.Addr())
		Expect(err).ToNot(HaveOccurred())
		port, err := strconv.Atoi(strPort)
		Expect(err).ToNot(HaveOccurred())
		testServerEndpoint = route.NewEndpoint("foo", host, uint16(port), "", "", nil, -1, "", models.ModificationTag{})
		_ = pool.Put(testServerEndpoint)
		reg.LookupStub = func(uri route.Uri) *route.Pool {
			if uri.String() == testServerRoute {
				return pool
			}
			return nil
		}

		handler = proxy.NewFastReverseProxy(reg)

		nextCalled = false
	})

	AfterEach(func() {
		Expect(nextCalled).To(BeTrue())
	})

	It("routes the request to the correct backend", func() {
		testBody := "Successfully got foo."
		testServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/foo"),
				ghttp.RespondWith(200, testBody),
			),
		)
		req := test_util.NewRequest("GET", testServerRoute, "/foo", nil)
		handler.ServeHTTP(resp, req, nextHandler)

		Expect(testServer.ReceivedRequests()).To(HaveLen(1))
		Expect(resp.Code).To(Equal(200))

		Expect(resp.Body.String()).To(Equal(testBody))
	})

	It("transparently sends end-to-end headers to the backend", func() {
		testServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/"),
				ghttp.VerifyHeaderKV("X-Foo-Header", "foo"),
				ghttp.VerifyHeaderKV("X-Bar-Header", "bar"),
				ghttp.RespondWith(200, nil),
			),
		)
		req := test_util.NewRequest("GET", testServerRoute, "/", nil)
		req.Header.Add("X-Foo-Header", "foo")
		req.Header.Add("X-Bar-Header", "bar")
		handler.ServeHTTP(resp, req, nextHandler)

		Expect(testServer.ReceivedRequests()).To(HaveLen(1))
	})

	It("transparently returns end-to-end headers from the backend", func() {
		testServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/"),
				func(w http.ResponseWriter, req *http.Request) {
					w.Header().Add("X-Foo-Header", "foo")
					w.Header().Add("X-Bar-Header", "bar")
					w.WriteHeader(200)
				},
			),
		)
		req := test_util.NewRequest("GET", testServerRoute, "/", nil)
		handler.ServeHTTP(resp, req, nextHandler)

		Expect(testServer.ReceivedRequests()).To(HaveLen(1))
		Expect(resp.Code).To(Equal(200))
		Expect(resp.Header().Get("X-Foo-Header")).To(Equal("foo"))
		Expect(resp.Header().Get("X-Bar-Header")).To(Equal("bar"))
	})

	XIt("transparently returns trailers from the backend", func() {
		testServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/"),
				func(w http.ResponseWriter, req *http.Request) {
					w.Header().Set("Trailer", "X-Foo-Trailer")
					w.WriteHeader(200)
					flusher, ok := w.(http.Flusher)
					Expect(ok).To(BeTrue(), "Expected http.ResponseWriter to be an http.Flusher")
					for i := 1; i <= 2; i++ {
						fmt.Fprintf(w, "Chunk #%d\n", i)
						flusher.Flush() // Trigger "chunked" encoding and send a chunk...
					}
					fmt.Fprintf(w, "\r\n")
					flusher.Flush() // Trigger "chunked" encoding and send a chunk...

					w.Header().Add("X-Foo-Trailer", "foo")
				},
			),
			func(w http.ResponseWriter, req *http.Request) {
			},
		)
		req := test_util.NewRequest("POST", testServerRoute, "/", nil)
		handler.ServeHTTP(resp, req, nextHandler)

		Expect(resp.Code).To(Equal(200))
		Expect(testServer.ReceivedRequests()).To(HaveLen(1))
		// Expect(resp.Header().Get("X-Foo-Trailer")).To(Equal("foo"))
		// Expect(resp.Header().Get("X-Bar-Trailer")).To(Equal("bar"))
	})

	It("strips hop-by-hop headers from the incoming request", func() {
		testServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/"),
				func(w http.ResponseWriter, req *http.Request) {
					for _, h := range proxy.HopHeaders {
						key := http.CanonicalHeaderKey(h)
						Expect(req.Header).ToNot(HaveKey(key), "Found unwanted key `%s` in request", key)
					}
				},
				ghttp.RespondWith(200, nil),
			),
		)
		req := test_util.NewRequest("GET", testServerRoute, "/", nil)
		for _, h := range proxy.HopHeaders {
			req.Header.Add(h, "some-value")
		}
		handler.ServeHTTP(resp, req, nextHandler)

		Expect(testServer.ReceivedRequests()).To(HaveLen(1))
	})

	It("strips hop-by-hop headers from the response", func() {
		testServer.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/"),
				func(w http.ResponseWriter, req *http.Request) {
					for _, h := range proxy.HopHeaders {
						w.Header().Add(h, "some-value")
					}
				},
			),
		)
		req := test_util.NewRequest("GET", testServerRoute, "/", nil)
		handler.ServeHTTP(resp, req, nextHandler)

		Expect(testServer.ReceivedRequests()).To(HaveLen(1))
		Expect(resp.Code).To(Equal(200))
		for _, h := range proxy.HopHeaders {
			Expect(resp.Header()).ToNot(HaveKey(h))
		}
	})

	Context("when a connection attempt to a backend fails", func() {
		BeforeEach(func() {
			pool := route.NewPool(1*time.Second, "")
			badEndpoint1 := route.NewEndpoint("foo", "192.0.2.1", uint16(80), "", "", nil, -1, "", models.ModificationTag{})
			badEndpoint2 := route.NewEndpoint("foo", "192.0.2.2", uint16(80), "", "", nil, -1, "", models.ModificationTag{})
			_ = pool.Put(badEndpoint1)
			_ = pool.Put(badEndpoint2)
			_ = pool.Put(testServerEndpoint)
			reg.LookupStub = func(uri route.Uri) *route.Pool {
				if uri.String() == testServerRoute {
					return pool
				}
				return nil
			}
		})
		It("retries the connection with other backends", func() {
			testServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/"),
					ghttp.RespondWith(200, nil),
				),
			)
			req := test_util.NewRequest("GET", testServerRoute, "/", nil)
			for _, h := range proxy.HopHeaders {
				req.Header.Add(h, "some-value")
			}
			handler.ServeHTTP(resp, req, nextHandler)

			Expect(testServer.ReceivedRequests()).To(HaveLen(1))
		})
	})
})

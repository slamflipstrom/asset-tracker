package telemetry

import (
	"expvar"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var (
	apiRequestsTotal           = expvar.NewInt("api_requests_total")
	apiRequestsErrorsTotal     = expvar.NewInt("api_requests_errors_total")
	apiRequestLatencyMsTotal   = expvar.NewInt("api_request_latency_ms_total")
	apiRequestLatencySamples   = expvar.NewInt("api_request_latency_samples_total")
	apiRequestsByRoute         = expvar.NewMap("api_requests_by_route")
	apiRequestErrorsByRoute    = expvar.NewMap("api_request_errors_by_route")
	wsConnectionsActive        = expvar.NewInt("ws_connections_active")
	wsConnectionsTotal         = expvar.NewInt("ws_connections_total")
	wsAuthFailuresTotal        = expvar.NewInt("ws_auth_failures_total")
	wsSessionInitFailuresTotal = expvar.NewInt("ws_session_init_failures_total")
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// APIRequestMetricsMiddleware records request volume, error rate, and latency for /api/v1 routes.
func APIRequestMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		route := requestRoute(r)
		key := strings.TrimSpace(r.Method + " " + route)
		if key == "" {
			key = r.Method + " /unknown"
		}

		apiRequestsTotal.Add(1)
		apiRequestsByRoute.Add(key, 1)

		if recorder.status >= http.StatusBadRequest {
			apiRequestsErrorsTotal.Add(1)
			apiRequestErrorsByRoute.Add(key, 1)
		}

		apiRequestLatencyMsTotal.Add(time.Since(start).Milliseconds())
		apiRequestLatencySamples.Add(1)
	})
}

func requestRoute(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := strings.TrimSpace(rctx.RoutePattern()); pattern != "" {
			return pattern
		}
	}
	return strings.TrimSpace(r.URL.Path)
}

func WSConnectionOpened() {
	wsConnectionsTotal.Add(1)
	wsConnectionsActive.Add(1)
}

func WSConnectionClosed() {
	wsConnectionsActive.Add(-1)
}

func WSAuthFailure() {
	wsAuthFailuresTotal.Add(1)
}

func WSSessionInitFailure() {
	wsSessionInitFailuresTotal.Add(1)
}

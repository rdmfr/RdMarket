package monitoring

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"
)

type RegistryConfig struct {
	Registerer prometheus.Registerer
	Gatherer   prometheus.Gatherer
}

type HTTPMetrics struct {
	reqCounter      *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestInFlight prometheus.Gauge
	responseSize    *prometheus.HistogramVec
}

type Registry struct {
	mu       sync.RWMutex
	metrics  *HTTPMetrics
	gatherer prometheus.Gatherer
	pipeline *PipelineMetrics
}

func NewRegistry(cfg RegistryConfig) *Registry {
	registerer := cfg.Registerer
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}
	gatherer := cfg.Gatherer
	if gatherer == nil {
		gatherer = prometheus.DefaultGatherer
	}
	reg := &Registry{gatherer: gatherer}
	reg.metrics = newHTTPMetrics(registerer)
	reg.pipeline = NewPipelineMetrics(registerer)
	return reg
}

func (r *Registry) HTTP() *HTTPMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metrics
}

func newHTTPMetrics(reg prometheus.Registerer) *HTTPMetrics {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "Total number of HTTP requests."},
		[]string{"method", "route", "status"},
	)
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "HTTP request latency in seconds.", Buckets: prometheus.DefBuckets},
		[]string{"method", "route"},
	)
	inFlight := prometheus.NewGauge(prometheus.GaugeOpts{Name: "http_requests_in_flight", Help: "Number of HTTP requests currently in flight."})
	responseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "http_response_size_bytes", Help: "HTTP response size in bytes.", Buckets: []float64{128, 512, 1024, 4096, 16384, 65536, 262144, 1048576}},
		[]string{"method", "route", "status"},
	)
	for _, collector := range []prometheus.Collector{counter, histogram, inFlight, responseSize} {
		if err := reg.Register(collector); err != nil && !strings.Contains(err.Error(), "already registered") {
			panic(err)
		}
	}
	return &HTTPMetrics{reqCounter: counter, requestDuration: histogram, requestInFlight: inFlight, responseSize: responseSize}
}

type Config struct {
	Token string
}

func RegisterRoutes(app *fiber.App, cfg *Config) {
	app.Get("/metrics", func(c *fiber.Ctx) error {
		provided := c.Get("X-Metrics-Token")
		if cfg == nil || cfg.Token == "" || provided != cfg.Token {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		if c.Method() != fiber.MethodGet {
			return c.SendStatus(fiber.StatusMethodNotAllowed)
		}
		var out bytes.Buffer
		mfs, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("failed to gather metrics")
		}
		for _, mf := range mfs {
			if _, err := expfmt.MetricFamilyToText(&out, mf); err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString("failed to encode metrics")
			}
		}
		c.Type("text/plain; version=0.0.4; charset=utf-8")
		return c.Send(out.Bytes())
	})
}

func routePattern(c *fiber.Ctx) string {
	if route := c.Route().Path; route != "" {
		return route
	}
	return c.Path()
}

func HTTPMiddleware(reg *Registry) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		reg.HTTP().requestInFlight.Inc()
		defer reg.HTTP().requestInFlight.Dec()

		if err := c.Next(); err != nil {
			return err
		}

		status := fmt.Sprintf("%d", c.Response().StatusCode())
		method := c.Method()
		route := routePattern(c)
		reg.HTTP().reqCounter.WithLabelValues(method, route, status).Inc()
		reg.HTTP().requestDuration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
		reg.HTTP().responseSize.WithLabelValues(method, route, status).Observe(float64(len(c.Response().Body())))
		return nil
	}
}

func (r *Registry) RegisterRuntime() {
	for _, collector := range []prometheus.Collector{prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}), prometheus.NewGoCollector()} {
		if err := prometheus.DefaultRegisterer.Register(collector); err != nil && !strings.Contains(err.Error(), "already registered") {
			panic(err)
		}
	}
}

type PipelineMetrics struct {
	ingestionRunsTotal          *prometheus.CounterVec
	ingestionDurationSeconds    *prometheus.HistogramVec
	observationsImportedTotal   *prometheus.CounterVec
	observationsRejectedTotal   *prometheus.CounterVec
	observationsRevisedTotal    *prometheus.CounterVec
	observationFreshnessSeconds prometheus.GaugeVec
	flaggedObservationsTotal    *prometheus.CounterVec
}

func NewPipelineMetrics(reg prometheus.Registerer) *PipelineMetrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	metrics := &PipelineMetrics{
		ingestionRunsTotal:          prometheus.NewCounterVec(prometheus.CounterOpts{Name: "economic_ingestion_runs_total", Help: "Number of economic ingestion runs by outcome."}, []string{"provider", "series", "status"}),
		ingestionDurationSeconds:    prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "economic_ingestion_duration_seconds", Help: "Economic ingestion runtime in seconds.", Buckets: prometheus.DefBuckets}, []string{"provider", "series", "status"}),
		observationsImportedTotal:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "economic_observations_imported_total", Help: "Imported economic observations by provider, series, and outcome."}, []string{"provider", "series", "outcome"}),
		observationsRejectedTotal:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "economic_observations_rejected_total", Help: "Rejected economic observations by provider and series."}, []string{"provider", "series"}),
		observationsRevisedTotal:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "economic_observations_revised_total", Help: "Revised economic observations by provider and series."}, []string{"provider", "series"}),
		observationFreshnessSeconds: *prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "economic_observation_freshness_seconds", Help: "Age in seconds of the latest stored observation."}, []string{"provider", "series"}),
		flaggedObservationsTotal:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "economic_observations_flagged_total", Help: "Flagged economic observations by provider and series."}, []string{"provider", "series"}),
	}
	collectors := []prometheus.Collector{metrics.ingestionRunsTotal, metrics.ingestionDurationSeconds, metrics.observationsImportedTotal, metrics.observationsRejectedTotal, metrics.observationsRevisedTotal, &metrics.observationFreshnessSeconds, metrics.flaggedObservationsTotal}
	for _, collector := range collectors {
		if err := reg.Register(collector); err != nil && !strings.Contains(err.Error(), "already registered") {
			panic(err)
		}
	}
	return metrics
}

func (m *PipelineMetrics) ObserveIngestionRun(provider, series, status string, durationSeconds float64, accepted, rejected, revised int) {
	if m == nil {
		return
	}
	m.ingestionRunsTotal.WithLabelValues(provider, series, status).Inc()
	m.ingestionDurationSeconds.WithLabelValues(provider, series, status).Observe(durationSeconds)
	m.observationsImportedTotal.WithLabelValues(provider, series, "accepted").Add(float64(accepted))
	m.observationsRejectedTotal.WithLabelValues(provider, series).Add(float64(rejected))
	m.observationsRevisedTotal.WithLabelValues(provider, series).Add(float64(revised))
	m.flaggedObservationsTotal.WithLabelValues(provider, series).Add(float64(rejected))
}

func (m *PipelineMetrics) ObserveObservationFreshness(series, provider string, ageSeconds float64) {
	if m == nil {
		return
	}
	m.observationFreshnessSeconds.WithLabelValues(provider, series).Set(ageSeconds)
}

func (m *PipelineMetrics) ObserveFlaggedObservations(series, provider string, count int) {
	if m == nil {
		return
	}
	m.flaggedObservationsTotal.WithLabelValues(provider, series).Add(float64(count))
}

func (r *Registry) MetricsHandler() http.Handler { return promhttp.Handler() }
func (r *Registry) MetricsPath() string          { return "/metrics" }
func (r *Registry) Pipeline() *PipelineMetrics   { return r.pipeline }

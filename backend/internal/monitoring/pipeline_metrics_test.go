package monitoring

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPipelineMetricsRecordIngestionRun(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewPipelineMetrics(reg)
	metrics.ObserveIngestionRun("bls_api", "SERIES", "succeeded", 1.25, 5, 2, 1)
	metrics.ObserveObservationFreshness("SERIES", "bls_api", 45.5)

	if got := testutil.ToFloat64(metrics.ingestionRunsTotal.WithLabelValues("bls_api", "SERIES", "succeeded")); got != 1 {
		t.Fatalf("expected one successful run, got %v", got)
	}
	if got := testutil.ToFloat64(metrics.observationsImportedTotal.WithLabelValues("bls_api", "SERIES", "accepted")); got != 5 {
		t.Fatalf("expected 5 accepted observations, got %v", got)
	}
	if got := testutil.ToFloat64(metrics.observationFreshnessSeconds.WithLabelValues("bls_api", "SERIES")); got != 45.5 {
		t.Fatalf("expected 45.5 seconds freshness, got %v", got)
	}
	if got := testutil.ToFloat64(metrics.flaggedObservationsTotal.WithLabelValues("bls_api", "SERIES")); got != 2 {
		t.Fatalf("expected 2 flagged observations, got %v", got)
	}
}

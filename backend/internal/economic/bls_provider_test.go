package economic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBLSProviderParsesMonthlyObservationsWithoutInventingReleaseTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected request method: %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"status":"REQUEST_SUCCEEDED","Results":{"series":[{"seriesID":"CUSR0000SA0","data":[{"year":"2026","period":"M02","value":"319.8"},{"year":"2026","period":"M13","value":"318.2"},{"year":"2026","period":"M01","value":"318.4"}]}]}}`))
	}))
	defer server.Close()

	provider := NewBLSProvider("")
	provider.endpoint = server.URL
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)
	observations, err := provider.Fetch(context.Background(), "CUSR0000SA0", start, end)
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 2 || observations[0].ReferenceDate.Month() != time.January || observations[1].ReferenceDate.Month() != time.February {
		t.Fatalf("unexpected observations: %+v", observations)
	}
	if observations[0].ReleaseTimestamp != nil || observations[0].PeriodEnd == nil {
		t.Fatalf("provider must leave unknown release time empty and include period end: %+v", observations[0])
	}
}

func TestBLSProviderRejectsInvalidRangesAndAPIStatus(t *testing.T) {
	provider := NewBLSProvider("")
	start := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if _, err := provider.Fetch(context.Background(), "CUSR0000SA0", start, end); err == nil {
		t.Fatal("expected range validation error")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"REQUEST_FAILED","Results":{"series":[]}}`))
	}))
	defer server.Close()
	provider.endpoint = server.URL
	start = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end = time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)
	if _, err := provider.Fetch(context.Background(), "CUSR0000SA0", start, end); err == nil || !strings.Contains(err.Error(), "not successful") {
		t.Fatalf("expected API status failure, got %v", err)
	}
}

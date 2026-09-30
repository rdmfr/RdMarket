package economic

import (
	"context"
	"errors"
	"rdmarket-intelligence/backend/internal/events"
	"strings"
	"sync"
	"testing"
	"time"
)

type testObservationStore struct {
	mu        sync.Mutex
	saveCalls int
	failCalls int
	observed  []ImportedObservation
}

func (s *testObservationStore) SaveObservations(_ context.Context, _, _ string, observations []ImportedObservation, _ int) ([]events.EconomicObservationsUpdated, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saveCalls++
	s.observed = observations
	return nil, nil
}

func (s *testObservationStore) RecordFailedRun(context.Context, string, string, error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failCalls++
	return nil
}

type testEconomicProvider struct {
	entered chan struct{}
	release chan struct{}
	err     error
}

func (p *testEconomicProvider) GetProviderName() string { return "test" }
func (p *testEconomicProvider) Fetch(ctx context.Context, _ string, _, _ time.Time) ([]ImportedObservation, error) {
	if p.entered != nil {
		p.entered <- struct{}{}
	}
	if p.release != nil {
		select {
		case <-p.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if p.err != nil {
		return nil, p.err
	}
	return []ImportedObservation{}, nil
}

func TestIngestionSerializesRunsPerSeries(t *testing.T) {
	store := &testObservationStore{}
	provider := &testEconomicProvider{entered: make(chan struct{}, 1), release: make(chan struct{})}
	service := NewIngestionService(store, map[string]EconomicDataProvider{"test": provider}, nil)
	series := SeriesDefinition{Code: "SERIES", SourceProvider: "test", SourceSeriesID: "series"}
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- service.Sync(context.Background(), series, time.Now().AddDate(-1, 0, 0), time.Now())
	}()
	<-provider.entered
	if err := service.Sync(context.Background(), series, time.Now().AddDate(-1, 0, 0), time.Now()); !errors.Is(err, ErrIngestionInProgress) {
		t.Fatalf("expected overlapping run to be rejected, got %v", err)
	}
	close(provider.release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if store.saveCalls != 1 {
		t.Fatalf("expected one persisted run, got %d", store.saveCalls)
	}
}

func TestIngestionFailureIsRecordedAndCSVdryRunDoesNotPersist(t *testing.T) {
	store := &testObservationStore{}
	failure := errors.New("provider unavailable")
	service := NewIngestionService(store, map[string]EconomicDataProvider{"test": &testEconomicProvider{err: failure}}, nil)
	series := SeriesDefinition{Code: "SERIES", SourceProvider: "test", SourceSeriesID: "series"}
	if err := service.Sync(context.Background(), series, time.Now().AddDate(-1, 0, 0), time.Now()); !errors.Is(err, failure) {
		t.Fatalf("expected provider error, got %v", err)
	}
	if store.failCalls != 1 || store.saveCalls != 0 {
		t.Fatalf("failed provider run was not isolated: %+v", store)
	}

	manual := SeriesDefinition{Code: "MANUAL", SourceProvider: "csv_import", SourceSeriesID: "manual"}
	result, err := service.ImportCSV(context.Background(), manual, strings.NewReader("reference_date,value\n2026-09-01,2.1\n"), 1024, true)
	if err != nil || result.Accepted != 1 {
		t.Fatalf("unexpected dry-run result: %+v, %v", result, err)
	}
	if store.saveCalls != 0 {
		t.Fatal("dry-run must not persist observations")
	}
}

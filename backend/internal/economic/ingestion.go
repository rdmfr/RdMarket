package economic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"rdmarket-intelligence/backend/internal/events"
)

var ErrIngestionInProgress = errors.New("economic series ingestion is already running")

type EconomicDataProvider interface {
	GetProviderName() string
	Fetch(context.Context, string, time.Time, time.Time) ([]ImportedObservation, error)
}

type ObservationStore interface {
	SaveObservations(context.Context, string, string, []ImportedObservation, int) ([]events.EconomicObservationsUpdated, error)
	RecordFailedRun(context.Context, string, string, error) error
}

type IngestionService struct {
	store      ObservationStore
	providers  map[string]EconomicDataProvider
	dispatcher events.Dispatcher
	mu         sync.Mutex
	seriesLock map[string]*sync.Mutex
}

func NewIngestionService(store ObservationStore, providers map[string]EconomicDataProvider, dispatcher events.Dispatcher) *IngestionService {
	return &IngestionService{store: store, providers: providers, dispatcher: dispatcher, seriesLock: make(map[string]*sync.Mutex)}
}

func (s *IngestionService) Sync(ctx context.Context, series SeriesDefinition, start, end time.Time) error {
	provider, ok := s.providers[series.SourceProvider]
	if !ok {
		return fmt.Errorf("economic provider %q is unavailable", series.SourceProvider)
	}
	lock := s.lockFor(series.Code)
	if !lock.TryLock() {
		return ErrIngestionInProgress
	}
	defer lock.Unlock()

	observations, err := provider.Fetch(ctx, series.SourceSeriesID, start, end)
	if err != nil {
		_ = s.store.RecordFailedRun(ctx, series.Code, provider.GetProviderName(), err)
		return err
	}
	changed, err := s.store.SaveObservations(ctx, series.Code, provider.GetProviderName(), observations, 0)
	if err != nil {
		_ = s.store.RecordFailedRun(ctx, series.Code, provider.GetProviderName(), err)
		return err
	}
	s.publish(changed)
	return nil
}

func (s *IngestionService) ImportCSV(ctx context.Context, series SeriesDefinition, reader io.Reader, maxBytes int64, dryRun bool) (CSVImportResult, error) {
	result, err := (CSVImportProvider{}).Parse(reader, maxBytes)
	if err != nil || dryRun {
		return result, err
	}
	observations := make([]ImportedObservation, 0, result.Accepted)
	for _, row := range result.Rows {
		if row.Observation != nil {
			observations = append(observations, *row.Observation)
		}
	}
	changed, err := s.store.SaveObservations(ctx, series.Code, "csv_import", observations, result.Rejected)
	if err != nil {
		return CSVImportResult{}, err
	}
	s.publish(changed)
	return result, nil
}

func (s *IngestionService) Run(ctx context.Context, catalog []SeriesDefinition, interval time.Duration) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	for _, series := range catalog {
		if !series.IsActive || s.providers[series.SourceProvider] == nil {
			continue
		}
		go func(definition SeriesDefinition) {
			s.syncRecentHistory(ctx, definition)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					s.syncRecentHistory(ctx, definition)
				}
			}
		}(series)
	}
}

func (s *IngestionService) syncRecentHistory(ctx context.Context, series SeriesDefinition) {
	end := time.Now().UTC()
	start := end.AddDate(-24, 0, 0)
	if err := s.Sync(ctx, series, start, end); err != nil {
		log.Printf("event=economic_ingestion_failed series=%s provider=%s error=%q", series.Code, series.SourceProvider, err.Error())
	}
}

func (s *IngestionService) lockFor(code string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seriesLock[code] == nil {
		s.seriesLock[code] = &sync.Mutex{}
	}
	return s.seriesLock[code]
}

func (s *IngestionService) publish(changed []events.EconomicObservationsUpdated) {
	if s.dispatcher == nil {
		return
	}
	for _, event := range changed {
		s.dispatcher.PublishEconomicObservationsUpdated(event)
	}
}

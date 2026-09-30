package economic

import (
	"context"
	"fmt"
	"time"
)

type IndicatorView struct {
	Code               string     `json:"code"`
	Name               string     `json:"name"`
	Country            string     `json:"country"`
	Category           string     `json:"category"`
	Unit               string     `json:"unit"`
	Frequency          string     `json:"frequency"`
	SeasonalAdjustment bool       `json:"seasonal_adjustment"`
	SourceProvider     string     `json:"source_provider"`
	SourceSeriesID     string     `json:"source_series_id"`
	SourceURL          string     `json:"source_url"`
	LicenseNote        string     `json:"license_note"`
	Description        string     `json:"description"`
	ChangeMode         string     `json:"change_mode"`
	PublicationLagDays int        `json:"publication_lag_days"`
	LatestValue        *string    `json:"latest_value"`
	PreviousValue      *string    `json:"previous_value"`
	Change             *string    `json:"change"`
	ReferenceDate      *time.Time `json:"reference_date"`
	PeriodEnd          *time.Time `json:"period_end"`
	ReleaseTimestamp   *time.Time `json:"release_timestamp"`
	RetrievedAt        *time.Time `json:"retrieved_at"`
	Stale              *bool      `json:"stale"`
	StalenessDays      *int       `json:"staleness_days"`
	Trend              string     `json:"trend"`
	Sparkline          []string   `json:"sparkline"`
}

type ObservationView struct {
	ReferenceDate    time.Time  `json:"reference_date"`
	PeriodEnd        *time.Time `json:"period_end"`
	Value            string     `json:"value"`
	ReleaseTimestamp *time.Time `json:"release_timestamp"`
	RetrievedAt      time.Time  `json:"retrieved_at"`
	Revision         int        `json:"revision"`
	IsLatest         bool       `json:"is_latest"`
}

type APIService interface {
	ListIndicators(context.Context) ([]IndicatorView, error)
	GetIndicator(context.Context, string) (IndicatorView, error)
	GetHistory(context.Context, string, time.Time, time.Time, bool) ([]ObservationView, error)
}

type DefaultAPIService struct {
	repository    *Repository
	catalogByCode map[string]SeriesDefinition
}

func NewAPIService(repository *Repository, catalog []SeriesDefinition) *DefaultAPIService {
	definitions := make(map[string]SeriesDefinition, len(catalog))
	for _, definition := range catalog {
		definitions[definition.Code] = definition
	}
	return &DefaultAPIService{repository: repository, catalogByCode: definitions}
}

func (s *DefaultAPIService) ListIndicators(ctx context.Context) ([]IndicatorView, error) {
	series, err := s.repository.ListSeries(ctx, true)
	if err != nil {
		return nil, err
	}
	views := make([]IndicatorView, 0, len(series))
	for _, record := range series {
		view, err := s.buildIndicatorView(ctx, record)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *DefaultAPIService) GetIndicator(ctx context.Context, code string) (IndicatorView, error) {
	series, err := s.repository.GetSeries(ctx, code)
	if err != nil {
		return IndicatorView{}, err
	}
	return s.buildIndicatorView(ctx, series)
}

func (s *DefaultAPIService) GetHistory(ctx context.Context, code string, start, end time.Time, revisions bool) ([]ObservationView, error) {
	series, err := s.repository.GetSeries(ctx, code)
	if err != nil {
		return nil, err
	}
	records, err := s.repository.History(ctx, series.ID, start, end, revisions)
	if err != nil {
		return nil, err
	}
	views := make([]ObservationView, 0, len(records))
	for _, record := range records {
		views = append(views, ObservationView{
			ReferenceDate: record.ReferenceDate, PeriodEnd: record.PeriodEnd, Value: record.Value,
			ReleaseTimestamp: record.ReleaseTimestamp, RetrievedAt: record.RetrievedAt,
			Revision: record.Revision, IsLatest: record.IsLatest,
		})
	}
	return views, nil
}

func (s *DefaultAPIService) buildIndicatorView(ctx context.Context, record SeriesRecord) (IndicatorView, error) {
	view := IndicatorView{
		Code: record.Code, Name: record.Name, Country: record.Country, Category: record.Category,
		Unit: record.Unit, Frequency: record.Frequency, SeasonalAdjustment: record.SeasonalAdjustment,
		SourceProvider: record.SourceProvider, SourceSeriesID: record.SourceSeriesID,
		Description: record.Description, ChangeMode: record.ChangeMode,
		PublicationLagDays: record.PublicationLagDays, Trend: "not_enough_data", Sparkline: []string{},
	}
	if definition, ok := s.catalogByCode[record.Code]; ok {
		view.SourceURL = definition.SourceURL
		view.LicenseNote = definition.LicenseNote
	}
	observations, err := s.repository.LatestObservations(ctx, record.ID, 30)
	if err != nil {
		return IndicatorView{}, err
	}
	if len(observations) == 0 {
		return view, nil
	}
	stored := make([]StoredObservation, 0, len(observations))
	for _, observation := range observations {
		stored = append(stored, StoredObservation{
			ImportedObservation: ImportedObservation{
				ReferenceDate: observation.ReferenceDate, PeriodEnd: observation.PeriodEnd,
				Value: observation.Value, ReleaseTimestamp: observation.ReleaseTimestamp,
			},
			Revision: observation.Revision, RetrievedAt: observation.RetrievedAt,
		})
	}
	definition := definitionForRecord(record, s.catalogByCode)
	snapshot, err := BuildSnapshot(definition, stored, time.Now().UTC(), definition.TrendThreshold)
	if err != nil {
		return IndicatorView{}, fmt.Errorf("build economic indicator snapshot: %w", err)
	}
	view.LatestValue = stringPointer(snapshot.LatestValue)
	if snapshot.PreviousValue != "" {
		view.PreviousValue = stringPointer(snapshot.PreviousValue)
	}
	if snapshot.Change != "" {
		view.Change = stringPointer(snapshot.Change)
	}
	view.ReferenceDate = &snapshot.ReferenceDate
	view.ReleaseTimestamp = snapshot.ReleaseTimestamp
	view.RetrievedAt = &snapshot.RetrievedAt
	view.Stale = boolPointer(snapshot.Stale)
	view.StalenessDays = intPointer(snapshot.StalenessDays)
	if snapshot.Trend != "" {
		view.Trend = snapshot.Trend
	}
	view.Sparkline = snapshot.Sparkline
	return view, nil
}

func definitionForRecord(record SeriesRecord, definitions map[string]SeriesDefinition) SeriesDefinition {
	if definition, ok := definitions[record.Code]; ok {
		return definition
	}
	return SeriesDefinition{
		Code: record.Code, Frequency: record.Frequency, ChangeMode: record.ChangeMode,
		SourceProvider: record.SourceProvider, SourceSeriesID: record.SourceSeriesID,
		PublicationLagDays: record.PublicationLagDays, TrendThreshold: "0.1",
	}
}

func stringPointer(value string) *string { return &value }
func boolPointer(value bool) *bool       { return &value }
func intPointer(value int) *int          { return &value }

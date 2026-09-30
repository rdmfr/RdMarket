package economic

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"rdmarket-intelligence/backend/internal/events"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SeriesRecord struct {
	ID                 uint64 `gorm:"primaryKey"`
	Code               string `gorm:"uniqueIndex;size:64"`
	Name               string `gorm:"size:160"`
	Country            string `gorm:"size:80"`
	Category           string `gorm:"size:32"`
	Unit               string `gorm:"size:32"`
	Frequency          string `gorm:"size:16"`
	SeasonalAdjustment bool
	SourceProvider     string `gorm:"size:64"`
	SourceSeriesID     string `gorm:"size:128"`
	Description        string
	ChangeMode         string `gorm:"size:16"`
	PublicationLagDays int
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (SeriesRecord) TableName() string { return "economic_series" }

type ObservationRecord struct {
	ID               uint64 `gorm:"primaryKey"`
	SeriesID         uint64
	ReferenceDate    time.Time
	PeriodEnd        *time.Time
	Value            string `gorm:"type:numeric(24,10)"`
	ReleaseTimestamp *time.Time
	RetrievedAt      time.Time
	Revision         int
	IsLatest         bool
	CreatedAt        time.Time
}

func (ObservationRecord) TableName() string { return "economic_observations" }

type IngestionRunRecord struct {
	ID           uint64 `gorm:"primaryKey"`
	SeriesID     *uint64
	Provider     string
	Status       string
	RowsFetched  int
	RowsInserted int
	RowsRevised  int
	ErrorCode    *string
	ErrorMessage *string
	StartedAt    time.Time
	FinishedAt   *time.Time
}

func (IngestionRunRecord) TableName() string { return "economic_ingestion_runs" }

type MarketEventRecord struct {
	ID             uint64 `gorm:"primaryKey"`
	SeriesID       *uint64
	EventType      string
	EventTimestamp time.Time
	Title          string
	Detail         string
	ValueBefore    *string `gorm:"type:numeric(24,10)"`
	ValueAfter     *string `gorm:"type:numeric(24,10)"`
	CreatedAt      time.Time
}

func (MarketEventRecord) TableName() string { return "market_events" }

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) SeedCatalog(ctx context.Context, catalog []SeriesDefinition) error {
	for _, definition := range catalog {
		if err := definition.Validate(); err != nil {
			return fmt.Errorf("invalid catalog entry %q: %w", definition.Code, err)
		}
		record := SeriesRecord{
			Code: definition.Code, Name: definition.Name, Country: definition.Country,
			Category: definition.Category, Unit: definition.Unit, Frequency: definition.Frequency,
			SeasonalAdjustment: definition.SeasonalAdjustment, SourceProvider: definition.SourceProvider,
			SourceSeriesID: definition.SourceSeriesID, Description: definition.Description,
			ChangeMode: definition.ChangeMode, PublicationLagDays: definition.PublicationLagDays,
			IsActive: definition.IsActive, UpdatedAt: time.Now().UTC(),
		}
		columns := []string{"name", "country", "category", "unit", "frequency", "seasonal_adjustment", "source_provider", "source_series_id", "description", "change_mode", "publication_lag_days", "is_active", "updated_at"}
		err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "code"}}, DoUpdates: clause.AssignmentColumns(columns),
		}).Create(&record).Error
		if err != nil {
			return fmt.Errorf("seed economic series %s: %w", definition.Code, err)
		}
	}
	return nil
}

func (r *Repository) ListSeries(ctx context.Context, activeOnly bool) ([]SeriesRecord, error) {
	query := r.db.WithContext(ctx).Order("country, category, code")
	if activeOnly {
		query = query.Where("is_active = TRUE")
	}
	var records []SeriesRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) GetSeries(ctx context.Context, code string) (SeriesRecord, error) {
	var record SeriesRecord
	err := r.db.WithContext(ctx).Where("code = ? AND is_active = TRUE", code).Take(&record).Error
	return record, err
}

func (r *Repository) History(ctx context.Context, seriesID uint64, start, end time.Time, revisions bool) ([]ObservationRecord, error) {
	query := r.db.WithContext(ctx).Where("series_id = ? AND reference_date >= ? AND reference_date <= ?", seriesID, start, end)
	if !revisions {
		query = query.Where("is_latest = TRUE")
	}
	var records []ObservationRecord
	if err := query.Order("reference_date ASC, revision ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) LatestObservations(ctx context.Context, seriesID uint64, limit int) ([]ObservationRecord, error) {
	if limit < 1 {
		limit = 2
	}
	var records []ObservationRecord
	err := r.db.WithContext(ctx).Where("series_id = ? AND is_latest = TRUE", seriesID).
		Order("reference_date DESC").Limit(limit).Find(&records).Error
	return records, err
}

func (r *Repository) SaveObservations(ctx context.Context, code, provider string, observations []ImportedObservation, rejected int) ([]events.EconomicObservationsUpdated, error) {
	changed := make([]events.EconomicObservationsUpdated, 0)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var series SeriesRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ? AND is_active = TRUE", code).Take(&series).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		run := IngestionRunRecord{SeriesID: &series.ID, Provider: provider, Status: "running", RowsFetched: len(observations) + rejected, StartedAt: now}
		if err := tx.Create(&run).Error; err != nil {
			return fmt.Errorf("record ingestion run: %w", err)
		}
		for _, observation := range observations {
			if !validDecimal(observation.Value) {
				return fmt.Errorf("invalid decimal in observation for %s", code)
			}
			var previous ObservationRecord
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("series_id = ? AND reference_date = ? AND is_latest = TRUE", series.ID, observation.ReferenceDate).Take(&previous).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
			if err == nil && sameObservation(previous, observation) {
				continue
			}
			revision := 0
			var before *string
			if err == nil {
				revision = previous.Revision + 1
				before = &previous.Value
				if updateErr := tx.Model(&ObservationRecord{}).Where("id = ?", previous.ID).Update("is_latest", false).Error; updateErr != nil {
					return updateErr
				}
				run.RowsRevised++
			} else {
				run.RowsInserted++
			}
			record := ObservationRecord{SeriesID: series.ID, ReferenceDate: observation.ReferenceDate.UTC(), PeriodEnd: observation.PeriodEnd, Value: observation.Value, ReleaseTimestamp: observation.ReleaseTimestamp, RetrievedAt: now, Revision: revision, IsLatest: true, CreatedAt: now}
			if err := tx.Create(&record).Error; err != nil {
				return fmt.Errorf("store observation for %s: %w", code, err)
			}
			if series.Category == "monetary_policy" && before != nil && !decimalEqual(*before, observation.Value) {
				eventAt := now
				if observation.ReleaseTimestamp != nil {
					eventAt = observation.ReleaseTimestamp.UTC()
				}
				after := observation.Value
				event := MarketEventRecord{SeriesID: &series.ID, EventType: "policy_rate_change", EventTimestamp: eventAt, Title: "Observed policy-rate change", Detail: fmt.Sprintf("%s changed from %s to %s.", series.Name, *before, observation.Value), ValueBefore: before, ValueAfter: &after, CreatedAt: now}
				if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "series_id"}, {Name: "event_type"}, {Name: "event_timestamp"}}, DoNothing: true}).Create(&event).Error; err != nil {
					return fmt.Errorf("store policy-rate event: %w", err)
				}
			}
			changed = append(changed, events.EconomicObservationsUpdated{SeriesCode: code, ReferenceDate: observation.ReferenceDate.UTC(), Revision: revision, ReleaseAt: observation.ReleaseTimestamp})
		}
		finished := time.Now().UTC()
		run.FinishedAt = &finished
		run.Status = "succeeded"
		if rejected > 0 {
			run.Status = "partial"
		}
		return tx.Save(&run).Error
	})
	if err != nil {
		return nil, err
	}
	return changed, nil
}

func (r *Repository) RecordFailedRun(ctx context.Context, code, provider string, cause error) error {
	var series SeriesRecord
	if err := r.db.WithContext(ctx).Where("code = ?", code).Take(&series).Error; err != nil {
		return err
	}
	now := time.Now().UTC()
	status := "failed"
	errorCode := "PROVIDER_FAILURE"
	message := cause.Error()
	if len(message) > 512 {
		message = message[:512]
	}
	run := IngestionRunRecord{
		SeriesID: &series.ID, Provider: provider, Status: status, ErrorCode: &errorCode,
		ErrorMessage: &message, StartedAt: now, FinishedAt: &now,
	}
	return r.db.WithContext(ctx).Create(&run).Error
}

func sameObservation(previous ObservationRecord, next ImportedObservation) bool {
	return decimalEqual(previous.Value, next.Value) && sameTime(previous.PeriodEnd, next.PeriodEnd) && sameTime(previous.ReleaseTimestamp, next.ReleaseTimestamp)
}

func sameTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func decimalEqual(left, right string) bool {
	a, okA := new(big.Rat).SetString(left)
	b, okB := new(big.Rat).SetString(right)
	return okA && okB && a.Cmp(b) == 0
}

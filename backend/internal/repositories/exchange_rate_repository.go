package repositories

import (
	"errors"
	"math"
	"rdmarket-intelligence/backend/internal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExchangeRateRepository interface {
	Ping() error
	Save(rate *models.ExchangeRate) error
	SaveBatch(rates []models.ExchangeRate) error
	GetLatest(currencyPair string) (*models.ExchangeRate, error)
	GetPreviousClose(currencyPair string, beforeTime time.Time) (*models.ExchangeRate, error)
	GetHistory(currencyPair string, start time.Time, end time.Time) ([]models.ExchangeRate, error)
	GetStatistics(currencyPair string) (*models.StatisticsData, error)
	CountObservations(currencyPair string) (int64, error)
	GetLastSuccessfulUpdate(currencyPair string) (*time.Time, error)
}

type GormExchangeRateRepository struct {
	db *gorm.DB
}

func (r *GormExchangeRateRepository) Ping() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func NewExchangeRateRepository(db *gorm.DB) *GormExchangeRateRepository {
	return &GormExchangeRateRepository{db: db}
}

func (r *GormExchangeRateRepository) Save(rate *models.ExchangeRate) error {
	// Prevent duplicate for currency_pair + timestamp + source (UTC normalized)
	rate.Timestamp = rate.Timestamp.UTC()
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "currency_pair"}, {Name: "timestamp"}, {Name: "source"}},
		DoUpdates: clause.AssignmentColumns([]string{"rate", "updated_at"}),
	}).Create(rate).Error
}

func (r *GormExchangeRateRepository) SaveBatch(rates []models.ExchangeRate) error {
	if len(rates) == 0 {
		return nil
	}
	for i := range rates {
		rates[i].Timestamp = rates[i].Timestamp.UTC()
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "currency_pair"}, {Name: "timestamp"}, {Name: "source"}},
		DoUpdates: clause.AssignmentColumns([]string{"rate", "updated_at"}),
	}).CreateInBatches(rates, 100).Error
}

func (r *GormExchangeRateRepository) GetLatest(currencyPair string) (*models.ExchangeRate, error) {
	var rate models.ExchangeRate
	err := r.db.Where("currency_pair = ?", currencyPair).
		Where("quality_status = ?", "ok").
		Order("timestamp desc").
		First(&rate).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rate, nil
}

func (r *GormExchangeRateRepository) GetPreviousClose(currencyPair string, beforeTime time.Time) (*models.ExchangeRate, error) {
	var rate models.ExchangeRate
	err := r.db.Where("currency_pair = ? AND timestamp < ? AND quality_status = ?", currencyPair, beforeTime.UTC(), "ok").
		Order("timestamp desc").
		First(&rate).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rate, nil
}

func (r *GormExchangeRateRepository) GetHistory(currencyPair string, start time.Time, end time.Time) ([]models.ExchangeRate, error) {
	var rates []models.ExchangeRate
	err := r.db.Where("currency_pair = ? AND timestamp >= ? AND timestamp <= ? AND quality_status = ?", currencyPair, start.UTC(), end.UTC(), "ok").
		Order("timestamp asc").
		Find(&rates).Error
	return rates, err
}

func (r *GormExchangeRateRepository) CountObservations(currencyPair string) (int64, error) {
	var count int64
	err := r.db.Model(&models.ExchangeRate{}).Where("currency_pair = ?", currencyPair).Count(&count).Error
	return count, err
}

func (r *GormExchangeRateRepository) GetLastSuccessfulUpdate(currencyPair string) (*time.Time, error) {
	var latest models.ExchangeRate
	err := r.db.Where("currency_pair = ?", currencyPair).Order("timestamp desc").First(&latest).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	t := latest.Timestamp.UTC()
	return &t, nil
}

func (r *GormExchangeRateRepository) GetStatistics(currencyPair string) (*models.StatisticsData, error) {
	latest, err := r.GetLatest(currencyPair)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		return &models.StatisticsData{
			CurrencyPair:      currencyPair,
			HasSufficientData: false,
		}, nil
	}

	prev, err := r.GetPreviousClose(currencyPair, latest.Timestamp)
	if err != nil {
		return nil, err
	}

	now := latest.Timestamp.UTC()
	oneYearAgo := now.AddDate(-1, 0, 0)
	oneWeekAgo := now.AddDate(0, 0, -7)
	oneMonthAgo := now.AddDate(0, -1, 0)

	// Fetch 1-year history for aggregates
	var yearRates []models.ExchangeRate
	err = r.db.Where("currency_pair = ? AND timestamp >= ? AND quality_status = ?", currencyPair, oneYearAgo, "ok").
		Order("timestamp asc").
		Find(&yearRates).Error
	if err != nil {
		return nil, err
	}

	totalObs, _ := r.CountObservations(currencyPair)
	if len(yearRates) < 2 {
		return &models.StatisticsData{
			CurrencyPair:      currencyPair,
			ObservationCount:  totalObs,
			HasSufficientData: false,
		}, nil
	}

	curRate := latest.Rate
	var prevClose *float64
	var dailyChange *float64
	var dailyChangePct *float64
	if prev != nil {
		pRate := prev.Rate
		prevClose = &pRate
		chg := curRate - pRate
		dailyChange = &chg
		if pRate > 0 {
			pct := (chg / pRate) * 100.0
			dailyChangePct = &pct
		}
	}

	// 52-week High, Low, Average, Min, Max
	high52 := -math.MaxFloat64
	low52 := math.MaxFloat64
	sum := 0.0
	for _, r := range yearRates {
		if r.Rate > high52 {
			high52 = r.Rate
		}
		if r.Rate < low52 {
			low52 = r.Rate
		}
		sum += r.Rate
	}
	avg := sum / float64(len(yearRates))

	// Weekly change
	var weeklyChangePct *float64
	weekRate, _ := r.GetPreviousClose(currencyPair, oneWeekAgo)
	if weekRate != nil && weekRate.Rate > 0 {
		pct := ((curRate - weekRate.Rate) / weekRate.Rate) * 100.0
		weeklyChangePct = &pct
	}

	// Monthly change
	var monthlyChangePct *float64
	monthRate, _ := r.GetPreviousClose(currencyPair, oneMonthAgo)
	if monthRate != nil && monthRate.Rate > 0 {
		pct := ((curRate - monthRate.Rate) / monthRate.Rate) * 100.0
		monthlyChangePct = &pct
	}

	return &models.StatisticsData{
		CurrencyPair:         currencyPair,
		CurrentRate:          &curRate,
		PreviousClose:        prevClose,
		DailyChange:          dailyChange,
		DailyChangePercent:   dailyChangePct,
		WeeklyChangePercent:  weeklyChangePct,
		MonthlyChangePercent: monthlyChangePct,
		High52Week:           &high52,
		Low52Week:            &low52,
		AverageRate:          &avg,
		MinimumRate:          &low52,
		MaximumRate:          &high52,
		ObservationCount:     totalObs,
		HasSufficientData:    true,
	}, nil
}

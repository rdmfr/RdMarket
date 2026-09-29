package models

import (
	"time"
)

// ExchangeRate represents a stored exchange rate observation in PostgreSQL
type ExchangeRate struct {
	ID            uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	CurrencyPair  string     `gorm:"size:10;not null;index:idx_currency_pair;index:idx_pair_time,priority:1;uniqueIndex:idx_pair_time_source,priority:1" json:"currency_pair"`
	Timestamp     time.Time  `gorm:"not null;index:idx_timestamp;index:idx_pair_time,priority:2;uniqueIndex:idx_pair_time_source,priority:2" json:"timestamp"`
	Rate          float64    `gorm:"type:numeric(16,4);not null" json:"rate"`
	Source        string     `gorm:"size:64;not null;uniqueIndex:idx_pair_time_source,priority:3" json:"source"`
	QualityStatus string     `gorm:"size:16;not null;default:ok" json:"quality_status"`
	QualityReason *string    `gorm:"size:255" json:"quality_reason,omitempty"`
	FetchedAt     *time.Time `json:"fetched_at,omitempty"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ExchangeRate) TableName() string {
	return "exchange_rates"
}

// CurrentRateData represents the payload for GET /api/v1/market/usdidr/current
type CurrentRateData struct {
	CurrencyPair       string    `json:"currency_pair"`
	CurrentRate        float64   `json:"current_rate"`
	PreviousClose      float64   `json:"previous_close"`
	DailyChange        float64   `json:"daily_change"`
	DailyChangePercent float64   `json:"daily_change_percent"`
	Timestamp          time.Time `json:"timestamp"`
	Source             string    `json:"source"`
	IsStale            bool      `json:"is_stale"`
}

// HistoricalPoint represents a single data point in historical series
type HistoricalPoint struct {
	Timestamp int64    `json:"timestamp"` // Unix timestamp in milliseconds
	Rate      *float64 `json:"rate"`      // Nullable for explicit missing data
}

// HistoricalRateData represents the payload for GET /api/v1/market/usdidr/history
type HistoricalRateData struct {
	CurrencyPair string            `json:"currency_pair"`
	Range        string            `json:"range"`
	Points       []HistoricalPoint `json:"points"`
	TotalPoints  int               `json:"total_points"`
	MissingCount int               `json:"missing_count"`
}

// StatisticsData represents statistical aggregates for GET /api/v1/market/usdidr/statistics
type StatisticsData struct {
	CurrencyPair         string   `json:"currency_pair"`
	CurrentRate          *float64 `json:"current_rate"`
	PreviousClose        *float64 `json:"previous_close"`
	DailyChange          *float64 `json:"daily_change"`
	DailyChangePercent   *float64 `json:"daily_change_percent"`
	WeeklyChangePercent  *float64 `json:"weekly_change_percent"`
	MonthlyChangePercent *float64 `json:"monthly_change_percent"`
	High52Week           *float64 `json:"high_52_week"`
	Low52Week            *float64 `json:"low_52_week"`
	AverageRate          *float64 `json:"average_rate"`
	MinimumRate          *float64 `json:"minimum_rate"`
	MaximumRate          *float64 `json:"maximum_rate"`
	ObservationCount     int64    `json:"observation_count"`
	HasSufficientData    bool     `json:"has_sufficient_data"`
}

// IndicatorSeries represents an indicator series (SMA, Return, etc.)
type IndicatorSeries struct {
	Name   string            `json:"name"`
	Points []HistoricalPoint `json:"points"`
}

// MarketCondition represents transparent rule-based condition labels
type MarketCondition struct {
	ShortTermCondition string  `json:"short_term_condition"` // "Positive", "Negative", "Neutral"
	Trend30Day         string  `json:"trend_30_day"`         // "Upward", "Downward", "Sideways"
	VolatilityLevel    string  `json:"volatility_level"`     // "Low", "Moderate", "High"
	RecentReturn7D     float64 `json:"recent_return_7d"`
	Return30D          float64 `json:"return_30d"`
	ObservedVolatility float64 `json:"observed_volatility"`
	Disclaimer         string  `json:"disclaimer"`
}

// IndicatorsData represents the payload for GET /api/v1/market/usdidr/indicators
type IndicatorsData struct {
	CurrencyPair    string            `json:"currency_pair"`
	Range           string            `json:"range"`
	SMA7            []HistoricalPoint `json:"sma_7"`
	SMA30           []HistoricalPoint `json:"sma_30"`
	SMA90           []HistoricalPoint `json:"sma_90"`
	DailyReturn     *float64          `json:"daily_return"`
	WeeklyReturn    *float64          `json:"weekly_return"`
	RollingVol30    *float64          `json:"rolling_volatility_30d"`
	MarketCondition MarketCondition   `json:"market_condition"`
}

// DataSourceInfo represents provider telemetry for GET /api/v1/data-sources
type DataSourceInfo struct {
	Provider             string      `json:"provider"`
	Instrument           string      `json:"instrument"`
	Status               string      `json:"status"` // "Connected", "Degraded", "Offline"
	LastSuccessfulUpdate *time.Time  `json:"last_successful_update"`
	DataFrequency        string      `json:"data_frequency"`
	NumberOfObservations int64       `json:"number_of_observations"`
	BaseCurrency         string      `json:"base_currency"`
	TargetCurrency       string      `json:"target_currency"`
	Capabilities         interface{} `json:"capabilities"`
}

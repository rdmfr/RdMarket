package services

import (
	"errors"
	"math"
	"rdmarket-intelligence/backend/internal/config"
	"rdmarket-intelligence/backend/internal/events"
	"rdmarket-intelligence/backend/internal/models"
	"rdmarket-intelligence/backend/internal/providers"
	"rdmarket-intelligence/backend/internal/repositories"
	"strings"
	"time"
)

var ErrInvalidRange = errors.New("invalid market data range")

type MarketService interface {
	Ready() error
	GetCurrentRate(currencyPair string) (*models.CurrentRateData, error)
	GetHistory(currencyPair, rangeStr, startStr, endStr string) (*models.HistoricalRateData, error)
	GetStatistics(currencyPair string) (*models.StatisticsData, error)
	GetIndicators(currencyPair, rangeStr string) (*models.IndicatorsData, error)
	GetCondition(currencyPair, rangeStr string) (*models.MarketCondition, error)
	GetDataSources() ([]models.DataSourceInfo, error)
	SyncExternalData(currencyPair string) error
	SeedInitialDataIfEmpty(currencyPair string) error
}

func (s *DefaultMarketService) Ready() error {
	return s.repo.Ping()
}

type DefaultMarketService struct {
	repo       repositories.ExchangeRateRepository
	provider   providers.ExchangeRateProvider
	cfg        *config.Config
	dispatcher events.Dispatcher
}

func NewMarketService(
	repo repositories.ExchangeRateRepository,
	provider providers.ExchangeRateProvider,
	cfg *config.Config,
) *DefaultMarketService {
	return &DefaultMarketService{
		repo:     repo,
		provider: provider,
		cfg:      cfg,
	}
}

func (s *DefaultMarketService) SetEventDispatcher(dispatcher events.Dispatcher) {
	s.dispatcher = dispatcher
}

func (s *DefaultMarketService) GetCurrentRate(currencyPair string) (*models.CurrentRateData, error) {
	currencyPair = strings.ToUpper(strings.TrimSpace(currencyPair))
	latest, err := s.repo.GetLatest(currencyPair)
	if err != nil {
		return nil, err
	}
	if latest == nil {
		if err := s.SyncExternalData(currencyPair); err != nil {
			return nil, err
		}
		latest, err = s.repo.GetLatest(currencyPair)
		if err != nil || latest == nil {
			return nil, err
		}
	}

	prev, _ := s.repo.GetPreviousClose(currencyPair, latest.Timestamp)
	var prevClose float64
	var dailyChange float64
	var dailyChangePct float64

	if prev != nil && prev.Rate > 0 {
		prevClose = prev.Rate
		dailyChange = latest.Rate - prev.Rate
		dailyChangePct = (dailyChange / prev.Rate) * 100.0
	} else {
		prevClose = latest.Rate
		dailyChange = 0.0
		dailyChangePct = 0.0
	}

	// Data is considered stale if older than 48 hours (excluding weekend gaps)
	isStale := time.Since(latest.Timestamp) > 48*time.Hour

	return &models.CurrentRateData{
		CurrencyPair:       currencyPair,
		CurrentRate:        latest.Rate,
		PreviousClose:      prevClose,
		DailyChange:        dailyChange,
		DailyChangePercent: dailyChangePct,
		Timestamp:          latest.Timestamp.UTC(),
		Source:             latest.Source,
		IsStale:            isStale,
	}, nil
}

func (s *DefaultMarketService) parseRange(rangeStr, startStr, endStr string) (time.Time, time.Time) {
	now := time.Now().UTC()
	var start, end time.Time

	if startStr != "" && endStr != "" {
		if sDate, err := time.Parse("2006-01-02", startStr); err == nil {
			start = sDate.UTC()
		}
		if eDate, err := time.Parse("2006-01-02", endStr); err == nil {
			end = eDate.UTC().Add(23*time.Hour + 59*time.Minute)
		}
		if !start.IsZero() && !end.IsZero() {
			return start, end
		}

	}

	end = now
	switch strings.ToUpper(rangeStr) {
	case "1D":
		start = now.AddDate(0, 0, -2) // Include prior day for change
	case "7D":
		start = now.AddDate(0, 0, -7)
	case "1M":
		start = now.AddDate(0, -1, 0)
	case "3M":
		start = now.AddDate(0, -3, 0)
	case "6M":
		start = now.AddDate(0, -6, 0)
	case "1Y":
		start = now.AddDate(-1, 0, 0)
	case "5Y":
		start = now.AddDate(-5, 0, 0)
	default:
		start = now.AddDate(0, -1, 0) // Default 1M
	}

	return start, end
}

func (s *DefaultMarketService) parseRequestedRange(rangeStr, startStr, endStr string) (time.Time, time.Time, error) {
	if startStr != "" || endStr != "" {
		if startStr == "" || endStr == "" {
			return time.Time{}, time.Time{}, ErrInvalidRange
		}
		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			return time.Time{}, time.Time{}, ErrInvalidRange
		}
		end, err := time.Parse("2006-01-02", endStr)
		if err != nil || end.Before(start) {
			return time.Time{}, time.Time{}, ErrInvalidRange
		}
		return start.UTC(), end.UTC().Add(24*time.Hour - time.Nanosecond), nil
	}
	switch strings.ToUpper(rangeStr) {
	case "1D", "7D", "1M", "3M", "6M", "1Y", "5Y":
		start, end := s.parseRange(rangeStr, "", "")
		return start, end, nil
	default:
		return time.Time{}, time.Time{}, ErrInvalidRange
	}
}

func (s *DefaultMarketService) GetHistory(currencyPair, rangeStr, startStr, endStr string) (*models.HistoricalRateData, error) {
	currencyPair = strings.ToUpper(strings.TrimSpace(currencyPair))
	start, end, err := s.parseRequestedRange(rangeStr, startStr, endStr)
	if err != nil {
		return nil, err
	}

	rates, err := s.repo.GetHistory(currencyPair, start, end)
	if err != nil {
		return nil, err
	}

	if len(rates) == 0 {
		return &models.HistoricalRateData{
			CurrencyPair: currencyPair, Range: rangeStr, Points: []models.HistoricalPoint{},
			Resolution: "raw", AggregationMethod: "none",
		}, nil
	}

	sourcePointCount := len(rates)
	resolution := "raw"
	aggregationMethod := "none"
	pointLimit := s.cfg.HistoryPointLimit
	if pointLimit <= 0 {
		pointLimit = 1500
	}
	if len(rates) > pointLimit {
		bucketSize := (len(rates) + pointLimit - 1) / pointLimit
		downsampled := make([]models.ExchangeRate, 0, pointLimit)
		for i := 0; i < len(rates); i += bucketSize {
			end := i + bucketSize
			if end > len(rates) {
				end = len(rates)
			}
			downsampled = append(downsampled, rates[end-1])
		}
		rates = downsampled
		resolution = "bucketed"
		aggregationMethod = "last_value"
	}
	var points []models.HistoricalPoint

	for _, r := range rates {
		rateVal := r.Rate
		points = append(points, models.HistoricalPoint{
			Timestamp: r.Timestamp.UnixMilli(),
			Rate:      &rateVal,
		})
	}

	return &models.HistoricalRateData{
		CurrencyPair:      currencyPair,
		Range:             rangeStr,
		Points:            points,
		TotalPoints:       len(points),
		SourcePointCount:  sourcePointCount,
		MissingCount:      0,
		Resolution:        resolution,
		AggregationMethod: aggregationMethod,
	}, nil
}

func (s *DefaultMarketService) GetStatistics(currencyPair string) (*models.StatisticsData, error) {
	currencyPair = strings.ToUpper(strings.TrimSpace(currencyPair))
	return s.repo.GetStatistics(currencyPair)
}

func (s *DefaultMarketService) GetIndicators(currencyPair, rangeStr string) (*models.IndicatorsData, error) {
	currencyPair = strings.ToUpper(strings.TrimSpace(currencyPair))
	// Need at least 90+ days prior to range start for SMA 90
	now := time.Now().UTC()
	start, _, err := s.parseRequestedRange(rangeStr, "", "")
	if err != nil {
		return nil, err
	}
	fetchStart := start.AddDate(0, 0, -120) // Buffer for 90-day moving average calculation

	rates, err := s.repo.GetHistory(currencyPair, fetchStart, now)
	if err != nil {
		return nil, err
	}

	sma7 := calculateSMA(rates, 7, start)
	sma30 := calculateSMA(rates, 30, start)
	sma90 := calculateSMA(rates, 90, start)

	var dailyReturn *float64
	var weeklyReturn *float64
	var rollingVol30 *float64

	n := len(rates)
	if n >= 2 {
		dRet := (rates[n-1].Rate - rates[n-2].Rate) / rates[n-2].Rate
		dailyReturn = &dRet
	}
	if n >= 8 {
		wRet := (rates[n-1].Rate - rates[n-8].Rate) / rates[n-8].Rate
		weeklyReturn = &wRet
	}

	// Calculate 30-day volatility from daily returns
	if n >= 31 {
		var dailyReturns []float64
		for i := n - 30; i < n; i++ {
			if rates[i-1].Rate > 0 {
				ret := (rates[i].Rate - rates[i-1].Rate) / rates[i-1].Rate
				dailyReturns = append(dailyReturns, ret)
			}
		}
		if len(dailyReturns) > 0 {
			vol := calculateStandardDeviation(dailyReturns)
			rollingVol30 = &vol
		}
	}

	// Transparent rule-based Market Condition
	var recent7D float64
	if weeklyReturn != nil {
		recent7D = *weeklyReturn
	}

	var ret30D float64
	if n >= 22 && rates[n-22].Rate > 0 { // ~22 trading days in 30 calendar days
		ret30D = (rates[n-1].Rate - rates[n-22].Rate) / rates[n-22].Rate
	}

	var obsVol float64
	if rollingVol30 != nil {
		obsVol = *rollingVol30
	}

	shortTerm := "Neutral"
	if recent7D > s.cfg.ShortTermThreshold {
		shortTerm = "Positive"
	} else if recent7D < -s.cfg.ShortTermThreshold {
		shortTerm = "Negative"
	}

	trend30 := "Sideways"
	if ret30D > s.cfg.Trend30DThreshold {
		trend30 = "Upward"
	} else if ret30D < -s.cfg.Trend30DThreshold {
		trend30 = "Downward"
	}

	volLevel := "Moderate"
	if obsVol > 0 {
		if obsVol < s.cfg.VolatilityLowLimit {
			volLevel = "Low"
		} else if obsVol > s.cfg.VolatilityHighLimit {
			volLevel = "High"
		}
	}

	condition := models.MarketCondition{
		ShortTermCondition: shortTerm,
		Trend30Day:         trend30,
		VolatilityLevel:    volLevel,
		RecentReturn7D:     recent7D,
		Return30D:          ret30D,
		ObservedVolatility: obsVol,
		Disclaimer:         "Descriptive statistics only. Not financial advice.",
	}

	return &models.IndicatorsData{
		CurrencyPair:    currencyPair,
		Range:           rangeStr,
		SMA7:            sma7,
		SMA30:           sma30,
		SMA90:           sma90,
		DailyReturn:     dailyReturn,
		WeeklyReturn:    weeklyReturn,
		RollingVol30:    rollingVol30,
		MarketCondition: condition,
	}, nil
}

func (s *DefaultMarketService) GetCondition(currencyPair, rangeStr string) (*models.MarketCondition, error) {
	indicators, err := s.GetIndicators(currencyPair, rangeStr)
	if err != nil {
		return nil, err
	}
	return &indicators.MarketCondition, nil
}

func calculateSMA(rates []models.ExchangeRate, window int, filterStart time.Time) []models.HistoricalPoint {
	var points []models.HistoricalPoint
	if len(rates) < window {
		return points
	}

	for i := window - 1; i < len(rates); i++ {
		t := rates[i].Timestamp
		if t.Before(filterStart) {
			continue
		}
		sum := 0.0
		for j := i - window + 1; j <= i; j++ {
			sum += rates[j].Rate
		}
		avg := sum / float64(window)
		rounded := math.Round(avg*100) / 100
		points = append(points, models.HistoricalPoint{
			Timestamp: t.UnixMilli(),
			Rate:      &rounded,
		})
	}
	return points
}

func calculateStandardDeviation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	var sumSqDiff float64
	for _, v := range values {
		diff := v - mean
		sumSqDiff += diff * diff
	}
	return math.Sqrt(sumSqDiff / float64(len(values)-1))
}

func (s *DefaultMarketService) GetDataSources() ([]models.DataSourceInfo, error) {
	lastUpdate, _ := s.repo.GetLastSuccessfulUpdate("USD/IDR")
	count, _ := s.repo.CountObservations("USD/IDR")

	status := "Connected"
	if count == 0 {
		status = "Degraded"
	}

	return []models.DataSourceInfo{
		{
			Provider:             s.provider.GetProviderName(),
			Instrument:           "USD/IDR",
			Status:               status,
			LastSuccessfulUpdate: lastUpdate,
			DataFrequency:        "Daily / Market Close",
			NumberOfObservations: count,
			BaseCurrency:         "USD",
			TargetCurrency:       "IDR",
			Capabilities:         s.provider.Capabilities(),
		},
	}, nil
}

func (s *DefaultMarketService) SyncExternalData(currencyPair string) error {
	parts := strings.Split(currencyPair, "/")
	if len(parts) != 2 {
		return nil
	}
	base := parts[0]
	target := parts[1]

	now := time.Now().UTC()
	start := now.AddDate(-2, 0, 0) // Fetch up to 2 years

	rates, err := s.provider.FetchHistoricalRates(base, target, start, now)
	if err != nil {
		return err
	}

	if len(rates) > 0 {
		if err := s.repo.SaveBatch(rates); err != nil {
			return err
		}
		for i := range rates {
			s.publishExchangeRateUpdated(&rates[i])
		}
	}

	// Also fetch current
	latest, err := s.provider.FetchCurrentRate(base, target)
	if err == nil && latest != nil {
		if err := s.repo.Save(latest); err != nil {
			return err
		}
		s.publishExchangeRateUpdated(latest)
	}

	return nil
}

func (s *DefaultMarketService) publishExchangeRateUpdated(rate *models.ExchangeRate) {
	if s.dispatcher == nil || rate == nil {
		return
	}
	s.dispatcher.PublishExchangeRateUpdated(events.ExchangeRateUpdated{
		CurrencyPair: rate.CurrencyPair,
		ObservedAt:   rate.Timestamp.UTC(),
		Source:       rate.Source,
	})
}

func (s *DefaultMarketService) SeedInitialDataIfEmpty(currencyPair string) error {
	count, err := s.repo.CountObservations(currencyPair)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// First try external provider
	err = s.SyncExternalData(currencyPair)
	if err == nil {
		count, _ = s.repo.CountObservations(currencyPair)
		if count > 0 {
			return nil
		}
	}

	return err
}

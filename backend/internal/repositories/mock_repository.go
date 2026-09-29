package repositories

import (
	"math"
	"rdmarket-intelligence/backend/internal/models"
	"sort"
	"sync"
	"time"
)

type MockExchangeRateRepository struct {
	mu    sync.RWMutex
	rates []models.ExchangeRate
}

func NewMockExchangeRateRepository() *MockExchangeRateRepository {
	return &MockExchangeRateRepository{
		rates: make([]models.ExchangeRate, 0),
	}
}

func (m *MockExchangeRateRepository) Ping() error {
	return nil
}

func (m *MockExchangeRateRepository) Save(rate *models.ExchangeRate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rateCopy := *rate
	rateCopy.Timestamp = rateCopy.Timestamp.UTC()

	// Update if exists or append
	updated := false
	for i, r := range m.rates {
		if r.CurrencyPair == rateCopy.CurrencyPair && r.Timestamp.Equal(rateCopy.Timestamp) && r.Source == rateCopy.Source {
			m.rates[i] = rateCopy
			updated = true
			break
		}
	}
	if !updated {
		m.rates = append(m.rates, rateCopy)
	}

	sort.Slice(m.rates, func(i, j int) bool {
		return m.rates[i].Timestamp.Before(m.rates[j].Timestamp)
	})
	return nil
}

func (m *MockExchangeRateRepository) SaveBatch(rates []models.ExchangeRate) error {
	for _, r := range rates {
		if err := m.Save(&r); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockExchangeRateRepository) GetLatest(currencyPair string) (*models.ExchangeRate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := len(m.rates) - 1; i >= 0; i-- {
		if m.rates[i].CurrencyPair == currencyPair {
			r := m.rates[i]
			return &r, nil
		}
	}
	return nil, nil
}

func (m *MockExchangeRateRepository) GetPreviousClose(currencyPair string, beforeTime time.Time) (*models.ExchangeRate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	beforeUTC := beforeTime.UTC()
	for i := len(m.rates) - 1; i >= 0; i-- {
		if m.rates[i].CurrencyPair == currencyPair && m.rates[i].Timestamp.Before(beforeUTC) {
			r := m.rates[i]
			return &r, nil
		}
	}
	return nil, nil
}

func (m *MockExchangeRateRepository) GetHistory(currencyPair string, start time.Time, end time.Time) ([]models.ExchangeRate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []models.ExchangeRate
	startUTC := start.UTC()
	endUTC := end.UTC()
	for _, r := range m.rates {
		if r.CurrencyPair == currencyPair && !r.Timestamp.Before(startUTC) && !r.Timestamp.After(endUTC) {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *MockExchangeRateRepository) CountObservations(currencyPair string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var count int64
	for _, r := range m.rates {
		if r.CurrencyPair == currencyPair {
			count++
		}
	}
	return count, nil
}

func (m *MockExchangeRateRepository) GetLastSuccessfulUpdate(currencyPair string) (*time.Time, error) {
	latest, err := m.GetLatest(currencyPair)
	if err != nil || latest == nil {
		return nil, err
	}
	t := latest.Timestamp.UTC()
	return &t, nil
}

func (m *MockExchangeRateRepository) GetStatistics(currencyPair string) (*models.StatisticsData, error) {
	latest, err := m.GetLatest(currencyPair)
	if err != nil || latest == nil {
		return &models.StatisticsData{
			CurrencyPair:      currencyPair,
			HasSufficientData: false,
		}, nil
	}

	prev, _ := m.GetPreviousClose(currencyPair, latest.Timestamp)
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

	totalObs, _ := m.CountObservations(currencyPair)
	now := latest.Timestamp.UTC()
	yearAgo := now.AddDate(-1, 0, 0)
	history, _ := m.GetHistory(currencyPair, yearAgo, now)

	if len(history) < 2 {
		return &models.StatisticsData{
			CurrencyPair:      currencyPair,
			ObservationCount:  totalObs,
			HasSufficientData: false,
		}, nil
	}

	high52 := -math.MaxFloat64
	low52 := math.MaxFloat64
	sum := 0.0
	for _, r := range history {
		if r.Rate > high52 {
			high52 = r.Rate
		}
		if r.Rate < low52 {
			low52 = r.Rate
		}
		sum += r.Rate
	}
	avg := sum / float64(len(history))

	return &models.StatisticsData{
		CurrencyPair:       currencyPair,
		CurrentRate:        &curRate,
		PreviousClose:      prevClose,
		DailyChange:        dailyChange,
		DailyChangePercent: dailyChangePct,
		High52Week:         &high52,
		Low52Week:          &low52,
		AverageRate:        &avg,
		MinimumRate:        &low52,
		MaximumRate:        &high52,
		ObservationCount:   totalObs,
		HasSufficientData:  true,
	}, nil
}

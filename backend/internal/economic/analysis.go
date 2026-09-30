package economic

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
	"time"
)

type StoredObservation struct {
	ImportedObservation
	Revision    int
	RetrievedAt time.Time
}

type IndicatorSnapshot struct {
	Code               string
	LatestValue        string
	PreviousValue      string
	Change             string
	ReferenceDate      time.Time
	ReleaseTimestamp   *time.Time
	RetrievedAt        time.Time
	SourceProvider     string
	SourceSeriesID     string
	Stale              bool
	StalenessDays      int
	Trend              string
	Sparkline          []string
	PublicationLagDays int
}

func BuildSnapshot(series SeriesDefinition, observations []StoredObservation, now time.Time, trendThreshold string) (IndicatorSnapshot, error) {
	if len(observations) == 0 {
		return IndicatorSnapshot{}, fmt.Errorf("Not enough data")
	}
	sorted := append([]StoredObservation(nil), observations...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ReferenceDate.Before(sorted[j].ReferenceDate)
	})
	latest := sorted[len(sorted)-1]
	if !validDecimal(latest.Value) {
		return IndicatorSnapshot{}, fmt.Errorf("stored value is not a valid decimal")
	}
	snapshot := IndicatorSnapshot{
		Code:               series.Code,
		LatestValue:        latest.Value,
		ReferenceDate:      latest.ReferenceDate,
		ReleaseTimestamp:   latest.ReleaseTimestamp,
		RetrievedAt:        latest.RetrievedAt,
		SourceProvider:     series.SourceProvider,
		SourceSeriesID:     series.SourceSeriesID,
		PublicationLagDays: series.PublicationLagDays,
		Sparkline:          make([]string, 0, 30),
	}
	if len(sorted) > 1 {
		previous := sorted[len(sorted)-2]
		snapshot.PreviousValue = previous.Value
		change, err := calculateChange(latest.Value, previous.Value, series.ChangeMode)
		if err == nil {
			snapshot.Change = change
			snapshot.Trend = classifyTrend(change, trendThreshold)
		}
	}
	start := len(sorted) - 30
	if start < 0 {
		start = 0
	}
	for _, observation := range sorted[start:] {
		snapshot.Sparkline = append(snapshot.Sparkline, observation.Value)
	}
	lastKnownAt := latest.ReferenceDate
	if latest.PeriodEnd != nil {
		lastKnownAt = *latest.PeriodEnd
	}
	if now.After(lastKnownAt) {
		snapshot.StalenessDays = int(now.Sub(lastKnownAt).Hours() / 24)
		snapshot.Stale = snapshot.StalenessDays > stalenessTolerance(series.Frequency)
	}
	return snapshot, nil
}

func calculateChange(current, previous, mode string) (string, error) {
	currentRat, ok := new(big.Rat).SetString(current)
	if !ok {
		return "", fmt.Errorf("invalid current decimal")
	}
	previousRat, ok := new(big.Rat).SetString(previous)
	if !ok {
		return "", fmt.Errorf("invalid previous decimal")
	}
	delta := new(big.Rat).Sub(currentRat, previousRat)
	switch mode {
	case "level":
	case "percent":
		if previousRat.Sign() == 0 {
			return "", fmt.Errorf("percent change from zero is undefined")
		}
		delta.Quo(delta, previousRat)
		delta.Mul(delta, big.NewRat(100, 1))
	case "bps":
		delta.Mul(delta, big.NewRat(100, 1))
	default:
		return "", fmt.Errorf("unsupported change mode %q", mode)
	}
	return strings.TrimRight(strings.TrimRight(delta.FloatString(10), "0"), "."), nil
}

func classifyTrend(change, threshold string) string {
	value, ok := new(big.Rat).SetString(change)
	if !ok {
		return "not_enough_data"
	}
	limit, ok := new(big.Rat).SetString(threshold)
	if !ok || limit.Sign() < 0 {
		limit = new(big.Rat)
	}
	if value.Cmp(limit) > 0 {
		return "rising"
	}
	if value.Cmp(new(big.Rat).Neg(limit)) < 0 {
		return "falling"
	}
	return "flat"
}

func stalenessTolerance(frequency string) int {
	switch frequency {
	case "daily":
		return 4
	case "weekly":
		return 14
	case "monthly":
		return 62
	case "quarterly":
		return 140
	default:
		return 0
	}
}

type DatedValue struct {
	Date  time.Time
	Value float64
}

type LagCorrelation struct {
	Lag         int
	SampleCount int
	Coefficient *float64
}

type Relationship struct {
	Method             string
	Window             int
	SampleCount        int
	MinimumSampleCount int
	RollingCorrelation *float64
	Lagged             []LagCorrelation
	NotEnoughData      bool
}

func ComputeRelationship(indicator, exchangeRate []DatedValue, window, maxLag, minSamples int) Relationship {
	result := Relationship{
		Method:             "period-end percentage changes; exact-date alignment; no forward-fill",
		Window:             window,
		MinimumSampleCount: minSamples,
		Lagged:             make([]LagCorrelation, 0, maxLag*2+1),
	}
	if window <= 0 || maxLag < 0 || minSamples < 2 {
		result.NotEnoughData = true
		return result
	}
	pairs := alignedReturns(indicator, exchangeRate)
	result.SampleCount = len(pairs)
	start := len(pairs) - window
	if start < 0 {
		start = 0
	}
	rolling := correlation(pairs[start:])
	if len(pairs[start:]) >= minSamples && rolling != nil {
		result.RollingCorrelation = rolling
	} else {
		result.NotEnoughData = true
	}
	for lag := -maxLag; lag <= maxLag; lag++ {
		lagged := correlationForLag(pairs, lag)
		if len(lagged) < minSamples {
			result.Lagged = append(result.Lagged, LagCorrelation{Lag: lag, SampleCount: len(lagged)})
			continue
		}
		result.Lagged = append(result.Lagged, LagCorrelation{Lag: lag, SampleCount: len(lagged), Coefficient: correlation(lagged)})
	}
	return result
}

type returnPair struct {
	date      time.Time
	indicator float64
	rate      float64
}

func alignedReturns(indicator, exchangeRate []DatedValue) []returnPair {
	indicatorByDate := make(map[string]float64, len(indicator))
	rateByDate := make(map[string]float64, len(exchangeRate))
	for _, value := range indicator {
		if finite(value.Value) {
			indicatorByDate[value.Date.UTC().Format("2006-01-02")] = value.Value
		}
	}
	for _, value := range exchangeRate {
		if finite(value.Value) {
			rateByDate[value.Date.UTC().Format("2006-01-02")] = value.Value
		}
	}
	dates := make([]time.Time, 0)
	for key := range indicatorByDate {
		if _, ok := rateByDate[key]; ok {
			date, _ := time.Parse("2006-01-02", key)
			dates = append(dates, date)
		}
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	pairs := make([]returnPair, 0, len(dates)-1)
	for index := 1; index < len(dates); index++ {
		previousDate, currentDate := dates[index-1], dates[index]
		previousIndicator := indicatorByDate[previousDate.Format("2006-01-02")]
		previousRate := rateByDate[previousDate.Format("2006-01-02")]
		currentIndicator := indicatorByDate[currentDate.Format("2006-01-02")]
		currentRate := rateByDate[currentDate.Format("2006-01-02")]
		if previousIndicator == 0 || previousRate == 0 {
			continue
		}
		pairs = append(pairs, returnPair{
			date:      currentDate,
			indicator: (currentIndicator - previousIndicator) / math.Abs(previousIndicator),
			rate:      (currentRate - previousRate) / math.Abs(previousRate),
		})
	}
	return pairs
}

func correlationForLag(pairs []returnPair, lag int) []returnPair {
	var matched []returnPair
	for i := range pairs {
		j := i + lag
		if j < 0 || j >= len(pairs) {
			continue
		}
		matched = append(matched, returnPair{indicator: pairs[i].indicator, rate: pairs[j].rate})
	}
	return matched
}

func correlation(pairs []returnPair) *float64 {
	if len(pairs) < 2 {
		return nil
	}
	var meanX, meanY float64
	for _, pair := range pairs {
		meanX += pair.indicator
		meanY += pair.rate
	}
	meanX /= float64(len(pairs))
	meanY /= float64(len(pairs))
	var covariance, varianceX, varianceY float64
	for _, pair := range pairs {
		dx, dy := pair.indicator-meanX, pair.rate-meanY
		covariance += dx * dy
		varianceX += dx * dx
		varianceY += dy * dy
	}
	if varianceX == 0 || varianceY == 0 {
		return nil
	}
	value := covariance / math.Sqrt(varianceX*varianceY)
	return &value
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

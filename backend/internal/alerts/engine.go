package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"regexp"
)

type RuleType string

var decimalPattern = regexp.MustCompile(`^[+-]?[0-9]+(?:\.[0-9]+)?$`)

const (
	PriceThresholdType         RuleType = "price_threshold"
	PercentChangeType          RuleType = "percent_change"
	VolatilityRegimeType       RuleType = "volatility_regime"
	SMACrossType               RuleType = "sma_cross"
	IndicatorReleaseType       RuleType = "indicator_release"
	IndicatorStaleType         RuleType = "indicator_stale"
	PolicyRateChangeType       RuleType = "policy_rate_change"
	DataSourceHealthType       RuleType = "data_source_health"
	ForecastIntervalBreachType RuleType = "forecast_interval_breach"
)

type Params interface{ ruleType() RuleType }

type PriceThresholdParams struct {
	Direction string `json:"direction"`
	Level     string `json:"level"`
}

func (PriceThresholdParams) ruleType() RuleType { return PriceThresholdType }

type PercentChangeParams struct {
	PeriodDays int    `json:"period_days"`
	Direction  string `json:"direction"`
	Threshold  string `json:"threshold_percent"`
}

func (PercentChangeParams) ruleType() RuleType { return PercentChangeType }

type VolatilityRegimeParams struct {
	Regime string `json:"regime"`
}

func (VolatilityRegimeParams) ruleType() RuleType { return VolatilityRegimeType }

type SMACrossParams struct {
	Fast      int    `json:"fast"`
	Slow      int    `json:"slow"`
	Direction string `json:"direction"`
}

func (SMACrossParams) ruleType() RuleType { return SMACrossType }

type IndicatorReleaseParams struct {
	SeriesCode       string  `json:"series_code"`
	MinimumAbsChange *string `json:"minimum_abs_change,omitempty"`
}

func (IndicatorReleaseParams) ruleType() RuleType { return IndicatorReleaseType }

type IndicatorStaleParams struct {
	SeriesCode string `json:"series_code"`
	MaxAgeDays int    `json:"max_age_days"`
}

func (IndicatorStaleParams) ruleType() RuleType { return IndicatorStaleType }

type PolicyRateChangeParams struct {
	SeriesCode string `json:"series_code,omitempty"`
}

func (PolicyRateChangeParams) ruleType() RuleType { return PolicyRateChangeType }

type DataSourceHealthParams struct {
	MaxAgeMinutes  int `json:"max_age_minutes,omitempty"`
	FailuresInARow int `json:"failures_in_a_row,omitempty"`
}

func (DataSourceHealthParams) ruleType() RuleType { return DataSourceHealthType }

type ForecastIntervalBreachParams struct{}

func (ForecastIntervalBreachParams) ruleType() RuleType { return ForecastIntervalBreachType }

type Rule struct {
	Type          RuleType
	SchemaVersion int
	Params        Params
}

func ParseRule(ruleType RuleType, schemaVersion int, raw json.RawMessage) (Rule, error) {
	if schemaVersion != 1 {
		return Rule{}, fmt.Errorf("unsupported rule schema version %d", schemaVersion)
	}
	var params Params
	switch ruleType {
	case PriceThresholdType:
		params = &PriceThresholdParams{}
	case PercentChangeType:
		params = &PercentChangeParams{}
	case VolatilityRegimeType:
		params = &VolatilityRegimeParams{}
	case SMACrossType:
		params = &SMACrossParams{}
	case IndicatorReleaseType:
		params = &IndicatorReleaseParams{}
	case IndicatorStaleType:
		params = &IndicatorStaleParams{}
	case PolicyRateChangeType:
		params = &PolicyRateChangeParams{}
	case DataSourceHealthType:
		params = &DataSourceHealthParams{}
	case ForecastIntervalBreachType:
		params = &ForecastIntervalBreachParams{}
	default:
		return Rule{}, fmt.Errorf("unsupported rule type %q", ruleType)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(params); err != nil {
		return Rule{}, fmt.Errorf("invalid %s parameters: %w", ruleType, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Rule{}, fmt.Errorf("invalid %s parameters: trailing data", ruleType)
	}
	if err := validateParams(params); err != nil {
		return Rule{}, err
	}
	return Rule{Type: ruleType, SchemaVersion: schemaVersion, Params: params}, nil
}

func validateParams(params Params) error {
	switch value := params.(type) {
	case *PriceThresholdParams:
		if !validDirection(value.Direction) || !validDecimalNumber(value.Level) || !positiveDecimal(value.Level) {
			return fmt.Errorf("invalid price threshold parameters")
		}
	case *PercentChangeParams:
		if value.PeriodDays < 1 || value.PeriodDays > 365 || !validDirection(value.Direction) || !validDecimalNumber(value.Threshold) || !nonNegativeDecimal(value.Threshold) {
			return fmt.Errorf("invalid percent change parameters")
		}
	case *VolatilityRegimeParams:
		if value.Regime != "low" && value.Regime != "moderate" && value.Regime != "elevated" && value.Regime != "high" {
			return fmt.Errorf("invalid volatility regime")
		}
	case *SMACrossParams:
		validPair := (value.Fast == 7 && value.Slow == 30) || (value.Fast == 30 && value.Slow == 90)
		if !validPair || (value.Direction != "up" && value.Direction != "down") {
			return fmt.Errorf("invalid SMA crossover parameters")
		}
	case *IndicatorReleaseParams:
		if value.SeriesCode == "" || (value.MinimumAbsChange != nil && (!validDecimalNumber(*value.MinimumAbsChange) || !nonNegativeDecimal(*value.MinimumAbsChange))) {
			return fmt.Errorf("invalid indicator release parameters")
		}
	case *IndicatorStaleParams:
		if value.SeriesCode == "" || value.MaxAgeDays < 1 || value.MaxAgeDays > 3650 {
			return fmt.Errorf("invalid indicator stale parameters")
		}
	case *PolicyRateChangeParams:
	case *DataSourceHealthParams:
		if value.MaxAgeMinutes < 0 || value.FailuresInARow < 0 || (value.MaxAgeMinutes == 0 && value.FailuresInARow == 0) {
			return fmt.Errorf("data source health requires a positive age or failure threshold")
		}
	case *ForecastIntervalBreachParams:
	default:
		return fmt.Errorf("unsupported parameter type")
	}
	return nil
}

func validDirection(direction string) bool {
	return direction == "above" || direction == "below"
}

func validDecimalNumber(value string) bool {
	if !decimalPattern.MatchString(value) {
		return false
	}
	_, ok := new(big.Rat).SetString(value)
	return ok
}

func positiveDecimal(value string) bool {
	number, ok := new(big.Rat).SetString(value)
	return ok && number.Sign() > 0
}

func nonNegativeDecimal(value string) bool {
	number, ok := new(big.Rat).SetString(value)
	return ok && number.Sign() >= 0
}

type NumericFact struct {
	Value  string
	Source string
}

type SMAPoint struct {
	PreviousFast string
	PreviousSlow string
	CurrentFast  string
	CurrentSlow  string
}

type IndicatorFact struct {
	SeriesCode string
	Change     *string
	Stale      bool
	AgeDays    int
}

type ForecastFact struct {
	Actual  string
	Lower95 string
	Upper95 string
}

type Snapshot struct {
	Rate                   *NumericFact
	PercentChanges         map[int]NumericFact
	VolatilityRegime       string
	SMAs                   map[string]SMAPoint
	Indicator              *IndicatorFact
	PolicyRateChanges      map[string]bool
	DataAgeMinutes         *int
	ProviderFailuresInARow *int
	Forecast               *ForecastFact
}

type Result struct {
	Evaluable    bool
	ConditionMet bool
	Reason       string
	Facts        map[string]string
}

type Evaluator interface {
	Type() RuleType
	Evaluate(Params, Snapshot) Result
}

type Registry struct {
	evaluators map[RuleType]Evaluator
}

func NewRegistry() *Registry {
	registry := &Registry{evaluators: make(map[RuleType]Evaluator)}
	for _, evaluator := range []Evaluator{
		priceThresholdEvaluator{}, percentChangeEvaluator{}, volatilityRegimeEvaluator{}, smaCrossEvaluator{},
		indicatorReleaseEvaluator{}, indicatorStaleEvaluator{}, policyRateChangeEvaluator{},
		dataSourceHealthEvaluator{}, forecastIntervalBreachEvaluator{},
	} {
		registry.Register(evaluator)
	}
	return registry
}

func (r *Registry) Register(evaluator Evaluator) {
	r.evaluators[evaluator.Type()] = evaluator
}

func (r *Registry) Evaluate(rule Rule, snapshot Snapshot) Result {
	evaluator, ok := r.evaluators[rule.Type]
	if !ok || rule.Params == nil || rule.Params.ruleType() != rule.Type {
		return notEvaluable("rule evaluator is unavailable")
	}
	return evaluator.Evaluate(rule.Params, snapshot)
}

func notEvaluable(reason string) Result {
	return Result{Reason: reason, Facts: map[string]string{}}
}

func evaluated(condition bool, facts map[string]string) Result {
	return Result{Evaluable: true, ConditionMet: condition, Facts: facts}
}

func compareDecimal(left, right string) (int, bool) {
	l, ok := new(big.Rat).SetString(left)
	if !ok {
		return 0, false
	}
	r, ok := new(big.Rat).SetString(right)
	if !ok {
		return 0, false
	}
	return l.Cmp(r), true
}

type priceThresholdEvaluator struct{}

func (priceThresholdEvaluator) Type() RuleType { return PriceThresholdType }
func (priceThresholdEvaluator) Evaluate(params Params, snapshot Snapshot) Result {
	p := params.(*PriceThresholdParams)
	if snapshot.Rate == nil {
		return notEvaluable("latest USD/IDR rate is unavailable")
	}
	comparison, ok := compareDecimal(snapshot.Rate.Value, p.Level)
	if !ok {
		return notEvaluable("rate or threshold is not a valid decimal")
	}
	met := comparison > 0
	if p.Direction == "below" {
		met = comparison < 0
	}
	return evaluated(met, map[string]string{"rate": snapshot.Rate.Value, "threshold": p.Level, "source": snapshot.Rate.Source})
}

type percentChangeEvaluator struct{}

func (percentChangeEvaluator) Type() RuleType { return PercentChangeType }
func (percentChangeEvaluator) Evaluate(params Params, snapshot Snapshot) Result {
	p := params.(*PercentChangeParams)
	fact, ok := snapshot.PercentChanges[p.PeriodDays]
	if !ok {
		return notEvaluable("requested percent-change window is unavailable")
	}
	comparison, ok := compareDecimal(fact.Value, p.Threshold)
	if !ok {
		return notEvaluable("change or threshold is not a valid decimal")
	}
	met := comparison > 0
	if p.Direction == "below" {
		met = comparison < 0
	}
	return evaluated(met, map[string]string{"observed_change_percent": fact.Value, "threshold_percent": p.Threshold, "period_days": fmt.Sprint(p.PeriodDays), "source": fact.Source})
}

type volatilityRegimeEvaluator struct{}

func (volatilityRegimeEvaluator) Type() RuleType { return VolatilityRegimeType }
func (volatilityRegimeEvaluator) Evaluate(params Params, snapshot Snapshot) Result {
	p := params.(*VolatilityRegimeParams)
	if snapshot.VolatilityRegime == "" {
		return notEvaluable("volatility regime is unavailable")
	}
	return evaluated(snapshot.VolatilityRegime == p.Regime, map[string]string{"observed_regime": snapshot.VolatilityRegime, "requested_regime": p.Regime})
}

type smaCrossEvaluator struct{}

func (smaCrossEvaluator) Type() RuleType { return SMACrossType }
func (smaCrossEvaluator) Evaluate(params Params, snapshot Snapshot) Result {
	p := params.(*SMACrossParams)
	key := fmt.Sprintf("%d/%d", p.Fast, p.Slow)
	point, ok := snapshot.SMAs[key]
	if !ok {
		return notEvaluable("SMA history is insufficient")
	}
	previousComparison, okPrevious := compareDecimal(point.PreviousFast, point.PreviousSlow)
	currentComparison, okCurrent := compareDecimal(point.CurrentFast, point.CurrentSlow)
	if !okPrevious || !okCurrent {
		return notEvaluable("SMA values are invalid")
	}
	met := previousComparison <= 0 && currentComparison > 0
	if p.Direction == "down" {
		met = previousComparison >= 0 && currentComparison < 0
	}
	return evaluated(met, map[string]string{
		"previous_fast_sma": point.PreviousFast, "previous_slow_sma": point.PreviousSlow,
		"current_fast_sma": point.CurrentFast, "current_slow_sma": point.CurrentSlow,
		"direction": p.Direction,
	})
}

type indicatorReleaseEvaluator struct{}

func (indicatorReleaseEvaluator) Type() RuleType { return IndicatorReleaseType }
func (indicatorReleaseEvaluator) Evaluate(params Params, snapshot Snapshot) Result {
	p := params.(*IndicatorReleaseParams)
	if snapshot.Indicator == nil || snapshot.Indicator.SeriesCode != p.SeriesCode {
		return notEvaluable("indicator release is unavailable")
	}
	if p.MinimumAbsChange == nil {
		return evaluated(true, map[string]string{"series_code": p.SeriesCode})
	}
	if snapshot.Indicator.Change == nil {
		return notEvaluable("indicator change is unavailable")
	}
	change, ok := new(big.Rat).SetString(*snapshot.Indicator.Change)
	threshold, thresholdOK := new(big.Rat).SetString(*p.MinimumAbsChange)
	if !ok || !thresholdOK {
		return notEvaluable("indicator change or threshold is invalid")
	}
	change.Abs(change)
	return evaluated(change.Cmp(threshold) >= 0, map[string]string{"series_code": p.SeriesCode, "absolute_change": change.FloatString(10), "minimum_absolute_change": *p.MinimumAbsChange})
}

type indicatorStaleEvaluator struct{}

func (indicatorStaleEvaluator) Type() RuleType { return IndicatorStaleType }
func (indicatorStaleEvaluator) Evaluate(params Params, snapshot Snapshot) Result {
	p := params.(*IndicatorStaleParams)
	if snapshot.Indicator == nil || snapshot.Indicator.SeriesCode != p.SeriesCode {
		return notEvaluable("indicator staleness is unavailable")
	}
	return evaluated(snapshot.Indicator.AgeDays > p.MaxAgeDays, map[string]string{"series_code": p.SeriesCode, "age_days": fmt.Sprint(snapshot.Indicator.AgeDays), "maximum_age_days": fmt.Sprint(p.MaxAgeDays)})
}

type policyRateChangeEvaluator struct{}

func (policyRateChangeEvaluator) Type() RuleType { return PolicyRateChangeType }
func (policyRateChangeEvaluator) Evaluate(params Params, snapshot Snapshot) Result {
	p := params.(*PolicyRateChangeParams)
	if len(snapshot.PolicyRateChanges) == 0 {
		return notEvaluable("policy-rate change data is unavailable")
	}
	if p.SeriesCode != "" {
		changed, ok := snapshot.PolicyRateChanges[p.SeriesCode]
		if !ok {
			return notEvaluable("policy-rate series is unavailable")
		}
		return evaluated(changed, map[string]string{"series_code": p.SeriesCode, "changed": fmt.Sprint(changed)})
	}
	for code, changed := range snapshot.PolicyRateChanges {
		if changed {
			return evaluated(true, map[string]string{"series_code": code, "changed": "true"})
		}
	}
	return evaluated(false, map[string]string{"changed": "false"})
}

type dataSourceHealthEvaluator struct{}

func (dataSourceHealthEvaluator) Type() RuleType { return DataSourceHealthType }
func (dataSourceHealthEvaluator) Evaluate(params Params, snapshot Snapshot) Result {
	p := params.(*DataSourceHealthParams)
	if p.MaxAgeMinutes > 0 && snapshot.DataAgeMinutes == nil {
		return notEvaluable("USD/IDR data age is unavailable")
	}
	if p.FailuresInARow > 0 && snapshot.ProviderFailuresInARow == nil {
		return notEvaluable("provider failure count is unavailable")
	}
	facts := make(map[string]string, 2)
	met := false
	if snapshot.DataAgeMinutes != nil {
		facts["data_age_minutes"] = fmt.Sprint(*snapshot.DataAgeMinutes)
		met = met || (p.MaxAgeMinutes > 0 && *snapshot.DataAgeMinutes > p.MaxAgeMinutes)
	}
	if snapshot.ProviderFailuresInARow != nil {
		facts["provider_failures_in_a_row"] = fmt.Sprint(*snapshot.ProviderFailuresInARow)
		met = met || (p.FailuresInARow > 0 && *snapshot.ProviderFailuresInARow >= p.FailuresInARow)
	}
	return evaluated(met, facts)
}

type forecastIntervalBreachEvaluator struct{}

func (forecastIntervalBreachEvaluator) Type() RuleType { return ForecastIntervalBreachType }
func (forecastIntervalBreachEvaluator) Evaluate(_ Params, snapshot Snapshot) Result {
	if snapshot.Forecast == nil {
		return notEvaluable("fresh forecast interval is unavailable")
	}
	below, okBelow := compareDecimal(snapshot.Forecast.Actual, snapshot.Forecast.Lower95)
	above, okAbove := compareDecimal(snapshot.Forecast.Actual, snapshot.Forecast.Upper95)
	if !okBelow || !okAbove {
		return notEvaluable("actual or prediction interval is invalid")
	}
	return evaluated(below < 0 || above > 0, map[string]string{"actual": snapshot.Forecast.Actual, "lower_95": snapshot.Forecast.Lower95, "upper_95": snapshot.Forecast.Upper95})
}

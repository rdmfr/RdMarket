package alerts

import (
	"encoding/json"
	"testing"
)

func TestRegistryEvaluatesEveryRuleTypeAndReportsMissingData(t *testing.T) {
	zero := 0
	five := 90
	change := "2.5"
	cases := []struct {
		name     RuleType
		params   string
		snapshot Snapshot
	}{
		{PriceThresholdType, `{"direction":"above","level":"16000"}`, Snapshot{Rate: &NumericFact{Value: "16001", Source: "test"}}},
		{PercentChangeType, `{"period_days":7,"direction":"above","threshold_percent":"1"}`, Snapshot{PercentChanges: map[int]NumericFact{7: {Value: "1.2", Source: "test"}}}},
		{VolatilityRegimeType, `{"regime":"high"}`, Snapshot{VolatilityRegime: "high"}},
		{SMACrossType, `{"fast":7,"slow":30,"direction":"up"}`, Snapshot{SMAs: map[string]SMAPoint{"7/30": {PreviousFast: "100", PreviousSlow: "101", CurrentFast: "102", CurrentSlow: "101"}}}},
		{IndicatorReleaseType, `{"series_code":"US_CPI","minimum_abs_change":"1"}`, Snapshot{Indicator: &IndicatorFact{SeriesCode: "US_CPI", Change: &change}}},
		{IndicatorStaleType, `{"series_code":"US_CPI","max_age_days":30}`, Snapshot{Indicator: &IndicatorFact{SeriesCode: "US_CPI", AgeDays: 45, Stale: true}}},
		{PolicyRateChangeType, `{}`, Snapshot{PolicyRateChanges: map[string]bool{"ID_BI_RATE": true}}},
		{DataSourceHealthType, `{"max_age_minutes":60}`, Snapshot{DataAgeMinutes: &five, ProviderFailuresInARow: &zero}},
		{ForecastIntervalBreachType, `{}`, Snapshot{Forecast: &ForecastFact{Actual: "105", Lower95: "90", Upper95: "100"}}},
	}
	registry := NewRegistry()
	for _, test := range cases {
		t.Run(string(test.name), func(t *testing.T) {
			rule, err := ParseRule(test.name, 1, json.RawMessage(test.params))
			if err != nil {
				t.Fatal(err)
			}
			result := registry.Evaluate(rule, test.snapshot)
			if !result.Evaluable || !result.ConditionMet || len(result.Facts) == 0 {
				t.Fatalf("unexpected evaluator result: %+v", result)
			}
			if missing := registry.Evaluate(rule, Snapshot{}); missing.Evaluable || missing.Reason == "" {
				t.Fatalf("missing data must be not evaluable: %+v", missing)
			}
		})
	}
}

func TestParseRuleRejectsUnknownFieldsBadThresholdAndVersion(t *testing.T) {
	cases := []struct {
		typeName RuleType
		version  int
		params   string
	}{
		{PriceThresholdType, 1, `{"direction":"above","level":"0"}`},
		{PriceThresholdType, 1, `{"direction":"above","level":"16000","execute":"true"}`},
		{PriceThresholdType, 2, `{"direction":"above","level":"16000"}`},
	}
	for _, test := range cases {
		if _, err := ParseRule(test.typeName, test.version, json.RawMessage(test.params)); err == nil {
			t.Errorf("expected rule validation failure for %+v", test)
		}
	}
}

func TestSMACrossDownAndHealthThresholds(t *testing.T) {
	rule, err := ParseRule(SMACrossType, 1, json.RawMessage(`{"fast":30,"slow":90,"direction":"down"}`))
	if err != nil {
		t.Fatal(err)
	}
	result := NewRegistry().Evaluate(rule, Snapshot{SMAs: map[string]SMAPoint{"30/90": {PreviousFast: "101", PreviousSlow: "100", CurrentFast: "99", CurrentSlow: "100"}}})
	if !result.Evaluable || !result.ConditionMet {
		t.Fatalf("expected downward crossover: %+v", result)
	}
	failures := 3
	healthRule, err := ParseRule(DataSourceHealthType, 1, json.RawMessage(`{"failures_in_a_row":3}`))
	if err != nil {
		t.Fatal(err)
	}
	health := NewRegistry().Evaluate(healthRule, Snapshot{ProviderFailuresInARow: &failures})
	if !health.Evaluable || !health.ConditionMet || len(health.Facts) != 1 {
		t.Fatalf("unexpected health result: %+v", health)
	}
}

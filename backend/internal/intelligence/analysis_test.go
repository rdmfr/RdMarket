package intelligence

import (
	"math"
	"testing"
	"time"
)

func TestEvaluateRegimeExposesInputsAndInsufficientSamples(t *testing.T) {
	summary := EvaluateRegime(RegimeInput{
		Trend:          TrendInput{CurrentRate: "16000", SMA30: "15800", SMA90: "15500", Slope30: "0.02", Threshold: "0.01"},
		Volatility:     VolatilityInput{Current: 0.04, Historical: []float64{0.01, 0.02, 0.03, 0.05}, LowPercentileMax: 25, ModeratePercentileMax: 50, ElevatedPercentileMax: 75},
		Move:           MoveInput{CurrentAbsoluteReturn: 0.03, HistoricalAbsolute: []float64{0.01, 0.02, 0.03, 0.04}},
		MinimumSamples: 3,
	})
	if summary.Trend.Label != "up" || summary.Volatility.Label != "elevated" || summary.MovePercentile == nil || math.Abs(*summary.MovePercentile-75) > 1e-9 {
		t.Fatalf("unexpected regime summary: %+v", summary)
	}
	if summary.Trend.Inputs["sma_30"] != "15800" || summary.Volatility.Inputs["historical_sample_count"] != "4" {
		t.Fatalf("regime label inputs were not exposed: %+v", summary)
	}
	insufficient := EvaluateRegime(RegimeInput{MinimumSamples: 3, Move: MoveInput{HistoricalAbsolute: []float64{0.01}}})
	if insufficient.Volatility.Label != "not_enough_data" || insufficient.MovePercentile != nil {
		t.Fatalf("expected insufficient sample labels: %+v", insufficient)
	}
}

func TestRenderBriefIsDeterministicAndOmitsEmptySections(t *testing.T) {
	observed := time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)
	facts := []BriefFact{
		{ID: "b", Section: "moves", Sentence: "Observed move was 0.2 percent.", Source: "market", ObservedAt: observed},
		{ID: "a", Section: "snapshot", Sentence: "Latest USD/IDR observation was 16,000.", Source: "market", ObservedAt: observed},
	}
	first, err := RenderBrief(facts)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderBrief(facts)
	if err != nil {
		t.Fatal(err)
	}
	if first.Text != second.Text || len(first.Facts) != 2 || first.Facts[0].ID != "a" || first.TemplateVersion != "v1" {
		t.Fatalf("brief is not deterministic: %+v", first)
	}
	if len(first.Text) == 0 || first.Text == "No new releases" {
		t.Fatalf("renderer added content without a matching fact: %q", first.Text)
	}
}

func TestRenderBriefRejectsUntraceableFacts(t *testing.T) {
	if _, err := RenderBrief([]BriefFact{{ID: "x", Section: "forecast", Sentence: "Text without source."}}); err == nil {
		t.Fatal("expected incomplete fact to be rejected")
	}
}

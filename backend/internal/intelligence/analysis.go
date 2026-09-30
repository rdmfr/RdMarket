package intelligence

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TrendInput struct {
	CurrentRate string
	SMA30       string
	SMA90       string
	Slope30     string
	Threshold   string
}

type VolatilityInput struct {
	Current               float64
	Historical            []float64
	LowPercentileMax      float64
	ModeratePercentileMax float64
	ElevatedPercentileMax float64
}

type MoveInput struct {
	CurrentAbsoluteReturn float64
	HistoricalAbsolute    []float64
}

type RegimeInput struct {
	Trend          TrendInput
	Volatility     VolatilityInput
	Move           MoveInput
	MinimumSamples int
}

type RegimeComponent struct {
	Label  string            `json:"label"`
	Inputs map[string]string `json:"inputs"`
	Reason string            `json:"reason,omitempty"`
}

type RegimeSummary struct {
	Trend           RegimeComponent `json:"trend"`
	Volatility      RegimeComponent `json:"volatility"`
	MovePercentile  *float64        `json:"move_percentile"`
	MoveSampleCount int             `json:"move_sample_count"`
}

func EvaluateRegime(input RegimeInput) RegimeSummary {
	minimum := input.MinimumSamples
	if minimum < 1 {
		minimum = 1
	}
	trend := evaluateTrend(input.Trend)
	volatility := evaluateVolatility(input.Volatility, minimum)
	movePercentile := percentile(input.Move.CurrentAbsoluteReturn, input.Move.HistoricalAbsolute)
	if len(input.Move.HistoricalAbsolute) < minimum {
		movePercentile = nil
	}
	return RegimeSummary{
		Trend: trend, Volatility: volatility,
		MovePercentile: movePercentile, MoveSampleCount: len(input.Move.HistoricalAbsolute),
	}
}

func evaluateTrend(input TrendInput) RegimeComponent {
	component := RegimeComponent{
		Label: "not_enough_data",
		Inputs: map[string]string{
			"current_rate": input.CurrentRate, "sma_30": input.SMA30,
			"sma_90": input.SMA90, "slope_30": input.Slope30,
			"slope_threshold": input.Threshold,
		},
		Reason: "Not enough data",
	}
	if input.CurrentRate == "" || input.SMA30 == "" || input.SMA90 == "" || input.Slope30 == "" || input.Threshold == "" {
		return component
	}
	current, ok1 := parseDecimal(input.CurrentRate)
	sma30, ok2 := parseDecimal(input.SMA30)
	sma90, ok3 := parseDecimal(input.SMA90)
	slope, ok4 := parseDecimal(input.Slope30)
	threshold, ok5 := parseDecimal(input.Threshold)
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		return component
	}
	if current > sma30 && current > sma90 && slope > threshold {
		component.Label = "up"
	} else if current < sma30 && current < sma90 && slope < -threshold {
		component.Label = "down"
	} else {
		component.Label = "sideways"
	}
	component.Reason = ""
	return component
}

func evaluateVolatility(input VolatilityInput, minimum int) RegimeComponent {
	component := RegimeComponent{
		Label: "not_enough_data",
		Inputs: map[string]string{
			"current_volatility":      formatNumber(input.Current),
			"historical_sample_count": fmt.Sprint(len(input.Historical)),
			"low_percentile_max":      formatNumber(input.LowPercentileMax),
			"moderate_percentile_max": formatNumber(input.ModeratePercentileMax),
			"elevated_percentile_max": formatNumber(input.ElevatedPercentileMax),
		},
		Reason: "Not enough data",
	}
	if !finite(input.Current) || input.Current < 0 || len(input.Historical) < minimum || !validPercentileThresholds(input) {
		return component
	}
	for _, value := range input.Historical {
		if !finite(value) || value < 0 {
			return component
		}
	}
	percentileRank := percentile(*absPointer(input.Current), input.Historical)
	switch {
	case *percentileRank <= input.LowPercentileMax:
		component.Label = "low"
	case *percentileRank <= input.ModeratePercentileMax:
		component.Label = "moderate"
	case *percentileRank <= input.ElevatedPercentileMax:
		component.Label = "elevated"
	default:
		component.Label = "high"
	}
	component.Inputs["historical_percentile"] = formatNumber(*percentileRank)
	component.Reason = ""
	return component
}

func validPercentileThresholds(input VolatilityInput) bool {
	return input.LowPercentileMax >= 0 && input.LowPercentileMax < input.ModeratePercentileMax && input.ModeratePercentileMax < input.ElevatedPercentileMax && input.ElevatedPercentileMax <= 100
}

func percentile(value float64, historical []float64) *float64 {
	if !finite(value) || len(historical) == 0 {
		return nil
	}
	lessOrEqual := 0
	for _, previous := range historical {
		if !finite(previous) {
			return nil
		}
		if previous <= value {
			lessOrEqual++
		}
	}
	rank := float64(lessOrEqual) * 100 / float64(len(historical))
	return &rank
}

func parseDecimal(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || !finite(parsed) {
		return 0, false
	}
	return parsed, true
}

func formatNumber(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.8f", value), "0"), ".")
}

func absPointer(value float64) *float64 {
	absolute := math.Abs(value)
	return &absolute
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

type BriefFact struct {
	ID         string    `json:"id"`
	Section    string    `json:"section"`
	Sentence   string    `json:"sentence"`
	Source     string    `json:"source"`
	ObservedAt time.Time `json:"observed_at"`
}

type RenderedBrief struct {
	Text            string      `json:"rendered_text"`
	Facts           []BriefFact `json:"facts"`
	TemplateVersion string      `json:"template_version"`
}

var briefSectionOrder = []string{
	"snapshot", "regime", "moves", "releases", "policy_rate_changes", "alerts", "data_quality", "forecast",
}

func RenderBrief(facts []BriefFact) (RenderedBrief, error) {
	knownSections := make(map[string]int, len(briefSectionOrder))
	for index, section := range briefSectionOrder {
		knownSections[section] = index
	}
	seen := make(map[string]struct{}, len(facts))
	ordered := append([]BriefFact(nil), facts...)
	for _, fact := range ordered {
		if fact.ID == "" || fact.Sentence == "" || fact.Source == "" || fact.ObservedAt.IsZero() {
			return RenderedBrief{}, fmt.Errorf("brief facts require an id, sentence, source, and observation time")
		}
		if _, ok := knownSections[fact.Section]; !ok {
			return RenderedBrief{}, fmt.Errorf("unsupported brief section %q", fact.Section)
		}
		if _, exists := seen[fact.ID]; exists {
			return RenderedBrief{}, fmt.Errorf("duplicate brief fact id %q", fact.ID)
		}
		seen[fact.ID] = struct{}{}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		left, right := knownSections[ordered[i].Section], knownSections[ordered[j].Section]
		if left != right {
			return left < right
		}
		return ordered[i].ID < ordered[j].ID
	})
	var lines []string
	for _, section := range briefSectionOrder {
		var sentences []string
		for _, fact := range ordered {
			if fact.Section == section {
				sentences = append(sentences, fact.Sentence)
			}
		}
		if len(sentences) > 0 {
			lines = append(lines, strings.Join(sentences, " "))
		}
	}
	return RenderedBrief{Text: strings.Join(lines, "\n"), Facts: ordered, TemplateVersion: "v1"}, nil
}

package economic

import (
	"math"
	"testing"
	"time"
)

func TestBuildSnapshotUsesStoredValuesAndFrequencyTolerance(t *testing.T) {
	series := SeriesDefinition{Code: "X", Frequency: "monthly", ChangeMode: "percent", SourceProvider: "test", SourceSeriesID: "x", PublicationLagDays: 31}
	observations := []StoredObservation{
		{ImportedObservation: ImportedObservation{ReferenceDate: date(2026, 7, 1), Value: "100"}},
		{ImportedObservation: ImportedObservation{ReferenceDate: date(2026, 8, 1), PeriodEnd: datePtr(2026, 8, 31), Value: "105"}},
	}
	snapshot, err := BuildSnapshot(series, observations, date(2026, 10, 1), "1")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Change != "5" || snapshot.Trend != "rising" || snapshot.Stale {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}

func TestCalculateChangeAndTrend(t *testing.T) {
	change, err := calculateChange("1.25", "1", "bps")
	if err != nil || change != "25" {
		t.Fatalf("bps change = %q, %v", change, err)
	}
	if _, err := calculateChange("1", "0", "percent"); err == nil {
		t.Fatal("expected percent change from zero to be undefined")
	}
	if got := classifyTrend("-2", "1"); got != "falling" {
		t.Fatalf("trend = %q", got)
	}
}

func TestComputeRelationshipRequiresEnoughSamplesAndDoesNotFillGaps(t *testing.T) {
	dates := []time.Time{date(2026, 1, 1), date(2026, 2, 1), date(2026, 3, 1), date(2026, 4, 1), date(2026, 5, 1)}
	indicator := []DatedValue{{dates[0], 100}, {dates[1], 102}, {dates[2], 101}, {dates[3], 104}, {dates[4], 105}}
	rate := []DatedValue{{dates[0], 15000}, {dates[1], 15300}, {dates[2], 15150}, {dates[3], 15600}, {dates[4], 15750}}
	result := ComputeRelationship(indicator, rate, 10, 1, 3)
	if result.SampleCount != 4 || result.RollingCorrelation == nil || math.Abs(*result.RollingCorrelation-1) > 1e-9 {
		t.Fatalf("unexpected relationship: %+v", result)
	}
	indicator = indicator[:2]
	result = ComputeRelationship(indicator, rate, 10, 1, 3)
	if !result.NotEnoughData || result.RollingCorrelation != nil {
		t.Fatalf("expected insufficient sample result: %+v", result)
	}
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func datePtr(year int, month time.Month, day int) *time.Time {
	value := date(year, month, day)
	return &value
}

package timeutil

import (
	"testing"
	"time"
)

func TestParseISO_And_FormatISO(t *testing.T) {
	input := "2026-03-29T10:00:00Z"
	parsed, err := ParseISO(input)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if parsed.Location() != time.UTC {
		t.Errorf("parsed time should be in UTC, got: %v", parsed.Location())
	}
	formatted := FormatISO(parsed)
	if formatted != input {
		t.Errorf("expected %s, got %s", input, formatted)
	}
}

func TestToDisplayTime_AsiaJakarta(t *testing.T) {
	// 2026-03-29 00:00:00 UTC should be 07:00:00 WIB
	utcTime := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	displayTime := ToDisplayTime(utcTime)

	if displayTime.Hour() != 7 {
		t.Errorf("expected 7:00 in Jakarta, got %d", displayTime.Hour())
	}

	displayStr := FormatDisplay(utcTime)
	expectedStr := "29 Mar 2026 07:00 WIB"
	if displayStr != expectedStr {
		t.Errorf("expected %q, got %q", expectedStr, displayStr)
	}
}

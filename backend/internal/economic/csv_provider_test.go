package economic

import (
	"errors"
	"strings"
	"testing"
)

func TestCSVImportProviderReturnsAcceptedAndRejectedRows(t *testing.T) {
	input := "reference_date,value,release_timestamp,period_end\n" +
		"2026-09-01,4.25,2026-09-15T08:00:00Z,2026-09-30\n" +
		"2026-09-02,not-a-number,,\n" +
		"2026-09-01,4.50,,\n" +
		"2026-09-03,4.30,2026-09-02T08:00:00Z,\n"

	result, err := (CSVImportProvider{}).Parse(strings.NewReader(input), 1024)
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted != 1 || result.Rejected != 3 || len(result.Rows) != 4 {
		t.Fatalf("unexpected import counts: %+v", result)
	}
	if result.Rows[0].Observation == nil || result.Rows[0].Observation.Value != "4.25" {
		t.Fatalf("unexpected accepted row: %+v", result.Rows[0])
	}
	if result.Rows[1].Reason == "" || result.Rows[2].Reason != "Duplicate reference_date in this file" || result.Rows[3].Reason == "" {
		t.Fatalf("rejected rows need explicit reasons: %+v", result.Rows)
	}
}

func TestCSVImportProviderRejectsInvalidHeadersAndOversizedFiles(t *testing.T) {
	provider := CSVImportProvider{}
	if _, err := provider.Parse(strings.NewReader("date,value\n2026-01-01,1"), 1024); err == nil {
		t.Fatal("expected unsupported header error")
	}
	if _, err := provider.Parse(strings.NewReader("reference_date,value\n2026-01-01,1"), 8); !errors.Is(err, ErrCSVTooLarge) {
		t.Fatalf("expected size limit error, got %v", err)
	}
}

func TestValidDecimalHonorsNUMERICPrecision(t *testing.T) {
	for _, value := range []string{"-0.25", "0", "99999999999999.1234567890"} {
		if !validDecimal(value) {
			t.Errorf("expected %q to be accepted", value)
		}
	}
	for _, value := range []string{"1e3", "100000000000000.1", "1.12345678901", "NaN"} {
		if validDecimal(value) {
			t.Errorf("expected %q to be rejected", value)
		}
	}
}

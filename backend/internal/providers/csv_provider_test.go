package providers

import (
	"strings"
	"testing"
	"time"
)

func TestCsvImportProvider_ParseCSV(t *testing.T) {
	csvData := `date,rate
2026-03-25,16100.50
2026-03-26,16150.00
2026-03-27,16200.75
`
	prov := NewCsvImportProvider()
	rates, err := prov.ParseCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(rates) != 3 {
		t.Fatalf("expected 3 rates, got %d", len(rates))
	}

	if rates[0].Rate != 16100.50 {
		t.Errorf("expected 16100.50, got %f", rates[0].Rate)
	}
	expectedDate, _ := time.Parse("2006-01-02", "2026-03-25")
	if !rates[0].Timestamp.Equal(expectedDate) {
		t.Errorf("expected %v, got %v", expectedDate, rates[0].Timestamp)
	}
}

func TestCsvImportProvider_Capabilities(t *testing.T) {
	prov := NewCsvImportProvider()
	caps := prov.Capabilities()
	if caps.SupportsIntraday {
		t.Errorf("csv provider should not claim intraday support")
	}
	if !caps.SupportsHistory {
		t.Errorf("csv provider should support history")
	}
}

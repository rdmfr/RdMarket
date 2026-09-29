package providers

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"rdmarket-intelligence/backend/internal/models"
	"strconv"
	"strings"
	"time"
)

type CsvImportProvider struct {
	Source string
	rates  []models.ExchangeRate
}

func NewCsvImportProvider() *CsvImportProvider {
	return &CsvImportProvider{
		Source: "CSV Import",
		rates:  make([]models.ExchangeRate, 0),
	}
}

func (p *CsvImportProvider) GetProviderName() string {
	return p.Source
}

func (p *CsvImportProvider) Capabilities() Capabilities {
	return Capabilities{
		SupportsIntraday: false,
		SupportsHistory:  true,
		MaxHistoryDays:   3650,
		Granularity:      "daily",
		RateLimit:        "local file, no network quota",
		RequiresAPIKey:   false,
		LicenseNote:      "Historical data loaded from local CSV files",
	}
}

func (p *CsvImportProvider) LoadFromCSVFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open csv file: %w", err)
	}
	defer func() { _ = f.Close() }()

	rates, err := p.ParseCSV(f)
	if err != nil {
		return err
	}
	p.rates = rates
	return nil
}

func (p *CsvImportProvider) ParseCSV(r io.Reader) ([]models.ExchangeRate, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv records: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("csv file is empty or missing data rows")
	}

	header := records[0]
	dateIdx, rateIdx := -1, -1
	for i, h := range header {
		clean := strings.ToLower(strings.TrimSpace(h))
		if clean == "date" || clean == "timestamp" || clean == "time" {
			dateIdx = i
		} else if clean == "rate" || clean == "close" || clean == "price" || clean == "value" {
			rateIdx = i
		}
	}

	if dateIdx == -1 || rateIdx == -1 {
		return nil, fmt.Errorf("csv must contain 'date' and 'rate' columns")
	}

	var results []models.ExchangeRate
	now := time.Now().UTC()

	for lineNum, row := range records[1:] {
		if len(row) <= dateIdx || len(row) <= rateIdx {
			continue
		}
		dateStr := strings.TrimSpace(row[dateIdx])
		rateStr := strings.TrimSpace(row[rateIdx])
		if dateStr == "" || rateStr == "" {
			continue
		}

		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			t, err = time.Parse(time.RFC3339, dateStr)
			if err != nil {
				continue // skip invalid row
			}
		}

		rateVal, err := strconv.ParseFloat(rateStr, 64)
		if err != nil || rateVal <= 0 {
			continue
		}

		results = append(results, models.ExchangeRate{
			CurrencyPair:  "USD/IDR",
			Timestamp:     t.UTC(),
			Rate:          rateVal,
			Source:        p.Source,
			QualityStatus: "ok",
			FetchedAt:     &now,
		})
		_ = lineNum
	}

	return results, nil
}

func (p *CsvImportProvider) FetchCurrentRate(base, target string) (*models.ExchangeRate, error) {
	if len(p.rates) == 0 {
		return nil, fmt.Errorf("no exchange rate records loaded in CSV provider")
	}
	latest := p.rates[len(p.rates)-1]
	return &latest, nil
}

func (p *CsvImportProvider) FetchHistoricalRates(base, target string, start, end time.Time) ([]models.ExchangeRate, error) {
	var filtered []models.ExchangeRate
	for _, r := range p.rates {
		if !r.Timestamp.Before(start) && !r.Timestamp.After(end) {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

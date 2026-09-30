package economic

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
)

//go:embed indicators.json
var catalogFiles embed.FS

type SeriesDefinition struct {
	Code               string `json:"code"`
	Name               string `json:"name"`
	Country            string `json:"country"`
	Category           string `json:"category"`
	Unit               string `json:"unit"`
	Frequency          string `json:"frequency"`
	SeasonalAdjustment bool   `json:"seasonal_adjustment"`
	SourceProvider     string `json:"source_provider"`
	SourceSeriesID     string `json:"source_series_id"`
	SourceURL          string `json:"source_url"`
	LicenseNote        string `json:"license_note"`
	Description        string `json:"description"`
	ChangeMode         string `json:"change_mode"`
	TrendThreshold     string `json:"trend_threshold"`
	PublicationLagDays int    `json:"publication_lag_days"`
	IsActive           bool   `json:"is_active"`
}

func LoadCatalog() ([]SeriesDefinition, error) {
	file, err := catalogFiles.Open("indicators.json")
	if err != nil {
		return nil, fmt.Errorf("open economic catalog: %w", err)
	}
	defer func() { _ = file.Close() }()
	return DecodeCatalog(file)
}

func DecodeCatalog(reader io.Reader) ([]SeriesDefinition, error) {
	var series []SeriesDefinition
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&series); err != nil {
		return nil, fmt.Errorf("decode economic catalog: %w", err)
	}
	if len(series) == 0 {
		return nil, fmt.Errorf("economic catalog must contain at least one series")
	}
	seen := make(map[string]struct{}, len(series))
	for _, item := range series {
		if err := item.Validate(); err != nil {
			return nil, fmt.Errorf("series %q: %w", item.Code, err)
		}
		if _, exists := seen[item.Code]; exists {
			return nil, fmt.Errorf("duplicate economic series code %q", item.Code)
		}
		seen[item.Code] = struct{}{}
	}
	return series, nil
}

func (s SeriesDefinition) Validate() error {
	if s.Code == "" || s.Name == "" || s.Country == "" || s.Unit == "" || s.SourceProvider == "" || s.SourceSeriesID == "" || s.Description == "" || s.SourceURL == "" || s.LicenseNote == "" || s.TrendThreshold == "" {
		return fmt.Errorf("required catalog field is empty")
	}
	switch s.Category {
	case "monetary_policy", "inflation", "growth", "trade", "rates", "external", "market":
	default:
		return fmt.Errorf("unsupported category %q", s.Category)
	}
	switch s.Frequency {
	case "daily", "weekly", "monthly", "quarterly":
	default:
		return fmt.Errorf("unsupported frequency %q", s.Frequency)
	}
	switch s.ChangeMode {
	case "level", "percent", "bps":
	default:
		return fmt.Errorf("unsupported change mode %q", s.ChangeMode)
	}
	threshold, validThreshold := new(big.Rat).SetString(s.TrendThreshold)
	if !validDecimal(s.TrendThreshold) || !validThreshold || threshold.Sign() < 0 {
		return fmt.Errorf("trend threshold must be a non-negative decimal")
	}
	if s.PublicationLagDays < 1 {
		return fmt.Errorf("publication lag must be at least one day")
	}
	return nil
}

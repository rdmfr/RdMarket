package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"rdmarket-intelligence/backend/internal/models"
	"time"
)

type FrankfurterProvider struct {
	baseURL    string
	httpClient *http.Client
}

type frankfurterLatestResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

type frankfurterHistoryResponse struct {
	Amount    float64                       `json:"amount"`
	Base      string                        `json:"base"`
	StartDate string                        `json:"start_date"`
	EndDate   string                        `json:"end_date"`
	Rates     map[string]map[string]float64 `json:"rates"`
}

func NewFrankfurterProvider(baseURL string) *FrankfurterProvider {
	if baseURL == "" {
		baseURL = "https://api.frankfurter.dev/v1"
	}
	return &FrankfurterProvider{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *FrankfurterProvider) GetProviderName() string {
	return "Frankfurter (ECB)"
}

func (p *FrankfurterProvider) Capabilities() Capabilities {
	return Capabilities{
		SupportsHistory: true,
		Granularity:     "daily working-day reference rate",
		RateLimit:       "provider terms apply",
		LicenseNote:     "ECB reference rates via Frankfurter; verify terms before public deployment",
	}
}

func (p *FrankfurterProvider) FetchCurrentRate(base, target string) (*models.ExchangeRate, error) {
	url := fmt.Sprintf("%s/latest?base=%s&symbols=%s", p.baseURL, base, target)
	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("external provider connection error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("external provider HTTP error status: %d", resp.StatusCode)
	}

	var data frankfurterLatestResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode provider response: %w", err)
	}

	rateVal, ok := data.Rates[target]
	if !ok {
		return nil, fmt.Errorf("target currency %s not found in rates", target)
	}

	date, err := time.Parse("2006-01-02", data.Date)
	if err != nil {
		date = time.Now().UTC()
	}

	return &models.ExchangeRate{
		CurrencyPair: fmt.Sprintf("%s/%s", base, target),
		Timestamp:    date.UTC(),
		Rate:         rateVal,
		Source:       p.GetProviderName(),
	}, nil
}

func (p *FrankfurterProvider) FetchHistoricalRates(base, target string, start, end time.Time) ([]models.ExchangeRate, error) {
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")
	url := fmt.Sprintf("%s/%s..%s?base=%s&symbols=%s", p.baseURL, startStr, endStr, base, target)

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history from provider: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider returned status: %d", resp.StatusCode)
	}

	var data frankfurterHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode history response: %w", err)
	}

	var rates []models.ExchangeRate
	for dateStr, symbolRates := range data.Rates {
		if rateVal, ok := symbolRates[target]; ok {
			parsedDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}
			rates = append(rates, models.ExchangeRate{
				CurrencyPair: fmt.Sprintf("%s/%s", base, target),
				Timestamp:    parsedDate.UTC(),
				Rate:         rateVal,
				Source:       p.GetProviderName(),
			})
		}
	}

	return rates, nil
}

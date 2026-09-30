package economic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const blsAPIEndpoint = "https://api.bls.gov/publicAPI/v2/timeseries/data/"

var blsSeriesPattern = regexp.MustCompile(`^[A-Z0-9]{8,32}$`)

type BLSProvider struct {
	apiKey   string
	client   *http.Client
	endpoint string
}

func NewBLSProvider(apiKey string) *BLSProvider {
	return &BLSProvider{
		apiKey:   apiKey,
		client:   &http.Client{Timeout: 15 * time.Second},
		endpoint: blsAPIEndpoint,
	}
}

func (p *BLSProvider) GetProviderName() string {
	return "bls_api"
}

func (p *BLSProvider) Fetch(ctx context.Context, seriesID string, start, end time.Time) ([]ImportedObservation, error) {
	seriesID = strings.TrimSpace(seriesID)
	if !blsSeriesPattern.MatchString(seriesID) {
		return nil, fmt.Errorf("invalid BLS series id")
	}
	if start.IsZero() || end.IsZero() || end.Before(start) || end.Year()-start.Year() > 24 {
		return nil, fmt.Errorf("BLS date range must be valid and no longer than 25 calendar years")
	}
	requestData := map[string]any{
		"seriesid":  []string{seriesID},
		"startyear": fmt.Sprintf("%04d", start.Year()),
		"endyear":   fmt.Sprintf("%04d", end.Year()),
	}
	if p.apiKey != "" {
		requestData["registrationkey"] = p.apiKey
	}
	body, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("encode BLS request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create BLS request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	response, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request BLS data: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BLS returned HTTP %d", response.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, 4*1024*1024+1))
	if err != nil {
		return nil, fmt.Errorf("read BLS response: %w", err)
	}
	if len(content) > 4*1024*1024 {
		return nil, fmt.Errorf("BLS response exceeds 4 MiB")
	}
	var payload blsResponse
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, fmt.Errorf("decode BLS response: %w", err)
	}
	if payload.Status != "REQUEST_SUCCEEDED" || len(payload.Results.Series) != 1 || payload.Results.Series[0].SeriesID != seriesID {
		return nil, fmt.Errorf("BLS request was not successful")
	}

	observations := make([]ImportedObservation, 0, len(payload.Results.Series[0].Data))
	for _, row := range payload.Results.Series[0].Data {
		if !strings.HasPrefix(row.Period, "M") || row.Period == "M13" {
			continue
		}
		monthText := strings.TrimPrefix(row.Period, "M")
		month, parseErr := time.Parse("01", monthText)
		year, yearErr := strconv.Atoi(row.Year)
		if parseErr != nil || yearErr != nil || len(monthText) != 2 || len(row.Year) != 4 {
			return nil, fmt.Errorf("BLS returned an invalid monthly period")
		}
		referenceDate := time.Date(year, month.Month(), 1, 0, 0, 0, 0, time.UTC)
		if referenceDate.Before(start) || referenceDate.After(end) {
			continue
		}
		if !validDecimal(row.Value) {
			return nil, fmt.Errorf("BLS returned a non-decimal value for %s", referenceDate.Format("2006-01"))
		}
		periodEnd := referenceDate.AddDate(0, 1, -1)
		observations = append(observations, ImportedObservation{
			ReferenceDate: referenceDate,
			PeriodEnd:     &periodEnd,
			Value:         row.Value,
		})
	}
	sort.Slice(observations, func(i, j int) bool {
		return observations[i].ReferenceDate.Before(observations[j].ReferenceDate)
	})
	return observations, nil
}

type blsResponse struct {
	Status  string `json:"status"`
	Results struct {
		Series []struct {
			SeriesID string `json:"seriesID"`
			Data     []struct {
				Year   string `json:"year"`
				Period string `json:"period"`
				Value  string `json:"value"`
			} `json:"data"`
		} `json:"series"`
	} `json:"Results"`
}

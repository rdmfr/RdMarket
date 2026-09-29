package forecasting

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type JobSubmission struct {
	JobID string `json:"job_id"`
}

type ForecastingClient interface {
	SubmitJob(context.Context, JobSubmission) error
}

type ModelCatalogClient interface {
	Models(context.Context) ([]ModelInfo, error)
}

type HTTPForecastingClient struct {
	baseURL string
	token   string
	client  *http.Client
}

type MockForecastingClient struct {
	SubmitError error
	Catalog     []ModelInfo
	ModelError  error
}

func (c *MockForecastingClient) SubmitJob(context.Context, JobSubmission) error {
	return c.SubmitError
}

func (c *MockForecastingClient) Models(context.Context) ([]ModelInfo, error) {
	if c.ModelError != nil {
		return nil, c.ModelError
	}
	return append([]ModelInfo(nil), c.Catalog...), nil
}

func NewHTTPForecastingClient(baseURL, token string) *HTTPForecastingClient {
	return &HTTPForecastingClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *HTTPForecastingClient) SubmitJob(ctx context.Context, submission JobSubmission) (resultErr error) {
	if c.token == "" {
		return fmt.Errorf("forecast internal token is not configured")
	}
	body, err := json.Marshal(submission)
	if err != nil {
		return fmt.Errorf("encode forecast job submission: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/jobs", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create forecast job request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("submit forecast job: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close forecast job response: %w", err))
		}
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("forecast service rejected job with status %d", resp.StatusCode)
	}
	return nil
}

func (c *HTTPForecastingClient) Models(ctx context.Context) (models []ModelInfo, resultErr error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create forecast models request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("retrieve forecast models: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close forecast models response: %w", err))
		}
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("forecast service returned model catalog status %d", resp.StatusCode)
	}
	var payload struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode forecast model catalog: %w", err)
	}
	if payload.Data == nil {
		return nil, fmt.Errorf("forecast service returned an invalid model catalog")
	}
	return payload.Data, nil
}

package forecasting

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name    string
		request JobRequest
		wantErr bool
	}{
		{
			name:    "valid forecast",
			request: JobRequest{JobType: "forecast", Models: []string{"naive", "ets"}, Horizon: 5},
		},
		{
			name:    "valid experimental model",
			request: JobRequest{JobType: "forecast", Models: []string{"lstm", "ridge", "gradient_boosting"}, Horizon: 5},
		},
		{
			name:    "invalid job type",
			request: JobRequest{JobType: "other", Models: []string{"naive"}, Horizon: 1},
			wantErr: true,
		},
		{
			name:    "unsupported horizon",
			request: JobRequest{JobType: "forecast", Models: []string{"naive"}, Horizon: 2},
			wantErr: true,
		},
		{
			name:    "unknown model",
			request: JobRequest{JobType: "forecast", Models: []string{"invented"}, Horizon: 1},
			wantErr: true,
		},
		{
			name:    "duplicate model",
			request: JobRequest{JobType: "backtest", Models: []string{"naive", "naive"}, Horizon: 1},
			wantErr: true,
		},
		{
			name:    "bounded params reject fractions",
			request: JobRequest{JobType: "backtest", Models: []string{"naive"}, Horizon: 1, Params: map[string]any{"max_folds": 2.5}},
			wantErr: true,
		},
		{
			name:    "bounded params accept valid integer",
			request: JobRequest{JobType: "backtest", Models: []string{"naive"}, Horizon: 1, Params: map[string]any{"max_folds": 20}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRequest(test.request)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateRequest() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}

func TestValidateAvailableModelsRejectsModelsMissingFromInstalledBuild(t *testing.T) {
	catalog := []ModelInfo{
		{Name: "naive", Available: true},
		{Name: "lstm", Available: false},
	}
	if err := validateAvailableModels([]string{"naive"}, catalog); err != nil {
		t.Fatalf("available baseline rejected: %v", err)
	}
	if err := validateAvailableModels([]string{"lstm"}, catalog); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("unavailable LSTM error = %v, want ErrInvalidRequest", err)
	}
	if err := validateAvailableModels([]string{"unknown"}, catalog); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("unknown model error = %v, want ErrInvalidRequest", err)
	}
}

func TestHTTPForecastingClientSubmitsAuthenticatedJob(t *testing.T) {
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/internal/jobs" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := NewHTTPForecastingClient(server.URL, "test-token")
	if err := client.SubmitJob(context.Background(), JobSubmission{JobID: "job-id"}); err != nil {
		t.Fatalf("SubmitJob() error = %v", err)
	}
	if gotAuthorization != "Bearer test-token" {
		t.Fatalf("Authorization = %q, want bearer token", gotAuthorization)
	}
}

func TestHTTPForecastingClientSurfacesUnavailableService(t *testing.T) {
	client := NewHTTPForecastingClient("http://127.0.0.1:1", "test-token")
	if err := client.SubmitJob(context.Background(), JobSubmission{JobID: "job-id"}); err == nil {
		t.Fatal("SubmitJob() expected service failure")
	}
}

func TestHTTPForecastingClientLoadsModelAvailability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/models" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"name":"lstm","version":"keras-lstm-v1","stability":"experimental","label":"experimental","available":false,"unavailable_reason":"Not installed: tensorflow","min_history":750,"minimum_observations":750,"description":"LSTM"}]}`))
	}))
	defer server.Close()

	catalog, err := NewHTTPForecastingClient(server.URL, "").Models(context.Background())
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}
	if len(catalog) != 1 || catalog[0].Name != "lstm" || catalog[0].Available || catalog[0].UnavailableReason == nil {
		t.Fatalf("Models() = %#v, want unavailable LSTM with reason", catalog)
	}
}

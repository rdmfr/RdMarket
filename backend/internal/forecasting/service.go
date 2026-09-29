package forecasting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"rdmarket-intelligence/backend/internal/config"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidRequest = errors.New("invalid forecast request")
	ErrDuplicateJob   = errors.New("an equivalent forecast job is already running")
	ErrNotFound       = errors.New("forecast result not found")
)

var supportedHorizons = map[int]struct{}{1: {}, 5: {}, 10: {}, 21: {}, 63: {}}

var supportedModels = map[string]ModelInfo{
	"naive":  {Name: "naive", Version: "1", Stability: "stable", Label: "baseline", Available: true, MinHistory: 2, MinimumObservations: 2, Description: "Last observed rate carried forward."},
	"drift":  {Name: "drift", Version: "1", Stability: "stable", Label: "baseline", Available: true, MinHistory: 3, MinimumObservations: 3, Description: "Naive forecast with the average observed change."},
	"ets":    {Name: "ets", Version: "1", Stability: "stable", Label: "stable", Available: true, MinHistory: 30, MinimumObservations: 30, Description: "Exponential smoothing on the training observations."},
	"arima":  {Name: "arima", Version: "1", Stability: "stable", Label: "stable", Available: true, MinHistory: 250, MinimumObservations: 250, Description: "ARIMA with bounded AIC search on training data."},
	"sarima": {Name: "sarima", Version: "1", Label: "stable", MinHistory: 500, Description: "Seasonal ARIMA with bounded training-only search."},
}

func init() {
	supportedModels["seasonal_naive"] = ModelInfo{
		Name: "seasonal_naive", Version: "1", Stability: "stable", Label: "baseline",
		Available: true, MinHistory: 7, MinimumObservations: 7,
		Description: "Repeats the latest weekly seasonal pattern.",
	}
	supportedModels["ridge"] = ModelInfo{
		Name: "ridge", Version: "scikit-learn", Stability: "experimental", Label: "experimental",
		MinHistory: 250, MinimumObservations: 250,
		Description: "Ridge regression on lagged log-returns and rolling statistics.",
	}
	supportedModels["gradient_boosting"] = ModelInfo{
		Name: "gradient_boosting", Version: "scikit-learn", Stability: "experimental", Label: "experimental",
		MinHistory: 250, MinimumObservations: 250,
		Description: "Gradient boosting on lagged log-returns and rolling statistics.",
	}
	supportedModels["lstm"] = ModelInfo{
		Name: "lstm", Version: "keras", Stability: "experimental", Label: "experimental",
		MinHistory: 750, MinimumObservations: 750,
		Description: "Keras LSTM on sliding windows of log-returns.",
	}
}

type ModelInfo struct {
	Name                string  `json:"name"`
	Version             string  `json:"version"`
	Stability           string  `json:"stability"`
	Label               string  `json:"label"`
	Available           bool    `json:"available"`
	UnavailableReason   *string `json:"unavailable_reason"`
	MinHistory          int     `json:"min_history"`
	MinimumObservations int     `json:"minimum_observations"`
	Description         string  `json:"description"`
}

type JobRequest struct {
	JobType string         `json:"job_type"`
	Models  []string       `json:"models"`
	Horizon int            `json:"horizon"`
	Params  map[string]any `json:"params"`
	Force   bool           `json:"force"`
}

type Job struct {
	ID            string     `json:"id"`
	JobType       string     `json:"job_type"`
	Status        string     `json:"status"`
	ProgressDone  int        `json:"progress_done"`
	ProgressTotal int        `json:"progress_total"`
	ErrorCode     *string    `json:"error_code"`
	ErrorMessage  *string    `json:"error_message"`
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
}

type forecastRun struct {
	ID                  string          `gorm:"column:id" json:"id"`
	ModelName           string          `gorm:"column:model_name" json:"model_name"`
	ModelVersion        string          `gorm:"column:model_version" json:"model_version"`
	ForecastOrigin      time.Time       `gorm:"column:forecast_origin" json:"forecast_origin"`
	Horizon             int             `gorm:"column:horizon" json:"horizon"`
	TrainingStart       time.Time       `gorm:"column:training_start" json:"training_start"`
	TrainingEnd         time.Time       `gorm:"column:training_end" json:"training_end"`
	TrainingRows        int             `gorm:"column:training_rows" json:"training_rows"`
	MissingObservations int             `gorm:"column:missing_observations" json:"missing_observations"`
	DataFingerprint     string          `gorm:"column:data_fingerprint" json:"data_fingerprint"`
	IntervalLevels      json.RawMessage `gorm:"column:interval_levels" json:"interval_levels"`
	GeneratedAt         time.Time       `gorm:"column:created_at" json:"generated_at"`
}

type Service struct {
	db     *gorm.DB
	client ForecastingClient
	cfg    *config.Config
}

func NewService(db *gorm.DB, client ForecastingClient, cfg *config.Config) *Service {
	return &Service{db: db, client: client, cfg: cfg}
}

func (s *Service) Models(ctx context.Context) ([]ModelInfo, error) {
	catalog, ok := s.client.(ModelCatalogClient)
	if !ok {
		return nil, fmt.Errorf("forecast client does not support model catalog")
	}
	models, err := catalog.Models(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}
	return models, nil
}

func validateRequest(req JobRequest) error {
	if req.JobType != "forecast" && req.JobType != "backtest" {
		return fmt.Errorf("%w: unsupported job_type", ErrInvalidRequest)
	}
	if _, ok := supportedHorizons[req.Horizon]; !ok {
		return fmt.Errorf("%w: unsupported horizon", ErrInvalidRequest)
	}
	if len(req.Models) == 0 || len(req.Models) > len(supportedModels) {
		return fmt.Errorf("%w: select one or more supported models", ErrInvalidRequest)
	}
	seen := make(map[string]struct{}, len(req.Models))
	for _, name := range req.Models {
		if _, ok := supportedModels[name]; !ok {
			return fmt.Errorf("%w: unsupported model", ErrInvalidRequest)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("%w: duplicate model", ErrInvalidRequest)
		}
		seen[name] = struct{}{}
	}
	if req.Params == nil {
		req.Params = make(map[string]any)
	}
	for _, key := range []string{"initial_train_size", "step_size", "max_folds", "window_size"} {
		if value, ok := req.Params[key]; ok {
			if !validBoundedInteger(value) {
				return fmt.Errorf("%w: invalid %s", ErrInvalidRequest, key)
			}
		}
	}
	return nil
}

func validateAvailableModels(names []string, catalog []ModelInfo) error {
	available := make(map[string]bool, len(catalog))
	for _, model := range catalog {
		available[model.Name] = model.Available
	}
	for _, name := range names {
		if !available[name] {
			return fmt.Errorf("%w: selected model is unavailable", ErrInvalidRequest)
		}
	}
	return nil
}

func validBoundedInteger(value any) bool {
	switch n := value.(type) {
	case float64:
		return n >= 1 && n <= 10000 && n == float64(int(n))
	case int:
		return n >= 1 && n <= 10000
	case int64:
		return n >= 1 && n <= 10000
	default:
		return false
	}
}

func (s *Service) CreateJob(ctx context.Context, request JobRequest, requestedBy *string) (*Job, error) {
	if err := validateRequest(request); err != nil {
		return nil, err
	}
	catalog, err := s.Models(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateAvailableModels(request.Models, catalog); err != nil {
		return nil, err
	}
	id := uuid.NewString()
	modelsJSON, err := json.Marshal(request.Models)
	if err != nil {
		return nil, fmt.Errorf("encode forecast models: %w", err)
	}
	params := request.Params
	if params == nil {
		params = make(map[string]any)
	}
	params["horizon"] = request.Horizon
	params["force"] = request.Force
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("encode forecast parameters: %w", err)
	}
	job := Job{ID: id, JobType: request.JobType, Status: "queued", CreatedAt: time.Now().UTC()}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(541996771)").Error; err != nil {
			return fmt.Errorf("serialize forecast job creation: %w", err)
		}
		var running int64
		if err := tx.Raw("SELECT COUNT(*) FROM forecast_jobs WHERE status IN ('queued', 'running')").Scan(&running).Error; err != nil {
			return fmt.Errorf("count active forecast jobs: %w", err)
		}
		if running >= int64(s.cfg.ForecastMaxConcurrent) {
			return ErrDuplicateJob
		}
		var duplicate bool
		if err := tx.Raw(`SELECT EXISTS (
			SELECT 1 FROM forecast_jobs
			WHERE status IN ('queued', 'running') AND job_type = ?
			AND model_names = ?::jsonb AND (params - 'force') = (?::jsonb - 'force')
		)`, request.JobType, string(modelsJSON), string(paramsJSON)).Scan(&duplicate).Error; err != nil {
			return fmt.Errorf("check duplicate forecast job: %w", err)
		}
		if duplicate {
			return ErrDuplicateJob
		}
		return tx.Exec(
			`INSERT INTO forecast_jobs (id, job_type, status, currency_pair, model_names, params, progress_done, progress_total, requested_by, created_at)
			 VALUES (?::uuid, ?, 'queued', 'USD/IDR', ?::jsonb, ?::jsonb, 0, ?, ?, ?)`,
			id, request.JobType, string(modelsJSON), string(paramsJSON), len(request.Models), requestedBy, job.CreatedAt,
		).Error
	}); err != nil {
		if errors.Is(err, ErrDuplicateJob) {
			return nil, ErrDuplicateJob
		}
		return nil, fmt.Errorf("create forecast job: %w", err)
	}
	if err := s.client.SubmitJob(ctx, JobSubmission{JobID: id}); err != nil {
		message := "The forecasting service is temporarily unavailable"
		code := "FORECAST_SERVICE_UNAVAILABLE"
		if updateErr := s.db.Exec(
			`UPDATE forecast_jobs SET status = 'failed', error_code = ?, error_message = ?, finished_at = ? WHERE id = ?::uuid`,
			code, message, time.Now().UTC(), id,
		).Error; updateErr != nil {
			return nil, fmt.Errorf("forecast service submission failed (%v); record failure: %w", err, updateErr)
		}
		return nil, fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
	}
	return &job, nil
}

var ErrServiceUnavailable = errors.New("forecast service unavailable")

func (s *Service) Job(id string) (*Job, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrNotFound
	}
	var job Job
	if err := s.db.Raw(`SELECT id::text, job_type, status, progress_done, progress_total, error_code, error_message, created_at, started_at, finished_at
		FROM forecast_jobs WHERE id = ?::uuid`, id).Scan(&job).Error; err != nil {
		return nil, fmt.Errorf("load forecast job: %w", err)
	}
	if job.ID == "" {
		return nil, ErrNotFound
	}
	return &job, nil
}

func (s *Service) Jobs(limit, offset int) ([]Job, error) {
	var jobs []Job
	if err := s.db.Raw(`SELECT id::text, job_type, status, progress_done, progress_total, error_code, error_message, created_at, started_at, finished_at
		FROM forecast_jobs ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset).Scan(&jobs).Error; err != nil {
		return nil, fmt.Errorf("list forecast jobs: %w", err)
	}
	return jobs, nil
}

func (s *Service) Latest(model string, horizon int) (map[string]any, error) {
	if _, ok := supportedModels[model]; !ok {
		return nil, ErrInvalidRequest
	}
	if _, ok := supportedHorizons[horizon]; !ok {
		return nil, ErrInvalidRequest
	}
	var run forecastRun
	if err := s.db.Raw(`SELECT id::text, model_name, model_version, forecast_origin, horizon, training_start, training_end,
		training_rows, missing_observations, data_fingerprint, interval_levels, created_at
		FROM forecast_runs WHERE currency_pair = 'USD/IDR' AND model_name = ? AND horizon = ?
		ORDER BY created_at DESC LIMIT 1`, model, horizon).Scan(&run).Error; err != nil {
		return nil, fmt.Errorf("load latest forecast: %w", err)
	}
	if run.ID == "" {
		return nil, ErrNotFound
	}
	return s.runResponse(run)
}

func (s *Service) Run(id string) (map[string]any, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrNotFound
	}
	var run forecastRun
	if err := s.db.Raw(`SELECT id::text, model_name, model_version, forecast_origin, horizon, training_start, training_end,
		training_rows, missing_observations, data_fingerprint, interval_levels, created_at
		FROM forecast_runs WHERE id = ?::uuid`, id).Scan(&run).Error; err != nil {
		return nil, fmt.Errorf("load forecast run: %w", err)
	}
	if run.ID == "" {
		return nil, ErrNotFound
	}
	return s.runResponse(run)
}

func (s *Service) runResponse(run forecastRun) (map[string]any, error) {
	var levels []float64
	if len(run.IntervalLevels) > 0 {
		if err := json.Unmarshal(run.IntervalLevels, &levels); err != nil {
			return nil, fmt.Errorf("decode forecast interval levels: %w", err)
		}
	}
	var forecasts []map[string]any
	if err := s.db.Raw(`SELECT target_timestamp, step, point::float8 AS point,
		lower_80::float8 AS lower_80, upper_80::float8 AS upper_80,
		lower_95::float8 AS lower_95, upper_95::float8 AS upper_95
		FROM forecasts WHERE run_id = ?::uuid ORDER BY step`, run.ID).Scan(&forecasts).Error; err != nil {
		return nil, fmt.Errorf("load forecast points: %w", err)
	}
	var history []map[string]any
	if err := s.db.Raw(`SELECT timestamp, rate FROM (
		SELECT timestamp, rate::float8 AS rate FROM exchange_rates
		WHERE currency_pair = 'USD/IDR' AND quality_status = 'ok' AND timestamp <= ?
		ORDER BY timestamp DESC LIMIT 120
	) observed ORDER BY timestamp`, run.ForecastOrigin).Scan(&history).Error; err != nil {
		return nil, fmt.Errorf("load forecast history: %w", err)
	}
	var baselineSkill *float64
	err := s.db.Raw(`SELECT metric_value::float8 FROM model_evaluations e JOIN backtest_runs b ON b.id = e.backtest_run_id
		WHERE b.currency_pair = 'USD/IDR' AND b.model_name = ? AND e.horizon = ?
		AND e.metric_name = 'skill_score_vs_naive' ORDER BY b.created_at DESC LIMIT 1`,
		run.ModelName, run.Horizon).Scan(&baselineSkill).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load naive baseline comparison: %w", err)
	}
	var beatsNaive *bool
	if baselineSkill != nil {
		value := *baselineSkill > 0
		beatsNaive = &value
	}
	metadata := map[string]any{
		"id": run.ID, "model_name": run.ModelName, "model_version": run.ModelVersion,
		"forecast_origin": run.ForecastOrigin, "horizon": run.Horizon, "training_start": run.TrainingStart,
		"training_end": run.TrainingEnd, "training_rows": run.TrainingRows,
		"missing_observations": run.MissingObservations, "data_fingerprint": run.DataFingerprint,
		"interval_levels": levels, "generated_at": run.GeneratedAt,
	}
	return map[string]any{
		"run":                 metadata,
		"forecasts":           forecasts,
		"history":             history,
		"baseline_comparison": map[string]any{"skill_score_vs_naive": baselineSkill, "beats_naive": beatsNaive},
	}, nil
}

func (s *Service) Leaderboard(horizon int) (map[string]any, error) {
	if _, ok := supportedHorizons[horizon]; !ok {
		return nil, ErrInvalidRequest
	}
	var rows []map[string]any
	err := s.db.Raw(`WITH latest AS (
		SELECT DISTINCT ON (b.model_name) b.id, b.model_name
		FROM backtest_runs b WHERE b.currency_pair = 'USD/IDR' ORDER BY b.model_name, b.created_at DESC
	), metrics AS (
		SELECT l.model_name,
			(MAX(e.metric_value) FILTER (WHERE e.metric_name = 'mae'))::float8 AS mae,
			(MAX(e.metric_value) FILTER (WHERE e.metric_name = 'rmse'))::float8 AS rmse,
			(MAX(e.metric_value) FILTER (WHERE e.metric_name = 'mape'))::float8 AS mape,
			(MAX(e.metric_value) FILTER (WHERE e.metric_name = 'mase'))::float8 AS mase,
			(MAX(e.metric_value) FILTER (WHERE e.metric_name = 'directional_accuracy'))::float8 AS directional_accuracy,
			(MAX(e.metric_value) FILTER (WHERE e.metric_name = 'interval_coverage'))::float8 AS interval_coverage,
			(MAX(e.metric_value) FILTER (WHERE e.metric_name = 'skill_score_vs_naive'))::float8 AS skill_score_vs_naive
		FROM latest l JOIN model_evaluations e ON e.backtest_run_id = l.id
		WHERE e.horizon = ? GROUP BY l.model_name
	)
	SELECT model_name, ? AS horizon, mae, rmse, mape, mase, directional_accuracy, interval_coverage,
		skill_score_vs_naive, (model_name IN ('naive', 'drift')) AS is_baseline
	FROM metrics ORDER BY is_baseline DESC, skill_score_vs_naive DESC NULLS LAST, model_name`,
		horizon, horizon).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load forecast leaderboard: %w", err)
	}
	hasEvaluations := false
	noModelBeatsNaive := true
	for _, row := range rows {
		if row["model_name"] != "naive" && row["model_name"] != "drift" {
			if score, ok := row["skill_score_vs_naive"].(float64); ok && score > 0 {
				noModelBeatsNaive = false
			}
			if _, ok := row["skill_score_vs_naive"].(float64); ok {
				hasEvaluations = true
			}
		}
	}
	for _, baseline := range []string{"naive", "drift"} {
		found := false
		for _, row := range rows {
			if row["model_name"] == baseline {
				found = true
				break
			}
		}
		if !found {
			rows = append(rows, map[string]any{
				"model_name": baseline, "horizon": horizon, "mae": nil, "rmse": nil,
				"mape": nil, "mase": nil, "directional_accuracy": nil, "interval_coverage": nil,
				"skill_score_vs_naive": nil, "is_baseline": true,
			})
		}
	}
	noModelBeatsNaive = hasEvaluations && noModelBeatsNaive
	return map[string]any{
		"horizon": horizon, "items": rows, "no_model_beats_naive": noModelBeatsNaive,
		"has_evaluations": hasEvaluations,
	}, nil
}

func (s *Service) Backtests(limit, offset int) ([]map[string]any, error) {
	var runs []map[string]any
	err := s.db.Raw(`SELECT b.id::text, b.model_name, b.strategy, b.horizon, b.n_folds, b.created_at,
		COALESCE(jsonb_object_agg(e.metric_name, e.metric_value) FILTER (WHERE e.metric_name IS NOT NULL), '{}'::jsonb) AS metrics
		FROM backtest_runs b LEFT JOIN model_evaluations e ON e.backtest_run_id = b.id AND e.horizon = b.horizon
		WHERE b.currency_pair = 'USD/IDR' GROUP BY b.id ORDER BY b.created_at DESC LIMIT ? OFFSET ?`, limit, offset).Scan(&runs).Error
	if err != nil {
		return nil, fmt.Errorf("list backtests: %w", err)
	}
	return runs, nil
}

func (s *Service) Backtest(id string) (map[string]any, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrNotFound
	}
	var run map[string]any
	if err := s.db.Raw(`SELECT id::text, model_name, strategy, initial_train_size, step_size, horizon, window_size,
		n_folds, started_at, finished_at, data_fingerprint, hyperparameters, created_at
		FROM backtest_runs WHERE id = ?::uuid`, id).Scan(&run).Error; err != nil {
		return nil, fmt.Errorf("load backtest: %w", err)
	}
	if run["id"] == nil {
		return nil, ErrNotFound
	}
	var metrics []map[string]any
	if err := s.db.Raw(`SELECT model_name, horizon, metric_name, metric_value, n_observations
		FROM model_evaluations WHERE backtest_run_id = ?::uuid ORDER BY horizon, metric_name`, id).Scan(&metrics).Error; err != nil {
		return nil, fmt.Errorf("load backtest metrics: %w", err)
	}
	run["metrics"] = metrics
	return run, nil
}

func (s *Service) Predictions(id string, limit, offset int) ([]map[string]any, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrNotFound
	}
	var exists bool
	if err := s.db.Raw(`SELECT EXISTS(SELECT 1 FROM backtest_runs WHERE id = ?::uuid)`, id).Scan(&exists).Error; err != nil {
		return nil, fmt.Errorf("check backtest: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}
	var predictions []map[string]any
	if err := s.db.Raw(`SELECT fold, forecast_origin, target_timestamp, step, actual, predicted, lower_95, upper_95
		FROM backtest_predictions WHERE backtest_run_id = ?::uuid ORDER BY fold, step LIMIT ? OFFSET ?`, id, limit, offset).Scan(&predictions).Error; err != nil {
		return nil, fmt.Errorf("load backtest predictions: %w", err)
	}
	return predictions, nil
}

func parsePage(value string, fallback, maximum int) int {
	n := fallback
	if value != "" {
		if parsed, err := fmt.Sscanf(value, "%d", &n); err != nil || parsed != 1 {
			return fallback
		}
	}
	if n < 1 {
		return fallback
	}
	if n > maximum {
		return maximum
	}
	return n
}

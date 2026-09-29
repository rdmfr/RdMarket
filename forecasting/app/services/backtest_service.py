from __future__ import annotations

from decimal import Decimal
from time import monotonic

import numpy as np
import pandas as pd
from sqlalchemy import text

from app.config import Settings
from app.domain.errors import InsufficientDataError, JobTimeoutError, ModelExecutionError
from app.domain.fingerprint import data_fingerprint
from app.domain.metrics import calculate_metrics
from app.domain.splits import walk_forward_folds
from app.models.registry import create_model, minimum_observations
from app.repositories.database import db_json
from app.services.forecast_service import _model_series, _to_rate


def execute_backtest(
    connection, job: dict, series: pd.Series, settings: Settings, progress, deadline: float | None = None
) -> None:
    params = dict(job.get("params") or {})
    names = list(params.get("model_names") or job.get("model_names") or ["naive", "drift", "ets"])
    if "naive" not in names:
        names.insert(0, "naive")
    horizon = int(params.get("horizon", 21))
    strategy = params.get("strategy", "expanding")
    initial = int(params.get("initial_train_size", 250))
    step_size = int(params.get("step_size", horizon))
    max_folds = min(int(params.get("max_folds", settings.max_backtest_folds)), settings.max_backtest_folds)
    window_size = params.get("window_size")
    if len(set(names)) != len(names) or horizon > settings.max_horizon:
        raise ModelExecutionError("Invalid backtest job parameters")
    folds = walk_forward_folds(len(series), initial, step_size, horizon, max_folds, strategy, window_size)
    model_folds = {
        name: folds[-min(len(folds), settings.lstm_max_folds):] if name == "lstm" else folds
        for name in names
    }
    total = sum(len(selected) for selected in model_folds.values())
    completed = 0
    fingerprint = data_fingerprint(series)
    transform = bool(params.get("log_transform", settings.log_transform))

    # Evaluate the naive baseline first so every model receives the same-fold skill comparator.
    naive_fold_errors: dict[int, float] = {}
    staged: dict[str, list[dict]] = {}
    for model_name in names:
        if len(series) < minimum_observations(model_name, settings):
            raise InsufficientDataError(f"Model {model_name} has insufficient total history")
        staged[model_name] = []
        model = create_model(model_name, settings)
        refit_frequency = (
            int(model.hyperparameters.get("refit_every_n_folds", 1))
            if model_name == "lstm"
            else 1
        )
        selected_folds = model_folds[model_name]
        for fold_index, fold in enumerate(selected_folds):
            if deadline is not None and monotonic() >= deadline:
                raise JobTimeoutError()
            training_raw = series.iloc[fold.train_start:fold.train_end]
            training = _model_series(training_raw, transform)
            try:
                if fold_index % refit_frequency == 0:
                    model.fit(training)
                elif hasattr(model, "advance_origin"):
                    model.advance_origin(training)
                result = model.predict(horizon)
            except Exception as exc:
                if isinstance(exc, InsufficientDataError | ModelExecutionError):
                    raise
                raise ModelExecutionError(f"Model {model_name} failed during backtest fold {fold.number}") from exc
            actual = series.iloc[fold.test_start:fold.test_end].to_numpy(dtype=float)
            predicted = np.asarray([_to_rate(value, transform) for value in result.point], dtype=float)
            lower95 = np.asarray([_to_rate(value, transform) for value in result.lower[0.95]], dtype=float)
            upper95 = np.asarray([_to_rate(value, transform) for value in result.upper[0.95]], dtype=float)
            scale = float(np.mean(np.abs(np.diff(training_raw.to_numpy(dtype=float))))) if len(training_raw) > 1 else 0
            metrics = calculate_metrics(
                actual,
                predicted,
                lower95,
                upper95,
                scale=scale,
                previous_actual=np.asarray([training_raw.iloc[-1]] * len(actual)),
            )
            staged[model_name].append(
                {
                    "fold": fold, "model": model, "actual": actual, "predicted": predicted,
                    "lower": lower95, "upper": upper95, "metrics": metrics,
                    "training_start": training_raw.index[0].to_pydatetime(),
                    "training_end": training_raw.index[-1].to_pydatetime(),
                }
            )
            if model_name == "naive":
                naive_fold_errors[fold.number] = metrics["mae"]
            if deadline is not None and monotonic() >= deadline:
                raise JobTimeoutError()
            completed += 1
            progress(completed, total)

    for name in names:
        model_records = staged[name]
        run_hyperparameters = dict(model_records[0]["model"].hyperparameters)
        run_hyperparameters["max_folds"] = len(model_records)
        if name == "lstm":
            run_hyperparameters["requested_max_folds"] = max_folds
            run_hyperparameters["configured_max_folds"] = settings.lstm_max_folds
        run_id = connection.execute(
            text(
                """INSERT INTO backtest_runs
                (job_id,currency_pair,model_name,model_version,strategy,initial_train_size,
                 step_size,horizon,window_size,n_folds,started_at,finished_at,data_fingerprint,hyperparameters)
                VALUES (:job_id,:pair,:model,:version,:strategy,:initial,:step,:horizon,:window,:folds,
                 CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,:fingerprint,CAST(:params AS jsonb)) RETURNING id"""
            ),
            {
                "job_id": job["id"], "pair": job["currency_pair"], "model": name,
                "version": model_records[0]["model"].version, "strategy": strategy, "initial": initial,
                "step": step_size, "horizon": horizon, "window": window_size,
                "folds": len(model_records),
                "fingerprint": fingerprint,
                "params": db_json(run_hyperparameters),
            },
        ).scalar_one()
        predictions = []
        metric_totals: dict[str, list[float]] = {}
        for record in model_records:
            fold = record["fold"]
            actual = record["actual"]
            predicted = record["predicted"]
            lower, upper = record["lower"], record["upper"]
            test_dates = series.index[fold.test_start:fold.test_end]
            for step_index, timestamp in enumerate(test_dates):
                predictions.append(
                    {
                        "backtest_run_id": run_id, "fold": fold.number,
                        "forecast_origin": series.index[fold.train_end - 1].to_pydatetime(),
                        "target_timestamp": timestamp.to_pydatetime(), "step": step_index + 1,
                        "actual": Decimal(str(actual[step_index])),
                        "predicted": Decimal(str(predicted[step_index])),
                        "lower_95": Decimal(str(lower[step_index])),
                        "upper_95": Decimal(str(upper[step_index])),
                    }
                )
            comparator = naive_fold_errors.get(fold.number)
            metrics = calculate_metrics(
                actual, predicted, lower, upper, naive_mae=comparator,
                scale=float(np.mean(np.abs(np.diff(series.iloc[fold.train_start:fold.train_end].to_numpy(dtype=float))))) if fold.train_end - fold.train_start > 1 else 0,
                previous_actual=np.asarray([series.iloc[fold.train_end - 1]] * len(actual)),
            )
            for metric_name, value in metrics.items():
                metric_totals.setdefault(metric_name, []).append(value)
        connection.execute(
            text(
                """INSERT INTO backtest_predictions
                (backtest_run_id,fold,forecast_origin,target_timestamp,step,actual,predicted,lower_95,upper_95)
                VALUES (:backtest_run_id,:fold,:forecast_origin,:target_timestamp,:step,:actual,:predicted,:lower_95,:upper_95)"""
            ),
            predictions,
        )
        evaluations = [
            {
                "backtest_run_id": run_id, "model_name": name, "horizon": horizon,
                "metric_name": metric, "metric_value": Decimal(str(float(np.mean(values)))),
                "n_observations": len(predictions),
            }
            for metric, values in metric_totals.items()
        ]
        connection.execute(
            text(
                """INSERT INTO model_evaluations
                (backtest_run_id,model_name,horizon,metric_name,metric_value,n_observations)
                VALUES (:backtest_run_id,:model_name,:horizon,:metric_name,:metric_value,:n_observations)"""
            ),
            evaluations,
        )

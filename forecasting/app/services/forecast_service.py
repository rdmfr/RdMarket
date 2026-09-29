from __future__ import annotations

from decimal import Decimal
from time import monotonic

import numpy as np
import pandas as pd
from sqlalchemy import text

from app.config import Settings
from app.domain.errors import InsufficientDataError, JobTimeoutError, ModelExecutionError
from app.domain.fingerprint import data_fingerprint
from app.models.registry import create_model, minimum_observations
from app.repositories.database import db_json


def _model_series(series: pd.Series, log_transform: bool) -> pd.Series:
    if series.empty:
        raise InsufficientDataError("No accepted exchange-rate observations are available")
    if not np.all(np.isfinite(series.to_numpy())) or np.any(series.to_numpy() <= 0):
        raise ModelExecutionError("Accepted exchange-rate observations are invalid for modeling")
    return np.log(series) if log_transform else series.copy()


def _to_rate(value: float, transformed: bool) -> float:
    return float(np.exp(value)) if transformed else float(value)


def _target_dates(last_timestamp: pd.Timestamp, horizon: int, calendar_days: bool) -> list[pd.Timestamp]:
    start = last_timestamp + pd.Timedelta(days=1)
    if calendar_days:
        return list(pd.date_range(start, periods=horizon, freq="D", tz="UTC"))
    return list(pd.bdate_range(start, periods=horizon, tz="UTC"))


def execute_forecast(connection, job: dict, series: pd.Series, settings: Settings, deadline: float | None = None) -> None:
    params = dict(job.get("params") or {})
    horizon = int(params.get("horizon", 21))
    names = list(params.get("model_names") or job.get("model_names") or ["naive", "drift", "ets"])
    if horizon < 1 or horizon > settings.max_horizon or len(set(names)) != len(names):
        raise ModelExecutionError("Invalid forecast job parameters")
    transform = bool(params.get("log_transform", settings.log_transform))
    levels = tuple(sorted(float(level) for level in params.get("interval_levels", settings.default_intervals)))
    fingerprint = data_fingerprint(series)
    missing = _weekday_gaps(series.index)
    origin = series.index[-1]
    dates = _target_dates(origin, horizon, settings.calendar_days)
    training_start, training_end = series.index[0], series.index[-1]

    for name in names:
        if deadline is not None and monotonic() >= deadline:
            raise JobTimeoutError()
        model = create_model(name, settings)
        if len(series) < minimum_observations(name, settings):
            raise InsufficientDataError(f"Model {name} requires at least {minimum_observations(name, settings)} observations")
        training = _model_series(series, transform)
        try:
            model.fit(training)
            result = model.predict(horizon)
        except Exception as exc:
            if isinstance(exc, InsufficientDataError | ModelExecutionError):
                raise
            raise ModelExecutionError(f"Model {name} failed to fit") from exc
        if deadline is not None and monotonic() >= deadline:
            raise JobTimeoutError()

        points = np.asarray([_to_rate(item, transform) for item in result.point], dtype=float)
        bounds = {
            level: (
                np.asarray([_to_rate(item, transform) for item in result.lower[level]], dtype=float),
                np.asarray([_to_rate(item, transform) for item in result.upper[level]], dtype=float),
            )
            for level in levels
        }
        run_id = connection.execute(
            text(
                """INSERT INTO forecast_runs
                (job_id, currency_pair, model_name, model_version, forecast_origin, horizon,
                 interval_levels, training_start, training_end, training_rows, missing_observations,
                 data_fingerprint, hyperparameters, log_transform, code_version)
                VALUES (:job_id, :pair, :model, :version, :origin, :horizon, CAST(:levels AS jsonb),
                 :training_start, :training_end, :rows, :missing, :fingerprint,
                 CAST(:params AS jsonb), :log_transform, :code_version) RETURNING id"""
            ),
            {
                "job_id": job["id"], "pair": job["currency_pair"], "model": name, "version": model.version,
                "origin": origin.to_pydatetime(), "horizon": horizon, "levels": db_json(levels),
                "training_start": training_start.to_pydatetime(), "training_end": training_end.to_pydatetime(),
                "rows": len(series), "missing": missing, "fingerprint": fingerprint,
                "params": db_json(model.hyperparameters), "log_transform": transform,
                "code_version": settings.code_version,
            },
        ).scalar_one()
        forecast_rows = []
        for step in range(horizon):
            lower_80, upper_80 = bounds.get(0.8, (np.full(horizon, np.nan), np.full(horizon, np.nan)))
            lower_95, upper_95 = bounds.get(0.95, (np.full(horizon, np.nan), np.full(horizon, np.nan)))
            forecast_rows.append(
                {
                    "run_id": run_id,
                    "target_timestamp": dates[step].to_pydatetime(),
                    "step": step + 1,
                    "point": Decimal(str(points[step])),
                    "lower_80": Decimal(str(lower_80[step])),
                    "upper_80": Decimal(str(upper_80[step])),
                    "lower_95": Decimal(str(lower_95[step])),
                    "upper_95": Decimal(str(upper_95[step])),
                }
            )
        connection.execute(
            text(
                """INSERT INTO forecasts
                (run_id,target_timestamp,step,point,lower_80,upper_80,lower_95,upper_95)
                VALUES (:run_id,:target_timestamp,:step,:point,:lower_80,:upper_80,:lower_95,:upper_95)"""
            ),
            forecast_rows,
        )


def _weekday_gaps(index: pd.DatetimeIndex) -> int:
    if len(index) < 2:
        return 0
    days = pd.bdate_range(index[0], index[-1], tz="UTC")
    observed = set(pd.DatetimeIndex(index).normalize())
    return sum(day not in observed for day in days)

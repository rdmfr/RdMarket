from __future__ import annotations

import os
from dataclasses import dataclass
from urllib.parse import quote, urlparse


class ConfigurationError(ValueError):
    pass


def _positive_int(env: dict[str, str], name: str, default: int, maximum: int) -> int:
    raw = env.get(name, str(default))
    try:
        value = int(raw)
    except ValueError as exc:
        raise ConfigurationError(f"{name} must be an integer") from exc
    if value < 1 or value > maximum:
        raise ConfigurationError(f"{name} must be between 1 and {maximum}")
    return value


@dataclass(frozen=True)
class Settings:
    database_url: str
    internal_token: str
    max_workers: int = 1
    queue_size: int = 4
    job_timeout_seconds: int = 1800
    min_ets_observations: int = 30
    min_arima_observations: int = 250
    min_sarima_observations: int = 500
    min_ml_observations: int = 250
    min_lstm_observations: int = 750
    max_backtest_folds: int = 100
    max_horizon: int = 63
    default_intervals: tuple[float, ...] = (0.8, 0.95)
    log_transform: bool = True
    calendar_days: bool = False
    enable_ml: bool = True
    random_seed: int = 42
    lstm_max_epochs: int = 40
    lstm_window: int = 20
    lstm_layers: int = 1
    lstm_units: int = 32
    lstm_dropout: float = 0.1
    lstm_learning_rate: float = 0.001
    lstm_batch_size: int = 32
    lstm_refit_every_n_folds: int = 5
    lstm_max_folds: int = 20
    code_version: str = "0.1.0"

    @classmethod
    def from_env(cls, env: dict[str, str] | None = None) -> Settings:
        source = os.environ if env is None else env
        db_url = source.get("FORECAST_DATABASE_URL") or source.get("DATABASE_URL", "")
        if not db_url and source.get("FORECAST_DB_USER") and source.get("FORECAST_DB_PASSWORD"):
            host = source.get("FORECAST_DB_HOST", "postgres")
            port = source.get("FORECAST_DB_PORT", "5432")
            name = source.get("FORECAST_DB_NAME", "rdmarket")
            db_url = (
                f"postgresql+psycopg://{quote(source['FORECAST_DB_USER'], safe='')}:"
                f"{quote(source['FORECAST_DB_PASSWORD'], safe='')}@{host}:{port}/{name}"
            )
        token = source.get("FORECAST_INTERNAL_TOKEN", "")
        if not db_url:
            raise ConfigurationError("FORECAST_DATABASE_URL is required")
        if not token or len(token) < 32:
            raise ConfigurationError("FORECAST_INTERNAL_TOKEN must contain at least 32 characters")
        parsed = urlparse(db_url)
        if parsed.scheme not in {"postgresql", "postgresql+psycopg"}:
            raise ConfigurationError("FORECAST_DATABASE_URL must use PostgreSQL with psycopg")
        raw_levels = source.get("FORECAST_DEFAULT_INTERVALS", "0.8,0.95")
        try:
            levels = tuple(sorted({float(part.strip()) for part in raw_levels.split(",")}))
        except ValueError as exc:
            raise ConfigurationError("FORECAST_DEFAULT_INTERVALS must be comma-separated probabilities") from exc
        if not levels or any(level <= 0 or level >= 1 for level in levels):
            raise ConfigurationError("forecast interval probabilities must be between 0 and 1")
        if 0.8 not in levels or 0.95 not in levels:
            raise ConfigurationError("FORECAST_DEFAULT_INTERVALS must include 0.8 and 0.95")

        def bool_setting(name: str, default: bool) -> bool:
            raw = source.get(name, str(default)).lower()
            if raw not in {"true", "false"}:
                raise ConfigurationError(f"{name} must be true or false")
            return raw == "true"

        return cls(
            database_url=db_url.replace("postgresql://", "postgresql+psycopg://", 1),
            internal_token=token,
            max_workers=_positive_int(dict(source), "FORECAST_MAX_CONCURRENT_JOBS", 1, 8),
            queue_size=_positive_int(dict(source), "FORECAST_QUEUE_SIZE", 4, 100),
            job_timeout_seconds=_positive_int(dict(source), "FORECAST_JOB_TIMEOUT_SECONDS", 1800, 86400),
            min_ets_observations=_positive_int(dict(source), "FORECAST_MIN_ETS_OBSERVATIONS", 30, 100000),
            min_arima_observations=_positive_int(dict(source), "FORECAST_MIN_ARIMA_OBSERVATIONS", 250, 100000),
            min_sarima_observations=_positive_int(dict(source), "FORECAST_MIN_SARIMA_OBSERVATIONS", 500, 100000),
            min_ml_observations=_positive_int(dict(source), "FORECAST_MIN_ML_OBSERVATIONS", 250, 100000),
            min_lstm_observations=_positive_int(dict(source), "FORECAST_MIN_LSTM_OBSERVATIONS", 750, 100000),
            max_backtest_folds=_positive_int(dict(source), "FORECAST_MAX_BACKTEST_FOLDS", 100, 1000),
            max_horizon=_positive_int(dict(source), "FORECAST_MAX_HORIZON", 63, 365),
            default_intervals=levels,
            log_transform=bool_setting("FORECAST_LOG_TRANSFORM", True),
            calendar_days=bool_setting("FORECAST_CALENDAR_DAYS", False),
            enable_ml=bool_setting("FORECAST_ENABLE_ML", True),
            random_seed=_bounded_int(dict(source), "FORECAST_RANDOM_SEED", 42, 0, 2**31 - 1),
            lstm_max_epochs=_positive_int(dict(source), "FORECAST_LSTM_MAX_EPOCHS", 40, 1000),
            lstm_window=_positive_int(dict(source), "FORECAST_LSTM_WINDOW", 20, 250),
            lstm_layers=_bounded_int(dict(source), "FORECAST_LSTM_LAYERS", 1, 1, 2),
            lstm_units=_bounded_int(dict(source), "FORECAST_LSTM_UNITS", 32, 32, 64),
            lstm_dropout=_bounded_float(dict(source), "FORECAST_LSTM_DROPOUT", 0.1, 0.0, 0.5),
            lstm_learning_rate=_bounded_float(dict(source), "FORECAST_LSTM_LEARNING_RATE", 0.001, 1e-6, 0.1),
            lstm_batch_size=_positive_int(dict(source), "FORECAST_LSTM_BATCH_SIZE", 32, 512),
            lstm_refit_every_n_folds=_positive_int(dict(source), "FORECAST_LSTM_REFIT_EVERY_N_FOLDS", 5, 1000),
            lstm_max_folds=_positive_int(dict(source), "FORECAST_LSTM_MAX_FOLDS", 20, 1000),
            code_version=source.get("FORECAST_CODE_VERSION", "0.1.0"),
        )


def _bounded_int(env: dict[str, str], name: str, default: int, minimum: int, maximum: int) -> int:
    raw = env.get(name, str(default))
    try:
        value = int(raw)
    except ValueError as exc:
        raise ConfigurationError(f"{name} must be an integer") from exc
    if not minimum <= value <= maximum:
        raise ConfigurationError(f"{name} must be between {minimum} and {maximum}")
    return value


def _bounded_float(env: dict[str, str], name: str, default: float, minimum: float, maximum: float) -> float:
    raw = env.get(name, str(default))
    try:
        value = float(raw)
    except ValueError as exc:
        raise ConfigurationError(f"{name} must be a number") from exc
    if not minimum <= value <= maximum:
        raise ConfigurationError(f"{name} must be between {minimum} and {maximum}")
    return value

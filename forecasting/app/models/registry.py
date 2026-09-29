from __future__ import annotations

import importlib.util

from app.config import Settings
from app.domain.errors import ModelExecutionError
from app.models.baselines import DriftModel, NaiveModel, SeasonalNaiveModel
from app.models.machine_learning import GradientBoostingModel, LSTMModel, RidgeModel
from app.models.statistical import ARIMAModel, ETSModel, SARIMAModel


MODEL_DESCRIPTIONS = {
    "naive": ("stable", "Last observed rate; random-walk normal prediction intervals."),
    "drift": ("stable", "Last observed rate adjusted by the average historical change."),
    "seasonal_naive": ("stable", "Repeats the latest five-session seasonal pattern."),
    "ets": ("stable", "Damped additive-trend exponential smoothing."),
    "arima": ("stable", "Autoregressive integrated moving-average model with bounded AIC search."),
    "sarima": ("stable", "Seasonal ARIMA with bounded AIC search and five-session seasonality."),
    "ridge": ("experimental", "Ridge regression on lagged log-returns and rolling statistics."),
    "gradient_boosting": ("experimental", "Gradient boosting on lagged log-returns and rolling statistics."),
    "lstm": ("experimental", "Keras LSTM on a sliding window of log-returns."),
}
ML_MODELS = {"ridge", "gradient_boosting", "lstm"}


def model_available(name: str, settings: Settings) -> tuple[bool, str | None]:
    if name not in ML_MODELS:
        return True, None
    if not settings.enable_ml:
        return False, "Machine-learning dependency group disabled by FORECAST_ENABLE_ML"
    required = ("sklearn",) if name in {"ridge", "gradient_boosting"} else ("sklearn", "keras", "tensorflow")
    missing = [package for package in required if importlib.util.find_spec(package) is None]
    if missing:
        return False, f"Not installed: {', '.join(missing)}"
    if name == "lstm":
        try:
            import keras
            import tensorflow  # noqa: F401

            if keras.backend.backend() != "tensorflow":
                return False, "Keras TensorFlow backend is unavailable"
        except (ImportError, RuntimeError, ValueError):
            return False, "Keras TensorFlow backend is unavailable"
    return True, None


def create_model(name: str, settings: Settings):
    if name in ML_MODELS:
        available, reason = model_available(name, settings)
        if not available:
            raise ModelExecutionError(f"Model {name} is unavailable: {reason}")
        if name == "ridge":
            return RidgeModel(window=settings.lstm_window, seed=settings.random_seed)
        if name == "gradient_boosting":
            return GradientBoostingModel(window=settings.lstm_window, seed=settings.random_seed)
        return LSTMModel(
            window=settings.lstm_window,
            units=settings.lstm_units,
            dropout=settings.lstm_dropout,
            learning_rate=settings.lstm_learning_rate,
            batch_size=settings.lstm_batch_size,
            max_epochs=settings.lstm_max_epochs,
            refit_every_n_folds=settings.lstm_refit_every_n_folds,
            layers=settings.lstm_layers,
            seed=settings.random_seed,
        )

    constructors = {
        "naive": NaiveModel,
        "drift": DriftModel,
        "seasonal_naive": SeasonalNaiveModel,
        "ets": ETSModel,
        "arima": ARIMAModel,
        "sarima": SARIMAModel,
    }
    try:
        return constructors[name]()
    except KeyError as exc:
        raise ValueError(f"Unsupported model: {name}") from exc


def minimum_observations(name: str, settings: Settings) -> int:
    minimums = {
        "naive": 2,
        "drift": 3,
        "seasonal_naive": 7,
        "ets": settings.min_ets_observations,
        "arima": settings.min_arima_observations,
        "sarima": settings.min_sarima_observations,
        "ridge": settings.min_ml_observations,
        "gradient_boosting": settings.min_ml_observations,
        "lstm": settings.min_lstm_observations,
    }
    try:
        return minimums[name]
    except KeyError as exc:
        raise ValueError(f"Unsupported model: {name}") from exc


def model_catalog(settings: Settings) -> list[dict[str, object]]:
    versions = {
        "naive": "1",
        "drift": "1",
        "seasonal_naive": "1",
        "ets": "statsmodels-0.14.4",
        "arima": "statsmodels-0.14.4",
        "sarima": "statsmodels-0.14.4",
        "ridge": RidgeModel.version,
        "gradient_boosting": GradientBoostingModel.version,
        "lstm": LSTMModel.version,
    }
    catalog = []
    for name, (stability, description) in MODEL_DESCRIPTIONS.items():
        available, unavailable_reason = model_available(name, settings)
        catalog.append(
            {
                "name": name,
                "version": versions[name],
                "stability": stability,
                "label": "baseline" if name in {"naive", "drift", "seasonal_naive"} else stability,
                "available": available,
                "unavailable_reason": unavailable_reason,
                "min_history": minimum_observations(name, settings),
                "minimum_observations": minimum_observations(name, settings),
                "description": description,
            }
        )
    return catalog

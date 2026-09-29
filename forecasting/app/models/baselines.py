from __future__ import annotations

import numpy as np
import pandas as pd

from app.domain.intervals import ForecastResult, normal_intervals


class NaiveModel:
    name = "naive"
    version = "1"

    def fit(self, series: pd.Series) -> None:
        self.values = np.asarray(series, dtype=float)
        self.last = float(self.values[-1])
        self.sigma = float(np.std(np.diff(self.values), ddof=1)) if len(self.values) > 2 else 0.0

    @property
    def hyperparameters(self) -> dict[str, object]:
        return {"method": "last_observation", "interval": "random_walk_normal"}

    def predict(self, horizon: int, alpha: float = 0.05) -> ForecastResult:
        point = np.full(horizon, self.last, dtype=float)
        standard_error = self.sigma * np.sqrt(np.arange(1, horizon + 1))
        lower, upper = normal_intervals(point, standard_error)
        return ForecastResult(point, lower, upper)


class DriftModel(NaiveModel):
    name = "drift"

    def fit(self, series: pd.Series) -> None:
        super().fit(series)
        differences = np.diff(self.values)
        self.drift = float((self.values[-1] - self.values[0]) / (len(self.values) - 1))
        residuals = differences - self.drift
        self.sigma = float(np.std(residuals, ddof=1)) if len(residuals) > 1 else 0.0

    @property
    def hyperparameters(self) -> dict[str, object]:
        return {"method": "linear_drift", "interval": "drift_residual_normal"}

    def predict(self, horizon: int, alpha: float = 0.05) -> ForecastResult:
        steps = np.arange(1, horizon + 1, dtype=float)
        point = self.last + self.drift * steps
        standard_error = self.sigma * np.sqrt(steps)
        lower, upper = normal_intervals(point, standard_error)
        return ForecastResult(point, lower, upper)


class SeasonalNaiveModel(NaiveModel):
    name = "seasonal_naive"

    def fit(self, series: pd.Series) -> None:
        super().fit(series)
        if len(self.values) < 7:
            raise ValueError("seasonal_naive requires at least seven observations")
        self.seasonal = self.values[-7:]
        seasonal_differences = self.values[7:] - self.values[:-7]
        self.sigma = float(np.std(seasonal_differences, ddof=1)) if len(seasonal_differences) > 1 else 0.0

    @property
    def hyperparameters(self) -> dict[str, object]:
        return {"season_length": 7, "interval": "seasonal_error_normal"}

    def predict(self, horizon: int, alpha: float = 0.05) -> ForecastResult:
        point = np.resize(self.seasonal, horizon).astype(float)
        standard_error = self.sigma * np.sqrt(np.arange(1, horizon + 1))
        lower, upper = normal_intervals(point, standard_error)
        return ForecastResult(point, lower, upper)

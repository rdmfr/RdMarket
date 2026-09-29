from __future__ import annotations

import itertools
import warnings

import numpy as np
import pandas as pd
from statsmodels.tsa.holtwinters import ExponentialSmoothing
from statsmodels.tsa.statespace.sarimax import SARIMAX

from app.domain.errors import ModelExecutionError
from app.domain.intervals import ForecastResult, normal_intervals


class ETSModel:
    name = "ets"
    version = "statsmodels-0.14.4"

    def fit(self, series: pd.Series) -> None:
        values = np.asarray(series, dtype=float)
        with warnings.catch_warnings():
            warnings.simplefilter("ignore")
            try:
                model = ExponentialSmoothing(
                    values,
                    trend="add",
                    damped_trend=True,
                    seasonal=None,
                    initialization_method="estimated",
                )
                self.result = model.fit(optimized=True, use_brute=False)
                self.residual_sigma = float(np.sqrt(np.mean(np.square(self.result.resid))))
            except Exception as exc:
                raise ModelExecutionError() from exc

    @property
    def hyperparameters(self) -> dict[str, object]:
        return {"trend": "add", "damped_trend": True, "seasonal": None}

    def predict(self, horizon: int, alpha: float = 0.05) -> ForecastResult:
        point = np.asarray(self.result.forecast(horizon), dtype=float)
        standard_error = self.residual_sigma * np.sqrt(np.arange(1, horizon + 1))
        lower, upper = normal_intervals(point, standard_error)
        return ForecastResult(point, lower, upper)


class _InformationCriterionModel:
    def _fit_candidates(self, values: np.ndarray, candidates: list[dict[str, object]]) -> None:
        best = None
        best_aic = float("inf")
        with warnings.catch_warnings():
            warnings.simplefilter("ignore")
            for parameters in candidates:
                try:
                    fitted = SARIMAX(
                        values,
                        enforce_stationarity=False,
                        enforce_invertibility=False,
                        **parameters,
                    ).fit(disp=False, maxiter=100)
                    if np.isfinite(fitted.aic) and fitted.aic < best_aic:
                        best, best_aic = fitted, float(fitted.aic)
                except Exception:
                    continue
        if best is None:
            raise ModelExecutionError()
        self.result = best
        self.selected_aic = best_aic

    def predict(self, horizon: int, alpha: float = 0.05) -> ForecastResult:
        try:
            forecast = self.result.get_forecast(steps=horizon)
            point = np.asarray(forecast.predicted_mean, dtype=float)
            standard_error = np.asarray(forecast.se_mean, dtype=float)
            lower, upper = normal_intervals(point, standard_error)
            return ForecastResult(point, lower, upper)
        except Exception as exc:
            raise ModelExecutionError() from exc


class ARIMAModel(_InformationCriterionModel):
    name = "arima"
    version = "statsmodels-0.14.4"

    def fit(self, series: pd.Series) -> None:
        values = np.asarray(series, dtype=float)
        candidates = [
            {"order": (p, d, q), "seasonal_order": (0, 0, 0, 0), "trend": "c" if d == 0 else "n"}
            for p, d, q in itertools.product(range(3), range(2), range(3))
        ]
        self.selected_order = self._choose(values, candidates)
        self._fit_candidates(values, [self.selected_order])

    @staticmethod
    def _choose(values: np.ndarray, candidates: list[dict[str, object]]) -> dict[str, object]:
        # Search is exhaustive over a fixed 18-candidate grid; it only sees the supplied training slice.
        best = None
        best_aic = float("inf")
        with warnings.catch_warnings():
            warnings.simplefilter("ignore")
            for parameters in candidates:
                try:
                    fitted = SARIMAX(
                        values,
                        enforce_stationarity=False,
                        enforce_invertibility=False,
                        **parameters,
                    ).fit(disp=False, maxiter=100)
                    if np.isfinite(fitted.aic) and fitted.aic < best_aic:
                        best, best_aic = parameters, float(fitted.aic)
                except Exception:
                    continue
        if best is None:
            raise ModelExecutionError()
        return best

    @property
    def hyperparameters(self) -> dict[str, object]:
        return {"order": list(self.selected_order["order"]), "search": "AIC; p,q=0..2; d=0..1"}


class SARIMAModel(_InformationCriterionModel):
    name = "sarima"
    version = "statsmodels-0.14.4"

    def fit(self, series: pd.Series) -> None:
        values = np.asarray(series, dtype=float)
        candidates = [
            {
                "order": (p, d, q),
                "seasonal_order": (P, 0, Q, 5),
                "trend": "c" if d == 0 else "n",
            }
            for p, d, q, P, Q in itertools.product(range(2), range(2), range(2), range(2), range(2))
        ]
        # Restrict to eight bounded candidates while covering non-seasonal and seasonal terms.
        selected = candidates[:4] + candidates[8:12]
        self.selected_order = ARIMAModel._choose(values, selected)
        self._fit_candidates(values, [self.selected_order])

    @property
    def hyperparameters(self) -> dict[str, object]:
        return {
            "order": list(self.selected_order["order"]),
            "seasonal_order": list(self.selected_order["seasonal_order"]),
            "search": "bounded AIC; seasonal period 5",
        }

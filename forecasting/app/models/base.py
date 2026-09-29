from __future__ import annotations

from typing import Protocol

import pandas as pd

from app.domain.intervals import ForecastResult


class ForecastModel(Protocol):
    name: str
    version: str

    def fit(self, series: pd.Series) -> None: ...

    def predict(self, horizon: int, alpha: float = 0.05) -> ForecastResult: ...

    @property
    def hyperparameters(self) -> dict[str, object]: ...

from __future__ import annotations

from dataclasses import dataclass
from statistics import NormalDist

import numpy as np


@dataclass(frozen=True)
class ForecastResult:
    point: np.ndarray
    lower: dict[float, np.ndarray]
    upper: dict[float, np.ndarray]


def normal_intervals(
    point: np.ndarray,
    standard_error: np.ndarray,
    levels: tuple[float, ...] = (0.8, 0.95),
) -> tuple[dict[float, np.ndarray], dict[float, np.ndarray]]:
    lower: dict[float, np.ndarray] = {}
    upper: dict[float, np.ndarray] = {}
    for level in levels:
        z_score = NormalDist().inv_cdf((1 + level) / 2)
        lower[level] = point - z_score * standard_error
        upper[level] = point + z_score * standard_error
    return lower, upper

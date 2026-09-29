from __future__ import annotations

import numpy as np


def calculate_metrics(
    actual: np.ndarray,
    predicted: np.ndarray,
    lower_95: np.ndarray | None = None,
    upper_95: np.ndarray | None = None,
    naive_mae: float | None = None,
    scale: float | None = None,
    previous_actual: np.ndarray | None = None,
) -> dict[str, float]:
    actual = np.asarray(actual, dtype=float)
    predicted = np.asarray(predicted, dtype=float)
    errors = predicted - actual
    abs_errors = np.abs(errors)
    denominator = np.abs(actual)
    nonzero = denominator > np.finfo(float).eps
    smape_denominator = np.abs(actual) + np.abs(predicted)
    valid_smape = smape_denominator > np.finfo(float).eps
    result = {
        "mae": float(np.mean(abs_errors)),
        "rmse": float(np.sqrt(np.mean(np.square(errors)))),
        "mape": float(np.mean(abs_errors[nonzero] / denominator[nonzero]) * 100) if np.any(nonzero) else 0.0,
        "smape": float(np.mean(2 * abs_errors[valid_smape] / smape_denominator[valid_smape]) * 100)
        if np.any(valid_smape)
        else 0.0,
    }
    if scale is not None:
        result["mase"] = float(np.mean(abs_errors) / scale) if scale > 0 else 0.0
    if previous_actual is not None and len(actual) > 1:
        actual_directions = np.sign(np.diff(np.r_[previous_actual[0], actual]))
        predicted_directions = np.sign(np.diff(np.r_[previous_actual[0], predicted]))
        result["directional_accuracy"] = float(np.mean(actual_directions == predicted_directions) * 100)
    else:
        result["directional_accuracy"] = 0.0
    if lower_95 is not None and upper_95 is not None:
        result["interval_coverage_95"] = float(np.mean((actual >= lower_95) & (actual <= upper_95)) * 100)
        result["average_interval_width_95"] = float(np.mean(upper_95 - lower_95))
    if naive_mae is not None:
        result["skill_vs_naive"] = float(1 - result["mae"] / naive_mae) if naive_mae > 0 else 0.0
    return result

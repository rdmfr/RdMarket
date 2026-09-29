import numpy as np
import pandas as pd
import pytest

from app.domain.errors import InsufficientDataError
from app.domain.fingerprint import data_fingerprint
from app.domain.metrics import calculate_metrics
from app.domain.splits import walk_forward_folds


def test_walk_forward_expanding_folds_are_chronological_and_bounded():
    folds = walk_forward_folds(31, initial_train_size=10, step_size=5, horizon=3, max_folds=4)
    assert [(fold.train_start, fold.train_end, fold.test_start, fold.test_end) for fold in folds] == [
        (0, 10, 10, 13),
        (0, 15, 15, 18),
        (0, 20, 20, 23),
        (0, 25, 25, 28),
    ]
    assert all(fold.train_end <= fold.test_start for fold in folds)


def test_walk_forward_rolling_window_keeps_fixed_train_width():
    folds = walk_forward_folds(40, 10, 5, 3, 5, "rolling", 12)
    assert [(fold.train_start, fold.train_end) for fold in folds] == [
        (0, 10), (3, 15), (8, 20), (13, 25), (18, 30)
    ]


def test_walk_forward_rejects_insufficient_history_and_bad_strategy():
    with pytest.raises(InsufficientDataError):
        walk_forward_folds(12, 10, 2, 3, 5)
    with pytest.raises(ValueError, match="strategy"):
        walk_forward_folds(20, 5, 1, 1, 5, "random")


def test_metrics_known_values_and_guard_zero_denominators():
    result = calculate_metrics(
        np.array([0.0, 10.0, 20.0]),
        np.array([1.0, 8.0, 22.0]),
        np.array([-1.0, 7.0, 19.0]),
        np.array([2.0, 9.0, 23.0]),
        naive_mae=3.0,
        scale=2.0,
    )
    assert result["mae"] == pytest.approx(5 / 3)
    assert result["rmse"] == pytest.approx(np.sqrt(3))
    assert result["mape"] == pytest.approx(15)
    assert result["interval_coverage_95"] == pytest.approx(200 / 3)
    assert result["skill_vs_naive"] == pytest.approx(1 - (5 / 3) / 3)


def test_data_fingerprint_is_stable_and_sensitive_to_observations():
    index = pd.date_range("2024-01-01", periods=3, tz="UTC")
    original = pd.Series([15000.0, 15001.0, 15002.0], index=index)
    assert data_fingerprint(original) == data_fingerprint(original.copy())
    changed = original.copy()
    changed.iloc[-1] += 1
    assert data_fingerprint(original) != data_fingerprint(changed)

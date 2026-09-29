from __future__ import annotations

from dataclasses import dataclass
from typing import Iterator

from app.domain.errors import InsufficientDataError


@dataclass(frozen=True)
class Fold:
    number: int
    train_start: int
    train_end: int
    test_start: int
    test_end: int


def walk_forward_folds(
    row_count: int,
    initial_train_size: int,
    step_size: int,
    horizon: int,
    max_folds: int,
    strategy: str = "expanding",
    window_size: int | None = None,
) -> list[Fold]:
    if strategy not in {"expanding", "rolling"}:
        raise ValueError("strategy must be expanding or rolling")
    if min(row_count, initial_train_size, step_size, horizon, max_folds) < 1:
        raise ValueError("fold inputs must be positive")
    if strategy == "rolling" and (window_size is None or window_size < initial_train_size):
        raise ValueError("rolling strategy requires window_size >= initial_train_size")
    if row_count < initial_train_size + horizon:
        raise InsufficientDataError("Not enough observations to create a walk-forward fold")

    folds: list[Fold] = []
    train_end = initial_train_size
    while train_end + horizon <= row_count and len(folds) < max_folds:
        train_start = max(0, train_end - window_size) if strategy == "rolling" and window_size else 0
        folds.append(
            Fold(
                number=len(folds) + 1,
                train_start=train_start,
                train_end=train_end,
                test_start=train_end,
                test_end=train_end + horizon,
            )
        )
        train_end += step_size
    if not folds:
        raise InsufficientDataError("Not enough observations to create a walk-forward fold")
    return folds

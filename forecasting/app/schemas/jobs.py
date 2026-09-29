from __future__ import annotations

from typing import Any, Literal
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field, field_validator


class JobSubmission(BaseModel):
    model_config = ConfigDict(extra="forbid")
    job_id: UUID


class JobAccepted(BaseModel):
    job_id: UUID
    status: Literal["queued"]


class JobParameters(BaseModel):
    model_config = ConfigDict(extra="forbid")
    horizon: int = Field(default=21, ge=1, le=365)
    model_names: list[str] = Field(default_factory=lambda: ["naive", "drift", "ets"], min_length=1, max_length=6)
    force: bool = False
    log_transform: bool = True
    interval_levels: list[float] = Field(default_factory=lambda: [0.8, 0.95])
    strategy: Literal["expanding", "rolling"] = "expanding"
    initial_train_size: int = Field(default=250, ge=7, le=100000)
    step_size: int = Field(default=21, ge=1, le=365)
    max_folds: int = Field(default=20, ge=1, le=1000)
    window_size: int | None = Field(default=None, ge=7, le=100000)

    @field_validator("interval_levels")
    @classmethod
    def validate_intervals(cls, value: list[float]) -> list[float]:
        if any(level <= 0 or level >= 1 for level in value) or not {0.8, 0.95}.issubset(value):
            raise ValueError("interval_levels must include 0.8 and 0.95 and contain probabilities")
        return sorted(set(value))

    @field_validator("model_names")
    @classmethod
    def validate_models(cls, value: list[str]) -> list[str]:
        allowed = {"naive", "drift", "seasonal_naive", "ets", "arima", "sarima"}
        if any(name not in allowed for name in value) or len(set(value)) != len(value):
            raise ValueError("model_names contains an unsupported or duplicate model")
        return value


class HealthResponse(BaseModel):
    status: Literal["ok"]
    service: str = "rdmarket-forecasting"
    version: str


class ErrorResponse(BaseModel):
    error: dict[str, Any]

from __future__ import annotations

from collections.abc import Iterator

from sqlalchemy import Engine, create_engine, text
from sqlalchemy.engine import Connection


def create_database_engine(database_url: str) -> Engine:
    return create_engine(database_url, pool_pre_ping=True, pool_size=5, max_overflow=0)


class ForecastRepository:
    def __init__(self, engine: Engine):
        self.engine = engine

    def transaction(self) -> Connection:
        return self.engine.connect()

    def recover_orphaned_jobs(self) -> None:
        with self.engine.begin() as connection:
            connection.execute(
                text(
                    """
                    UPDATE forecast_jobs
                    SET status = 'failed', error_code = 'SERVICE_RESTARTED',
                        error_message = 'Service restarted while the job was running',
                        finished_at = CURRENT_TIMESTAMP
                    WHERE status = 'running'
                    """
                )
            )

    def get_job(self, job_id: str) -> dict | None:
        with self.engine.connect() as connection:
            row = connection.execute(
                text(
                    """SELECT id, job_type, status, currency_pair, model_names, params
                       FROM forecast_jobs WHERE id = :job_id"""
                ),
                {"job_id": job_id},
            ).mappings().first()
            return dict(row) if row else None

    def mark_running(self, job_id: str, total: int) -> bool:
        with self.engine.begin() as connection:
            result = connection.execute(
                text(
                    """UPDATE forecast_jobs SET status='running', started_at=CURRENT_TIMESTAMP,
                       progress_done=0, progress_total=:total, error_code=NULL, error_message=NULL
                       WHERE id=:job_id AND status='queued'"""
                ),
                {"job_id": job_id, "total": total},
            )
            return result.rowcount == 1

    def update_progress(self, job_id: str, done: int, total: int) -> None:
        with self.engine.begin() as connection:
            connection.execute(
                text("UPDATE forecast_jobs SET progress_done=:done, progress_total=:total WHERE id=:job_id"),
                {"job_id": job_id, "done": done, "total": total},
            )

    def finish_job(self, job_id: str) -> None:
        with self.engine.begin() as connection:
            connection.execute(
                text(
                    """UPDATE forecast_jobs SET status='succeeded', progress_done=progress_total,
                       finished_at=CURRENT_TIMESTAMP, error_code=NULL, error_message=NULL WHERE id=:job_id"""
                ),
                {"job_id": job_id},
            )

    def fail_job(self, job_id: str, code: str, message: str) -> None:
        with self.engine.begin() as connection:
            connection.execute(
                text(
                    """UPDATE forecast_jobs SET status='failed', error_code=:code, error_message=:message,
                       finished_at=CURRENT_TIMESTAMP WHERE id=:job_id"""
                ),
                {"job_id": job_id, "code": code, "message": message[:255]},
            )

    def load_modeling_series(self, currency_pair: str):
        import pandas as pd

        with self.engine.connect() as connection:
            rows = connection.execute(
                text(
                    """SELECT timestamp, rate FROM exchange_rates
                       WHERE currency_pair=:pair AND quality_status='ok'
                       ORDER BY timestamp ASC, id ASC"""
                ),
                {"pair": currency_pair},
            ).all()
        if not rows:
            return pd.Series(dtype=float)
        frame = pd.DataFrame(rows, columns=["timestamp", "rate"])
        frame["timestamp"] = pd.to_datetime(frame["timestamp"], utc=True)
        frame["day"] = frame["timestamp"].dt.floor("D")
        frame = frame.sort_values(["timestamp"]).drop_duplicates("day", keep="last")
        frame = frame[frame["day"].dt.dayofweek < 5]
        values = frame["rate"].astype(float).to_numpy()
        return pd.Series(values, index=pd.DatetimeIndex(frame["day"]), dtype=float, name="rate")


def db_json(value):
    import json

    return json.dumps(value, separators=(",", ":"), sort_keys=True)

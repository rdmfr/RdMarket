from __future__ import annotations

import asyncio
import logging
from time import monotonic
from concurrent.futures import ThreadPoolExecutor
from uuid import UUID

from app.config import Settings
from app.domain.errors import ForecastingError, JobTimeoutError
from app.repositories.database import ForecastRepository
from app.services.backtest_service import execute_backtest
from app.services.forecast_service import execute_forecast

logger = logging.getLogger(__name__)


class JobRunner:
    def __init__(self, repository: ForecastRepository, settings: Settings):
        self.repository = repository
        self.settings = settings
        self.queue: asyncio.Queue[str] = asyncio.Queue(maxsize=settings.queue_size)
        self.executor = ThreadPoolExecutor(max_workers=settings.max_workers, thread_name_prefix="forecast-job")
        self.tasks: list[asyncio.Task] = []
        self.submitted: set[str] = set()
        self.submitted_lock = __import__("threading").Lock()

    async def start(self) -> None:
        self.repository.recover_orphaned_jobs()
        self.tasks = [asyncio.create_task(self._worker(), name=f"forecast-worker-{i}") for i in range(self.settings.max_workers)]

    async def stop(self) -> None:
        for _ in self.tasks:
            await self.queue.put(None)
        await asyncio.gather(*self.tasks, return_exceptions=True)
        self.executor.shutdown(wait=True, cancel_futures=True)

    def submit(self, job_id: UUID) -> bool:
        job_key = str(job_id)
        with self.submitted_lock:
            if job_key in self.submitted:
                return False
        try:
            self.queue.put_nowait(job_key)
            with self.submitted_lock:
                self.submitted.add(job_key)
            return True
        except asyncio.QueueFull:
            return False

    async def _worker(self) -> None:
        loop = asyncio.get_running_loop()
        while True:
            job_id = await self.queue.get()
            try:
                if job_id is None:
                    return
                await loop.run_in_executor(self.executor, self._run_job, job_id)
            except Exception:
                logger.exception("Forecast worker failed")
            finally:
                if job_id is not None:
                    with self.submitted_lock:
                        self.submitted.discard(job_id)
                self.queue.task_done()

    def _run_job(self, job_id: str) -> None:
        job = self.repository.get_job(job_id)
        if not job or job["status"] != "queued":
            return
        try:
            if job["job_type"] not in {"forecast", "backtest"}:
                raise ForecastingError("INVALID_JOB_TYPE", "Unsupported forecasting job type")
            names = list(job.get("model_names") or (job.get("params") or {}).get("model_names") or ["naive"])
            horizon = int((job.get("params") or {}).get("horizon", 21))
            total = len(names) if job["job_type"] == "forecast" else len(names) * int((job.get("params") or {}).get("max_folds", 20))
            if not self.repository.mark_running(job_id, total):
                return
            deadline = monotonic() + self.settings.job_timeout_seconds
            series = self.repository.load_modeling_series(job["currency_pair"])
            with self.repository.transaction() as connection:
                with connection.begin():
                    if job["job_type"] == "forecast":
                        execute_forecast(connection, job, series, self.settings, deadline)
                        self.repository.update_progress(job_id, len(names), len(names))
                    else:
                        execute_backtest(
                            connection,
                            job,
                            series,
                            self.settings,
                            lambda done, actual_total: self.repository.update_progress(job_id, done, actual_total),
                            deadline,
                        )
                    if monotonic() >= deadline:
                        raise JobTimeoutError()
            self.repository.finish_job(job_id)
        except ForecastingError as exc:
            self.repository.fail_job(job_id, exc.code, exc.message)
        except Exception:
            logger.exception("Forecast job %s failed", job_id)
            self.repository.fail_job(job_id, "JOB_FAILED", "The forecasting job failed")

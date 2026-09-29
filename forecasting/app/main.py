from __future__ import annotations

from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.api.routes import router
from app.config import Settings
from app.repositories.database import ForecastRepository, create_database_engine
from app.services.job_runner import JobRunner


def create_app(settings: Settings | None = None, repository: ForecastRepository | None = None) -> FastAPI:
    resolved_settings = settings or Settings.from_env()
    resolved_repository = repository or ForecastRepository(create_database_engine(resolved_settings.database_url))
    runner = JobRunner(resolved_repository, resolved_settings)

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        app.state.settings = resolved_settings
        app.state.repository = resolved_repository
        app.state.runner = runner
        await runner.start()
        yield
        await runner.stop()

    application = FastAPI(title="RdMarket Forecasting", version=resolved_settings.code_version, lifespan=lifespan)
    application.state.settings = resolved_settings
    application.state.repository = resolved_repository
    application.state.runner = runner
    application.include_router(router)
    return application


app = create_app()

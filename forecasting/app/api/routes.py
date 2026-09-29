from __future__ import annotations

import hmac
from uuid import UUID

from fastapi import APIRouter, Depends, Header, HTTPException, Request, status

from app.models.registry import model_catalog
from app.schemas.jobs import HealthResponse, JobAccepted, JobSubmission

router = APIRouter()


def internal_auth(request: Request, authorization: str | None = Header(default=None)) -> None:
    expected = f"Bearer {request.app.state.settings.internal_token}"
    if not authorization or not hmac.compare_digest(authorization, expected):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Unauthorized")


@router.get("/health", response_model=HealthResponse)
def health(request: Request):
    return HealthResponse(status="ok", version=request.app.state.settings.code_version)


@router.get("/models")
def models(request: Request):
    return {"data": model_catalog(request.app.state.settings)}


@router.post(
    "/internal/jobs",
    response_model=JobAccepted,
    status_code=status.HTTP_202_ACCEPTED,
    dependencies=[Depends(internal_auth)],
)
def submit_job(payload: JobSubmission, request: Request):
    job = request.app.state.repository.get_job(str(payload.job_id))
    if not job:
        raise HTTPException(status_code=404, detail="Job not found")
    if job["status"] != "queued":
        raise HTTPException(status_code=409, detail="Job is not queued")
    if not request.app.state.runner.submit(payload.job_id):
        raise HTTPException(status_code=503, detail="Forecast worker capacity is full")
    return JobAccepted(job_id=payload.job_id, status="queued")


@router.get("/internal/jobs/{job_id}", dependencies=[Depends(internal_auth)])
def get_job(job_id: UUID, request: Request):
    job = request.app.state.repository.get_job(str(job_id))
    if not job:
        raise HTTPException(status_code=404, detail="Job not found")
    return {
        "id": str(job["id"]),
        "job_type": job["job_type"],
        "status": job["status"],
        "currency_pair": job["currency_pair"],
        "model_names": job["model_names"],
        "params": job["params"],
    }

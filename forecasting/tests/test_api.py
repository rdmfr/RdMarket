from uuid import uuid4

from fastapi.testclient import TestClient

from app.config import Settings
from app.main import create_app


class FakeRunner:
    def __init__(self):
        self.submitted = []

    def submit(self, job_id):
        self.submitted.append(job_id)
        return True


class FakeRepository:
    def __init__(self, job):
        self.job = job

    def recover_orphaned_jobs(self):
        pass

    def get_job(self, job_id):
        if str(self.job["id"]) == job_id:
            return self.job
        return None


def test_health_models_and_internal_bearer_auth():
    job_id = uuid4()
    repo = FakeRepository(
        {
            "id": job_id,
            "job_type": "forecast",
            "status": "queued",
            "currency_pair": "USD/IDR",
            "model_names": ["naive"],
            "params": {},
        }
    )
    settings = Settings(
        database_url="postgresql+psycopg://forecast:test@localhost/db",
        internal_token="unit-test-token-with-more-than-thirty-two-characters",
    )
    api = create_app(settings, repo)
    api.state.runner = FakeRunner()

    with TestClient(api) as client:
        assert client.get("/health").json()["status"] == "ok"
        assert "naive" in {model["name"] for model in client.get("/models").json()["data"]}
        assert client.post("/internal/jobs", json={"job_id": str(job_id)}).status_code == 401
        accepted = client.post(
            "/internal/jobs",
            json={"job_id": str(job_id)},
            headers={"Authorization": f"Bearer {settings.internal_token}"},
        )
        assert accepted.status_code == 202
        assert accepted.json()["status"] == "queued"

import os

os.environ.setdefault("FORECAST_DATABASE_URL", "postgresql+psycopg://forecast:test@localhost/forecast")
os.environ.setdefault("FORECAST_INTERNAL_TOKEN", "unit-test-token-with-more-than-thirty-two-characters")

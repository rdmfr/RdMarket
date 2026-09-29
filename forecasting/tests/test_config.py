import pytest

from app.config import ConfigurationError, Settings


def test_settings_require_database_and_strong_internal_token():
    with pytest.raises(ConfigurationError, match="FORECAST_DATABASE_URL"):
        Settings.from_env({"FORECAST_INTERNAL_TOKEN": "a" * 40})
    with pytest.raises(ConfigurationError, match="at least 32"):
        Settings.from_env({"FORECAST_DATABASE_URL": "postgresql://db/name", "FORECAST_INTERNAL_TOKEN": "weak"})


def test_settings_build_database_url_from_dedicated_credentials():
    settings = Settings.from_env(
        {
            "FORECAST_DB_USER": "read user",
            "FORECAST_DB_PASSWORD": "secret@value",
            "FORECAST_INTERNAL_TOKEN": "x" * 40,
        }
    )
    assert "read%20user:secret%40value@postgres:5432/rdmarket" in settings.database_url
    assert settings.max_workers == 1


def test_settings_reject_invalid_boolean_and_intervals():
    env = {
        "FORECAST_DATABASE_URL": "postgresql://db/name",
        "FORECAST_INTERNAL_TOKEN": "x" * 40,
        "FORECAST_LOG_TRANSFORM": "sometimes",
    }
    with pytest.raises(ConfigurationError, match="true or false"):
        Settings.from_env(env)


def test_lstm_configuration_enforces_supported_architecture_bounds():
    env = {
        "FORECAST_DATABASE_URL": "postgresql://db/name",
        "FORECAST_INTERNAL_TOKEN": "x" * 40,
        "FORECAST_LSTM_LAYERS": "3",
    }
    with pytest.raises(ConfigurationError, match="between 1 and 2"):
        Settings.from_env(env)

    env["FORECAST_LSTM_LAYERS"] = "2"
    env["FORECAST_LSTM_UNITS"] = "31"
    with pytest.raises(ConfigurationError, match="between 32 and 64"):
        Settings.from_env(env)

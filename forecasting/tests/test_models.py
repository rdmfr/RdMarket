import numpy as np
import pandas as pd
import pytest
from app.config import Settings
from app.models.machine_learning import GradientBoostingModel, RidgeModel, _features
from app.models.registry import model_catalog

from app.models.baselines import DriftModel, NaiveModel, SeasonalNaiveModel
from app.models.statistical import ARIMAModel, ETSModel, SARIMAModel


def _series(size=75):
    x = np.arange(size, dtype=float)
    return pd.Series(15000 + 0.7 * x + 12 * np.sin(2 * np.pi * x / 5))


def test_baseline_models_return_deterministic_ordered_intervals():
    for model in (NaiveModel(), DriftModel(), SeasonalNaiveModel()):
        model.fit(_series())
        first = model.predict(5)
        second = model.predict(5)
        assert len(first.point) == 5
        np.testing.assert_array_equal(first.point, second.point)
        assert np.all(first.lower[0.8] <= first.point)
        assert np.all(first.point <= first.upper[0.8])
        assert np.all(first.lower[0.95] <= first.lower[0.8])
        assert np.all(first.upper[0.8] <= first.upper[0.95])


def test_ets_arima_and_sarima_produce_forecast_intervals():
    for model in (ETSModel(), ARIMAModel(), SARIMAModel()):
        model.fit(_series())
        result = model.predict(3)
        assert len(result.point) == 3
        assert np.all(np.isfinite(result.point))
        assert np.all(result.lower[0.8] <= result.point)
        assert np.all(result.point <= result.upper[0.8])


def test_arima_search_is_bounded_and_reproducible():
    series = _series()
    left, right = ARIMAModel(), ARIMAModel()
    left.fit(series)
    right.fit(series)
    assert left.hyperparameters == right.hyperparameters
    np.testing.assert_allclose(left.predict(2).point, right.predict(2).point)


def test_ml_features_at_origin_do_not_depend_on_future_observations():
    returns = np.arange(30, dtype=float)
    features_before, targets_before = _features(returns, 5)
    changed_future = returns.copy()
    changed_future[20:] += 10000
    features_after, targets_after = _features(changed_future, 5)
    np.testing.assert_array_equal(features_before[:15], features_after[:15])
    np.testing.assert_array_equal(targets_before[:15], targets_after[:15])


def test_disabled_ml_models_are_reported_unavailable_with_reason():
    settings = Settings(
        database_url="postgresql+psycopg://user:pass@localhost/db",
        internal_token="x" * 40,
        enable_ml=False,
    )
    catalog = {model["name"]: model for model in model_catalog(settings)}
    for name in ("ridge", "gradient_boosting", "lstm"):
        assert catalog[name]["available"] is False
        assert catalog[name]["unavailable_reason"]
        assert catalog[name]["stability"] == "experimental"


def test_sklearn_models_are_deterministic_and_fit_scalers_on_training_rows(monkeypatch):
    sklearn = pytest.importorskip("sklearn")
    from sklearn.preprocessing import StandardScaler

    observed_scaler_inputs = []
    original_fit = StandardScaler.fit

    def record_fit(scaler, values, y=None, sample_weight=None):
        observed_scaler_inputs.append(np.asarray(values).copy())
        return original_fit(scaler, values, y, sample_weight)

    monkeypatch.setattr(StandardScaler, "fit", record_fit)
    series = _series(450)
    returns = np.diff(series.to_numpy(dtype=float))
    window = 10
    features, targets = _features(returns, window)
    split = int(len(targets) * 0.8)

    for model_type in (RidgeModel, GradientBoostingModel):
        model = model_type(window=window, seed=7)
        model.fit(series)
        first = model.predict(5)
        second = model.predict(5)
        np.testing.assert_array_equal(first.point, second.point)
        assert np.all(first.lower[0.8] <= first.point)
        assert np.all(first.point <= first.upper[0.8])
        assert np.all(first.lower[0.95] <= first.lower[0.8])
        assert np.all(first.upper[0.8] <= first.upper[0.95])

        calibration_input = observed_scaler_inputs[-2]
        np.testing.assert_array_equal(calibration_input, features[:split])
        assert len(calibration_input) < len(features)
    assert sklearn.__version__


def test_lstm_is_seeded_and_returns_empirical_prediction_intervals():
    pytest.importorskip("tensorflow")
    pytest.importorskip("keras")
    from app.models.machine_learning import LSTMModel

    series = _series(180)
    options = {
        "window": 8,
        "units": 32,
        "dropout": 0.0,
        "batch_size": 16,
        "max_epochs": 2,
        "seed": 7,
    }
    first_model = LSTMModel(**options)
    first_model.fit(series)
    first = first_model.predict(3)
    second_model = LSTMModel(**options)
    second_model.fit(series)
    second = second_model.predict(3)

    np.testing.assert_allclose(first.point, second.point, rtol=1e-6, atol=1e-8)
    assert np.all(np.isfinite(first.point))
    assert np.all(first.lower[0.8] <= first.point)
    assert np.all(first.point <= first.upper[0.8])
    assert np.all(first.lower[0.95] <= first.lower[0.8])
    assert np.all(first.upper[0.8] <= first.upper[0.95])
    assert first_model.hyperparameters["seed"] == 7
    assert first_model.hyperparameters["refit_every_n_folds"] == 5

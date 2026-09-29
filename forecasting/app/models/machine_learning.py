from __future__ import annotations

import numpy as np
import pandas as pd

from app.domain.errors import ModelExecutionError
from app.domain.intervals import ForecastResult


def _features(returns: np.ndarray, window: int) -> tuple[np.ndarray, np.ndarray]:
    if len(returns) <= window:
        raise ModelExecutionError("Not enough return observations for the configured ML window")
    rows = []
    targets = []
    for target_index in range(window, len(returns)):
        history = returns[target_index - window:target_index]
        rolling = []
        for size in (5, 10, 20):
            available = history[-min(size, len(history)):]
            rolling.extend((float(np.mean(available)), float(np.std(available))))
        rows.append(np.concatenate((history, np.asarray(rolling, dtype=float))))
        targets.append(returns[target_index])
    return np.asarray(rows, dtype=float), np.asarray(targets, dtype=float)


class _SklearnReturnModel:
    name = "ml"
    version = "scikit-learn"
    estimator_type: str

    def __init__(self, window: int = 20, seed: int = 42):
        self.window = window
        self.seed = seed

    def _estimator(self):
        from sklearn.ensemble import GradientBoostingRegressor
        from sklearn.linear_model import Ridge
        from sklearn.pipeline import make_pipeline
        from sklearn.preprocessing import StandardScaler

        if self.estimator_type == "ridge":
            estimator = Ridge(alpha=1.0)
        else:
            estimator = GradientBoostingRegressor(
                n_estimators=100,
                max_depth=2,
                learning_rate=0.03,
                loss="huber",
                random_state=self.seed,
            )
        return make_pipeline(StandardScaler(), estimator)

    def fit(self, series: pd.Series) -> None:
        values = np.asarray(series, dtype=float)
        returns = np.diff(values)
        features, targets = _features(returns, self.window)
        split = max(1, int(len(targets) * 0.8))
        if len(targets) - split < 1:
            raise ModelExecutionError("Not enough training rows for time-ordered ML validation")

        self.interval_residuals: dict[float, np.ndarray] = {}
        calibration_model = self._estimator()
        calibration_model.fit(features[:split], targets[:split])
        residuals = np.abs(targets[split:] - calibration_model.predict(features[split:]))
        if not len(residuals) or not np.all(np.isfinite(residuals)):
            raise ModelExecutionError("ML validation residuals are not finite")
        self.interval_residuals = {0.8: residuals, 0.95: residuals}

        self.model = self._estimator()
        self.model.fit(features, targets)
        if not np.all(np.isfinite(np.asarray(self.model.predict(features[-1:]), dtype=float))):
            raise ModelExecutionError("ML model returned a non-finite training prediction")
        self.last_level = float(values[-1])
        self.return_history = list(returns[-self.window:].astype(float))
        self.training_rows = len(targets)

    def advance_origin(self, series: pd.Series) -> None:
        values = np.asarray(series, dtype=float)
        returns = np.diff(values)
        if len(returns) < self.window:
            raise ModelExecutionError("Not enough returns at the new forecast origin")
        self.last_level = float(values[-1])
        self.return_history = list(returns[-self.window:].astype(float))

    @property
    def hyperparameters(self) -> dict[str, object]:
        return {
            "window": self.window,
            "estimator": self.estimator_type,
            "seed": self.seed,
            "interval_method": "time-ordered validation absolute-return residual quantiles scaled by sqrt(step)",
            "validation_shuffle": False,
        }

    def predict(self, horizon: int, alpha: float = 0.05) -> ForecastResult:
        if horizon < 1:
            raise ModelExecutionError("Forecast horizon must be positive")
        returns = list(self.return_history)
        point = []
        level = self.last_level
        for _ in range(horizon):
            features, _ = _features(np.asarray([0.0] + returns, dtype=float), self.window)
            predicted_return = float(self.model.predict(features[-1:])[0])
            if not np.isfinite(predicted_return):
                raise ModelExecutionError("ML model returned a non-finite forecast")
            level += predicted_return
            point.append(level)
            returns.append(predicted_return)
            returns = returns[-self.window:]
        points = np.asarray(point, dtype=float)
        lower = {}
        upper = {}
        for coverage, residuals in self.interval_residuals.items():
            quantile = float(np.quantile(residuals, coverage, method="higher"))
            width = quantile * np.sqrt(np.arange(1, horizon + 1, dtype=float))
            lower[coverage] = points - width
            upper[coverage] = points + width
        return ForecastResult(points, lower, upper)


class RidgeModel(_SklearnReturnModel):
    name = "ridge"
    version = "scikit-learn-ridge-v1"
    estimator_type = "ridge"


class GradientBoostingModel(_SklearnReturnModel):
    name = "gradient_boosting"
    version = "scikit-learn-gradient-boosting-v1"
    estimator_type = "gradient_boosting"


class LSTMModel:
    name = "lstm"
    version = "keras-lstm-v1"

    def __init__(
        self,
        window: int = 20,
        units: int = 32,
        dropout: float = 0.1,
        learning_rate: float = 0.001,
        batch_size: int = 32,
        max_epochs: int = 40,
        refit_every_n_folds: int = 5,
        layers: int = 1,
        seed: int = 42,
    ):
        self.window = window
        self.units = units
        self.dropout = dropout
        self.learning_rate = learning_rate
        self.batch_size = batch_size
        self.max_epochs = max_epochs
        self.refit_every_n_folds = refit_every_n_folds
        self.layers = layers
        self.seed = seed

    def fit(self, series: pd.Series) -> None:
        try:
            import keras
            import tensorflow as tf
            from sklearn.preprocessing import StandardScaler
        except ImportError as exc:
            raise ModelExecutionError("ML dependencies are not installed") from exc

        values = np.asarray(series, dtype=float)
        returns = np.diff(values)
        features, targets = _features(returns, self.window)
        validation_size = max(1, int(len(targets) * 0.2))
        split = len(targets) - validation_size
        if split < 2 or validation_size < 1:
            raise ModelExecutionError("Not enough training rows for time-ordered LSTM validation")

        scaler = StandardScaler().fit(features[:split])
        train_x = scaler.transform(features[:split]).reshape((-1, self.window + 6, 1))
        validation_x = scaler.transform(features[split:]).reshape((-1, self.window + 6, 1))
        train_y = targets[:split]
        validation_y = targets[split:]

        keras.utils.set_random_seed(self.seed)
        try:
            tf.config.experimental.enable_op_determinism()
        except (AttributeError, RuntimeError, ValueError) as exc:
            raise ModelExecutionError("TensorFlow operation determinism could not be enabled") from exc

        calibration_layers = [keras.layers.Input(shape=(self.window + 6, 1))]
        for layer_index in range(self.layers):
            calibration_layers.append(
                keras.layers.LSTM(self.units, return_sequences=layer_index < self.layers - 1)
            )
            calibration_layers.append(keras.layers.Dropout(self.dropout))
        calibration_layers.append(keras.layers.Dense(1))
        calibration = keras.Sequential(calibration_layers)
        calibration.compile(optimizer=keras.optimizers.Adam(learning_rate=self.learning_rate), loss="mse")
        early_stopping = keras.callbacks.EarlyStopping(
            monitor="val_loss", patience=5, restore_best_weights=True
        )
        history = calibration.fit(
            train_x,
            train_y,
            validation_data=(validation_x, validation_y),
            epochs=self.max_epochs,
            batch_size=self.batch_size,
            shuffle=False,
            callbacks=[early_stopping],
            verbose=0,
        )
        loss_values = history.history.get("loss", [])
        if not loss_values or not np.all(np.isfinite(loss_values)):
            raise ModelExecutionError("LSTM training loss is not finite")
        residuals = np.abs(validation_y - calibration.predict(validation_x, verbose=0).reshape(-1))
        if not len(residuals) or not np.all(np.isfinite(residuals)):
            raise ModelExecutionError("LSTM validation residuals are not finite")
        self.interval_residuals = {0.8: residuals, 0.95: residuals}

        self.scaler = StandardScaler().fit(features)
        all_x = self.scaler.transform(features).reshape((-1, self.window + 6, 1))
        model_layers = [keras.layers.Input(shape=(self.window + 6, 1))]
        for layer_index in range(self.layers):
            model_layers.append(
                keras.layers.LSTM(self.units, return_sequences=layer_index < self.layers - 1)
            )
            model_layers.append(keras.layers.Dropout(self.dropout))
        model_layers.append(keras.layers.Dense(1))
        self.model = keras.Sequential(model_layers)
        self.model.compile(optimizer=keras.optimizers.Adam(learning_rate=self.learning_rate), loss="mse")
        keras.utils.set_random_seed(self.seed)
        fitted = self.model.fit(
            all_x,
            targets,
            epochs=max(1, early_stopping.best_epoch + 1),
            batch_size=self.batch_size,
            shuffle=False,
            verbose=0,
        )
        final_loss = np.asarray(fitted.history.get("loss", []), dtype=float)
        if not len(final_loss) or not np.all(np.isfinite(final_loss)):
            raise ModelExecutionError("LSTM training loss is not finite")
        self.last_level = float(values[-1])
        self.return_history = list(returns[-self.window:].astype(float))
        self.keras = keras
        self.training_rows = len(targets)

    def advance_origin(self, series: pd.Series) -> None:
        values = np.asarray(series, dtype=float)
        returns = np.diff(values)
        if len(returns) < self.window:
            raise ModelExecutionError("Not enough returns at the new forecast origin")
        self.last_level = float(values[-1])
        self.return_history = list(returns[-self.window:].astype(float))

    @property
    def hyperparameters(self) -> dict[str, object]:
        import keras
        import tensorflow as tf

        return {
            "window": self.window,
            "units": self.units,
            "layers": self.layers,
            "dropout": self.dropout,
            "learning_rate": self.learning_rate,
            "batch_size": self.batch_size,
            "max_epochs": self.max_epochs,
            "seed": self.seed,
            "keras_version": keras.__version__,
            "tensorflow_version": tf.__version__,
            "multi_step_strategy": "recursive",
            "interval_method": "empirical absolute time-ordered validation residual quantiles scaled by sqrt(step)",
            "validation_shuffle": False,
            "refit_every_n_folds": self.refit_every_n_folds,
        }

    def predict(self, horizon: int, alpha: float = 0.05) -> ForecastResult:
        if horizon < 1:
            raise ModelExecutionError("Forecast horizon must be positive")
        returns = list(self.return_history)
        level = self.last_level
        points = []
        for _ in range(horizon):
            features, _ = _features(np.asarray([0.0] + returns, dtype=float), self.window)
            row = self.scaler.transform(features[-1:]).reshape((1, self.window + 6, 1))
            predicted_return = float(self.model.predict(row, verbose=0).reshape(-1)[0])
            if not np.isfinite(predicted_return):
                raise ModelExecutionError("LSTM returned a non-finite forecast")
            level += predicted_return
            points.append(level)
            returns.append(predicted_return)
            returns = returns[-self.window:]
        point = np.asarray(points, dtype=float)
        lower = {}
        upper = {}
        for coverage, residuals in self.interval_residuals.items():
            width = float(np.quantile(residuals, coverage, method="higher")) * np.sqrt(
                np.arange(1, horizon + 1, dtype=float)
            )
            lower[coverage] = point - width
            upper[coverage] = point + width
        return ForecastResult(point, lower, upper)

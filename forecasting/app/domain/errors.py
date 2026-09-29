class ForecastingError(Exception):
    def __init__(self, code: str, message: str):
        super().__init__(message)
        self.code = code
        self.message = message


class InsufficientDataError(ForecastingError):
    def __init__(self, message: str = "Not enough accepted observations for the requested model"):
        super().__init__("INSUFFICIENT_DATA", message)


class ModelExecutionError(ForecastingError):
    def __init__(self, message: str = "The requested model could not be fitted"):
        super().__init__("MODEL_FAILED", message)


class JobTimeoutError(ForecastingError):
    def __init__(self):
        super().__init__("JOB_TIMEOUT", "The forecasting job exceeded its configured time limit")

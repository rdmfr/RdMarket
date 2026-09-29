FROM python:3.12.9-slim-bookworm AS build
ENV PIP_DISABLE_PIP_VERSION_CHECK=1 PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
ARG INSTALL_ML=true
WORKDIR /build
COPY pyproject.toml ./
COPY app ./app
RUN if [ "$INSTALL_ML" = "true" ]; then \
      pip install --no-cache-dir --prefix=/install ".[core,ml]"; \
    else \
      pip install --no-cache-dir --prefix=/install ".[core]"; \
    fi

FROM python:3.12.9-slim-bookworm
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 PATH="/home/forecast/.local/bin:$PATH"
ARG INSTALL_ML=true
RUN groupadd --system forecast && useradd --system --gid forecast --create-home forecast
WORKDIR /app
COPY --from=build /install /usr/local
COPY --chown=forecast:forecast app ./app
USER forecast
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8000/health', timeout=2)"
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]

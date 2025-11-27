FROM python:3.11-slim AS base

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PYTHONPATH=/app

# Install OS-level deps
RUN apt-get update && \
    apt-get install -y --no-install-recommends build-essential && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY requirements.txt /app/requirements.txt
RUN pip install --upgrade pip && \
    pip install -r requirements.txt && \
    pip install gunicorn

COPY . /app/ml_service
WORKDIR /app/ml_service

# Default artifacts dir inside container (can be overridden with volume)
ENV ML_ARTIFACTS_ROOT=/app/ml_service/ml_artifacts \
    PYTHONPATH=/app

EXPOSE 8085 6565

CMD ["bash", "-c", "\
uvicorn ml_service.monitoring:app --host 0.0.0.0 --port 6565 --log-level info & \
gunicorn ml_service.service:app -k uvicorn.workers.UvicornWorker --bind 0.0.0.0:8085 --workers 4 --threads 2 --timeout 120 & \
wait -n \
"]


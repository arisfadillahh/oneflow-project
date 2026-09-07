FROM python:3.12-slim
WORKDIR /app

COPY apps/worker/requirements.txt /app/requirements.txt
RUN python -m pip install --no-cache-dir --timeout 180 --retries 10 --progress-bar off -r requirements.txt

COPY apps/worker/app /app/app

CMD ["python", "-m", "app.main"]

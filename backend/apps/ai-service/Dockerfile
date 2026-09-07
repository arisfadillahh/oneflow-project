FROM python:3.12-slim
WORKDIR /app

COPY apps/ai-service/requirements.txt /app/requirements.txt
RUN python -m pip install --no-cache-dir --timeout 180 --retries 10 --progress-bar off -r requirements.txt

COPY apps/ai-service/app /app/app

EXPOSE 8000
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]

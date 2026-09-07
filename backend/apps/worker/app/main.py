import os
import time
import urllib.error
import urllib.request
import json
import math
import re
import smtplib
from io import BytesIO
from email.message import EmailMessage
from urllib.parse import urlparse
from datetime import datetime, timezone

import psycopg
from docx import Document
from minio import Minio
from pypdf import PdfReader


POSTGRES_DSN = os.getenv(
    "POSTGRES_DSN",
    "postgres://postgres:postgres@localhost:5432/oneflow?sslmode=disable",
)
WORKER_ENV = os.getenv("WORKER_ENV", "development")
ALERT_WEBHOOK_URL = os.getenv("ALERT_WEBHOOK_URL", "").strip()
ALERT_EMAIL_SMTP_HOST = os.getenv("ALERT_EMAIL_SMTP_HOST", "").strip()
ALERT_EMAIL_SMTP_PORT = int(os.getenv("ALERT_EMAIL_SMTP_PORT", "587"))
ALERT_EMAIL_SMTP_USERNAME = os.getenv("ALERT_EMAIL_SMTP_USERNAME", "").strip()
ALERT_EMAIL_SMTP_PASSWORD = os.getenv("ALERT_EMAIL_SMTP_PASSWORD", "").strip()
ALERT_EMAIL_FROM = os.getenv("ALERT_EMAIL_FROM", ALERT_EMAIL_SMTP_USERNAME).strip()
ALERT_EMAIL_TO = [item.strip() for item in os.getenv("ALERT_EMAIL_TO", "").split(",") if item.strip()]
ALERT_EMAIL_USE_TLS = os.getenv("ALERT_EMAIL_USE_TLS", "true").lower() == "true"
OBJECT_STORAGE_ENDPOINT = os.getenv("OBJECT_STORAGE_ENDPOINT", "http://minio:9000")
OBJECT_STORAGE_ACCESS_KEY = os.getenv("OBJECT_STORAGE_ACCESS_KEY", "minioadmin")
OBJECT_STORAGE_SECRET_KEY = os.getenv("OBJECT_STORAGE_SECRET_KEY", "minioadmin")
OBJECT_STORAGE_BUCKET = os.getenv("OBJECT_STORAGE_BUCKET", "oneflow")
OBJECT_STORAGE_USE_SSL = os.getenv("OBJECT_STORAGE_USE_SSL", "false").lower() == "true"
LAST_ALERT_SIGNATURE = ""
INGESTION_ERROR_PREFIX = "__INGESTION_ERROR__:"
EMBEDDING_DIMENSIONS = int(os.getenv("EMBEDDING_DIMENSIONS", "1536"))
EMBEDDING_MODEL = os.getenv("EMBEDDING_MODEL", "local/hash-1536").strip()
EMBEDDING_EXPECTED_DIMENSIONS = int(
    os.getenv(
        "EMBEDDING_EXPECTED_DIMENSIONS",
        "3072" if EMBEDDING_MODEL.endswith("text-embedding-3-large") else "1536" if EMBEDDING_MODEL.endswith("text-embedding-3-small") else str(EMBEDDING_DIMENSIONS),
    )
)
EMBEDDING_API_KEY = (os.getenv("OPENAI_API_KEY") or os.getenv("OPENROUTER_API_KEY") or "").strip()
EMBEDDING_BASE_URL = os.getenv(
    "EMBEDDING_BASE_URL",
    os.getenv("OPENAI_BASE_URL") or os.getenv("OPENROUTER_BASE_URL") or "https://api.openai.com/v1",
).rstrip("/")


def build_storage_client() -> Minio:
    endpoint = OBJECT_STORAGE_ENDPOINT.replace("http://", "").replace("https://", "").rstrip("/")
    return Minio(
        endpoint,
        access_key=OBJECT_STORAGE_ACCESS_KEY,
        secret_key=OBJECT_STORAGE_SECRET_KEY,
        secure=OBJECT_STORAGE_USE_SSL,
    )


def upsert_worker_status(connection: psycopg.Connection, details: dict | None = None) -> None:
    if details is None:
        details = {
            "mode": "scheduler",
            "jobs": ["heartbeat", "monthly_reset_check"],
            "lastHeartbeatAt": datetime.now(timezone.utc).isoformat(),
        }
    with connection.cursor() as cursor:
        cursor.execute(
            """
            INSERT INTO system_status (service_name, status, details, updated_at)
            VALUES ('worker', 'up', %s::jsonb, NOW())
            ON CONFLICT (service_name) DO UPDATE SET
              status = EXCLUDED.status,
              details = EXCLUDED.details,
              updated_at = NOW()
            """,
            (json.dumps(details, separators=(",", ":")),),
        )


def normalize_knowledge_text(text: str) -> str:
    value = (text or "").replace("\r\n", "\n").replace("\r", "\n").replace("\u00a0", " ").strip()
    if not value:
        return ""
    # Some imported docs arrive as one long line. Restore common structural markers
    # so headings and list items do not get mixed into the same retrieval chunk.
    value = re.sub(r"(?<!^)\s+(#{1,6}\s+)", r"\n\1", value)
    value = re.sub(r"(?<!#)\s+(?=(?:[-*]\s+|\d+[.)]\s+))", "\n", value)
    lines = [re.sub(r"[ \t]+", " ", line).strip() for line in value.split("\n")]
    return "\n".join(line for line in lines if line)


def split_structured_sections(text: str) -> list[str]:
    lines = normalize_knowledge_text(text).split("\n")
    sections: list[str] = []
    current: list[str] = []

    for line in lines:
        is_heading = bool(re.match(r"^#{1,6}\s+\S", line))
        if is_heading and current:
            sections.append("\n".join(current).strip())
            current = [line]
        else:
            current.append(line)

    if current:
        sections.append("\n".join(current).strip())
    return [section for section in sections if section and not is_standalone_heading(section)]


def is_standalone_heading(section: str) -> bool:
    lines = [line.strip() for line in section.split("\n") if line.strip()]
    if len(lines) != 1:
        return False
    line = lines[0]
    if not re.match(r"^#{1,6}\s+\S", line):
        return False
    heading_text = re.sub(r"^#{1,6}\s+", "", line).strip()
    return not any(char.islower() for char in heading_text)


def split_long_section(section: str, chunk_size: int, overlap: int) -> list[str]:
    if len(section) <= chunk_size:
        return [section]

    lines = section.split("\n")
    chunks: list[str] = []
    current: list[str] = []

    def current_text() -> str:
        return "\n".join(current).strip()

    for line in lines:
        proposed = "\n".join([*current, line]).strip() if current else line
        if current and len(proposed) > chunk_size:
            chunks.append(current_text())
            current = [line]
        else:
            current.append(line)

    if current:
        chunks.append(current_text())

    refined: list[str] = []
    for chunk in chunks:
        if len(chunk) <= chunk_size:
            refined.append(chunk)
            continue
        refined.extend(split_by_sentence_boundary(chunk, chunk_size, overlap))
    return refined


def split_by_sentence_boundary(text: str, chunk_size: int, overlap: int) -> list[str]:
    normalized = " ".join((text or "").split())
    if not normalized:
        return []

    chunks: list[str] = []
    start = 0
    while start < len(normalized):
        hard_end = min(len(normalized), start + chunk_size)
        end = hard_end
        if hard_end < len(normalized):
            window = normalized[start:hard_end]
            boundary_options = [
                window.rfind(". "),
                window.rfind("? "),
                window.rfind("! "),
                window.rfind("; "),
                window.rfind(" - "),
                window.rfind(" "),
            ]
            boundary = max(boundary_options)
            if boundary >= max(80, int(chunk_size * 0.55)):
                end = start + boundary + 1
        chunks.append(normalized[start:end].strip())
        if end >= len(normalized):
            break
        start = max(end - overlap, start + 1)
    return [chunk for chunk in chunks if chunk]


def chunk_text(text: str, chunk_size: int = 700, overlap: int = 80) -> list[str]:
    normalized = normalize_knowledge_text(text)
    if not normalized:
        return []
    if normalized.startswith(INGESTION_ERROR_PREFIX):
        return []

    sections = split_structured_sections(normalized)
    chunks: list[str] = []
    for section in sections:
        chunks.extend(split_long_section(section, chunk_size, overlap))
    return chunks


def tokenize(text: str) -> list[str]:
    return re.findall(r"[a-zA-Z0-9]+", (text or "").lower())


def char_ngrams(text: str, size: int = 3) -> list[str]:
    normalized = re.sub(r"\s+", " ", (text or "").lower()).strip()
    if not normalized:
        return []
    padded = f"  {normalized}  "
    return [padded[idx : idx + size] for idx in range(max(0, len(padded) - size + 1))]


def build_embedding(text: str, dimensions: int = EMBEDDING_DIMENSIONS) -> list[float]:
    vector = [0.0] * dimensions
    for token in tokenize(text):
        vector[hash(f"tok:{token}") % dimensions] += 1.0
    for gram in char_ngrams(text):
        vector[hash(f"tri:{gram}") % dimensions] += 0.35

    norm = math.sqrt(sum(value * value for value in vector))
    if norm == 0:
        return vector
    return [round(value / norm, 6) for value in vector]


def embedding_provider_enabled() -> bool:
    return bool(EMBEDDING_API_KEY and EMBEDDING_MODEL and not EMBEDDING_MODEL.startswith("local/"))


def build_provider_embedding(text: str) -> list[float]:
    body = {
        "model": EMBEDDING_MODEL,
        "input": text,
    }
    request = urllib.request.Request(
        f"{EMBEDDING_BASE_URL}/embeddings",
        data=json.dumps(body).encode("utf-8"),
        headers={
            "Authorization": f"Bearer {EMBEDDING_API_KEY}",
            "Content-Type": "application/json",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            payload = json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        error_body = exc.read().decode("utf-8", errors="ignore")
        raise RuntimeError(f"embedding provider returned {exc.code}: {error_body[:160]}") from exc
    data = payload.get("data") or []
    if not data:
        raise RuntimeError("embedding provider returned empty data")
    embedding = data[0].get("embedding")
    if not isinstance(embedding, list) or not embedding:
        raise RuntimeError("embedding provider returned invalid vector")
    return [float(value) for value in embedding]


def build_retrieval_embedding(text: str) -> list[float]:
    if embedding_provider_enabled():
        return build_provider_embedding(text)
    return build_embedding(text, EMBEDDING_EXPECTED_DIMENSIONS)


def vector_literal(embedding: list[float]) -> str:
    if len(embedding) != EMBEDDING_EXPECTED_DIMENSIONS:
        raise RuntimeError(f"embedding dimension mismatch: got {len(embedding)}, expected {EMBEDDING_EXPECTED_DIMENSIONS}")
    return json.dumps(embedding, separators=(",", ":"))


def object_key_from_url(file_url: str) -> str:
    value = (file_url or "").strip()
    if not value:
        return ""
    parsed = urlparse(value if "://" in value else f"http://placeholder/{value.lstrip('/')}")
    path = parsed.path.strip("/")
    prefix = f"{OBJECT_STORAGE_BUCKET}/"
    if path.startswith(prefix):
        return path[len(prefix) :]
    return path


def extract_text_from_bytes(file_url: str, content: bytes) -> str:
    suffix = os.path.splitext((file_url or "").lower())[1]
    if suffix in {".txt", ".md", ".csv", ".json"}:
        return content.decode("utf-8", errors="ignore").strip()
    if suffix == ".pdf":
        reader = PdfReader(BytesIO(content))
        return "\n".join((page.extract_text() or "").strip() for page in reader.pages).strip()
    if suffix == ".docx":
        document = Document(BytesIO(content))
        return "\n".join(paragraph.text.strip() for paragraph in document.paragraphs if paragraph.text.strip()).strip()
    return ""


def hydrate_uploaded_knowledge_text(connection: psycopg.Connection, storage_client: Minio) -> int:
    hydrated = 0
    with connection.cursor() as cursor:
        cursor.execute(
            """
            SELECT id, organization_id, COALESCE(file_url, '')
            FROM knowledge_documents
            WHERE status = 'published'
              AND COALESCE(file_url, '') <> ''
              AND COALESCE(raw_text, '') = ''
            ORDER BY updated_at DESC
            """
        )
        documents = cursor.fetchall()

        for document_id, organization_id, file_url in documents:
            object_key = object_key_from_url(file_url)
            if not object_key:
                continue
            response = None
            try:
                response = storage_client.get_object(OBJECT_STORAGE_BUCKET, object_key)
                content = response.read()
                extracted_text = extract_text_from_bytes(file_url, content)
                if not extracted_text:
                    continue
                cursor.execute(
                    """
                    UPDATE knowledge_documents
                    SET raw_text = %s,
                        updated_at = NOW()
                    WHERE id = %s AND organization_id = %s
                    """,
                    (extracted_text, document_id, organization_id),
                )
                hydrated += 1
            except Exception as exc:
                cursor.execute(
                    """
                    UPDATE knowledge_documents
                    SET raw_text = %s,
                        updated_at = NOW()
                    WHERE id = %s AND organization_id = %s
                    """,
                    (f"{INGESTION_ERROR_PREFIX}{str(exc)[:240]}", document_id, organization_id),
                )
                print(f"[worker] extraction error document={document_id} error={exc}", flush=True)
            finally:
                if response is not None:
                    response.close()
                    response.release_conn()
    return hydrated


def sync_embeddings(connection: psycopg.Connection) -> int:
    synced = 0
    with connection.cursor() as cursor:
        cursor.execute(
            """
            SELECT id, organization_id, question, answer
            FROM knowledge_faqs
            WHERE status = 'published'
              AND (
                embedding IS NULL
                OR vector_dims(embedding) <> %s
              )
            """
            ,
            (EMBEDDING_EXPECTED_DIMENSIONS,),
        )
        for faq_id, organization_id, question, answer in cursor.fetchall():
            embedding = build_retrieval_embedding(f"{question} {answer}")
            cursor.execute(
                """
                UPDATE knowledge_faqs
                SET embedding = %s::vector,
                    updated_at = NOW()
                WHERE id = %s AND organization_id = %s
                """,
                (vector_literal(embedding), faq_id, organization_id),
            )
            synced += 1

        cursor.execute(
            """
            SELECT c.id, d.organization_id, d.title, c.chunk_text
            FROM knowledge_chunks c
            JOIN knowledge_documents d ON d.id = c.knowledge_document_id
            WHERE d.status = 'published'
              AND (
                c.embedding IS NULL
                OR vector_dims(c.embedding) <> %s
              )
            """
            ,
            (EMBEDDING_EXPECTED_DIMENSIONS,),
        )
        for chunk_id, organization_id, title, chunk_text in cursor.fetchall():
            embedding = build_retrieval_embedding(f"{title} {chunk_text}")
            cursor.execute(
                """
                UPDATE knowledge_chunks
                SET embedding = %s::vector
                WHERE id = %s
                  AND EXISTS (
                    SELECT 1 FROM knowledge_documents d
                    WHERE d.id = knowledge_chunks.knowledge_document_id
                      AND d.organization_id = %s
                  )
                """,
                (vector_literal(embedding), chunk_id, organization_id),
            )
            synced += 1

        cursor.execute(
            """
            SELECT id, organization_id, title, COALESCE(location, ''), COALESCE(work_type, ''), COALESCE(short_description, ''), COALESCE(apply_link, '')
            FROM job_positions
            WHERE is_active = TRUE
              AND (
                embedding IS NULL
                OR vector_dims(embedding) <> %s
              )
            """
            ,
            (EMBEDDING_EXPECTED_DIMENSIONS,),
        )
        for position_id, organization_id, title, location, work_type, short_description, apply_link in cursor.fetchall():
            embedding = build_retrieval_embedding(f"{title} {location} {work_type} {short_description} {apply_link}")
            cursor.execute(
                """
                UPDATE job_positions
                SET embedding = %s::vector,
                    updated_at = NOW()
                WHERE id = %s AND organization_id = %s
                """,
                (vector_literal(embedding), position_id, organization_id),
            )
            synced += 1

        cursor.execute(
            """
            SELECT id, organization_id, record_type, title, COALESCE(content, ''), fields_json
            FROM knowledge_records
            WHERE status = 'published'
              AND (
                embedding IS NULL
                OR vector_dims(embedding) <> %s
              )
            """
            ,
            (EMBEDDING_EXPECTED_DIMENSIONS,),
        )
        for record_id, organization_id, record_type, title, content, fields_json in cursor.fetchall():
            fields = fields_json if isinstance(fields_json, dict) else {}
            embedding = build_retrieval_embedding(
                f"{record_type} {title} {content} {json.dumps(fields, ensure_ascii=False, sort_keys=True)}"
            )
            cursor.execute(
                """
                UPDATE knowledge_records
                SET embedding = %s::vector,
                    updated_at = NOW()
                WHERE id = %s AND organization_id = %s
                """,
                (vector_literal(embedding), record_id, organization_id),
            )
            synced += 1

    return synced


def sync_knowledge_chunks(connection: psycopg.Connection) -> int:
    created = 0
    with connection.cursor() as cursor:
        cursor.execute(
            """
            SELECT id, organization_id, COALESCE(raw_text, '')
            FROM knowledge_documents
            WHERE status = 'published'
            ORDER BY updated_at DESC
            """
        )
        documents = cursor.fetchall()
        for document_id, organization_id, raw_text in documents:
            parts = chunk_text(raw_text)
            cursor.execute(
                """
                SELECT chunk_text
                FROM knowledge_chunks
                WHERE knowledge_document_id = %s
                ORDER BY chunk_index ASC
                """,
                (document_id,),
            )
            existing_parts = [row[0] for row in cursor.fetchall()]
            if existing_parts == parts:
                continue
            cursor.execute(
                """
                DELETE FROM knowledge_chunks
                WHERE knowledge_document_id = %s
                  AND EXISTS (
                    SELECT 1 FROM knowledge_documents d
                    WHERE d.id = knowledge_chunks.knowledge_document_id
                      AND d.organization_id = %s
                  )
                """,
                (document_id, organization_id),
            )
            for idx, part in enumerate(parts):
                cursor.execute(
                    """
                    INSERT INTO knowledge_chunks (
                      knowledge_document_id,
                      chunk_index,
                      chunk_text,
                      embedding,
                      metadata_json,
                      created_at
                    )
                    VALUES (%s, %s, %s, NULL, %s::jsonb, NOW())
                    """,
                    (
                        document_id,
                        idx,
                        part,
                        f'{{"strategy":"smart_sections","chunkLength":{len(part)}}}',
                    ),
                )
            created += len(parts)
    return created


def collect_alerts(connection: psycopg.Connection) -> list[dict]:
    alerts: list[dict] = []
    with connection.cursor() as cursor:
        cursor.execute(
            """
            SELECT service_name, status, COALESCE(details, '{}'::jsonb)::text, updated_at
            FROM system_status
            ORDER BY service_name ASC
            """
        )
        for service_name, status, details_text, updated_at in cursor.fetchall():
            details = json.loads(details_text or "{}")
            if service_name in {"worker", "ai-service"} and updated_at:
                stale_seconds = (datetime.now(timezone.utc) - updated_at).total_seconds()
                if stale_seconds >= 90:
                    alerts.append(
                        {
                            "severity": "high",
                            "service": service_name,
                            "message": f"{service_name} heartbeat is stale for {int(stale_seconds // 60)} minutes.",
                        }
                    )
            if service_name == "wa-gateway" and status != "connected":
                alerts.append(
                    {
                        "severity": "high",
                        "service": service_name,
                        "message": "WhatsApp gateway is disconnected.",
                    }
                )
                if updated_at and (datetime.now(timezone.utc) - updated_at).total_seconds() >= 600:
                    alerts.append(
                        {
                            "severity": "high",
                            "service": service_name,
                            "message": f"WhatsApp gateway has remained disconnected for {int((datetime.now(timezone.utc) - updated_at).total_seconds() // 60)} minutes.",
                        }
                    )
            if service_name == "wa-gateway" and int(details.get("failedOutboundCount", 0) or 0) >= 3:
                alerts.append(
                    {
                        "severity": "high",
                        "service": service_name,
                        "message": f"WhatsApp gateway has {int(details.get('failedOutboundCount', 0))} consecutive failed outbound sends.",
                    }
                )
            elif status not in {"up", "connected"}:
                alerts.append(
                    {
                        "severity": "medium",
                        "service": service_name,
                        "message": f"Service {service_name} is reporting status={status}.",
                    }
                )
            if service_name == "worker":
                pending_documents = int(details.get("pendingKnowledgeDocuments", 0) or 0)
                pending_embeddings = int(details.get("pendingEmbeddings", 0) or 0)
                reset_error = str(details.get("lastMonthlyResetError", "") or "").strip()
                if pending_documents > 0:
                    alerts.append(
                        {
                            "severity": "medium",
                            "service": service_name,
                            "message": f"Worker has {pending_documents} published knowledge document(s) pending text extraction.",
                        }
                    )
                if pending_embeddings > 0:
                    alerts.append(
                        {
                            "severity": "medium",
                            "service": service_name,
                            "message": f"Worker has {pending_embeddings} knowledge record(s) pending embeddings.",
                        }
                    )
                if reset_error:
                    alerts.append(
                        {
                            "severity": "high",
                            "service": service_name,
                            "message": f"Monthly credit reset failure: {reset_error}",
                        }
                    )
    return alerts


def collect_worker_metrics(connection: psycopg.Connection) -> dict:
    with connection.cursor() as cursor:
        cursor.execute(
            """
            SELECT COUNT(*)
            FROM knowledge_documents
            WHERE status = 'published'
              AND COALESCE(file_url, '') <> ''
              AND COALESCE(raw_text, '') = ''
            """
        )
        pending_documents = int(cursor.fetchone()[0] or 0)

        cursor.execute(
            """
            SELECT
              (SELECT COUNT(*) FROM knowledge_faqs WHERE status = 'published' AND (
                embedding IS NULL OR vector_dims(embedding) <> %s
              )) +
              (SELECT COUNT(*)
               FROM knowledge_chunks c
               JOIN knowledge_documents d ON d.id = c.knowledge_document_id
               WHERE d.status = 'published' AND (
                 c.embedding IS NULL OR vector_dims(c.embedding) <> %s
               )) +
              (SELECT COUNT(*) FROM job_positions WHERE is_active = TRUE AND (
                embedding IS NULL OR vector_dims(embedding) <> %s
              )) +
              (SELECT COUNT(*) FROM knowledge_records WHERE status = 'published' AND (
                embedding IS NULL OR vector_dims(embedding) <> %s
              ))
            """,
            (
                EMBEDDING_EXPECTED_DIMENSIONS,
                EMBEDDING_EXPECTED_DIMENSIONS,
                EMBEDDING_EXPECTED_DIMENSIONS,
                EMBEDDING_EXPECTED_DIMENSIONS,
            ),
        )
        pending_embeddings = int(cursor.fetchone()[0] or 0)

        cursor.execute(
            """
            SELECT last_reset_at, next_reset_at
            FROM credit_wallet
            WHERE is_active = TRUE
            ORDER BY created_at ASC
            LIMIT 1
            """
        )
        wallet_row = cursor.fetchone()

    metrics = {
        "pendingKnowledgeDocuments": pending_documents,
        "pendingEmbeddings": pending_embeddings,
    }
    if wallet_row:
        last_reset_at, next_reset_at = wallet_row
        metrics["lastResetAt"] = iso_or_none(last_reset_at)
        metrics["nextResetAt"] = iso_or_none(next_reset_at)
    return metrics


def iso_or_none(value) -> str | None:
    if value is None:
        return None
    if hasattr(value, "isoformat"):
        return value.isoformat()
    return str(value)


def send_alert_webhook(alerts: list[dict]) -> None:
    global LAST_ALERT_SIGNATURE
    if not ALERT_WEBHOOK_URL or not alerts:
        return
    signature = "|".join(sorted(f"{item['service']}:{item['message']}" for item in alerts))
    if signature == LAST_ALERT_SIGNATURE:
        return
    payload = {
        "source": "oneflow-worker",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "alerts": alerts,
    }
    body = json_bytes(payload)
    request = urllib.request.Request(
        ALERT_WEBHOOK_URL,
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=5) as response:
        response.read()
    LAST_ALERT_SIGNATURE = signature


def send_alert_email(alerts: list[dict]) -> None:
    if not ALERT_EMAIL_SMTP_HOST or not ALERT_EMAIL_FROM or not ALERT_EMAIL_TO or not alerts:
        return

    message = EmailMessage()
    message["Subject"] = f"[Oneflow.id] {len(alerts)} operational alert(s)"
    message["From"] = ALERT_EMAIL_FROM
    message["To"] = ", ".join(ALERT_EMAIL_TO)
    lines = [
        "Oneflow.id worker detected operational alerts.",
        f"Timestamp: {datetime.now(timezone.utc).isoformat()}",
        "",
    ]
    for item in alerts:
        lines.append(f"- {item.get('service', 'unknown')}: {item.get('message', 'No message')}")
    message.set_content("\n".join(lines))

    with smtplib.SMTP(ALERT_EMAIL_SMTP_HOST, ALERT_EMAIL_SMTP_PORT, timeout=10) as smtp:
        if ALERT_EMAIL_USE_TLS:
            smtp.starttls()
        if ALERT_EMAIL_SMTP_USERNAME:
            smtp.login(ALERT_EMAIL_SMTP_USERNAME, ALERT_EMAIL_SMTP_PASSWORD)
        smtp.send_message(message)


def json_bytes(payload: dict) -> bytes:
    import json

    return json.dumps(payload).encode("utf-8")


def run_monthly_reset_if_due(connection: psycopg.Connection) -> None:
    with connection.cursor() as cursor:
        cursor.execute(
            """
            UPDATE credit_wallet wallet
            SET monthly_credits_used = 0,
                monthly_credits_remaining = wallet.monthly_credit_limit,
                last_reset_at = NOW(),
                next_reset_at = NOW() + INTERVAL '1 month',
                updated_at = NOW()
            WHERE wallet.is_active = TRUE
              AND wallet.next_reset_at IS NOT NULL
              AND wallet.next_reset_at <= NOW()
              AND EXISTS (
                SELECT 1
                FROM credit_purchases purchase
                JOIN credit_packages package
                  ON package.id = purchase.credit_package_id
                 AND package.organization_id = purchase.organization_id
                WHERE purchase.organization_id = wallet.organization_id
                  AND purchase.payment_status = 'confirmed'
                  AND COALESCE(package.billing_period, 'monthly') = 'monthly'
                  AND purchase.active_until > NOW()
              )
            RETURNING wallet.monthly_credit_limit
            """
        )
        reset_rows = cursor.fetchall()
        if not reset_rows:
            return

        now = datetime.now(timezone.utc)
        reset_wallet_count = len(reset_rows)
        reset_monthly_limit = sum(row[0] for row in reset_rows)
        cursor.execute(
            """
            UPDATE system_status
            SET details = %s::jsonb, updated_at = NOW()
            WHERE service_name = 'worker'
            """,
            (
                f'{{"mode":"scheduler","lastResetAt":"{now.isoformat()}","resetWalletCount":{reset_wallet_count},"resetMonthlyLimit":{reset_monthly_limit}}}',
            ),
        )
        print(f"[worker] monthly credit reset applied wallets={reset_wallet_count} at {now.isoformat()}", flush=True)


def main() -> None:
    print(f"[worker] boot env={WORKER_ENV}", flush=True)
    storage_client = build_storage_client()
    while True:
        try:
            with psycopg.connect(POSTGRES_DSN, autocommit=True) as connection:
                monthly_reset_error = ""
                try:
                    run_monthly_reset_if_due(connection)
                except Exception as reset_exc:
                    monthly_reset_error = str(reset_exc)[:240]
                    print(f"[worker] monthly reset error {monthly_reset_error}", flush=True)
                hydrated_count = hydrate_uploaded_knowledge_text(connection, storage_client)
                chunk_count = sync_knowledge_chunks(connection)
                embedding_count = sync_embeddings(connection)
                details = {
                    "mode": "scheduler",
                    "jobs": ["heartbeat", "monthly_reset_check", "knowledge_hydration", "chunking", "embeddings", "alerting"],
                    "lastHeartbeatAt": datetime.now(timezone.utc).isoformat(),
                    "hydratedThisRun": hydrated_count,
                    "chunksSyncedThisRun": chunk_count,
                    "embeddingsSyncedThisRun": embedding_count,
                    "alertCount": 0,
                    **collect_worker_metrics(connection),
                }
                if monthly_reset_error:
                    details["lastMonthlyResetError"] = monthly_reset_error
                upsert_worker_status(connection, details)
                alerts = collect_alerts(connection)
                details["alertCount"] = len(alerts)
                upsert_worker_status(connection, details)
                try:
                    send_alert_webhook(alerts)
                except Exception as webhook_exc:
                    print(f"[worker] alert webhook error {webhook_exc}", flush=True)
                try:
                    send_alert_email(alerts)
                except Exception as email_exc:
                    print(f"[worker] alert email error {email_exc}", flush=True)
                print(
                f"[worker] heartbeat {datetime.now(timezone.utc).isoformat()} hydrated={hydrated_count} chunks={chunk_count} embeddings={embedding_count} alerts={len(alerts)}",
                flush=True,
            )
        except Exception as exc:
            print(f"[worker] error {exc}", flush=True)
        time.sleep(20)


if __name__ == "__main__":
    main()

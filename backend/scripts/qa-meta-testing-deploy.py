"""Read-only QA by default; explicit fixture/retirement flags mutate test state.

Never pairs a real phone or sends customer messages.
"""
import base64
import hashlib
import hmac
import json
import subprocess
import sys
from pathlib import Path
import time
import urllib.error
import urllib.parse
import urllib.request


def environment(container):
    item = json.loads(subprocess.check_output(["docker", "inspect", container]))[0]
    return dict(value.split("=", 1) for value in item["Config"]["Env"] if "=" in value)


def request(path, expected, token=None, body=None, extra=None, base="https://oneflow.id"):
    headers = {"User-Agent": "Oneflow-Deployment-QA"}
    if token:
        headers["Authorization"] = "Bearer " + token
    if extra:
        headers.update(extra)
    req = urllib.request.Request(base + path, data=body, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            status, content = response.status, response.read()
    except urllib.error.HTTPError as error:
        status, content = error.code, error.read()
    assert status == expected, f"{path.split('?')[0]} expected {expected}, got {status}"
    print(f"PASS {path.split('?')[0]} HTTP {status}")
    return content


def encode(value):
    return base64.urlsafe_b64encode(value).rstrip(b"=")


def main():
    backend = environment("oneflow-app-backend")
    gateway = environment("oneflow-wa-gateway")
    postgres = environment("oneflow-postgres")
    if "--retire-legacy" in sys.argv:
        query = """
        WITH retired AS (
          UPDATE whatsapp_sessions
          SET status='disconnected', is_default=FALSE,
              details=COALESCE(details,'{}'::jsonb) || jsonb_build_object('legacyRetired',true),
              updated_at=NOW()
          WHERE COALESCE(provider,'whatsmeow') <> 'meta_cloud'
            AND deleted_at IS NULL
            AND (status <> 'disconnected' OR is_default OR NOT COALESCE(details,'{}'::jsonb) @> '{"legacyRetired":true}'::jsonb)
          RETURNING id
        ) SELECT count(*) FROM retired;
        """
        count = subprocess.check_output([
            "docker", "exec", "oneflow-postgres", "psql", "-v", "ON_ERROR_STOP=1", "-U", postgres["POSTGRES_USER"],
            "-d", postgres["POSTGRES_DB"], "-At", "-c", query,
        ], text=True).strip()
        print("Retired legacy sessions: " + count + "; history and credentials preserved, no data deletion")
        return
    if "--inspect-qa-fixture" in sys.argv:
        query = """
        SELECT json_build_object(
          'organization', o.name,
          'sessions', (SELECT count(*) FROM whatsapp_sessions w WHERE w.organization_id=o.id AND w.deleted_at IS NULL),
          'sessionStates', (SELECT json_agg(json_build_object('provider',w.provider,'status',w.status,'connected',w.last_connected_at IS NOT NULL,'historyStatus',w.meta_history_sync_status)) FROM whatsapp_sessions w WHERE w.organization_id=o.id AND w.deleted_at IS NULL),
          'fixtures', (SELECT json_agg(json_build_object('status',p.payment_status,'activeUntil',p.active_until,'packageMatchesOrg',cp.organization_id=p.organization_id,'whatsappLimit',cp.max_whatsapp_sessions))
            FROM credit_purchases p JOIN credit_packages cp ON cp.id=p.credit_package_id
            WHERE p.organization_id=o.id AND p.notes='QA_COEXISTENCE_FIXTURE_NO_PAYMENT'))
        FROM organizations o JOIN agents a ON a.id=o.created_by
        WHERE o.name='Oneflow QA Coexistence' AND a.username LIKE 'qa.coexistence.%';
        """
        result = subprocess.check_output([
            "docker", "exec", "oneflow-postgres", "psql", "-U", postgres["POSTGRES_USER"],
            "-d", postgres["POSTGRES_DB"], "-At", "-c", query,
        ], text=True)
        print(result.strip())
        return
    if "--prepare-qa-fixture" in sys.argv:
        subprocess.run([
            "docker", "exec", "-i", "oneflow-postgres", "psql", "-v", "ON_ERROR_STOP=1", "-U", postgres["POSTGRES_USER"], "-d", postgres["POSTGRES_DB"]
        ], input=Path(__file__).with_name("qa-coexistence-entitlement.sql").read_text(), text=True, check=True)
        print("Temporary zero-payment QA fixture prepared; other organizations unchanged")
        return
    request("/", 200)
    request("/login", 200)
    request("/health", 200)
    request("/api/whatsapp/meta/config", 401)
    request("/api/meta/webhook?hub.mode=subscribe&hub.verify_token=invalid&hub.challenge=qa", 403)
    body = b'{"object":"whatsapp_business_account","entry":[]}'
    request("/api/meta/webhook", 401, body=body, extra={"Content-Type": "application/json"})
    signature = "sha256=" + hmac.new(gateway["META_APP_SECRET"].encode(), body, hashlib.sha256).hexdigest()
    request("/api/meta/webhook", 200, body=body, extra={"Content-Type": "application/json", "X-Hub-Signature-256": signature})
    # A 60-second server-signed credential checks authenticated config only; never export it.
    row = subprocess.check_output([
        "docker", "exec", "oneflow-postgres", "psql", "-U", postgres["POSTGRES_USER"],
        "-d", postgres["POSTGRES_DB"], "-At", "-c",
        "SELECT a.id::text || '|' || m.organization_id::text || '|' || m.role FROM agents a JOIN organization_members m ON m.agent_id=a.id WHERE a.role='owner' AND m.status='active' LIMIT 1;",
    ], text=True).strip()
    if not row:
        raise SystemExit("No owner fixture available for read-only config QA")
    agent_id, organization_id, organization_role = row.split("|")
    payload = {"agent_id": agent_id, "sub": agent_id, "organization_id": organization_id,
               "role": "owner", "organization_role": organization_role, "exp": int(time.time()) + 60}
    unsigned = encode(b'{"alg":"HS256","typ":"JWT"}') + b"." + encode(json.dumps(payload).encode())
    token = (unsigned + b"." + encode(hmac.new(backend["JWT_SECRET"].encode(), unsigned, hashlib.sha256).digest())).decode()
    config = json.loads(request("/api/whatsapp/meta/config", 200, token=token))
    assert config["enabled"] is True and config["onboardingMode"] == "coexistence"
    assert config["appId"] == "1460232068764576" and config["configurationId"] == "1768328367945778"
    print("PASS authenticated config: enabled, coexistence, expected app/config IDs")
    request("/api/whatsapp/sessions", 410, token=token,
            body=b'{"provider":"whatsmeow","label":"rejected QA request"}',
            extra={"Content-Type": "application/json"})
    inspected = json.loads(subprocess.check_output(["docker", "inspect", "oneflow-wa-gateway"]))[0]
    address = next(value["IPAddress"] for value in inspected["NetworkSettings"]["Networks"].values() if value.get("IPAddress"))
    gateway_base = "http://" + address + ":" + gateway.get("WA_PORT", "8090")
    for path in ["/api/qr", "/api/qr-image", "/api/pair-phone", "/api/connect"]:
        request(path, 410, base=gateway_base)
    active_legacy = subprocess.check_output([
        "docker", "exec", "oneflow-postgres", "psql", "-U", postgres["POSTGRES_USER"],
        "-d", postgres["POSTGRES_DB"], "-At", "-c",
        "SELECT count(*) FROM whatsapp_sessions WHERE deleted_at IS NULL AND COALESCE(provider,'whatsmeow') <> 'meta_cloud' AND status IN ('connected','connecting','qr_pending');",
    ], text=True).strip()
    assert active_legacy == "0", "Legacy session state remains active"
    print("PASS no active legacy session rows")
    print("QA complete: no customer messages, no phone pairing, no data resets")


if __name__ == "__main__":
    main()

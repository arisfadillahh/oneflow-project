"""Validate existing server credentials and explicitly prepare the testing deploy."""
import argparse
import base64
import json
import os
from pathlib import Path
import secrets
import shutil
import subprocess
import urllib.error
import urllib.request

import yaml


def container_env(name):
    result = json.loads(subprocess.check_output(["docker", "inspect", name]))[0]
    return dict(item.split("=", 1) for item in result["Config"]["Env"] if "=" in item)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    root = Path("/home/goffath/oneflow-deploy")
    path = root / "docker-compose.yml"
    config = yaml.safe_load(path.read_text())
    gateway = container_env("oneflow-wa-gateway")
    backend = container_env("oneflow-app-backend")
    app_id = "1460232068764576"
    secret = gateway.get("META_APP_SECRET", "")
    if not secret or not gateway.get("META_WEBHOOK_VERIFY_TOKEN"):
        raise SystemExit("Existing Meta secret or webhook verify token is missing")
    request = urllib.request.Request(
        f"https://graph.facebook.com/v23.0/{app_id}?fields=id",
        headers={"Authorization": f"Bearer {app_id}|{secret}"},
    )
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            verified = json.load(response).get("id") == app_id
    except urllib.error.HTTPError as error:
        raise SystemExit(f"Meta app credential validation rejected: HTTP {error.code}") from None
    except Exception:
        raise SystemExit("Meta app credential validation unavailable") from None
    if not verified:
        raise SystemExit("Existing Meta secret did not verify against Oneflow app")
    key = backend.get("META_CREDENTIAL_ENCRYPTION_KEY") or gateway.get("META_CREDENTIAL_ENCRYPTION_KEY")
    if not key:
        postgres = container_env("oneflow-postgres")
        count = subprocess.check_output([
            "docker", "exec", "oneflow-postgres", "psql", "-U", postgres["POSTGRES_USER"],
            "-d", postgres["POSTGRES_DB"], "-At", "-c",
            "SELECT count(*) FROM whatsapp_sessions WHERE meta_business_token_ciphertext IS NOT NULL AND meta_business_token_ciphertext <> '';",
        ], text=True).strip()
        if count != "0":
            raise SystemExit("Existing encrypted Meta credentials require the original encryption key")
        key = base64.b64encode(secrets.token_bytes(32)).decode()
    if len(base64.b64decode(key, validate=True)) != 32:
        raise SystemExit("Existing Meta encryption key is invalid")
    print("Meta app credential verified; encryption configuration safe; no credentials printed")
    if not args.apply:
        print("Read-only preflight passed; use --apply after images are built")
        return
    backup = root / "rollback-coexistence-20260905"
    backup.mkdir(mode=0o700, exist_ok=True)
    backup_file = backup / "docker-compose.yml"
    if backup_file.exists():
        raise SystemExit("Rollback backup already exists; inspect prior deployment before retry")
    shutil.copy2(path, backup_file)
    os.chmod(backup_file, 0o600)
    for service in ("app-backend", "wa-gateway"):
        current = json.loads(subprocess.check_output(["docker", "inspect", f"oneflow-{service}"]))[0]
        subprocess.run(["docker", "tag", current["Image"], f"oneflow-{service}:rollback-coexistence-20260905"], check=True)
        config["services"][service]["image"] = f"oneflow-{service}:coexistence-20260905"
    for service in ("app-backend", "wa-gateway"):
        target = config["services"][service]
        env = target.get("environment", {})
        if isinstance(env, list):
            env = dict(item.split("=", 1) for item in env if "=" in item)
        env.update({"META_CLOUD_ENABLED": "true", "META_APP_SECRET": secret,
                    "META_GRAPH_API_VERSION": "v23.0", "META_CREDENTIAL_ENCRYPTION_KEY": key})
        if service == "app-backend":
            env.update({"META_APP_ID": app_id, "META_CONFIGURATION_ID": "1768328367945778"})
        else:
            env.update({"META_WEBHOOK_ENABLED": "true", "META_WEBHOOK_VERIFY_TOKEN": gateway["META_WEBHOOK_VERIFY_TOKEN"]})
        target["environment"] = env
    temporary = path.with_suffix(".pending")
    descriptor = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(descriptor, "w") as output:
        yaml.safe_dump(config, output, sort_keys=False)
    subprocess.run(["docker", "compose", "-f", str(temporary), "config", "--quiet"], check=True)
    temporary.replace(path)
    print("Compose prepared; rollback config and images retained; containers not restarted yet")


if __name__ == "__main__":
    main()

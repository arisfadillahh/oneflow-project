from __future__ import annotations

import json
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class Handler(BaseHTTPRequestHandler):
    def _json(self, payload: dict, status: int = 200) -> None:
        body = json.dumps(payload).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self) -> None:
        if self.path in {"/healthz", "/"}:
            self._json({"status": "ok", "service": "local-wa-stub"})
            return
        self._json({"error": "not found"}, 404)

    def do_POST(self) -> None:
        length = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(length) if length > 0 else b"{}"
        try:
            payload = json.loads(raw.decode("utf-8") or "{}")
        except Exception:
            payload = {}
        if self.path.endswith("/send-text") or self.path.endswith("/send-media"):
            self._json(
                {
                    "status": "sent",
                    "externalMessageId": f"local-wa-{int(time.time() * 1000)}",
                    "echo": {
                        "phone": payload.get("phone", ""),
                        "text": payload.get("text", ""),
                    },
                }
            )
            return
        if self.path.endswith("/chat-presence"):
            self._json({"status": "ok"})
            return
        self._json({"error": "not found", "path": self.path}, 404)

    def log_message(self, fmt: str, *args) -> None:
        return


if __name__ == "__main__":
    server = ThreadingHTTPServer(("127.0.0.1", 18090), Handler)
    print("local-wa-stub listening on http://127.0.0.1:18090", flush=True)
    server.serve_forever()

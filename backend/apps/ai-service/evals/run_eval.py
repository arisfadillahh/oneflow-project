import argparse
import json
import sys
import time
import urllib.request
from pathlib import Path


BLOCKED_ARTIFACTS = ["\nQ:", "\nA:", "[http://", "[https://", "Pertanyaan sumber:", "Jawaban sumber:"]


def post_decide(base_url: str, message: str) -> dict:
    body = json.dumps(
        {
            "conversation_id": "eval",
            "message_text": message,
            "customer_name": "Simulated Customer",
            "history": [],
        }
    ).encode("utf-8")
    request = urllib.request.Request(
        f"{base_url.rstrip('/')}/api/decide",
        data=body,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=60) as response:
        return json.loads(response.read().decode("utf-8"))


def check_case(base_url: str, case: dict) -> tuple[bool, list[str]]:
    response = post_decide(base_url, case["message"])
    answer = str(response.get("answer_text") or "")
    decision_text = " ".join(
        str(value or "")
        for value in [
            response.get("answer_text"),
            response.get("escalation_reason"),
            json.dumps(response.get("retrieval_metadata") or {}, ensure_ascii=False),
        ]
    )
    metadata = response.get("retrieval_metadata") or {}
    matches = metadata.get("matches") or []
    top_kind = matches[0].get("kind") if matches else metadata.get("topKind")
    failures: list[str] = []

    expected_decision = case.get("expected_decision", "answer")
    if response.get("decision") != expected_decision:
        failures.append(f"decision={response.get('decision')}, expected={expected_decision}")
    expected_kind = case.get("expected_kind")
    if isinstance(expected_kind, list):
        if top_kind not in expected_kind:
            failures.append(f"kind={top_kind}, expected_one_of={expected_kind}")
    elif expected_kind and top_kind != expected_kind:
        failures.append(f"kind={top_kind}, expected={expected_kind}")
    if case.get("must_include_any"):
        lowered = decision_text.lower()
        if not any(str(item).lower() in lowered for item in case["must_include_any"]):
            failures.append("missing expected semantic anchor")
    if any(artifact in f"\n{answer}" for artifact in BLOCKED_ARTIFACTS):
        failures.append("blocked output artifact found")
    return not failures, failures


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://localhost:8000")
    parser.add_argument("--cases", default=str(Path(__file__).with_name("golden_cases.json")))
    parser.add_argument("--include-extra", action="store_true")
    parser.add_argument("--delay", type=float, default=1.2)
    args = parser.parse_args()

    data = json.loads(Path(args.cases).read_text(encoding="utf-8"))
    cases = list(data.get("golden") or [])
    if args.include_extra:
        cases.extend(data.get("extra") or [])

    failed = 0
    for case in cases:
        ok, failures = check_case(args.base_url, case)
        status = "PASS" if ok else "FAIL"
        print(f"{status} {case['id']}: {case['message']}")
        if failures:
            failed += 1
            print(f"  - {'; '.join(failures)}")
        if args.delay > 0:
            time.sleep(args.delay)
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())

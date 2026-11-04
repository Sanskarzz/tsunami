#!/usr/bin/env python3
"""Read-only free-model and account checks; never emit key labels or values."""
import argparse
from datetime import datetime, timezone
from decimal import Decimal, InvalidOperation
import json
from pathlib import Path
import re
import sys
import urllib.error
import urllib.request

parser = argparse.ArgumentParser()
parser.add_argument("--key-file", default=".local/openrouter-api-key")
parser.add_argument("--model", required=True)
parser.add_argument("--out", default="artifacts/openrouter/preflight.json")
args = parser.parse_args()
if not re.fullmatch(r"[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+:free", args.model):
    sys.exit("An explicit :free model is required")
key = Path(args.key_file).read_text().strip()


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())


def get(path, authenticated=False):
    headers = {"Authorization": "Bearer " + key} if authenticated else {}
    req = urllib.request.Request("https://openrouter.ai/api/v1/" + path, headers=headers)
    try:
        with opener.open(req, timeout=20) as response:
            return json.loads(response.read(4 << 20))
    except urllib.error.HTTPError as error:
        sys.exit(f"OpenRouter metadata check failed: HTTP {error.code}")


catalog = get("models")["data"]
model = next((m for m in catalog if m["id"] == args.model), None)
if not model or "tools" not in model.get("supported_parameters", []):
    sys.exit("Requested free model is absent or does not advertise tools")
pricing = model.get("pricing", {})
try:
    zero = all(Decimal(str(pricing.get(name, "NaN"))) == 0 for name in ("prompt", "completion"))
    zero = zero and Decimal(str(pricing.get("request", "0"))) == 0
except InvalidOperation:
    zero = False
if not zero:
    sys.exit("Model pricing is not verified zero; no inference should be attempted")
account = get("key", True)["data"]
record = {
    "checked_at": datetime.now(timezone.utc).isoformat(),
    "model": args.model,
    "pricing": pricing,
    "tools_advertised": True,
    "is_free_tier": account.get("is_free_tier"),
    "usage": account.get("usage"),
    "usage_daily": account.get("usage_daily"),
    "free_model_daily_requests": account.get("free_model_daily_requests"),
}
out = Path(args.out)
out.parent.mkdir(parents=True, exist_ok=True)
out.write_text(json.dumps(record, indent=2) + "\n")
print(json.dumps(record, indent=2))

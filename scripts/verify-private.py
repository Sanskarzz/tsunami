#!/usr/bin/env python3
"""Run synthetic private assessments, tenant denial, and a connector outage.

Only the trusted operator uses port-forward; workers never receive these keys.
The normal five-case suite stays scripted even with a model configured.
"""
import base64
from contextlib import contextmanager
from datetime import datetime, timezone
import json
from deploy_images import deployed_image, with_image
import os
from pathlib import Path
import re
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request

context = os.environ.get("CONTEXT", "kind-tsunami-lab")
if context != "kind-tsunami-lab":
    sys.exit("Private verification is restricted to the dedicated lab")
base = ["kubectl", "--context", context]


def kubectl(*args, data=None):
    p = subprocess.run(base + list(args), input=data, text=True, capture_output=True)
    if p.returncode:
        raise RuntimeError("Private verification Kubernetes command failed (credential-bearing errors suppressed)")
    return p.stdout


@contextmanager
def forward(namespace, service):
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0)); port = sock.getsockname()[1]
    p = subprocess.Popen(base + ["-n", namespace, "port-forward", "--address=127.0.0.1", "service/"+service, f"{port}:8080"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        for _ in range(100):
            try:
                with socket.create_connection(("127.0.0.1", port), timeout=0.1):
                    break
            except OSError:
                if p.poll() is not None:
                    raise RuntimeError("Port-forward exited")
                time.sleep(0.05)
        else:
            raise RuntimeError("Port-forward readiness timed out")
        yield f"http://127.0.0.1:{port}"
    finally:
        p.terminate()
        p.wait(timeout=5)


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())


def api(url, token, method="GET", body=None, expected=200):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method, headers={"Authorization": "Bearer "+token, "Content-Type": "application/json"})
    try:
        with opener.open(req, timeout=5) as response:
            if response.status != expected:
                raise RuntimeError("Unexpected API status")
            return json.load(response)
    except urllib.error.HTTPError as err:
        if err.code == expected:
            return None
        raise RuntimeError(f"Unexpected API status {err.code}") from None


template = json.loads(kubectl("-n", "tsunami", "create", "-f", "deploy/verify.yaml", "--dry-run=client", "-o", "json"))
template = with_image(template, deployed_image(kubectl))
template["metadata"]["generateName"] = "tsunami-private-verify-"
template["spec"]["template"]["spec"]["containers"][0]["args"] = ["verify", "--private"]
job = json.loads(kubectl("-n", "tsunami", "create", "-f", "-", "-o", "json", data=json.dumps(template)))["metadata"]["name"]
out = Path("artifacts/private")/job; out.mkdir(parents=True, exist_ok=True)
state = {}
for _ in range(125):
    state = json.loads(kubectl("-n", "tsunami", "get", "job", job, "-o", "json")).get("status", {})
    if state.get("succeeded") or state.get("failed"):
        break
    time.sleep(1)
log = kubectl("-n", "tsunami", "logs", "job/"+job)
(out/"verification.log").write_text(log)
print(log, flush=True)
ids = list(dict.fromkeys(re.findall(r"VERIFY_CREATED run=([0-9a-f]{32})", log)))
if ids:
    subprocess.run(["go", "run", "./cmd/tsunami", "collect-k8s", "--context", context, "--runs", ",".join(ids), "--out", str(out)], check=True)
if not state.get("succeeded"):
    sys.exit(f"Private workflow failed; records retained in {out}")

secrets = json.loads(kubectl("-n", "tsunami", "get", "secret", "tsunami-operator", "-o", "json"))["data"]
admin_a, admin_b = (base64.b64decode(secrets[k]).decode() for k in ("admin-a", "admin-b"))
with forward("tsunami", "tsunami-control") as broker, forward("tsunami-customer-a", "tsunami-vulnerable") as target:
    api(broker+"/v1/runs", admin_b, "POST", {"scenario": "vulnerable", "connectivity": "private"}, expected=503)
    print("PASS tenant B cannot create a run using tenant A's private route", flush=True)
    before = api(target+"/stats", "")["requests"]
    run_id = None
    try:
        kubectl("-n", "tsunami-customer-a", "scale", "deployment/tsunami-connector", "--replicas=0")
        # Wait for actual process termination, not only desired replica count.
        kubectl("-n", "tsunami-customer-a", "wait", "--for=delete", "pod", "-l", "tsunami-role=connector,tsunami-instance=main", "--timeout=60s")
        r = api(broker+"/v1/runs", admin_a, "POST", {"scenario": "vulnerable", "connectivity": "private"}, expected=201)
        run_id = r["id"]
        for _ in range(400):
            r = api(broker+"/v1/runs/"+run_id, admin_a)
            if r["state"] in ("completed", "failed", "stopped"):
                break
            time.sleep(0.1)
        if r["state"] not in ("completed", "failed", "stopped") or any(c.get("status") for c in r["captures"]):
            raise RuntimeError("Connector outage was not closed safely")
        after = api(target+"/stats", "")["requests"]
        if after != before:
            raise RuntimeError("Private target received traffic during connector outage")
        (out/"outage.json").write_text(json.dumps({"checked_at": datetime.now(timezone.utc).isoformat(), "run": r, "target_requests_before": before, "target_requests_after": after, "no_direct_fallback": True, "cross_tenant_private_creation_denied": True}, indent=2)+"\n")
        print("PASS connector outage: no target traffic and no direct fallback", flush=True)
    finally:
        if run_id:
            api(broker+"/v1/runs/"+run_id+"/stop", admin_a, "POST")
        kubectl("-n", "tsunami-customer-a", "scale", "deployment/tsunami-connector", "--replicas=1")
        kubectl("-n", "tsunami-customer-a", "rollout", "status", "deployment/tsunami-connector", "--timeout=60s")
print(f"PASS private verification; evidence in {out}. No model calls made.")

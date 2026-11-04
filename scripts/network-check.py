#!/usr/bin/env python3
"""Trusted diagnostics for a local cluster; no agent or customer input."""
import copy
from datetime import datetime, timezone
import json
from deploy_images import deployed_image, with_image
import os
import socket
from pathlib import Path
import subprocess
import sys
import time

context = os.environ.get("CONTEXT", "kind-kind")
runtime = os.environ.get("RUNTIME_CLASS", "")
base = ["kubectl", "--context", context, "-n", "tsunami"]
out = Path(os.environ["NETWORK_OUTPUT_DIR"]) if os.environ.get("NETWORK_OUTPUT_DIR") else Path("artifacts/isolation") / context / (runtime or "runc")
out.mkdir(parents=True, exist_ok=True)


def kubectl(*args, data=None):
    return subprocess.check_output(base + list(args), input=data, text=True)


kubectl("rollout", "status", "deployment/tsunami-tripwire", "--timeout=40s")
services = json.loads(kubectl("get", "services", "-o", "json"))["items"]
ips = {s["metadata"]["name"]: s["spec"]["clusterIP"] for s in services}
pods = json.loads(kubectl("get", "pods", "-l", "tsunami-role=tripwire", "-o", "json"))["items"]
ready = [p for p in pods if any(c["type"] == "Ready" and c["status"] == "True" for c in p["status"].get("conditions", []))]
if len(ready) != 1:
    sys.exit("Expected exactly one ready tripwire")
api = json.loads(kubectl("get", "service", "kubernetes", "-n", "default", "-o", "json"))["spec"]["clusterIP"]
dns = json.loads(kubectl("get", "service", "kube-dns", "-n", "kube-system", "-o", "json"))["spec"]["clusterIP"]
node_name = ready[0]["spec"]["nodeName"]
node = json.loads(kubectl("get", "node", node_name, "-o", "json"))
node_ip = next(a["address"] for a in node["status"]["addresses"] if a["type"] == "InternalIP")
destinations = {
    "tripwire-service": f'http://{ips["tsunami-tripwire"]}:8080',
    "tripwire-pod": f'http://{ready[0]["status"]["podIP"]}:8080',
    "api-service": f"https://{api}:443",
    "node-api": f"https://{node_ip}:6443",
    "dns-tcp": f"http://{dns}:53",
}
if "tsunami-model-fixture" in ips:
    destinations["model-service"] = f'http://{ips["tsunami-model-fixture"]}:8080'
if os.environ.get("PROBE_PRIVATE") == "true":
    for ns, name, label in (("tsunami-customer-a", "tsunami-vulnerable", "private-a-web"), ("tsunami-customer-a", "tsunami-tripwire", "private-a-tripwire"), ("tsunami-customer-a", "tsunami-connector", "private-a-connector"), ("tsunami-customer-b", "tsunami-fixed", "private-b-web")):
        ip = kubectl("get", "service", name, "-n", ns, "-o", "jsonpath={.spec.clusterIP}").strip()
        destinations[label] = f"http://{ip}:8080"
if os.environ.get("PROBE_APPLICATIONS") == "true":
    for name in ["vampi-vulnerable", "vampi-fixed", "application-tripwire"]:
        ip = kubectl("get", "service", name, "-n", "tsunami-app-lab", "-o", "jsonpath={.spec.clusterIP}").strip()
        destinations[name] = f"http://{ip}:5000"
model_ip = ""
if os.environ.get("PROBE_OPENROUTER") == "true":
    # Resolve only this operator-approved model dependency. The runner gets a
    # numeric address, so its own DNS denial cannot mask the TCP policy check.
    model_ip = socket.getaddrinfo("openrouter.ai", 443, socket.AF_INET, socket.SOCK_STREAM)[0][4][0]
template = json.loads(kubectl("create", "-f", "deploy/network-probe.yaml", "--dry-run=client", "-o", "json"))
template = with_image(template, deployed_image(kubectl))
results = []
for role, positive in (("positive-before", True), ("runner", False), ("positive-after", True)):
    manifest = copy.deepcopy(template)
    manifest["metadata"]["generateName"] = f"tsunami-network-{role}-"
    spec = manifest["spec"]["template"]["spec"]
    manifest["spec"]["template"]["metadata"]["labels"]["tsunami-role"] = "verifier" if positive else "runner"
    if runtime and not positive:
        spec["runtimeClassName"] = runtime
    spec["containers"][0]["env"] = [
        {"name": "TSUNAMI_PROBE_BROKER", "value": f'http://{ips["tsunami-control"]}:8080'},
        {"name": "TSUNAMI_PROBE_DESTINATIONS", "value": json.dumps(destinations)},
        {"name": "TSUNAMI_PROBE_POSITIVE", "value": str(positive).lower()},
        {"name": "TSUNAMI_PROBE_DNS", "value": f"http://{dns}:53"},
        {"name": "TSUNAMI_PROBE_OPENROUTER_IPV4", "value": model_ip},
    ]
    job = json.loads(kubectl("create", "-f", "-", "-o", "json", data=json.dumps(manifest)))["metadata"]["name"]
    deadline = time.monotonic() + 100
    while time.monotonic() < deadline:
        state = json.loads(kubectl("get", "job", job, "-o", "json")).get("status", {})
        if state.get("succeeded") or state.get("failed"):
            break
        time.sleep(1)
    log = kubectl("logs", f"job/{job}")
    print(f"--- {role}: {job} ---\n{log}", flush=True)
    (out / f"{role}.log").write_text(log)
    pods = json.loads(kubectl("get", "pods", "-l", f"job-name={job}", "-o", "json"))["items"]
    image_id = pods[0].get("status", {}).get("containerStatuses", [{}])[0].get("imageID", "") if pods else ""
    results.append({"role": role, "job": job, "runtime_class": runtime if not positive else "", "image_id": image_id, "passed": bool(state.get("succeeded"))})
    if positive and not results[-1]["passed"]:
        break
(out / "network.json").write_text(json.dumps({"checked_at": datetime.now(timezone.utc).isoformat(), "context": context, "destinations": destinations, "udp_dns": f"{dns}:53", "openrouter_ipv4": model_ip or None, "checks": results}, indent=2) + "\n")
if len(results) != 3 or not all(r["passed"] for r in results):
    sys.exit("FAIL: network boundary not qualified; inspect startup and settled observations")
print(f"PASS tested TCP and UDP DNS paths; results in {out}. IPv6 and metadata are not covered.")

#!/usr/bin/env python3
"""Test the connector's actual network identity against another lab tenant.

Trusted positive controls bracket the connector-identity probe. No secret is
mounted, and TCP probes do not issue assessment requests.
"""
from datetime import datetime, timezone
import copy
import json
from deploy_images import deployed_image, with_image
import os
from pathlib import Path
import subprocess
import sys
import time

context = os.environ.get("CONTEXT", "kind-tsunami-lab")
if context != "kind-tsunami-lab":
    sys.exit("Connector diagnostics are restricted to the dedicated local lab")
base = ["kubectl", "--context", context]


def kubectl(*args, data=None):
    return subprocess.check_output(base + list(args), input=data, text=True)


def service(namespace, name):
    return kubectl("-n", namespace, "get", "service", name, "-o", "jsonpath={.spec.clusterIP}").strip()


own = "http://"+service("tsunami-customer-a", "tsunami-vulnerable")+":8080"
destinations = {"customer-b": "http://"+service("tsunami-customer-b", "tsunami-fixed")+":8080", "api-service": "https://"+service("default", "kubernetes")+":443"}
node = json.loads(kubectl("get", "nodes", "-o", "json"))["items"][0]
node_ip = next(a["address"] for a in node["status"]["addresses"] if a["type"] == "InternalIP")
destinations["node-api"] = f"https://{node_ip}:6443"
template = json.loads(kubectl("-n", "tsunami", "create", "--dry-run=client", "-f", "deploy/network-probe.yaml", "-o", "json"))
template = with_image(template, deployed_image(kubectl))
out = Path("artifacts/private/connector-network"); out.mkdir(parents=True, exist_ok=True)
checks = []
for role, namespace, positive in (("positive-before", "tsunami", True), ("connector", "tsunami-customer-a", False), ("positive-after", "tsunami", True)):
    manifest = copy.deepcopy(template)
    manifest["metadata"].pop("namespace", None)
    manifest["metadata"]["generateName"] = "tsunami-connector-probe-"
    spec = manifest["spec"]["template"]["spec"]
    manifest["spec"]["template"]["metadata"]["labels"] = {"tsunami-role": "verifier" if positive else "connector", "tsunami-instance": "probe"}
    if not positive:
        spec["serviceAccountName"] = "default"
    spec["containers"][0]["env"] = [{"name": "TSUNAMI_PROBE_BROKER", "value": own}, {"name": "TSUNAMI_PROBE_DESTINATIONS", "value": json.dumps(destinations)}, {"name": "TSUNAMI_PROBE_POSITIVE", "value": str(positive).lower()}]
    job = json.loads(kubectl("-n", namespace, "create", "-f", "-", "-o", "json", data=json.dumps(manifest)))["metadata"]["name"]
    state = {}
    for _ in range(100):
        state = json.loads(kubectl("-n", namespace, "get", "job", job, "-o", "json")).get("status", {})
        if state.get("succeeded") or state.get("failed"):
            break
        time.sleep(1)
    log = kubectl("-n", namespace, "logs", "job/"+job)
    (out/(role+".log")).write_text(log)
    pods = json.loads(kubectl("-n", namespace, "get", "pods", "-l", "job-name="+job, "-o", "json"))["items"]
    image_id = pods[0]["status"]["containerStatuses"][0]["imageID"] if pods else ""
    checks.append({"role": role, "namespace": namespace, "job": job, "passed": bool(state.get("succeeded")), "image_id": image_id})
    print(f"{role}: {job} passed={checks[-1]['passed']}", flush=True)
    if positive and not checks[-1]["passed"]:
        break
(out/"record.json").write_text(json.dumps({"checked_at": datetime.now(timezone.utc).isoformat(), "context": context, "allowed_health_check": own, "denied_destinations": destinations, "checks": checks}, indent=2)+"\n")
if len(checks) != 3 or not all(c["passed"] for c in checks):
    sys.exit(f"Connector network boundary unqualified; inspect {out}")
print("PASS connector identity can reach its own web fixture; tested customer B and API paths are denied")

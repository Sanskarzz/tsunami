#!/usr/bin/env python3
"""Prepare a synthetic held-request application only in the owned Kind lab."""
import sys
if __name__ == "__main__":
    sys.dont_write_bytecode = True

import argparse
from deployment_guard import require_secured_lab
import base64
import hashlib
import json
from pathlib import Path
import secrets
import subprocess

from deploy_images import deployed_image


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--context", default="kind-tsunami-lab")
    args = parser.parse_args()
    require_secured_lab(args.context)
    if args.context != "kind-tsunami-lab":
        parser.error("use only the dedicated kind-tsunami-lab context")
    base = ["kubectl", "--context", args.context]
    def kube(*argv, data=None):
        return subprocess.check_output(base + list(argv), input=data, text=True, stderr=subprocess.DEVNULL)
    def apply(value):
        kube("apply", "-f", "-", data=json.dumps(value))
    pods = json.loads(kube("-n", "tsunami", "get", "pods", "-l", "tsunami-role=runner", "-o", "json"))["items"]
    if any(p.get("status", {}).get("phase") in ("Running", "Pending") for p in pods):
        raise RuntimeError("finish active/pending workers before reconfiguring the lab")
    kube("get", "runtimeclass", "gvisor")
    image = deployed_image(kube)
    ns = "tsunami-stop-lab"
    apply({"apiVersion": "v1", "kind": "Namespace", "metadata": {"name": ns, "labels": {"pod-security.kubernetes.io/enforce": "restricted"}}})
    credentials = {alias: secrets.token_urlsafe(36) for alias in ("owner", "peer")}
    apply({"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "stop-fixture-credentials", "namespace": ns}, "data": {"credentials.json": base64.b64encode(json.dumps(credentials).encode()).decode()}})
    labels = {"app": "tsunami-stop-fixture"}
    container = {"name": "fixture", "image": image, "imagePullPolicy": "IfNotPresent", "args": ["stop-fixture"],
                 "env": [{"name": "TSUNAMI_ALLOW_STOP_LAB", "value": "true"}],
                 "securityContext": {"allowPrivilegeEscalation": False, "readOnlyRootFilesystem": True, "capabilities": {"drop": ["ALL"]}},
                 "volumeMounts": [{"name": "credentials", "mountPath": "/run/stop", "readOnly": True}],
                 "readinessProbe": {"httpGet": {"path": "/healthz", "port": 8080}},
                 "resources": {"requests": {"cpu": "10m", "memory": "16Mi"}, "limits": {"cpu": "200m", "memory": "64Mi"}}}
    spec = {"automountServiceAccountToken": False, "securityContext": {"runAsNonRoot": True, "runAsUser": 65532, "runAsGroup": 65532, "seccompProfile": {"type": "RuntimeDefault"}},
            "containers": [container], "volumes": [{"name": "credentials", "secret": {"secretName": "stop-fixture-credentials"}}]}
    apply({"apiVersion": "apps/v1", "kind": "Deployment", "metadata": {"name": "tsunami-stop-fixture", "namespace": ns}, "spec": {"replicas": 1, "selector": {"matchLabels": labels}, "template": {"metadata": {"labels": labels}, "spec": spec}}})
    apply({"apiVersion": "v1", "kind": "Service", "metadata": {"name": "tsunami-stop-fixture", "namespace": ns}, "spec": {"selector": labels, "ports": [{"port": 8080, "targetPort": 8080}]}})
    apply({"apiVersion": "networking.k8s.io/v1", "kind": "NetworkPolicy", "metadata": {"name": "stop-fixture", "namespace": ns}, "spec": {
        "podSelector": {"matchLabels": labels}, "policyTypes": ["Ingress", "Egress"], "egress": [],
        "ingress": [{"from": [{"namespaceSelector": {"matchLabels": {"kubernetes.io/metadata.name": "tsunami"}}, "podSelector": {"matchExpressions": [{"key": "tsunami-role", "operator": "In", "values": ["control", "verifier"]}]}}], "ports": [{"port": 8080, "protocol": "TCP"}]}]}})
    kube("-n", ns, "rollout", "restart", "deployment/tsunami-stop-fixture")
    kube("-n", ns, "rollout", "status", "deployment/tsunami-stop-fixture", "--timeout=60s")
    ip = kube("-n", ns, "get", "service", "tsunami-stop-fixture", "-o", "jsonpath={.spec.clusterIP}").strip()
    # Preserve existing application plans/credentials without printing them.
    secret = json.loads(kube("-n", "tsunami", "get", "secret", "tsunami-applications", "--ignore-not-found", "-o", "json") or "null")
    configs = json.loads(base64.b64decode(secret["data"]["config.json"])) if secret else []
    configs = [c for c in configs if not c["plan"]["id"].startswith("stop-lab-")]
    for case in ["run", "engagement", "api-outage", "restart"]:
        prefix = "/cases/" + case
        owner = {"account": "owner", "path": prefix + "/objects/owner", "equals": {"/id": "owner", "/owner": "owner"}, "sha256": {"/proof": hashlib.sha256(b"synthetic-stop-proof-owner").hexdigest()}}
        peer = {"account": "peer", "path": prefix + "/objects/peer", "equals": {"/id": "peer", "/owner": "peer"}, "sha256": {"/proof": hashlib.sha256(b"synthetic-stop-proof-peer").hexdigest()}}
        plan = {"version": "http-object-access/v2", "id": "stop-lab-" + case, "title": "Owned held-request Stop test", "origin": f"http://tsunami-stop-fixture.{ns}.svc.cluster.local:8080",
                "approved_addresses": [ip], "synthetic_data_only": True, "owner_pointer": "/owner", "identity": [{"account": a, "path": prefix + "/me", "equals": {"/name": a}} for a in ("owner", "peer")],
                "baselines": [owner, peer], "probe": dict(owner, account="peer"), "expected_access": "deny", "denial_statuses": [401, 403, 404]}
        configs.append({"tenant": "demo-a", "plan": plan, "credentials": credentials})
    local = Path(".local/stop-application-config.json")
    local.write_text(json.dumps(configs, indent=2) + "\n"); local.chmod(0o600)
    apply({"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "tsunami-applications", "namespace": "tsunami"}, "data": {"config.json": base64.b64encode(local.read_bytes()).decode()}})
    # Namespace-limited pod observation is needed to distinguish API cleanup
    # from actual pod/container termination observations.
    role = json.loads(kube("-n", "tsunami", "get", "role", "tsunami-job-controller", "-o", "json"))
    rules = [r for r in role["rules"] if "pods" not in r.get("resources", [])]
    rules.append({"apiGroups": [""], "resources": ["pods"], "verbs": ["get", "list"]})
    kube("-n", "tsunami", "patch", "role", "tsunami-job-controller", "--type=merge", "-p", json.dumps({"rules": rules}))
    control = json.loads(kube("-n", "tsunami", "get", "deployment", "tsunami-control", "-o", "json"))
    pod = control["spec"]["template"]["spec"]; c = pod["containers"][0]
    c["env"] = [e for e in c.get("env", []) if e["name"] != "TSUNAMI_APPLICATION_CONFIG_FILE"] + [{"name": "TSUNAMI_APPLICATION_CONFIG_FILE", "value": "/run/applications/config.json"}]
    c["volumeMounts"] = [v for v in c.get("volumeMounts", []) if v["name"] != "application-config"] + [{"name": "application-config", "mountPath": "/run/applications", "readOnly": True}]
    pod["volumes"] = [v for v in pod.get("volumes", []) if v["name"] != "application-config"] + [{"name": "application-config", "secret": {"secretName": "tsunami-applications"}}]
    c["readinessProbe"] = {"httpGet": {"path": "/readyz", "port": 8080}}
    c["livenessProbe"] = {"httpGet": {"path": "/livez", "port": 8080}, "initialDelaySeconds": 5, "periodSeconds": 5}
    kube("-n", "tsunami", "patch", "deployment", "tsunami-control", "--type=merge", "-p", json.dumps({"spec": {"template": {"spec": pod}}}))
    kube("-n", "tsunami", "rollout", "restart", "deployment/tsunami-control")
    kube("-n", "tsunami", "rollout", "status", "deployment/tsunami-control", "--timeout=90s")
    print("Owned held-request fixture and four bounded plans configured. Existing operator/model/private credentials preserved. No assessments or model calls made.")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, ValueError, OSError, subprocess.SubprocessError):
        raise SystemExit("Owned Stop lab setup failed; inspect resources (credential output suppressed).")

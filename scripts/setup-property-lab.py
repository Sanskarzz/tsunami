#!/usr/bin/env python3
"""Provision an owned property-exposure fixture and four exact read plans."""
import argparse
import base64
import json
from pathlib import Path
import secrets
import subprocess

from application_registry import merge_applications, write_private_json, binding_compiler, prepare_plan
from deploy_images import deployed_image


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--context", default="kind-tsunami-lab")
    args = parser.parse_args()
    if args.context != "kind-tsunami-lab":
        parser.error("use only the dedicated kind-tsunami-lab context")
    binding_compiler()
    base = ["kubectl", "--context", args.context]
    def kube(*argv, data=None):
        return subprocess.check_output(base + list(argv), input=data, text=True, stderr=subprocess.DEVNULL)
    def apply(value):
        return kube("apply", "-f", "-", data=json.dumps(value))
    active = json.loads(kube("-n", "tsunami", "get", "pods", "-l", "tsunami-role=runner", "-o", "json"))["items"]
    if any(p.get("status", {}).get("phase") in ("Running", "Pending") for p in active):
        raise RuntimeError("finish active/pending workers before lab reconfiguration")
    kube("get", "runtimeclass", "gvisor")
    image = deployed_image(kube)
    # Require an existing application mount; do not overwrite unrelated pod
    # configuration while provisioning another trusted lab application.
    control = json.loads(kube("-n", "tsunami", "get", "deployment", "tsunami-control", "-o", "json"))
    pod = control["spec"]["template"]["spec"]
    if not any(e["name"] == "TSUNAMI_APPLICATION_CONFIG_FILE" and e.get("value") == "/run/applications/config.json" for e in pod["containers"][0].get("env", [])) or not any(v.get("secret", {}).get("secretName") == "tsunami-applications" for v in pod.get("volumes", [])):
        raise RuntimeError("configure the owned application/Stop lab mount first")
    existing = json.loads(kube("-n", "tsunami", "get", "secret", "tsunami-applications", "-o", "json"))
    previous = json.loads(base64.b64decode(existing["data"]["config.json"]))
    cfg = {"credentials": {a: secrets.token_urlsafe(36) for a in ("owner", "peer")}, "records": {a: {"id": secrets.token_hex(12), "private_note": "synthetic-private-note-" + secrets.token_urlsafe(24)} for a in ("owner", "peer")}}
    # Validate capacity before making provisioning changes. Real service IP is
    # filled after its owned Service exists; no target requests are made here.
    merge_applications(previous, [{"tenant": "demo-a", "plan": {"id": "property-" + v}} for v in ("vulnerable", "filtered", "ambiguous", "stop")])
    ns = "tsunami-property-lab"
    apply({"apiVersion": "v1", "kind": "Namespace", "metadata": {"name": ns, "labels": {"pod-security.kubernetes.io/enforce": "restricted"}}})
    write_private_json(".local/property-fixture-config.json", cfg)
    apply({"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "property-fixture-config", "namespace": ns}, "data": {"config.json": base64.b64encode(json.dumps(cfg).encode()).decode()}})
    labels = {"app": "tsunami-property-fixture"}
    container = {"name": "fixture", "image": image, "imagePullPolicy": "IfNotPresent", "args": ["property-fixture"], "env": [{"name": "TSUNAMI_ALLOW_PROPERTY_LAB", "value": "true"}],
                 "securityContext": {"allowPrivilegeEscalation": False, "readOnlyRootFilesystem": True, "capabilities": {"drop": ["ALL"]}},
                 "volumeMounts": [{"name": "config", "mountPath": "/run/property", "readOnly": True}], "readinessProbe": {"httpGet": {"path": "/healthz", "port": 8080}},
                 "resources": {"requests": {"cpu": "10m", "memory": "16Mi"}, "limits": {"cpu": "200m", "memory": "64Mi"}}}
    spec = {"automountServiceAccountToken": False, "securityContext": {"runAsNonRoot": True, "runAsUser": 65532, "runAsGroup": 65532, "seccompProfile": {"type": "RuntimeDefault"}}, "containers": [container], "volumes": [{"name": "config", "secret": {"secretName": "property-fixture-config"}}]}
    apply({"apiVersion": "apps/v1", "kind": "Deployment", "metadata": {"name": "tsunami-property-fixture", "namespace": ns}, "spec": {"replicas": 1, "selector": {"matchLabels": labels}, "template": {"metadata": {"labels": labels}, "spec": spec}}})
    apply({"apiVersion": "v1", "kind": "Service", "metadata": {"name": "tsunami-property-fixture", "namespace": ns}, "spec": {"selector": labels, "ports": [{"port": 8080, "targetPort": 8080}]}})
    apply({"apiVersion": "networking.k8s.io/v1", "kind": "NetworkPolicy", "metadata": {"name": "property-fixture", "namespace": ns}, "spec": {"podSelector": {"matchLabels": labels}, "policyTypes": ["Ingress", "Egress"], "egress": [], "ingress": [{"from": [{"namespaceSelector": {"matchLabels": {"kubernetes.io/metadata.name": "tsunami"}}, "podSelector": {"matchExpressions": [{"key": "tsunami-role", "operator": "In", "values": ["control", "verifier"]}]}}], "ports": [{"port": 8080, "protocol": "TCP"}]}]}})
    kube("-n", ns, "rollout", "restart", "deployment/tsunami-property-fixture")
    kube("-n", ns, "rollout", "status", "deployment/tsunami-property-fixture", "--timeout=60s")
    ip = kube("-n", ns, "get", "service", "tsunami-property-fixture", "-o", "jsonpath={.spec.clusterIP}").strip()
    configured = []
    for variant in ("vulnerable", "filtered", "ambiguous", "stop"):
        prefix = "/cases/" + variant
        observations = {a: {"identity": {"status": 200, "body": {"name": a}}, "object": {"status": 200, "body": {"id": cfg["records"][a]["id"], "owner": a, "private_note": cfg["records"][a]["private_note"]}}} for a in ("owner", "peer")}
        plan = prepare_plan({"version": "tsunami.application-binding/v1", "profile": "http-property-access/v1", "id": "property-" + variant, "title": "Owner-only note exposed through a shared record", "origin": f"http://tsunami-property-fixture.{ns}.svc.cluster.local:8080", "approved_addresses": [ip], "synthetic_data_only": True, "expected_access": "filtered", "identity_path": prefix + "/me", "identity_pointer": "/name", "object_path_template": prefix + "/records/{id}", "object_id_pointer": "/id", "owner_pointer": "/owner", "protected_pointers": ["/private_note"], "owner": observations["owner"], "peer": observations["peer"]})
        configured.append({"tenant": "demo-a", "plan": plan, "credentials": cfg["credentials"]})
    registry = merge_applications(previous, configured)
    write_private_json(".local/property-application-config.json", registry)
    # A concurrent registry update must not be overwritten silently.
    kube("-n", "tsunami", "patch", "secret", "tsunami-applications", "--type=json", "--patch-file=/dev/stdin", data=json.dumps([
        {"op": "test", "path": "/metadata/resourceVersion", "value": existing["metadata"]["resourceVersion"]},
        {"op": "replace", "path": "/data/config.json", "value": base64.b64encode(json.dumps(registry).encode()).decode()}]))
    kube("-n", "tsunami", "rollout", "restart", "deployment/tsunami-control")
    kube("-n", "tsunami", "rollout", "status", "deployment/tsunami-control", "--timeout=90s")
    print("Owned property fixture and four bounded read plans configured. Other plans and model/operator/private credentials preserved. No assessments or model calls made.")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, ValueError, KeyError, OSError, subprocess.SubprocessError):
        raise SystemExit("Property lab setup failed; inspect owned resources (credential output suppressed).")

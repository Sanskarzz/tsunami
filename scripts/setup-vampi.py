#!/usr/bin/env python3
"""Provision and seed only the owned VAmPI lab. No agent or model calls."""
import argparse
import base64
from contextlib import contextmanager
import json
from deploy_images import deployed_image, with_image, manifest_json
from application_registry import merge_applications, write_private_json, binding_compiler, prepare_plan
import os
from pathlib import Path
import secrets
import socket
import subprocess
import time
import urllib.request

SOURCE = "https://github.com/erev0s/VAmPI.git"
COMMIT = "f16052dce83f05847133ec98f01c5193a41de7d8"
NAMESPACE = "tsunami-app-lab"
IMAGE = "tsunami-vampi:f16052d"


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        return None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--context", default="kind-tsunami-lab")
    args = parser.parse_args()
    if args.context != "kind-tsunami-lab":
        parser.error("use the dedicated kind-tsunami-lab context")
    binding_compiler()
    base = ["kubectl", "--context", args.context]

    def kube(*argv, data=None):
        return subprocess.check_output(base + list(argv), input=data, text=True, stderr=subprocess.DEVNULL)

    def apply(value):
        kube("apply", "-f", "-", data=json.dumps(value))

    active = json.loads(kube("-n", "tsunami", "get", "pods", "-l", "tsunami-role=runner", "--field-selector=status.phase=Running", "-o", "json"))["items"]
    if active:
        raise RuntimeError("finish active worker runs before reconfiguring applications")
    source = Path(".local/external/VAmPI")
    source.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    if not source.exists():
        subprocess.run(["git", "clone", "--quiet", SOURCE, str(source)], check=True)
        subprocess.run(["git", "-C", str(source), "checkout", "--quiet", "--detach", COMMIT], check=True)
    head = subprocess.check_output(["git", "-C", str(source), "rev-parse", "HEAD"], text=True).strip()
    dirty = subprocess.check_output(["git", "-C", str(source), "status", "--porcelain"], text=True).strip()
    if head != COMMIT or dirty:
        raise RuntimeError("expected a clean pinned VAmPI checkout; existing files were not overwritten")
    records = Path("artifacts/application-adapters")
    records.mkdir(parents=True, exist_ok=True, mode=0o700)
    with (records / "vampi-build.log").open("w") as log:
        subprocess.run(["docker", "build", "-f", "deploy/applications/VAmPI.Dockerfile", "-t", IMAGE, str(source)], stdout=log, stderr=subprocess.STDOUT, check=True)
    subprocess.run(["kind", "load", "docker-image", IMAGE, "--name", "tsunami-lab"], stdout=subprocess.DEVNULL, check=True)
    apply({"apiVersion": "v1", "kind": "Namespace", "metadata": {"name": NAMESPACE, "labels": {"pod-security.kubernetes.io/enforce": "restricted"}}})
    tripwire = manifest_json(kube("create", "--dry-run=client", "-f", "deploy/applications/network-tripwire.yaml", "-o", "json"))
    apply(with_image(tripwire, deployed_image(kube)))
    for name, vulnerable in [("vampi-vulnerable", "1"), ("vampi-fixed", "0")]:
        labels = {"app": name, "tsunami-app-lab": "vampi"}
        container = {"name": "vampi", "image": IMAGE, "imagePullPolicy": "IfNotPresent",
                     "env": [{"name": "vulnerable", "value": vulnerable}, {"name": "tokentimetolive", "value": "3600"}],
                     "ports": [{"containerPort": 5000}],
                     "securityContext": {"allowPrivilegeEscalation": False, "readOnlyRootFilesystem": True, "capabilities": {"drop": ["ALL"]}},
                     "resources": {"requests": {"cpu": "50m", "memory": "96Mi"}, "limits": {"cpu": "500m", "memory": "256Mi"}},
                     "readinessProbe": {"httpGet": {"path": "/", "port": 5000}, "initialDelaySeconds": 2, "periodSeconds": 2},
                     "volumeMounts": [{"name": "database", "mountPath": "/vampi/database"}, {"name": "tmp", "mountPath": "/tmp"}]}
        spec = {"automountServiceAccountToken": False, "securityContext": {"runAsNonRoot": True, "runAsUser": 10001, "runAsGroup": 10001, "fsGroup": 10001, "seccompProfile": {"type": "RuntimeDefault"}},
                "containers": [container], "volumes": [{"name": "database", "emptyDir": {"sizeLimit": "16Mi"}}, {"name": "tmp", "emptyDir": {"sizeLimit": "16Mi"}}]}
        apply({"apiVersion": "apps/v1", "kind": "Deployment", "metadata": {"name": name, "namespace": NAMESPACE}, "spec": {"replicas": 1, "selector": {"matchLabels": labels}, "template": {"metadata": {"labels": labels}, "spec": spec}}})
        apply({"apiVersion": "v1", "kind": "Service", "metadata": {"name": name, "namespace": NAMESPACE}, "spec": {"selector": labels, "ports": [{"port": 5000, "targetPort": 5000}]}})
    apply({"apiVersion": "networking.k8s.io/v1", "kind": "NetworkPolicy", "metadata": {"name": "vampi-lab", "namespace": NAMESPACE}, "spec": {
        "podSelector": {"matchLabels": {"tsunami-app-lab": "vampi"}}, "policyTypes": ["Ingress", "Egress"], "egress": [],
        "ingress": [{"from": [{"namespaceSelector": {"matchLabels": {"kubernetes.io/metadata.name": "tsunami"}}, "podSelector": {"matchExpressions": [{"key": "tsunami-role", "operator": "In", "values": ["control", "verifier"]}]}}], "ports": [{"protocol": "TCP", "port": 5000}]}]}})
    for name in ["vampi-vulnerable", "vampi-fixed"]:
        kube("-n", NAMESPACE, "rollout", "status", "deployment/" + name, "--timeout=90s")

    @contextmanager
    def forward(name):
        listener = socket.socket()
        listener.bind(("127.0.0.1", 0))
        port = listener.getsockname()[1]
        listener.close()
        proc = subprocess.Popen(base + ["-n", NAMESPACE, "port-forward", "--address=127.0.0.1", "service/" + name, f"{port}:5000"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
        url = f"http://127.0.0.1:{port}"

        def call(path, body=None, token=None, capture=False):
            data = None if body is None else json.dumps(body).encode()
            headers = {"Content-Type": "application/json"}
            if token:
                headers["Authorization"] = "Bearer " + token
            request = urllib.request.Request(url + path, data=data, headers=headers)
            with opener.open(request, timeout=5) as response:
                status = response.status
                raw = response.read((64 << 10) + 1)
            if len(raw) > 64 << 10:
                raise RuntimeError("setup response exceeds limit")
            parsed = json.loads(raw)
            return {"status": status, "body": parsed} if capture else parsed

        try:
            for _ in range(100):
                try:
                    call("/")
                    break
                except OSError:
                    if proc.poll() is not None:
                        raise RuntimeError("lab port-forward ended")
                    time.sleep(0.1)
            else:
                raise RuntimeError("lab target did not become ready")
            yield call
        finally:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait()

    configurations = []
    for name in ["vampi-vulnerable", "vampi-fixed"]:
        credentials = {}
        observations = {}
        with forward(name) as call:
            call("/createdb")  # Only the dedicated, disposable lab database.
            for alias in ["owner", "peer"]:
                username = "tsunami_" + alias + "_" + secrets.token_hex(4)
                password = secrets.token_urlsafe(24)
                title = "tsunami-" + alias + "-" + secrets.token_hex(6)
                proof = "synthetic-book-proof-" + secrets.token_hex(16)
                call("/users/v1/register", {"username": username, "password": password, "email": username + "@example.invalid"})
                token = call("/users/v1/login", {"username": username, "password": password}).get("auth_token")
                if not isinstance(token, str) or len(token) < 32:
                    raise RuntimeError("synthetic account login failed")
                call("/books/v1", {"book_title": title, "secret": proof}, token)
                credentials[alias] = token
                observations[alias] = {"identity": call("/me", token=token, capture=True), "object": call("/books/v1/" + title, token=token, capture=True)}
        ip = kube("-n", NAMESPACE, "get", "service", name, "-o", "jsonpath={.spec.clusterIP}").strip()
        plan = prepare_plan({"version": "tsunami.application-binding/v1", "profile": "http-object-access/v2", "id": name, "title": "Cross-user book secret access", "origin": f"http://{name}.{NAMESPACE}.svc.cluster.local:5000", "approved_addresses": [ip], "synthetic_data_only": True, "expected_access": "deny", "identity_path": "/me", "identity_pointer": "/data/username", "object_path_template": "/books/v1/{id}", "object_id_pointer": "/book_title", "owner_pointer": "/owner", "protected_pointers": ["/secret"], "owner": observations["owner"], "peer": observations["peer"]})
        configurations.append({"tenant": "demo-a", "plan": plan, "credentials": credentials})
    existing = json.loads(kube("-n", "tsunami", "get", "secret", "tsunami-applications", "--ignore-not-found", "-o", "json") or "null")
    previous = json.loads(base64.b64decode(existing["data"]["config.json"])) if existing else []
    configurations = merge_applications(previous, configurations)
    local = Path(".local/application-config.json")
    write_private_json(local, configurations)
    apply({"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "tsunami-applications", "namespace": "tsunami"}, "type": "Opaque", "data": {"config.json": base64.b64encode(local.read_bytes()).decode()}})
    deployment = json.loads(kube("-n", "tsunami", "get", "deployment", "tsunami-control", "-o", "json"))
    pod = deployment["spec"]["template"]["spec"]
    container = pod["containers"][0]
    container["env"] = [e for e in container.get("env", []) if e["name"] != "TSUNAMI_APPLICATION_CONFIG_FILE"] + [{"name": "TSUNAMI_APPLICATION_CONFIG_FILE", "value": "/run/applications/config.json"}]
    container["volumeMounts"] = [m for m in container.get("volumeMounts", []) if m["name"] != "application-config"] + [{"name": "application-config", "mountPath": "/run/applications", "readOnly": True}]
    pod["volumes"] = [v for v in pod.get("volumes", []) if v["name"] != "application-config"] + [{"name": "application-config", "secret": {"secretName": "tsunami-applications", "defaultMode": 0o444}}]
    kube("-n", "tsunami", "patch", "deployment", "tsunami-control", "--type=merge", "-p", json.dumps({"spec": {"template": {"spec": pod}}}))
    kube("-n", "tsunami", "rollout", "restart", "deployment/tsunami-control")
    kube("-n", "tsunami", "rollout", "status", "deployment/tsunami-control", "--timeout=90s")
    image_id = subprocess.check_output(["docker", "image", "inspect", IMAGE, "--format", "{{.Id}}"], text=True).strip()
    (records / "source.json").write_text(json.dumps({"source": SOURCE, "commit": COMMIT, "license": "MIT (upstream source retained locally, not vendored)", "image": image_id, "synthetic_data_only": True, "account_token_lifetime_seconds": 3600, "debugger_enabled": False}, indent=2) + "\n")
    print("Owned VAmPI lab seeded; four legitimate baseline GETs per variant compiled into draft registration plans. No cross-user probes or model calls made. Scope approval is still required.")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, KeyError, ValueError, subprocess.SubprocessError):
        raise SystemExit("VAmPI lab setup failed; inspect rollout/build records (credentials suppressed).")

#!/usr/bin/env python3
"""Install only synthetic private targets in the dedicated local cluster.

Secret values are held in memory and sent through stdin; never printed.
Generated non-secret resources are retained for review.
"""
import base64
import json
from deploy_images import deployed_image
import os
from pathlib import Path
import subprocess
import sys

context = os.environ.get("CONTEXT", "kind-tsunami-lab")
if context != "kind-tsunami-lab":
    sys.exit("Private lab setup is restricted to kind-tsunami-lab")
base = ["kubectl", "--context", context]


def kubectl(*args, data=None):
    p = subprocess.run(base + list(args), input=data, text=True, capture_output=True)
    if p.returncode:
        sys.exit("Private lab setup command failed; inspect resources with kubectl (Secret errors suppressed)")
    return p.stdout


def apply(obj):
    return kubectl("apply", "-f", "-", data=json.dumps(obj))


def obj(kind, name, ns, **kwargs):
    return {"apiVersion": {"Deployment": "apps/v1", "NetworkPolicy": "networking.k8s.io/v1"}.get(kind, "v1"), "kind": kind, "metadata": {"name": name, "namespace": ns}, **kwargs}


selected_image = os.environ.get("IMAGE") or deployed_image(kubectl)
if any(c.isspace() for c in selected_image):
    sys.exit("An explicit native image without whitespace is required")

resources = []
for ns in ("tsunami-customer-a", "tsunami-customer-b"):
    resource = {"apiVersion": "v1", "kind": "Namespace", "metadata": {"name": ns, "labels": {"app.kubernetes.io/part-of": "tsunami", "pod-security.kubernetes.io/enforce": "restricted"}}}
    apply(resource)
    resources.append(resource)

# Private key only in broker namespace; connector receives its verification key
# through config and a separate operator-only emergency credential.
generated = json.loads(subprocess.check_output(["./bin/tsunami", "bootstrap-connector-keys"], text=True))
pki_dir = Path(".local/private-pki")
existing_tls = json.loads(kubectl("-n", "tsunami", "get", "secret", "tsunami-private-broker-tls", "--ignore-not-found", "-o", "json") or "null")
if existing_tls:
    if not (pki_dir/"ca.crt").is_file() or not (pki_dir/"ca.key").is_file():
        sys.exit("Existing connector trust found but offline CA files are missing; restore them rather than replacing trust implicitly")
    approved_ca = base64.b64decode(existing_tls["data"]["ca.crt"])
    if approved_ca != (pki_dir/"ca.crt").read_bytes():
        sys.exit("Offline CA differs from installed connector trust; automatic CA rollover is refused")
pki_args = ["./bin/tsunami", "bootstrap-connector-tls", "--dir", str(pki_dir)]
if os.environ.get("RENEW_PRIVATE_TLS") == "true":
    pki_args.append("--renew")
pki_record = json.loads(subprocess.check_output(pki_args, text=True))
for namespace, name, role in (("tsunami", "tsunami-private-broker-tls", "broker"), ("tsunami-customer-a", "tsunami-private-server-tls", "connector")):
    apply(obj("Secret", name, namespace, type="kubernetes.io/tls", stringData={"ca.crt": (pki_dir/"ca.crt").read_text(), "tls.crt": (pki_dir/(role+".crt")).read_text(), "tls.key": (pki_dir/(role+".key")).read_text()}))
existing = json.loads(kubectl("-n", "tsunami", "get", "secret", "tsunami-private-signer", "--ignore-not-found", "-o", "json") or "null")
if existing:
    signing = {k: base64.b64decode(v).decode() for k, v in existing["data"].items()}
else:
    signing = {k: generated[k] for k in ("signing-key", "public-key")}
    apply(obj("Secret", "tsunami-private-signer", "tsunami", type="Opaque", stringData=signing))
existing = kubectl("-n", "tsunami-customer-a", "get", "secret", "tsunami-connector-emergency", "--ignore-not-found", "-o", "json")
if not existing.strip():
    apply(obj("Secret", "tsunami-connector-emergency", "tsunami-customer-a", type="Opaque", stringData={"emergency-key": generated["emergency-key"]}))

security = {"runAsNonRoot": True, "runAsUser": 65532, "runAsGroup": 65532, "fsGroup": 65532, "seccompProfile": {"type": "RuntimeDefault"}}
container_security = {"allowPrivilegeEscalation": False, "readOnlyRootFilesystem": True, "capabilities": {"drop": ["ALL"]}}


def deployment(name, ns, labels, command, env=None):
    container = {"name": "app", "image": selected_image, "imagePullPolicy": "IfNotPresent", "args": command, "env": env or [], "securityContext": container_security, "readinessProbe": {"httpGet": {"path": "/healthz", "port": 8080}}, "resources": {"requests": {"cpu": "10m", "memory": "16Mi"}, "limits": {"cpu": "200m", "memory": "128Mi"}}}
    return obj("Deployment", name, ns, spec={"replicas": 1, "strategy": {"type": "Recreate"}, "selector": {"matchLabels": dict(labels)}, "template": {"metadata": {"labels": dict(labels)}, "spec": {"automountServiceAccountToken": False, "securityContext": security, "containers": [container]}}})


for ns in ("tsunami-customer-a", "tsunami-customer-b"):
    names = ("vulnerable", "fixed", "tripwire") if ns.endswith("-a") else ("fixed",)
    for name in names:
        labels = {"tsunami-role": "private-tripwire" if name == "tripwire" else "private-web", "tsunami-fixture": name}
        env = [{"name": "LAB_REDIRECT_TO", "value": "http://tsunami-tripwire:8080/api/documents/doc-alice"}] if name == "vulnerable" else []
        resources.append(deployment("tsunami-"+name, ns, labels, ["lab"] + ([] if name == "vulnerable" else ["--fixed"]), env))
        resources.append(obj("Service", "tsunami-"+name, ns, spec={"selector": labels, "ports": [{"port": 8080, "targetPort": 8080}]}))
    # Diagnostic verifier is a trusted exception, not an agent identity.
    peers = [{"namespaceSelector": {"matchLabels": {"kubernetes.io/metadata.name": "tsunami"}}, "podSelector": {"matchLabels": {"tsunami-role": "verifier"}}}]
    if ns.endswith("-a"):
        peers.append({"podSelector": {"matchLabels": {"tsunami-role": "connector"}}})
    resources.append(obj("NetworkPolicy", "private-web", ns, spec={"podSelector": {"matchLabels": {"tsunami-role": "private-web"}}, "policyTypes": ["Ingress", "Egress"], "ingress": [{"from": peers, "ports": [{"protocol": "TCP", "port": 8080}]}], "egress": []}))
    if ns.endswith("-a"):
        # An unrestricted-ingress destination exposes runner egress failures.
        resources.append(obj("NetworkPolicy", "private-tripwire", ns, spec={"podSelector": {"matchLabels": {"tsunami-role": "private-tripwire"}}, "policyTypes": ["Ingress", "Egress"], "ingress": [{}], "egress": []}))

resources.append(obj("PersistentVolumeClaim", "tsunami-connector-state", "tsunami-customer-a", spec={"accessModes": ["ReadWriteOnce"], "resources": {"requests": {"storage": "64Mi"}}}))
resources.append(obj("Service", "tsunami-connector", "tsunami-customer-a", spec={"selector": {"tsunami-role": "connector", "tsunami-instance": "main"}, "ports": [{"port": 8080, "targetPort": 8080}]}))
trusted = {"namespaceSelector": {"matchLabels": {"kubernetes.io/metadata.name": "tsunami"}}, "podSelector": {"matchExpressions": [{"key": "tsunami-role", "operator": "In", "values": ["control", "verifier"]}]}}
resources.append(obj("NetworkPolicy", "connector-boundary", "tsunami-customer-a", spec={"podSelector": {"matchLabels": {"tsunami-role": "connector"}}, "policyTypes": ["Ingress", "Egress"], "ingress": [{"from": [trusted], "ports": [{"protocol": "TCP", "port": 8080}]}], "egress": [{"to": [{"podSelector": {"matchLabels": {"tsunami-role": "private-web"}}}], "ports": [{"protocol": "TCP", "port": 8080}]}, {"to": [{"namespaceSelector": {"matchLabels": {"kubernetes.io/metadata.name": "kube-system"}}, "podSelector": {"matchLabels": {"k8s-app": "kube-dns"}}}], "ports": [{"protocol": "UDP", "port": 53}, {"protocol": "TCP", "port": 53}]}]}))
apply({"apiVersion": "v1", "kind": "List", "items": resources})

cfg = {"tenant": "demo-a", "audience": "customer-a", "public_key": signing["public-key"], "endpoint": "https://tsunami-connector.tsunami-customer-a.svc.cluster.local:8080", "tls_directory": "/run/connector-tls", "targets": {}}
for name in ("vulnerable", "fixed"):
    ip = kubectl("-n", "tsunami-customer-a", "get", "service", "tsunami-"+name, "-o", "jsonpath={.spec.clusterIP}").strip()
    cfg["targets"][name] = {"url": f"http://tsunami-{name}.tsunami-customer-a.svc.cluster.local:8080", "addresses": [ip]}
for ns in ("tsunami", "tsunami-customer-a"):
    apply(obj("ConfigMap", "tsunami-private", ns, data={"config.json": json.dumps(cfg)}))

dep = deployment("tsunami-connector", "tsunami-customer-a", {"tsunami-role": "connector"}, ["connector"], [{"name": "TSUNAMI_ALLOW_PRIVATE_LAB", "value": "true"}])
spec = dep["spec"]["template"]["spec"]
dep["spec"]["template"]["metadata"]["labels"]["tsunami-instance"] = "main"
spec["containers"][0]["volumeMounts"] = [{"name": "config", "mountPath": "/run/private", "readOnly": True}, {"name": "emergency", "mountPath": "/run/connector", "readOnly": True}, {"name": "state", "mountPath": "/data"}]
spec["volumes"] = [{"name": "config", "configMap": {"name": "tsunami-private"}}, {"name": "emergency", "secret": {"secretName": "tsunami-connector-emergency", "defaultMode": 0o444}}, {"name": "state", "persistentVolumeClaim": {"claimName": "tsunami-connector-state"}}]
spec["containers"][0]["volumeMounts"].append({"name": "connector-tls", "mountPath": "/run/connector-tls", "readOnly": True})
spec["volumes"].append({"name": "connector-tls", "secret": {"secretName": "tsunami-private-server-tls", "defaultMode": 0o444}})
# This checks listener readiness only; the verifier checks real mTLS identity.
spec["containers"][0]["readinessProbe"] = {"tcpSocket": {"port": 8080}}
apply(dep)
resources.append(dep)

control = json.loads(kubectl("-n", "tsunami", "get", "deployment", "tsunami-control", "-o", "json"))
spec = control["spec"]["template"]["spec"]
container = spec["containers"][0]
env = [e for e in container.get("env", []) if e["name"] not in ("TSUNAMI_PRIVATE_CONFIG_FILE", "TSUNAMI_ALLOW_PRIVATE_HTTP_LAB", "TSUNAMI_ALLOW_PRIVATE_LAB", "TSUNAMI_IMAGE")]
env += [{"name": "TSUNAMI_PRIVATE_CONFIG_FILE", "value": "/run/private/config.json"}, {"name": "TSUNAMI_ALLOW_PRIVATE_LAB", "value": "true"}, {"name": "TSUNAMI_IMAGE", "value": selected_image}]
mounts = [v for v in container.get("volumeMounts", []) if v["name"] not in ("private-config", "private-signer", "private-tls")]
mounts += [{"name": "private-config", "mountPath": "/run/private", "readOnly": True}, {"name": "private-signer", "mountPath": "/run/connector", "readOnly": True}]
mounts.append({"name": "private-tls", "mountPath": "/run/connector-tls", "readOnly": True})
volumes = [v for v in spec.get("volumes", []) if v["name"] not in ("private-config", "private-signer", "private-tls")]
volumes += [{"name": "private-config", "configMap": {"name": "tsunami-private"}}, {"name": "private-signer", "secret": {"secretName": "tsunami-private-signer", "items": [{"key": "signing-key", "path": "signing-key"}], "defaultMode": 0o444}}]
volumes.append({"name": "private-tls", "secret": {"secretName": "tsunami-private-broker-tls", "defaultMode": 0o444}})
patch = [{"op": "test", "path": "/metadata/resourceVersion", "value": control["metadata"]["resourceVersion"]}, {"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": selected_image}, {"op": "add", "path": "/spec/template/spec/containers/0/env", "value": env}, {"op": "add", "path": "/spec/template/spec/containers/0/volumeMounts", "value": mounts}, {"op": "add", "path": "/spec/template/spec/volumes", "value": volumes}]
kubectl("-n", "tsunami", "patch", "deployment", "tsunami-control", "--type=json", "--patch-file=/dev/stdin", data=json.dumps(patch))
for ns, name in [("tsunami-customer-a", "tsunami-vulnerable"), ("tsunami-customer-a", "tsunami-fixed"), ("tsunami-customer-a", "tsunami-tripwire"), ("tsunami-customer-a", "tsunami-connector"), ("tsunami-customer-b", "tsunami-fixed"), ("tsunami", "tsunami-control")]:
    kubectl("-n", ns, "rollout", "restart", "deployment/"+name)
    kubectl("-n", ns, "rollout", "status", "deployment/"+name, "--timeout=60s")
out = Path("artifacts/private"); out.mkdir(parents=True, exist_ok=True)
(out/"resources.json").write_text(json.dumps({"apiVersion": "v1", "kind": "List", "items": resources}, indent=2)+"\n")
(out/"config.json").write_text(json.dumps(cfg, indent=2)+"\n")
(out/"tls-record.json").write_text(json.dumps(pki_record, indent=2)+"\n")
print("Private synthetic lab configured with mTLS. CA key and emergency TLS key stay local. Signing keys, replay state, and cutoff are preserved. No model calls made.")

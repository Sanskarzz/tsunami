#!/usr/bin/env python3
"""Opt-in model configuration. Key bytes travel through stdin, never argv/logs."""
import sys
if __name__ == "__main__":
    sys.dont_write_bytecode = True

import argparse
from deployment_guard import require_secured_lab
import base64
import json
from deploy_images import image_from_control, with_image, manifest_json
import re
from pathlib import Path
import subprocess
from urllib.parse import urlparse

parser = argparse.ArgumentParser()
parser.add_argument("--provider", choices=["fixture", "responses", "openrouter-free", "disabled"], required=True)
parser.add_argument("--model")
parser.add_argument("--base-url", default="https://api.openai.com/v1/")
parser.add_argument("--key-file")
parser.add_argument("--context", default="kind-tsunami-lab")
args = parser.parse_args()
try:
    require_secured_lab(args.context)
except RuntimeError as error:
    raise SystemExit(str(error))
if args.context != "kind-tsunami-lab":
    sys.exit("Model setup is restricted to the dedicated local kind-tsunami-lab cluster")
base = ["kubectl", "--context", args.context, "-n", "tsunami"]


def kubectl(*argv, data=None):
    result = subprocess.run(base + list(argv), input=data, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if result.returncode:
        # Never echo an API error that might include a submitted Secret object.
        sys.exit("Model configuration command failed; inspect the lab resources with kubectl")
    return result.stdout


deployment = json.loads(kubectl("get", "deployment", "tsunami-control", "-o", "json"))
spec = deployment["spec"]["template"]["spec"]
container = spec["containers"][0]
if container["name"] != "control":
    sys.exit("Unexpected lab control-plane layout")
existing_env = container.get("env", [])
if args.provider != "disabled" and not any(e["name"] == "TSUNAMI_RUNTIME_CLASS" and e.get("value") == "gvisor" for e in existing_env):
    sys.exit("Configure and qualify the gVisor lab before enabling the model path")
env = [e for e in existing_env if not e["name"].startswith("TSUNAMI_MODEL_") and e["name"] != "TSUNAMI_ALLOW_MODEL_CALLS"]
mounts = [v for v in container.get("volumeMounts", []) if v["name"] != "model-credentials"]
volumes = [v for v in spec.get("volumes", []) if v["name"] != "model-credentials"]
if args.provider == "fixture":
    fixture = manifest_json(kubectl("create", "--dry-run=client", "-f", "deploy/model-fixture.yaml", "-o", "json"))
    kubectl("apply", "-f", "-", data=json.dumps(with_image(fixture, image_from_control(deployment))))
    kubectl("rollout", "restart", "deployment/tsunami-model-fixture")
    kubectl("rollout", "status", "deployment/tsunami-model-fixture", "--timeout=60s")
    args.model = "tsunami-wire-fixture"
    args.base_url = "http://tsunami-model-fixture:8080/v1/"
elif args.provider in ("responses", "openrouter-free"):
    # Apply the same endpoint/model rule before copying any credential.
    if args.provider == "openrouter-free":
        if not args.model or not re.fullmatch(r"[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+:free", args.model):
            sys.exit("openrouter-free requires an explicit :free model ID")
        if args.base_url != "https://api.openai.com/v1/" and args.base_url.rstrip("/") != "https://openrouter.ai/api/v1":
            sys.exit("openrouter-free accepts only the official OpenRouter API endpoint")
        args.base_url = "https://openrouter.ai/api/v1/"
    if not args.model or not args.key_file:
        sys.exit("responses requires --model and --key-file; no default model or key is selected")
    endpoint = urlparse(args.base_url)
    if endpoint.scheme != "https" or not endpoint.hostname or endpoint.username or endpoint.password or endpoint.query or endpoint.fragment:
        sys.exit("A plain HTTPS API base URL is required")
    key = Path(args.key_file).read_text().strip()
    if not key or len(key) > 4096:
        sys.exit("Invalid model key file")
    if key.startswith("sk-or-") and args.provider != "openrouter-free":
        sys.exit("OpenRouter keys require the pinned openrouter-free profile")
    secret = {"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "tsunami-model", "namespace": "tsunami"}, "type": "Opaque", "data": {"api-key": base64.b64encode(key.encode()).decode()}}
    kubectl("apply", "-f", "-", data=json.dumps(secret))
    mounts.append({"name": "model-credentials", "mountPath": "/run/model", "readOnly": True})
    volumes.append({"name": "model-credentials", "secret": {"secretName": "tsunami-model", "defaultMode": 0o444}})
if args.provider != "disabled":
    env += [
        {"name": "TSUNAMI_MODEL_PROVIDER", "value": args.provider},
        {"name": "TSUNAMI_MODEL_ID", "value": args.model},
        {"name": "TSUNAMI_MODEL_BASE_URL", "value": args.base_url},
        {"name": "TSUNAMI_ALLOW_MODEL_CALLS", "value": "true"},
    ]
patch = [
    {"op": "test", "path": "/metadata/resourceVersion", "value": deployment["metadata"]["resourceVersion"]},
    {"op": "add", "path": "/spec/template/spec/containers/0/env", "value": env},
    {"op": "add", "path": "/spec/template/spec/containers/0/volumeMounts", "value": mounts},
    {"op": "add", "path": "/spec/template/spec/volumes", "value": volumes},
]
kubectl("patch", "deployment", "tsunami-control", "--type=json", "--patch-file=/dev/stdin", data=json.dumps(patch))
# Keys are loaded at process startup. A Secret-only rotation would otherwise
# leave the old key in memory because it does not change the Pod template.
kubectl("rollout", "restart", "deployment/tsunami-control")
kubectl("rollout", "status", "deployment/tsunami-control", "--timeout=60s")
if args.provider not in ("responses", "openrouter-free"):
    kubectl("delete", "secret", "tsunami-model", "--ignore-not-found")
print(f"Configured {args.provider}. Default assessments remain scripted. No external model call was made by setup.")

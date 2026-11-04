#!/usr/bin/env python3
"""Run the explicitly selected model path and collect those exact run IDs."""
import argparse
import json
from deploy_images import deployed_image, with_image
from pathlib import Path
import re
import subprocess
import sys
import time

parser = argparse.ArgumentParser()
parser.add_argument("--context", default="kind-tsunami-lab")
parser.add_argument("--fixture", action="store_true", help="require the synthetic Responses fixture")
parser.add_argument("--allow-model-calls", action="store_true", help="explicitly permit configured inference")
parser.add_argument("--stop-check", action="store_true", help="stop a fresh run during inference instead of evaluating a pair")
parser.add_argument("--free-only", action="store_true", help="require openrouter-free before and at run creation")
args = parser.parse_args()
if args.context != "kind-tsunami-lab":
    sys.exit("Evaluation is restricted to the dedicated local lab")
base = ["kubectl", "--context", args.context, "-n", "tsunami"]


def kubectl(*argv, data=None):
    return subprocess.check_output(base + list(argv), input=data, text=True)


control = json.loads(kubectl("get", "deployment", "tsunami-control", "-o", "json"))
env = {e["name"]: e.get("value", "") for e in control["spec"]["template"]["spec"]["containers"][0].get("env", [])}
provider = env.get("TSUNAMI_MODEL_PROVIDER")
if args.free_only and provider != "openrouter-free":
    sys.exit("Free-only evaluation refused: the configured provider is not openrouter-free")
if args.fixture:
    if provider != "fixture":
        sys.exit("Fixture-only evaluation refused: the configured provider is not fixture")
elif not args.allow_model_calls or provider not in ("responses", "openrouter-free"):
    sys.exit("Real evaluation requires configured inference and explicit --allow-model-calls")
template = json.loads(kubectl("create", "-f", "deploy/verify.yaml", "--dry-run=client", "-o", "json"))
template = with_image(template, deployed_image(kubectl))
template["metadata"]["generateName"] = "tsunami-model-eval-"
template["spec"]["activeDeadlineSeconds"] = 240
template["spec"]["template"]["spec"]["containers"][0]["args"] = ["live-stop-check" if args.stop_check else "live-eval", "--allow-model-calls"]
if args.free_only:
    template["spec"]["template"]["spec"]["containers"][0]["args"].append("--require-free")
job = json.loads(kubectl("create", "-f", "-", "-o", "json", data=json.dumps(template)))["metadata"]["name"]
deadline = time.monotonic() + 250
state = {}
while time.monotonic() < deadline:
    state = json.loads(kubectl("get", "job", job, "-o", "json")).get("status", {})
    if state.get("succeeded") or state.get("failed"):
        break
    time.sleep(1)
log = kubectl("logs", f"job/{job}")
print(log)
out = Path("artifacts/evaluations") / job
out.mkdir(parents=True, exist_ok=True)
(out / "verification.log").write_text(log)
ids = list(dict.fromkeys(re.findall(r"EVAL(?:_CREATED)? run=([0-9a-f]{32})", log)))
if ids:
    collected = subprocess.run(["go", "run", "./cmd/tsunami", "collect-k8s", "--context", args.context, "--execution", "live", "--runs", ",".join(ids), "--out", str(out)])
    if collected.returncode:
        sys.exit(f"Could not export run state; Job log and run IDs retained in {out}")
if not state.get("succeeded"):
    sys.exit(f"Evaluation failed or was inconclusive; available reports retained in {out}")
kind = "inference stop check" if args.stop_check else "paired evaluation"
print(f"PASS {kind}; reports in {out}. This guided lab is not a discovery benchmark.")

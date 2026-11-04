#!/usr/bin/env python3
"""Evaluate only closed synthetic procedures; defaults never call a model."""
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
parser.add_argument("--mode", choices=["scripted", "fixture", "free"], default="scripted")
parser.add_argument("--private", action="store_true")
parser.add_argument("--repeat", type=int, default=2)
parser.add_argument("--assessments", default="")
parser.add_argument("--allow-model-calls", action="store_true")
args = parser.parse_args()
if args.context != "kind-tsunami-lab" or args.repeat not in (1, 2, 3):
    sys.exit("Use the dedicated lab and 1-3 repetitions")
if args.private and args.mode != "scripted":
    sys.exit("Private catalog currently supports scripted evaluation only")
if args.mode == "free" and (not args.allow_model_calls or args.repeat != 1 or not args.assessments or len(args.assessments.split(",")) > 2):
    sys.exit("Free trial requires explicit model opt-in, one repeat, and at most two selected procedures")
base = ["kubectl", "--context", args.context, "-n", "tsunami"]


def kubectl(*argv, data=None):
    return subprocess.check_output(base+list(argv), input=data, text=True)


if args.mode != "scripted":
    control = json.loads(kubectl("get", "deployment", "tsunami-control", "-o", "json"))
    env = {e["name"]: e.get("value", "") for e in control["spec"]["template"]["spec"]["containers"][0]["env"]}
    expected = "fixture" if args.mode == "fixture" else "openrouter-free"
    if env.get("TSUNAMI_MODEL_PROVIDER") != expected:
        sys.exit("Configured provider differs from evaluation mode; no run created")
template = json.loads(kubectl("create", "--dry-run=client", "-f", "deploy/verify.yaml", "-o", "json"))
template = with_image(template, deployed_image(kubectl))
template["metadata"]["generateName"] = "tsunami-catalog-"
template["spec"]["activeDeadlineSeconds"] = 900
argv = ["catalog-eval", "--mode", args.mode, "--repeat", str(args.repeat)]
if args.private:
    argv += ["--connectivity", "private"]
if args.assessments:
    argv += ["--assessments", args.assessments]
if args.allow_model_calls:
    argv += ["--allow-model-calls"]
template["spec"]["template"]["spec"]["containers"][0]["args"] = argv
job = json.loads(kubectl("create", "-f", "-", "-o", "json", data=json.dumps(template)))["metadata"]["name"]
out = Path("artifacts/assessments")/job; out.mkdir(parents=True, exist_ok=True)
state = {}
for _ in range(910):
    state = json.loads(kubectl("get", "job", job, "-o", "json")).get("status", {})
    if state.get("succeeded") or state.get("failed"):
        break
    time.sleep(1)
log = kubectl("logs", "job/"+job)
(out/"verification.log").write_text(log)
for line in log.splitlines():
    if line.startswith("CATALOG_REPORT "):
        report = json.loads(line[len("CATALOG_REPORT "):])
        (out/"evaluation.json").write_text(json.dumps(report, indent=2)+"\n")
    else:
        print(line)
ids = list(dict.fromkeys(re.findall(r"CATALOG_CREATED run=([0-9a-f]{32})", log)))
if ids:
    subprocess.run(["go", "run", "./cmd/tsunami", "collect-k8s", "--context", args.context, "--execution", "all", "--runs", ",".join(ids), "--out", str(out)], check=True)
if not state.get("succeeded"):
    sys.exit(f"Catalog mismatches/failures retained in {out}")
print(f"PASS catalog evaluation; records in {out}. This is a guided synthetic suite, not a blind discovery benchmark.")

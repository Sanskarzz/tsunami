#!/usr/bin/env python3
"""Verify the actual runtime of a diagnostic pod on our dedicated Kind node."""
import json
from deploy_images import deployed_image, with_image
from datetime import datetime, timezone
import os
from pathlib import Path
import subprocess
import sys
import time

context = os.environ.get("CONTEXT", "kind-tsunami-lab")
if context != "kind-tsunami-lab":
    sys.exit("Runtime inspection is implemented only for the dedicated kind-tsunami-lab cluster")
base = ["kubectl", "--context", context, "-n", "tsunami"]


def kubectl(*args, data=None):
    return subprocess.check_output(base + list(args), input=data, text=True)


control = json.loads(kubectl("get", "deployment", "tsunami-control", "-o", "json"))
env = control["spec"]["template"]["spec"]["containers"][0].get("env", [])
if not any(e["name"] == "TSUNAMI_RUNTIME_CLASS" and e.get("value") == "gvisor" for e in env):
    sys.exit("The control plane is not configured to create gVisor workers")
template = json.loads(kubectl("create", "-f", "deploy/network-probe.yaml", "--dry-run=client", "-o", "json"))
template = with_image(template, deployed_image(kubectl))
template["metadata"]["generateName"] = "tsunami-runtime-probe-"
spec = template["spec"]["template"]["spec"]
spec["runtimeClassName"] = "gvisor"
spec["containers"][0]["args"] = ["runtime-probe"]
spec["containers"][0]["env"] = []
job = json.loads(kubectl("create", "-f", "-", "-o", "json", data=json.dumps(template)))["metadata"]["name"]
deadline = time.monotonic() + 90
pod = None
while time.monotonic() < deadline:
    pods = json.loads(kubectl("get", "pods", "-l", f"job-name={job}", "-o", "json"))["items"]
    if pods:
        pod = pods[0]
        phase = pod.get("status", {}).get("phase")
        if phase == "Failed":
            sys.exit(kubectl("describe", "pod", pod["metadata"]["name"]))
        if phase == "Running":
            break
    time.sleep(1)
if not pod or pod.get("status", {}).get("phase") != "Running":
    sys.exit("Runtime probe did not start; no runtime fallback is permitted")
node = pod["spec"]["nodeName"]
if node != "tsunami-lab-control-plane":
    sys.exit("Refusing runtime inspection of another node")
items = json.loads(subprocess.check_output(["docker", "exec", node, "crictl", "pods", "-o", "json"], text=True))["items"]
sandbox = next(p for p in items if p["metadata"]["uid"] == pod["metadata"]["uid"])
info = json.loads(subprocess.check_output(["docker", "exec", node, "crictl", "inspectp", sandbox["id"]], text=True))
handler = info.get("status", {}).get("runtimeHandler")
runtime_type = info.get("info", {}).get("runtimeType")
if handler != "runsc" or runtime_type != "io.containerd.runsc.v1":
    sys.exit(f"Unexpected runtime: handler={handler}, type={runtime_type}")
kubectl("wait", "--for=condition=complete", f"job/{job}", "--timeout=30s")
log = kubectl("logs", f"job/{job}")
print(log)
record = {"checked_at": datetime.now(timezone.utc).isoformat(),"context": context,"job": job,"node": node,"runtime_handler": handler,"runtime_type": runtime_type,"container_checks_passed": True}
out = Path("artifacts/isolation") / context
out.mkdir(parents=True, exist_ok=True)
(out / "runtime.json").write_text(json.dumps(record, indent=2)+"\n")
(out / "runtime.log").write_text(log)
print("PASS actual CRI runtime is runsc; this is a smoke test, not an escape-resistance proof")

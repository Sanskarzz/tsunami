#!/usr/bin/env python3
"""Credential-free curl bypass checks in the owned cluster; never an agent tool."""
import copy
import argparse
from datetime import datetime, timezone
import json
from deploy_images import deployed_tool_image
import os
from pathlib import Path
import subprocess
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", type=Path, help="new output directory; default is a timestamped artifact directory")
    args = parser.parse_args()
    context = os.environ.get("CONTEXT", "kind-tsunami-lab")
    if context != "kind-tsunami-lab":
        raise RuntimeError("use the dedicated kind-tsunami-lab context")
    base = ["kubectl", "--context", context, "-n", "tsunami"]
    def kube(*argv, data=None):
        return subprocess.check_output(base + list(argv), input=data, text=True, stderr=subprocess.DEVNULL)
    # Check infrastructure readiness before creating any diagnostic pod. The
    # pod's own first connection is still tested immediately, without a delay.
    kube("rollout", "status", "deployment/tsunami-control", "--timeout=90s")
    kube("rollout", "status", "deployment/tsunami-tripwire", "--timeout=40s")
    kube("rollout", "status", "deployment/application-tripwire", "-n", "tsunami-app-lab", "--timeout=40s")
    tool_image = deployed_tool_image(kube)
    services = json.loads(kube("get", "services", "-o", "json"))["items"]
    ips = {s["metadata"]["name"]: s["spec"]["clusterIP"] for s in services}
    tripwires = json.loads(kube("get", "pods", "-l", "tsunami-role=tripwire", "-o", "json"))["items"]
    ready = [p for p in tripwires if any(c["type"] == "Ready" and c["status"] == "True" for c in p["status"].get("conditions", []))]
    if len(ready) != 1:
        raise RuntimeError("one ready unrestricted-ingress tripwire required")
    node = json.loads(kube("get", "node", ready[0]["spec"]["nodeName"], "-o", "json"))
    node_ip = next(a["address"] for a in node["status"]["addresses"] if a["type"] == "InternalIP")
    api = kube("get", "service", "kubernetes", "-n", "default", "-o", "jsonpath={.spec.clusterIP}").strip()
    dns = kube("get", "service", "kube-dns", "-n", "kube-system", "-o", "jsonpath={.spec.clusterIP}").strip()
    app = kube("get", "service", "application-tripwire", "-n", "tsunami-app-lab", "-o", "jsonpath={.spec.clusterIP}").strip()
    destinations = {"tripwire-service": f'http://{ips["tsunami-tripwire"]}:8080',
                    "tripwire-pod": f'http://{ready[0]["status"]["podIP"]}:8080',
                    "api-service": f"https://{api}:443", "node-api": f"https://{node_ip}:6443",
                    "dns-tcp": f"http://{dns}:53", "application-tripwire": f"http://{app}:5000"}
    template = json.loads(kube("create", "--dry-run=client", "-f", "deploy/network-probe.yaml", "-o", "json"))
    out = args.out or Path("artifacts/tool-execution") / ("network-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ"))
    out.mkdir(parents=True, exist_ok=False, mode=0o700)
    report = {"checked_at": datetime.now(timezone.utc).isoformat(), "context": context,
              "destinations": destinations, "checks": [], "limitations": ["Selected IPv4/TCP paths only; UDP DNS is qualified by the separate Go diagnostic.", "Not arbitrary shell containment, IPv6, metadata or production capacity qualification."]}
    for role, positive in [("positive-before", True), ("runner", False), ("positive-after", True)]:
        manifest = copy.deepcopy(template)
        manifest["metadata"]["generateName"] = "tsunami-curl-" + role + "-"
        manifest["spec"]["activeDeadlineSeconds"] = 90
        pod = manifest["spec"]["template"]["spec"]
        # Same image, runtime and security settings on both sides; only the
        # policy identity differs. No run/operator/target credentials mounted.
        pod["runtimeClassName"] = "gvisor"
        manifest["spec"]["template"]["metadata"]["labels"]["tsunami-role"] = "verifier" if positive else "runner"
        container = pod["containers"][0]
        container["image"] = tool_image
        container["args"] = ["curl-network-probe"]
        container["resources"]["limits"] = {"cpu": "300m", "memory": "128Mi", "ephemeral-storage": "32Mi"}
        container["env"] = [{"name": "TSUNAMI_PROBE_BROKER", "value": f'http://{ips["tsunami-control"]}:8080'},
                            {"name": "TSUNAMI_PROBE_DESTINATIONS", "value": json.dumps(destinations)},
                            {"name": "TSUNAMI_PROBE_POSITIVE", "value": str(positive).lower()}]
        name = json.loads(kube("create", "-f", "-", "-o", "json", data=json.dumps(manifest)))["metadata"]["name"]
        state = {}
        runtime = None
        for _ in range(110):
            if runtime is None:
                live = json.loads(kube("get", "pods", "-l", "job-name=" + name, "-o", "json"))["items"]
                running = [p for p in live if p.get("status", {}).get("phase") == "Running"]
                if running:
                    pod = running[0]
                    node = pod["spec"]["nodeName"]
                    if node != "tsunami-lab-control-plane":
                        raise RuntimeError("refusing runtime inspection of another node")
                    sandboxes = json.loads(subprocess.check_output(["docker", "exec", node, "crictl", "pods", "-o", "json"], text=True))["items"]
                    sandbox = next(p for p in sandboxes if p["metadata"]["uid"] == pod["metadata"]["uid"])
                    info = json.loads(subprocess.check_output(["docker", "exec", node, "crictl", "inspectp", sandbox["id"]], text=True))
                    runtime = {"handler": info.get("status", {}).get("runtimeHandler"), "type": info.get("info", {}).get("runtimeType")}
                    if runtime != {"handler": "runsc", "type": "io.containerd.runsc.v1"}:
                        raise RuntimeError("unexpected actual runtime")
            state = json.loads(kube("get", "job", name, "-o", "json")).get("status", {})
            if state.get("succeeded") or state.get("failed"):
                break
            time.sleep(1)
        log = kube("logs", "job/" + name)
        (out / (role + ".log")).write_text(log)
        pods = json.loads(kube("get", "pods", "-l", "job-name=" + name, "-o", "json"))["items"]
        image = pods[0].get("status", {}).get("containerStatuses", [{}])[0].get("imageID", "") if pods else ""
        passed = bool(state.get("succeeded")) and runtime is not None
        report["checks"].append({"role": role, "job": name, "passed": passed, "runtime_class": "gvisor", "actual_runtime": runtime, "image_id": image})
        (out / "network.json").write_text(json.dumps(report, indent=2) + "\n")
        print(f"{role}: {'PASS' if passed else 'FAIL'}; {name}", flush=True)
        if not passed:
            raise RuntimeError("curl boundary unqualified; records retained at " + str(out))
    print("PASS selected curl bypass checks; records in " + str(out))


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, subprocess.SubprocessError) as error:
        raise SystemExit(str(error) if isinstance(error, RuntimeError) else "Curl diagnostics failed; inspect retained records.")

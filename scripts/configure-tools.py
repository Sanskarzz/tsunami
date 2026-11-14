#!/usr/bin/env python3
"""Enable the single reviewed adapter only in the owned gVisor lab."""
import sys
if __name__ == "__main__":
    sys.dont_write_bytecode = True

import argparse
from deployment_guard import require_secured_lab
import json
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--context", default="kind-tsunami-lab")
    parser.add_argument("--tool-image", default="tsunami-tools:dev")
    parser.add_argument("--native-image", help="already built/loaded native image; preserve other configuration")
    args = parser.parse_args()
    require_secured_lab(args.context)
    if not args.tool_image or any(c.isspace() for c in args.tool_image):
        parser.error("explicit tool image without whitespace required")
    if args.native_image is not None and (not args.native_image or any(c.isspace() for c in args.native_image)):
        parser.error("explicit native image without whitespace required")
    if args.context != "kind-tsunami-lab":
        parser.error("use the dedicated kind-tsunami-lab context")
    base = ["kubectl", "--context", args.context, "-n", "tsunami"]
    def kube(*argv, data=None):
        return subprocess.check_output(base + list(argv), input=data, text=True, stderr=subprocess.DEVNULL)
    kube("get", "runtimeclass", "gvisor")
    active = json.loads(kube("get", "pods", "-l", "tsunami-role=runner", "--field-selector=status.phase=Running", "-o", "json"))["items"]
    if active:
        raise RuntimeError("finish active worker runs before changing the executor")
    control = json.loads(kube("get", "deployment", "tsunami-control", "-o", "json"))
    container = control["spec"]["template"]["spec"]["containers"][0]
    env = container.get("env", [])
    if not any(e["name"] == "TSUNAMI_RUNTIME_CLASS" and e.get("value") == "gvisor" for e in env):
        raise RuntimeError("the control plane must already select gvisor")
    env = [e for e in env if e["name"] not in ("TSUNAMI_TOOL_IMAGE", "TSUNAMI_ENABLE_CURL_ADAPTER") and not (args.native_image and e["name"] == "TSUNAMI_IMAGE")]
    if args.native_image:
        env.append({"name": "TSUNAMI_IMAGE", "value": args.native_image})
    env += [{"name": "TSUNAMI_TOOL_IMAGE", "value": args.tool_image}, {"name": "TSUNAMI_ENABLE_CURL_ADAPTER", "value": "true"}]
    patch = [{"op": "test", "path": "/metadata/resourceVersion", "value": control["metadata"]["resourceVersion"]},
             {"op": "add", "path": "/spec/template/spec/containers/0/env", "value": env}]
    if args.native_image:
        patch.append({"op": "replace", "path": "/spec/template/spec/containers/0/image", "value": args.native_image})
    kube("patch", "deployment", "tsunami-control", "--type=json", "-p", json.dumps(patch))
    kube("rollout", "restart", "deployment/tsunami-control")
    kube("rollout", "status", "deployment/tsunami-control", "--timeout=90s")
    print("Reviewed curl-http/v1 adapter enabled in owned lab; qualification is separate. Other configuration preserved.")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, subprocess.SubprocessError):
        raise SystemExit("Tool configuration failed; inspect the deployment (credential output suppressed).")

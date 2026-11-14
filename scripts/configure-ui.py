#!/usr/bin/env python3
"""Build and enable the local-only UI in the existing idle owned Kind lab."""
import sys
if __name__ == "__main__":
    sys.dont_write_bytecode = True

from datetime import datetime, timezone
import base64
import json
import os
import ipaddress
from pathlib import Path
import subprocess

from showcase import Lab
from deployment_guard import require_secured_lab


def main():
    require_secured_lab()
    lab = Lab(False, 900)
    lab.foundation()
    lab.permission_idle(required=0)
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    out = Path("artifacts/ui") / ("deploy-" + stamp)
    out.mkdir(parents=True, mode=0o700, exist_ok=False)
    image = "tsunami:ui-" + stamp.lower()
    tool_image="tsunami-tools:ui-"+stamp.lower()
    print("Building local UI; no target reset, model call, new cluster or public exposure.", flush=True)
    lab.child(["make", "image", "tools-image", "IMAGE=" + image,"TOOL_IMAGE="+tool_image], out / "build.log")
    lab.child(["kind", "load", "docker-image", image,tool_image, "--name", "tsunami-lab"], out / "load.log")
    # Check again after building. Single-operator maintenance is still required;
    # this does not claim a distributed admission/maintenance lock.
    lab.idle()
    lab.permission_idle(required=0)
    # This durable key is generated once, outside the state database, and is
    # never logged or put in an image/command argument. Do not regenerate it on
    # restart: existing encrypted setup access/evidence depends on it.
    result=subprocess.run(lab.base+["-n","tsunami","get","secret","tsunami-onboarding","--ignore-not-found","-o","json"],env=lab.env,capture_output=True,text=True)
    if result.returncode: raise RuntimeError("cannot inspect owned onboarding key")
    if result.stdout.strip():
        existing=json.loads(result.stdout)
        if len(base64.b64decode(existing["data"]["encryption-key"],validate=True))!=32: raise RuntimeError("existing onboarding key is invalid; refusing to replace it")
    else:
        prior=lab.obj("tsunami","deployment","tsunami-control")
        if any(e["name"]=="TSUNAMI_ONBOARDING_KEY_FILE" for c in prior["spec"]["template"]["spec"]["containers"] for e in c.get("env",[])):
            raise RuntimeError("previous onboarding key is missing; restore it instead of generating a replacement")
        secret={"apiVersion":"v1","kind":"Secret","metadata":{"name":"tsunami-onboarding","namespace":"tsunami"},"immutable":True,"type":"Opaque","data":{"encryption-key":base64.b64encode(os.urandom(32)).decode()}}
        result=subprocess.run(lab.base+["create","-f","-"],env=lab.env,input=json.dumps(secret),capture_output=True,text=True)
        if result.returncode: raise RuntimeError("could not provision owned onboarding key")
    control = lab.obj("tsunami", "deployment", "tsunami-control")
    containers = control["spec"]["template"]["spec"]["containers"]
    index = next(i for i, c in enumerate(containers) if c["name"] == "control")
    container = containers[index]
    denied=set(next((e.get("value","").split(",") for e in container.get("env",[]) if e["name"]=="TSUNAMI_ONBOARDING_DENIED_CIDRS"),[]))
    nodes=json.loads(lab.kube("get","nodes","-o","json"))["items"]
    for node in nodes:
        for address in node["status"].get("addresses",[]):
            if address["type"] in ("InternalIP","ExternalIP"):
                ip=ipaddress.ip_address(address["address"]);denied.add(str(ip)+("/32" if ip.version==4 else "/128"))
    api_ip=lab.obj("default","service","kubernetes")["spec"]["clusterIP"]
    api_address=ipaddress.ip_address(api_ip)
    denied.add(str(api_address)+("/32" if api_address.version==4 else "/128"))
    denied.discard("")
    env = [e for e in container.get("env", []) if e["name"] not in ("TSUNAMI_ENABLE_UI", "TSUNAMI_IMAGE","TSUNAMI_TOOL_IMAGE","TSUNAMI_ONBOARDING_KEY_FILE","TSUNAMI_ONBOARDING_DENIED_CIDRS")]
    env += [{"name": "TSUNAMI_ENABLE_UI", "value": "true"}, {"name": "TSUNAMI_IMAGE", "value": image},{"name":"TSUNAMI_TOOL_IMAGE","value":tool_image},{"name":"TSUNAMI_ONBOARDING_KEY_FILE","value":"/run/onboarding/encryption-key"}]
    env.append({"name":"TSUNAMI_ONBOARDING_DENIED_CIDRS","value":",".join(sorted(denied))})
    mounts=[m for m in container.get("volumeMounts",[]) if m["name"]!="onboarding-key"]
    mounts.append({"name":"onboarding-key","mountPath":"/run/onboarding","readOnly":True})
    volumes=[v for v in control["spec"]["template"]["spec"].get("volumes",[]) if v["name"]!="onboarding-key"]
    volumes.append({"name":"onboarding-key","secret":{"secretName":"tsunami-onboarding","defaultMode":292}})
    prefix = f"/spec/template/spec/containers/{index}"
    patch = [{"op": "test", "path": "/metadata/resourceVersion", "value": control["metadata"]["resourceVersion"]},
             {"op": "replace", "path": prefix + "/image", "value": image},
             {"op": "add", "path": prefix + "/env", "value": env},
             {"op":"add","path":prefix+"/volumeMounts","value":mounts},
             {"op":"add","path":"/spec/template/spec/volumes","value":volumes}]
    # Patch is submitted on stdin, not a command argument that might contain
    # unrelated configuration. The control Secret data is never read here.
    lab.child(["kubectl", "--context", "kind-tsunami-lab", "-n", "tsunami", "rollout", "status", "deployment/tsunami-control", "--timeout=90s"], out / "before-rollout.log")
    result = subprocess.run(lab.base + ["-n", "tsunami", "patch", "deployment", "tsunami-control", "--type=json", "--patch-file=/dev/stdin"],
                            input=json.dumps(patch), text=True, env=lab.env, capture_output=True)
    if result.returncode:
        raise RuntimeError("UI deployment patch failed; inspect the owned deployment (configuration output suppressed)")
    lab.child(["kubectl", "--context", "kind-tsunami-lab", "-n", "tsunami", "rollout", "status", "deployment/tsunami-control", "--timeout=90s"], out / "rollout.log")
    with lab.forward() as origin:
        with lab.opener.open(origin + "/ui/", timeout=5) as response:
            if response.status != 200 or b"Local assessment workspace" not in response.read(128 << 10):
                raise RuntimeError("UI readiness check failed")
    (out / "deployment.json").write_text(json.dumps({"image": image,"tools_image":tool_image, "context": "kind-tsunami-lab", "ui": "/ui/", "target_requests": 0, "external_model_calls": 0,"onboarding_key":"existing_or_new_immutable_secret; never included"}, indent=2) + "\n")
    print("PASS local UI enabled. Use a loopback service port-forward and open http://127.0.0.1:8080/ui/")
    print("Deployment metadata: " + str(out))


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, ValueError, StopIteration, subprocess.SubprocessError):
        raise SystemExit("Local UI deployment failed; inspect owned lab state and private logs. No automatic retry or target reset.")

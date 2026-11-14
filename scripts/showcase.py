#!/usr/bin/env python3
"""Owned-lab showcase: inspect, explicitly refresh disposable data, then rehearse."""
import sys
if __name__ == "__main__":
    sys.dont_write_bytecode = True

import argparse
import base64
from contextlib import contextmanager
from datetime import datetime, timezone
import hashlib
import json
import math
import os
from pathlib import Path
import shutil
import socket
import subprocess
import time
import urllib.request
import urllib.error

from deploy_images import image_from_control
from deployment_guard import require_secured_lab


CONTEXT = "kind-tsunami-lab"
CORE = {
    "vampi-vulnerable": ("tsunami-app-lab", "vampi-vulnerable", 5000),
    "vampi-fixed": ("tsunami-app-lab", "vampi-fixed", 5000),
    "stop-lab-run": ("tsunami-stop-lab", "tsunami-stop-fixture", 8080),
}
PROPERTY = {"property-" + name: ("tsunami-property-lab", "tsunami-property-fixture", 8080) for name in ("vulnerable", "filtered", "ambiguous", "stop")}


def expiry_hint(token, now, minimum):
    """JWT claims are an offline expiry hint, never signature/auth verification."""
    try:
        if not isinstance(token, str) or len(token) > 4096:
            raise ValueError()
        parts = token.split(".")
        if len(parts) != 3:
            raise ValueError()
        payload = json.loads(base64.urlsafe_b64decode(parts[1] + "=" * (-len(parts[1]) % 4)))
        expires = payload["exp"]
        if type(expires) not in (int, float) or not -1e12 < expires < 1e12 or not math.isfinite(expires):
            raise ValueError()
        remaining = int(expires - now)
    except (ValueError, TypeError, KeyError, UnicodeError):
        raise RuntimeError("VAmPI expiry metadata unreadable; refresh the owned setup") from None
    if remaining < minimum:
        raise RuntimeError("VAmPI credentials expired or too near expiry; use showcase-prepare")
    return {"remaining_seconds": remaining, "scope": "unsigned_expiry_hint_not_account_validation"}


def credential_fingerprint(tenant, identifier, credentials):
    # Go registration marshals these exact fields in this order. The owned
    # fixture credentials are ASCII; no token values enter the public report.
    data = json.dumps({"Tenant": tenant, "ID": identifier, "Credentials": dict(sorted(credentials.items()))}, separators=(",", ":"), ensure_ascii=False).encode()
    return hashlib.sha256(data).hexdigest()


def owned_scope(plan, namespace, service, port, address):
    expected = f"http://{service}.{namespace}.svc.cluster.local:{port}"
    if plan.get("origin") != expected or plan.get("approved_addresses") != [address] or plan.get("synthetic_data_only") is not True:
        raise RuntimeError("selected application differs from the fixed owned-lab target; refusing showcase")
    if plan.get("version") not in ("http-object-access/v2", "http-property-access/v1"):
        raise RuntimeError("unsupported owned application profile")


def active_test_pod(pod):
    if pod.get("status", {}).get("phase") not in ("Pending", "Running"):
        return False
    role = pod.get("metadata", {}).get("labels", {}).get("tsunami-role")
    owners = pod.get("metadata", {}).get("ownerReferences", [])
    # Permanent trusted services such as the offline model fixture also use
    # verifier identity. Only verifier Jobs represent active diagnostics.
    return role == "runner" or (role == "verifier" and any(o.get("kind") == "Job" for o in owners))


def blocked_capture_ids(bundle):
    # Ordinary successful captures omit the false boolean in Go JSON.
    return [a["capture_id"] for a in bundle["artifacts"]
            if json.loads(base64.b64decode(a["data_base64"], validate=True))["capture"].get("blocked", False)]


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        return None


class Lab:
    def __init__(self, include_property, minimum):
        self.include_property, self.minimum = include_property, minimum
        self.env = dict(os.environ, KUBECONFIG=str(Path(".local/sandbox.kubeconfig").resolve()), CONTEXT=CONTEXT)
        if not Path(self.env["KUBECONFIG"]).is_file():
            raise RuntimeError("dedicated sandbox kubeconfig missing; provision the owned sandbox first")
        for program in ("kubectl", "docker", "kind", "go", "make"):
            if not shutil.which(program):
                raise RuntimeError("missing local prerequisite: " + program)
        self.base = ["kubectl", "--context", CONTEXT]
        self.opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())

    def kube(self, *args):
        try:
            return subprocess.check_output(self.base + list(args), env=self.env, text=True, stderr=subprocess.DEVNULL, timeout=15)
        except subprocess.SubprocessError:
            raise RuntimeError("owned Kubernetes resource unavailable; inspect the dedicated lab (response suppressed)") from None

    def obj(self, namespace, kind, name):
        return json.loads(self.kube("-n", namespace, "get", kind, name, "-o", "json"))

    def idle(self):
        pods = json.loads(self.kube("-n", "tsunami", "get", "pods", "-l", "tsunami-role in (runner,verifier)", "-o", "json"))["items"]
        if any(active_test_pod(p) for p in pods):
            raise RuntimeError("active/pending workers or diagnostics exist; finish or explicitly Stop them before showcase maintenance")

    def foundation(self):
        subprocess.run(["docker", "info", "--format", "{{.ServerVersion}}"], capture_output=True, env=self.env, check=True, timeout=15)
        self.idle()
        nodes = json.loads(self.kube("get", "nodes", "-o", "json"))["items"]
        if len(nodes) != 1 or nodes[0]["metadata"]["name"] != "tsunami-lab-control-plane":
            raise RuntimeError("refusing showcase outside the dedicated Kind node")
        marker = subprocess.check_output(["docker", "inspect", "--format", '{{index .Config.Labels "io.x-k8s.kind.cluster"}}', "tsunami-lab-control-plane"], env=self.env, text=True, stderr=subprocess.DEVNULL, timeout=15).strip()
        if marker != "tsunami-lab":
            raise RuntimeError("dedicated Kind ownership marker missing")
        runsc_version = subprocess.check_output(["docker", "exec", "tsunami-lab-control-plane", "/usr/local/bin/runsc", "--version"], env=self.env, text=True, stderr=subprocess.DEVNULL, timeout=15).strip()
        runtime = json.loads(self.kube("get", "runtimeclass", "gvisor", "-o", "json"))
        if runtime.get("handler") != "runsc":
            raise RuntimeError("gVisor RuntimeClass is not bound to runsc")
        cilium = self.obj("kube-system", "daemonset", "cilium")
        desired = cilium.get("status", {}).get("desiredNumberScheduled", 0)
        if desired != 1 or cilium["status"].get("numberReady") != desired:
            raise RuntimeError("this showcase requires the ready dedicated single-node Cilium lab")
        config = self.obj("kube-system", "configmap", "cilium-config")
        if config["data"].get("enable-policy") != "always":
            raise RuntimeError("Cilium always-enforcement prerequisite missing")
        control = self.obj("tsunami", "deployment", "tsunami-control")
        if control.get("status", {}).get("readyReplicas") != 1:
            raise RuntimeError("control deployment is not ready")
        image = image_from_control(control)
        env = {e["name"]: e.get("value") for e in control["spec"]["template"]["spec"]["containers"][0].get("env", [])}
        if env.get("TSUNAMI_RUNTIME_CLASS") != "gvisor" or env.get("TSUNAMI_ENABLE_CURL_ADAPTER") != "true" or not env.get("TSUNAMI_TOOL_IMAGE"):
            raise RuntimeError("configured gVisor/curl execution path required")
        return {"context": CONTEXT, "node_count": desired, "kubelet_version": nodes[0]["status"]["nodeInfo"]["kubeletVersion"], "runtime_class": "gvisor", "runtime_handler_configured": "runsc", "runsc_version": runsc_version, "cilium_policy": "always", "cilium_image": cilium["spec"]["template"]["spec"]["containers"][0]["image"], "native_image": image, "tools_image": env["TSUNAMI_TOOL_IMAGE"]}

    def permission_idle(self, required=None):
        if required is None:
            required = 7 if self.include_property else 3
        token = base64.b64decode(self.obj("tsunami", "secret", "tsunami-operator")["data"]["admin-a"]).decode()
        with self.forward() as origin:
            engagements = self.read(origin, "/v1/engagements", token)
            try:
                setups = self.read(origin, "/v1/onboarding", token)
            except urllib.error.HTTPError as error:
                if error.code != 404:
                    raise RuntimeError("cannot inspect setup permissions before maintenance") from None
                setups = []  # Compatibility with builds predating onboarding.
            if any(item["state"] == "validating" for item in setups):
                raise RuntimeError("active setup validation exists; inspect and explicitly Stop it before maintenance")
            if len(engagements) + required > 100:
                raise RuntimeError("lab history capacity insufficient; preserve/archive state through an explicit maintenance plan")
            cursor = ""
            while True:
                page = self.read(origin, "/v1/runs?limit=100" + ("&cursor=" + cursor if cursor else ""), token)
                if any(r["state"] in ("pending", "running") for r in page["items"]):
                    raise RuntimeError("nonterminal run permissions exist; inspect and explicitly Stop them before maintenance")
                cursor = page.get("next_cursor")
                if not cursor:
                    return

    @contextmanager
    def forward(self):
        sock = socket.socket(); sock.bind(("127.0.0.1", 0)); port = sock.getsockname()[1]; sock.close()
        proc = subprocess.Popen(self.base + ["-n", "tsunami", "port-forward", "--address=127.0.0.1", "service/tsunami-control", f"{port}:8080"], env=self.env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        origin = f"http://127.0.0.1:{port}"
        try:
            end = time.monotonic() + 10
            while True:
                try:
                    self.read(origin, "/readyz")
                    break
                except OSError:
                    if time.monotonic() > end or proc.poll() is not None:
                        raise RuntimeError("state/revocation API unavailable through loopback port-forward") from None
                    time.sleep(0.1)
            yield origin
        finally:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill(); proc.wait()

    def read(self, origin, path, token=None):
        req = urllib.request.Request(origin + path)
        if token:
            req.add_header("Authorization", "Bearer " + token)
        with self.opener.open(req, timeout=5) as response:
            data = response.read((2 << 20) + 1)
        if len(data) > 2 << 20:
            raise RuntimeError("diagnostic response exceeds limit")
        return json.loads(data)

    def check(self):
        report = self.foundation()
        if not Path("bin/tsunami").is_file():
            raise RuntimeError("build the local CLI or use showcase-prepare")
        secret = self.obj("tsunami", "secret", "tsunami-operator")
        token = base64.b64decode(secret["data"]["admin-a"]).decode()
        registry = json.loads(base64.b64decode(self.obj("tsunami", "secret", "tsunami-applications")["data"]["config.json"]))
        configs = {c["plan"]["id"]: c for c in registry if c["tenant"] == "demo-a"}
        selected = dict(CORE, **PROPERTY) if self.include_property else CORE
        report.update({"checked_at": datetime.now(timezone.utc).isoformat(), "target_requests": 0, "external_model_calls": 0, "applications": [], "warnings": []})
        with self.forward() as origin:
            status = self.read(origin, "/v1/status", token)
            for route in ("scheduler", "applications", "curl"):
                if status["execution_routes"].get(route, {}).get("state") != "ready":
                    raise RuntimeError("required execution route unavailable: " + route)
            for route in ("model", "private"):
                if status["execution_routes"].get(route, {}).get("state") != "ready":
                    report["warnings"].append(route + " route not ready; not used by this scripted public-lab showcase")
            plans = {p["id"]: p for p in self.read(origin, "/v1/applications", token)}
            engagements = self.read(origin, "/v1/engagements", token)
            required = 7 if self.include_property else 3
            report["remaining_lab_engagement_slots"] = 100 - len(engagements)
            if report["remaining_lab_engagement_slots"] < required:
                raise RuntimeError("lab history capacity insufficient; archive/restore through an explicit maintenance plan, never silently delete history")
            cursor = ""
            while True:
                page = self.read(origin, "/v1/runs?limit=100" + ("&cursor=" + cursor if cursor else ""), token)
                if any(r["state"] in ("pending", "running") for r in page["items"]):
                    raise RuntimeError("nonterminal run permissions exist; inspect and explicitly Stop them before showcase")
                cursor = page.get("next_cursor")
                if not cursor:
                    break
            for identifier, (namespace, service, port) in selected.items():
                if identifier not in configs or identifier not in plans:
                    raise RuntimeError("owned application missing; use showcase-prepare")
                address = self.obj(namespace, "service", service)["spec"]["clusterIP"]
                owned_scope(plans[identifier], namespace, service, port, address)
                deployment = self.obj(namespace, "deployment", service)
                if deployment.get("status", {}).get("readyReplicas") != 1:
                    raise RuntimeError("owned target deployment is not ready")
                config = configs[identifier]
                if credential_fingerprint(config["tenant"], identifier, config["credentials"]) != plans[identifier]["credential_set_id"]:
                    raise RuntimeError("mounted registry is stale or credentials differ; refresh before rehearsal")
                item = {"application": identifier, "owned_service": service, "origin": plans[identifier]["origin"]}
                if identifier.startswith("vampi-"):
                    item["credential_expiry_hints"] = [expiry_hint(c, time.time(), self.minimum) for c in config["credentials"].values()]
                else:
                    item["credential_expiry_scope"] = "opaque_lab_credentials_no_local_expiry_hint"
                report["applications"].append(item)
        return report

    def child(self, argv, logfile):
        with Path(logfile).open("w") as output:
            result = subprocess.run(argv, env=self.env, text=True, stdout=output, stderr=subprocess.STDOUT)
        if result.returncode:
            raise RuntimeError("showcase step failed; inspect private local log " + str(logfile))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["check", "prepare", "run"])
    parser.add_argument("--include-property", action="store_true", help="include all four field-level controls")
    parser.add_argument("--minimum-ttl", type=int, default=900, help="minimum hinted VAmPI lifetime in seconds")
    parser.add_argument("--toolkit", type=Path, help="optional local schema checkout; never downloaded or executed")
    args = parser.parse_args()
    if not 120 <= args.minimum_ttl <= 3600:
        parser.error("minimum TTL must be 120–3600 seconds")
    if args.action != "check":
        require_secured_lab()
    lab = Lab(args.include_property, args.minimum_ttl)
    if args.action == "check":
        report = lab.check()
        print(json.dumps(report, indent=2))
        print("PASS prerequisites; no target requests. JWT expiry is only a hint; execution still validates identities.")
        return
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    out = Path("artifacts/showcase") / (args.action + "-" + stamp)
    out.mkdir(parents=True, exist_ok=False, mode=0o700)
    if args.action == "prepare":
        lab.foundation()
        lab.permission_idle()
        # Unique tags avoid cached mutable-image ambiguity in Kind.
        native, tools = "tsunami:showcase-" + stamp.lower(), "tsunami-tools:showcase-" + stamp.lower()
        print("Refreshing only owned disposable lab data; existing run/evidence history and operator/model/private configuration are preserved.", flush=True)
        lab.child(["make", "image", "tools-image", "IMAGE=" + native, "TOOL_IMAGE=" + tools], out / "build.log")
        lab.child(["kind", "load", "docker-image", native, tools, "--name", "tsunami-lab"], out / "load.log")
        lab.child(["python3", "scripts/configure-tools.py", "--native-image", native, "--tool-image", tools], out / "deployment.log")
        lab.child(["python3", "scripts/setup-vampi.py"], out / "vampi-setup.log")
        lab.child(["python3", "scripts/setup-stop-lab.py"], out / "stop-setup.log")
        if args.include_property:
            lab.child(["python3", "scripts/setup-property-lab.py"], out / "property-setup.log")
        report = lab.check()
        (out / "prerequisites.json").write_text(json.dumps(report, indent=2) + "\n")
        print("PASS owned preparation; credentials refreshed, scope approval still required. Records: " + str(out))
        return
    prerequisite = lab.check()
    (out / "prerequisites.json").write_text(json.dumps(prerequisite, indent=2) + "\n")
    print("AUTOMATED OWNED-LAB APPROVAL: scripted demonstration only; no human finding review or customer authorization is implied.", flush=True)
    report = {"version": "tsunami.showcase/v1", "checked_at": datetime.now(timezone.utc).isoformat(), "mode": "scripted_mecatl", "approval": "automated_owned_lab", "external_model_calls": 0, "passed": False, "cases": [], "limitations": ["Owned deliberately vulnerable/fixed applications with supplied facts; no autonomous discovery or customer readiness.", "Selected single-node IPv4/TCP tests; no 1,000-agent or production Stop guarantee.", "Evidence hashes identify bytes, not authenticity against a compromised trusted service."]}
    summary_path = out / "showcase.json"
    def save():
        summary_path.write_text(json.dumps(report, indent=2) + "\n")
    save()
    try:
        report["phase"] = "network"
        save()
        lab.child(["python3", "scripts/tool-network-check.py", "--out", str(out / "network")], out / "network-step.log")
        for family in (["object", "property"] if args.include_property else ["object"]):
            report["phase"] = family
            save()
            command = ["python3", "scripts/verify-journey.py", "--profile", family, "--out", str(out / family)]
            if args.toolkit:
                command += ["--toolkit", str(args.toolkit)]
            lab.child(command, out / (family + "-step.log"))
            campaign = json.loads((out / family / "summary.json").read_text())
            report["binary_sha256"] = campaign["binary_sha256"]
            report["control_image_id"] = campaign["control_image_id"]
            for case in campaign["checks"]:
                bundle_path = out / family / case["application"] / "bundle.json"
                bundle = json.loads(bundle_path.read_text())
                blocked = blocked_capture_ids(bundle)
                if not case["passed"] or not blocked or bundle["toolkit_findings"]:
                    raise RuntimeError("showcase case lacks expected controls or contains an invented review")
                report["cases"].append({"application": case["application"], "engagement_id": case["engagement_id"], "run_id": case["run_id"], "verdict": case["result_status"].get("assessment", {}).get("verdict", "stopped"), "finding_candidates": case["candidates"], "reviewed_findings": 0, "blocked_capture_ids": blocked, "bundle": str(bundle_path.relative_to(out)), "report": str((bundle_path.parent / "report-draft.md").relative_to(out)), "cleanup": case["after_cleanup"]["cleanup"]})
            save()
        report["network_record"] = "network/network.json"
        lines = ["# Owned-lab showcase results", "", "Scripted Mecatl; automated lab approvals; no human finding review.", "", "| Application | Result | Candidates | Scope denials | Report |", "|---|---|---:|---:|---|"]
        for c in report["cases"]:
            lines.append(f"| {c['application']} | {c['verdict']} | {c['finding_candidates']} | {len(c['blocked_capture_ids'])} | [evidence]({c['report']}) |")
        lines += ["", "See [network checks](network/network.json), [prerequisites](prerequisites.json), and [full index](showcase.json).", "", "Controlled examples only; no production, capacity or autonomous-discovery claim."]
        (out / "index.md").write_text("\n".join(lines) + "\n")
        report["passed"] = True
        report["phase"] = "complete"
        save()
        print("PASS showcase rehearsal. Open " + str(out / "index.md"), flush=True)
    finally:
        save()


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, ValueError, KeyError, StopIteration, subprocess.SubprocessError) as error:
        raise SystemExit(str(error) if isinstance(error, RuntimeError) else "Showcase failed; inspect local records (credential output suppressed).")

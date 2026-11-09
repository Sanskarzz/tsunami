#!/usr/bin/env python3
"""Observe real gVisor/curl Stop, controller API denial and durable revocation."""
import argparse
import atexit
import base64
from contextlib import contextmanager, ExitStack
from datetime import datetime, timezone
import json
from pathlib import Path
import socket
import subprocess
import time
import urllib.error
import urllib.request


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        return None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--context", default="kind-tsunami-lab")
    parser.add_argument("--cases", default="run,engagement,api-outage,restart")
    args = parser.parse_args()
    cases = args.cases.split(",")
    if args.context != "kind-tsunami-lab" or not cases or any(c not in {"run", "engagement", "api-outage", "restart"} for c in cases) or len(set(cases)) != len(cases):
        parser.error("use the owned Kind lab and distinct supported cases")
    base = ["kubectl", "--context", args.context]
    def kube(*argv, data=None):
        return subprocess.check_output(base + list(argv), input=data, text=True, stderr=subprocess.DEVNULL)
    out = Path("artifacts/stop-lifecycle") / datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    out.mkdir(parents=True, exist_ok=False, mode=0o700)
    token = base64.b64decode(kube("-n", "tsunami", "get", "secret", "tsunami-operator", "-o", "jsonpath={.data.admin-a}")).decode()
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    def api(origin, path, method="GET", body=None, authenticated=False, expected=200):
        data = json.dumps(body).encode() if body is not None else None
        headers = {"Content-Type": "application/json"}
        if authenticated:
            headers["Authorization"] = "Bearer " + token
        request = urllib.request.Request(origin + path, data=data, method=method, headers=headers)
        try:
            with opener.open(request, timeout=5) as response:
                if response.status != expected:
                    raise RuntimeError("unexpected API status")
                raw = response.read((2 << 20) + 1)
                if len(raw) > 2 << 20:
                    raise RuntimeError("unexpectedly large diagnostic response")
                return json.loads(raw) if raw else None
        except urllib.error.HTTPError as error:
            if error.code == expected:
                return None
            raise RuntimeError("diagnostic API returned status " + str(error.code)) from None
    def wait(predicate, seconds=30):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            result = predicate()
            if result:
                return result
            time.sleep(0.1)
        raise RuntimeError("timed out waiting for required observation")
    @contextmanager
    def forward(namespace, service):
        sock = socket.socket(); sock.bind(("127.0.0.1", 0)); port = sock.getsockname()[1]; sock.close()
        proc = subprocess.Popen(base + ["-n", namespace, "port-forward", "--address=127.0.0.1", "service/" + service, f"{port}:8080"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        origin = f"http://127.0.0.1:{port}"
        try:
            deadline = time.monotonic() + 10
            while True:
                if proc.poll() is not None:
                    raise RuntimeError("diagnostic port-forward ended")
                try:
                    api(origin, "/healthz")
                    break
                except (OSError, RuntimeError):
                    if time.monotonic() >= deadline:
                        raise RuntimeError("diagnostic port-forward not ready")
                    time.sleep(0.1)
            yield origin
        finally:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill(); proc.wait()
    policy_name = "tsunami-stop-api-outage"
    fault = {"apiVersion": "cilium.io/v2", "kind": "CiliumNetworkPolicy", "metadata": {"name": policy_name, "namespace": "tsunami"}, "spec": {
        "endpointSelector": {"matchLabels": {"tsunami-role": "control"}},
        "egressDeny": [{"toEntities": ["kube-apiserver", "host", "remote-node"], "toPorts": [{"ports": [{"port": "443", "protocol": "TCP"}, {"port": "6443", "protocol": "TCP"}]}]}]}}
    # Never overwrite a fault policy owned by another diagnostic invocation.
    existing = kube("-n", "tsunami", "get", "ciliumnetworkpolicy", policy_name, "--ignore-not-found", "-o", "name").strip()
    if existing:
        raise RuntimeError("an existing API fault must be investigated before this campaign")
    report = {"checked_at": datetime.now(timezone.utc).isoformat(), "context": args.context, "external_model_calls": 0, "checks": [], "limitations": [
        "One owned Kind node; no arbitrary node partition, production load or hostile sandbox qualification.",
        "Core readiness is state/revocation readiness. Route status describes startup configuration, not live provider connectivity.",
        "API resource absence is not proof of process exit on a partitioned node. CRI observations are separately recorded.",
        "Times are local observations, not a production 60-second guarantee."]}
    def save():
        (out / "summary.json").write_text(json.dumps(report, indent=2) + "\n")
    atexit.register(save)
    def stop_parent(parent, entry):
        # Also revoke after a failed precondition, before leaving owned work
        # running until its normal deadline. Never hide the original failure.
        try:
            with forward("tsunami", "tsunami-control") as control:
                api(control, parent + "/stop", "POST", {}, True)
            entry["final_parent_revocation"] = "acknowledged"
        except (OSError, RuntimeError, subprocess.SubprocessError):
            entry["final_parent_revocation"] = "unavailable_requires_operator_check"
        save()
    save()
    with forward("tsunami-stop-lab", "tsunami-stop-fixture") as target:
        for case in cases:
            entry = {"case": case, "passed": False}
            report["checks"].append(entry); save()
            fault_installed = False
            with ExitStack() as cleanup, forward("tsunami", "tsunami-control") as control:
                health = api(control, "/v1/status", authenticated=True)
                entry["startup_routes"] = health["execution_routes"]
                controls = json.loads(kube("-n", "tsunami", "get", "pods", "-l", "tsunami-role=control", "-o", "json"))["items"]
                entry["control_image_id"] = next(p["status"]["containerStatuses"][0]["imageID"] for p in controls if not p["metadata"].get("deletionTimestamp") and p.get("status", {}).get("phase") == "Running")
                plans = api(control, "/v1/applications", authenticated=True)
                plan = next(p for p in plans if p["id"] == "stop-lab-" + case)
                before = api(target, "/cases/" + case + "/stats")
                entry["target_before"] = before
                engagement = api(control, "/v1/engagements", "POST", {"name": "Owned gVisor Stop qualification", "authorization_ref": "owned-lab/stop-qualification", "application": plan["id"], "executor": "curl-http/v1"}, True, 201)
                parent = "/v1/engagements/" + engagement["id"]
                cleanup.callback(stop_parent, parent, entry)
                entry["engagement_id"] = engagement["id"]
                api(control, parent + "/approve", "POST", {"scope_digest": engagement["scope_digest"]}, True)
                run = api(control, parent + "/runs", "POST", {"assessment": plan["id"] + "/object-access/v2", "execution": "scripted"}, True, 201)
                entry["run_id"] = run["id"]
                run_path = "/v1/runs/" + run["id"]
                def held():
                    stats = api(target, "/cases/" + case + "/stats")
                    return stats if stats["active"] == 1 and stats["probe_started"] == before["probe_started"] + 1 else None
                entry["target_held"] = wait(held, 60)
                pods = json.loads(kube("-n", "tsunami", "get", "pods", "-l", "tsunami-run=" + run["id"], "-o", "json"))["items"]
                if len(pods) != 1 or pods[0]["spec"].get("runtimeClassName") != "gvisor":
                    raise RuntimeError("expected one gVisor worker")
                pod = pods[0]; node = pod["spec"]["nodeName"]
                if node != "tsunami-lab-control-plane":
                    raise RuntimeError("refusing runtime access to another node")
                sandbox_list = json.loads(subprocess.check_output(["docker", "exec", node, "crictl", "pods", "-o", "json"], text=True))["items"]
                sandbox = next(p for p in sandbox_list if p["metadata"]["uid"] == pod["metadata"]["uid"])
                inspection = json.loads(subprocess.check_output(["docker", "exec", node, "crictl", "inspectp", sandbox["id"]], text=True))
                runtime = {"handler": inspection.get("status", {}).get("runtimeHandler"), "type": inspection.get("info", {}).get("runtimeType")}
                if runtime != {"handler": "runsc", "type": "io.containerd.runsc.v1"}:
                    raise RuntimeError("actual runtime differs from gVisor")
                entry["runtime"] = runtime
                entry["worker_image_id"] = pod["status"]["containerStatuses"][0]["imageID"]
                container_id = pod["status"]["containerStatuses"][0]["containerID"].split("://", 1)[1]
                save()
                try:
                    if case in ("api-outage", "restart"):
                        kube("apply", "-f", "-", data=json.dumps(fault)); fault_installed = True
                        def fault_observed():
                            status = api(control, run_path + "/status", authenticated=True)
                            return status if status.get("observation_code") == "executor_observation_unavailable" else None
                        entry["api_fault_observed"] = wait(fault_observed, 7)
                    current = held()
                    if not current:
                        raise RuntimeError("target ended before Stop; timeout is not a passing Stop result")
                    age = (datetime.now(timezone.utc) - datetime.fromisoformat(current["probe_started_at"].replace("Z", "+00:00"))).total_seconds()
                    entry["held_age_at_stop_ms"] = round(age * 1000, 2)
                    if age >= 8:
                        raise RuntimeError("too close to the ten-second request deadline to attribute cancellation to Stop")
                    requested = time.monotonic()
                    entry["stop_requested_at"] = datetime.now(timezone.utc).isoformat()
                    stop_path = run_path + "/stop" if case == "run" else parent + "/stop"
                    api(control, stop_path, "POST", {}, True)
                    entry["api_ack_ms"] = round((time.monotonic() - requested) * 1000, 2)
                    def drained():
                        stats = api(target, "/cases/" + case + "/stats")
                        if stats["active"] == 0 and stats["probe_cancelled"] == before["probe_cancelled"] + 1:
                            return stats
                    entry["target_cancelled"] = wait(drained, 5)
                    entry["target_cancel_observed_ms"] = round((time.monotonic() - requested) * 1000, 2)
                    def local_drained():
                        status = api(control, run_path + "/status", authenticated=True)
                        return status if status["permission"] == "revoked" and status["local_operations"] == 0 else None
                    status = wait(local_drained, 5)
                    entry["after_stop"] = status
                    if status["permission"] != "revoked" or status["local_operations"] != 0:
                        raise RuntimeError("Stop did not revoke and drain local work")
                    if status.get("last_admitted_at", "") > status["revoked_at"]:
                        raise RuntimeError("operation admitted after revocation")
                    if fault_installed and status["cleanup"] != "pending":
                        raise RuntimeError("cleanup claimed acceptance while its API was blocked")
                    # The retained CRI entry may be removed quickly; record the
                    # distinction instead of treating absence as exit proof.
                    def worker_ended():
                        result = subprocess.run(["docker", "exec", node, "crictl", "inspect", container_id], capture_output=True, text=True)
                        if result.returncode:
                            return {"state": "runtime_entry_absent", "exit_proven": False}
                        state = json.loads(result.stdout)["status"]["state"]
                        return {"state": state, "exit_proven": True} if state == "CONTAINER_EXITED" else None
                    entry["runtime_after_stop"] = wait(worker_ended, 10)
                    entry["runtime_end_observed_ms"] = round((time.monotonic() - requested) * 1000, 2)
                    time.sleep(0.5)
                    stable = api(target, "/cases/" + case + "/stats")
                    if stable["requests"] != entry["target_cancelled"]["requests"] or stable["probe_completed"] != before["probe_completed"]:
                        raise RuntimeError("target activity continued after the drain observation")
                    api(control, parent + "/stop", "POST", {}, True)
                    api(control, parent + "/runs", "POST", {"assessment": plan["id"] + "/object-access/v2"}, True, 409)
                    if case == "restart":
                        kube("-n", "tsunami", "rollout", "restart", "deployment/tsunami-control")
                        kube("-n", "tsunami", "rollout", "status", "deployment/tsunami-control", "--timeout=60s")
                        # service port-forward points at the previous pod;
                        # establish a new authenticated operator connection.
                        with forward("tsunami", "tsunami-control") as restarted:
                            entry["after_restart"] = api(restarted, run_path + "/status", authenticated=True)
                            if entry["after_restart"]["permission"] != "revoked":
                                raise RuntimeError("restart restored permission")
                            api(restarted, parent + "/runs", "POST", {"assessment": plan["id"] + "/object-access/v2"}, True, 409)
                finally:
                    if fault_installed:
                        kube("-n", "tsunami", "delete", "ciliumnetworkpolicy", policy_name, "--ignore-not-found")
                    save()
            # Also reconnect after normal cases; this makes restart handling
            # explicit and confirms cleanup retries after the fault is removed.
            with forward("tsunami", "tsunami-control") as recovered:
                def clean():
                    status = api(recovered, run_path + "/status", authenticated=True)
                    return status if status["cleanup"] == "api_resources_absent" else None
                entry["after_cleanup"] = wait(clean, 35)
                entry["target_final"] = api(target, "/cases/" + case + "/stats")
                if entry["target_final"]["requests"] != entry["target_cancelled"]["requests"]:
                    raise RuntimeError("late target activity after cleanup/restart")
            entry["passed"] = True
            save()
            print(f"PASS {case}: Stop ack {entry['api_ack_ms']} ms; target cancellation observed {entry['target_cancel_observed_ms']} ms; {entry['runtime_after_stop']['state']}", flush=True)
    print("PASS owned gVisor/curl Stop campaign; records in " + str(out))


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, ValueError, KeyError, OSError, subprocess.SubprocessError, StopIteration) as error:
        raise SystemExit(str(error) if isinstance(error, RuntimeError) else "Stop qualification failed; inspect retained records (credentials suppressed).")

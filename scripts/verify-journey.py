#!/usr/bin/env python3
"""Exercise the compiled operator CLI against owned applications and real Jobs."""
import argparse
import base64
from contextlib import contextmanager
from datetime import datetime, timezone
import hashlib
import json
import uuid
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import time
import urllib.request


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        return None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--context", default="kind-tsunami-lab")
    parser.add_argument("--binary", type=Path, default=Path("bin/tsunami"))
    parser.add_argument("--toolkit", type=Path, help="optional separately obtained local toolkit; read-only schema validation")
    parser.add_argument("--profile", choices=["object", "property"], default="object", help="owned application campaign; scripted only")
    parser.add_argument("--out", type=Path, help="new campaign directory; default is a fresh timestamped artifact directory")
    args = parser.parse_args()
    if args.context != "kind-tsunami-lab":
        parser.error("use only the dedicated kind-tsunami-lab context")
    binary = args.binary.resolve(strict=True)
    base = ["kubectl", "--context", args.context]
    def kube(*argv):
        return subprocess.check_output(base + list(argv), text=True, stderr=subprocess.DEVNULL)
    def wait(predicate, seconds=30):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            value = predicate()
            if value:
                return value
            time.sleep(0.1)
        raise RuntimeError("required owned-lab observation did not arrive")
    @contextmanager
    def forward(namespace, service):
        sock = socket.socket(); sock.bind(("127.0.0.1", 0)); port = sock.getsockname()[1]; sock.close()
        process = subprocess.Popen(base + ["-n", namespace, "port-forward", "--address=127.0.0.1", "service/" + service, f"{port}:8080"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        origin = f"http://127.0.0.1:{port}"
        try:
            wait(lambda: ready(origin), 10)
            yield origin
        finally:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill(); process.wait()
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
    def read(origin, path):
        with opener.open(origin + path, timeout=3) as response:
            data = response.read((64 << 10) + 1)
        if len(data) > 64 << 10:
            raise RuntimeError("diagnostic response too large")
        return json.loads(data) if data else None
    def ready(origin):
        try:
            read(origin, "/healthz")
            return True
        except OSError:
            return False

    property_profile = args.profile == "property"
    cases = [("property-vulnerable", "confirmed"), ("property-filtered", "no_issue_observed"), ("property-ambiguous", "inconclusive"), ("property-stop", "stopped")] if property_profile else [("vampi-vulnerable", "confirmed"), ("vampi-fixed", "no_issue_observed"), ("stop-lab-run", "stopped")]
    out = args.out or Path("artifacts/property-access" if property_profile else "artifacts/operator-journey") / datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    out.mkdir(parents=True, exist_ok=False, mode=0o700)
    report = {"checked_at": datetime.now(timezone.utc).isoformat(), "context": args.context,
              "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(), "external_model_calls": 0, "profile": args.profile,
              "checks": [], "limitations": ["Owned synthetic data with known identities and inventory; no blind discovery or customer readiness.",
              "Approvals in this campaign are automated lab actions, not customer authorization or human finding review.",
              "One node and selected CLI paths; production failures, load and broad sandbox containment remain unqualified."]}
    def save():
        (out / "summary.json").write_text(json.dumps(report, indent=2) + "\n")
    transcript = (out / "transcript.txt").open("w")
    token = base64.b64decode(kube("-n", "tsunami", "get", "secret", "tsunami-operator", "-o", "jsonpath={.data.admin-a}"))
    Path(".local").mkdir(exist_ok=True, mode=0o700)
    fd, token_path = tempfile.mkstemp(prefix="journey-operator-", dir=".local")
    with os.fdopen(fd, "wb") as f:
        f.write(token)
    pending = []
    save()
    try:
        control_pods = json.loads(kube("-n", "tsunami", "get", "pods", "-l", "tsunami-role=control", "-o", "json"))["items"]
        report["control_image_id"] = next(p["status"]["containerStatuses"][0]["imageID"] for p in control_pods if not p["metadata"].get("deletionTimestamp") and p.get("status", {}).get("phase") == "Running")
        with forward("tsunami", "tsunami-control") as control:
            def cli(command, *flags, structured=True, expected=0):
                argv = [str(binary), "assess", command, "--url", control, "--token-file", token_path, *flags]
                if structured:
                    argv.append("--json")
                result = subprocess.run(argv, text=True, capture_output=True, timeout=110)
                transcript.write(f"\n$ tsunami assess {command} {' '.join(flags)}\n" + result.stdout + result.stderr)
                transcript.flush()
                if token.decode() in result.stdout + result.stderr:
                    raise RuntimeError("CLI disclosed the operator credential; do not share these records")
                if result.returncode != expected:
                    raise RuntimeError("CLI step failed: " + command + "; retain the local transcript")
                return json.loads(result.stdout) if structured and expected == 0 else result.stdout
            try:
                plans = cli("applications")
                if not {c[0] for c in cases}.issubset(p["id"] for p in plans):
                    raise RuntimeError("seed the selected owned fixtures first")
                for application, want in cases:
                    entry = {"application": application, "passed": False}
                    report["checks"].append(entry); save()
                    held_case = want == "stopped"
                    target_context = forward("tsunami-property-lab", "tsunami-property-fixture") if property_profile else forward("tsunami-stop-lab", "tsunami-stop-fixture") if held_case else None
                    stats_path = "/cases/" + (application.removeprefix("property-") if property_profile else "run") + "/stats"
                    from contextlib import nullcontext
                    with target_context if target_context else nullcontext() as target:
                        before = read(target, stats_path) if target else None
                        if before is not None: entry["target_before"] = before
                        view = cli("plan", "--application", application, "--authorization-ref", "owned-lab/operator-journey", "--executor", "curl-http/v1")
                        parent = view["engagement"]["id"]; pending.append(parent)
                        entry["engagement_id"] = parent
                        cli("show", "--engagement", parent, structured=False)
                        cli("start", "--engagement", parent, "--request-id", uuid.uuid4().hex, expected=1)
                        cli("approve", "--engagement", parent, "--scope-digest", "0" * 64, expected=1)
                        if target and read(target, stats_path)["requests"] != before["requests"]:
                            raise RuntimeError("planning or rejected approval caused target requests")
                        cli("approve", "--engagement", parent, "--scope-digest", view["engagement"]["scope_digest"])
                        run = cli("start", "--engagement", parent, "--request-id", uuid.uuid4().hex)["run_id"]
                        entry["run_id"] = run
                        if held_case:
                            def held():
                                stats = read(target, stats_path)
                                return stats if stats["active"] == 1 and stats["probe_started"] == before["probe_started"] + 1 else None
                            entry["target_held"] = wait(held, 10)
                            pod = json.loads(kube("-n", "tsunami", "get", "pods", "-l", "tsunami-run=" + run, "-o", "json"))["items"][0]
                            node = pod["spec"]["nodeName"]
                            if node != "tsunami-lab-control-plane" or pod["spec"].get("runtimeClassName") != "gvisor":
                                raise RuntimeError("expected the dedicated gVisor worker")
                            sandboxes = json.loads(subprocess.check_output(["docker", "exec", node, "crictl", "pods", "-o", "json"], text=True))["items"]
                            sandbox = next(s for s in sandboxes if s["metadata"]["uid"] == pod["metadata"]["uid"])
                            runtime = json.loads(subprocess.check_output(["docker", "exec", node, "crictl", "inspectp", sandbox["id"]], text=True))
                            entry["runtime"] = {"handler": runtime["status"].get("runtimeHandler"), "type": runtime["info"].get("runtimeType")}
                            if entry["runtime"] != {"handler": "runsc", "type": "io.containerd.runsc.v1"}:
                                raise RuntimeError("worker did not use the actual gVisor runtime")
                            # The API can lag behind a running container. Use
                            # CRI rather than accepting an empty image-ID sample.
                            containers = json.loads(subprocess.check_output(["docker", "exec", node, "crictl", "ps", "-a", "--pod", sandbox["id"], "-o", "json"], text=True))["containers"]
                            container = next(c for c in containers if c["metadata"]["name"] == "agent")
                            observed = json.loads(subprocess.check_output(["docker", "exec", node, "crictl", "inspect", container["id"]], text=True))["status"]
                            entry["worker_image_id"] = observed.get("imageRef")
                            entry["worker_container_id"] = container["id"]
                            if not entry["worker_image_id"]:
                                raise RuntimeError("actual running worker image identity unavailable")
                            cli("watch", "--run", run, "--timeout", "1s", structured=False, expected=1)
                            if cli("status", "--run", run)["permission"] != "active" or not held():
                                raise RuntimeError("read-only watch revoked permission or request ended before Stop")
                            age = (datetime.now(timezone.utc) - datetime.fromisoformat(held()["probe_started_at"].replace("Z", "+00:00"))).total_seconds()
                            if age >= 8:
                                raise RuntimeError("request too close to its ten-second deadline to attribute cancellation to Stop")
                            cli("stop", "--engagement", parent, structured=False)
                            entry["target_cancelled"] = wait(lambda: (s if (s := read(target, stats_path))["active"] == 0 and s["probe_cancelled"] == before["probe_cancelled"] + 1 else None), 5)
                            if entry["target_cancelled"]["probe_completed"] != before["probe_completed"]:
                                raise RuntimeError("held target request completed rather than cancelled")
                        else:
                            cli("watch", "--run", run, "--timeout", "100s", structured=False)
                            cli("stop", "--engagement", parent, structured=False)
                            if target:
                                entry["target_completed"] = read(target, stats_path)
                                if entry["target_completed"]["requests"] != before["requests"] + 7 or entry["target_completed"]["probe_completed"] != before["probe_completed"] + 1:
                                    raise RuntimeError("actual target did not observe the seven approved reads and completed probe")
                        status = cli("status", "--run", run)
                        entry["result_status"] = status
                        if status["execution_mode"] != "scripted" or status["executor"] != "curl-http/v1":
                            raise RuntimeError("unexpected execution mode or executor")
                        if held_case:
                            if status["permission"] != "revoked":
                                raise RuntimeError("CLI Stop did not revoke permission")
                        elif not status.get("assessment") or status["assessment"]["verdict"] != want or status["target_requests_reserved"] != 7:
                            raise RuntimeError("application result did not match its owned control")
                        cli("start", "--engagement", parent, "--request-id", uuid.uuid4().hex, expected=1)
                        cli("export", "--engagement", parent, "--out", str(out / application), structured=False)
                        bundle = json.loads((out / application / "bundle.json").read_text())
                        if bundle["toolkit_findings"] or len(bundle["finding_candidates"]) != (1 if want == "confirmed" else 0):
                            raise RuntimeError("a CLI export invented review or miscounted candidates")
                        if args.toolkit:
                            from toolkit import validate_export
                            validate_export(args.toolkit, out / application / "bundle.json")
                        entry["candidates"] = len(bundle["finding_candidates"])
                        entry["artifacts"] = len(bundle["artifacts"])
                        def cleaned():
                            status = cli("status", "--run", run)
                            return status if status["cleanup"] == "api_resources_absent" else None
                        entry["after_cleanup"] = wait(cleaned, 35)
                        if target:
                            final = read(target, stats_path)
                            if final["requests"] != entry["target_cancelled" if held_case else "target_completed"]["requests"]:
                                raise RuntimeError("late target traffic after CLI Stop")
                            entry["target_final"] = final
                        entry["passed"] = True; save()
                        pending.remove(parent)
                        print("PASS CLI " + application + ": " + want + "; export saved; no automatic review", flush=True)
            finally:
                for parent in pending:
                    try:
                        cli("stop", "--engagement", parent, structured=False)
                    except (OSError, RuntimeError, subprocess.SubprocessError):
                        report["cleanup_warning"] = "Check pending engagement revocation with an operator."
                save()
    finally:
        os.unlink(token_path)
        transcript.close()
    print("PASS compiled CLI journey; records in " + str(out))


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError, ValueError, KeyError, StopIteration, subprocess.SubprocessError) as error:
        raise SystemExit(str(error) if isinstance(error, RuntimeError) else "CLI qualification failed; inspect local records (credentials suppressed).")

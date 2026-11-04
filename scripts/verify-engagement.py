#!/usr/bin/env python3
"""Run the scripted engagement flow in the dedicated lab; collect via operator API."""
import argparse
import base64
import json
from deploy_images import deployed_image, with_image
from pathlib import Path
import re
import socket
import subprocess
import time
import urllib.request

MAX_EXPORT_BYTES = 192 << 20  # platform.MaxInlineExportBytes


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, response, code, message, headers, url):
        return None


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--context", default="kind-tsunami-lab")
    parser.add_argument("--private", action="store_true")
    parser.add_argument("--application", action="store_true", help="owned VAmPI application campaign; scripted only")
    parser.add_argument("--executor", choices=["curl-http/v1"], default="")
    args = parser.parse_args()
    if args.executor and not args.application:
        parser.error("reviewed executor is only available for configured applications")
    if args.context != "kind-tsunami-lab":
        parser.error("use the dedicated kind-tsunami-lab context")
    if args.application and args.private:
        parser.error("private application adapters are not qualified")
    base = ["kubectl", "--context", args.context, "-n", "tsunami"]

    def kubectl(*argv, data=None):
        return subprocess.check_output(base + list(argv), input=data, text=True, stderr=subprocess.DEVNULL)

    job = json.loads(kubectl("create", "--dry-run=client", "-f", "deploy/verify.yaml", "-o", "json"))
    job = with_image(job, deployed_image(kubectl))
    job["metadata"]["generateName"] = "tsunami-application-" if args.application else "tsunami-engagement-"
    job["spec"]["activeDeadlineSeconds"] = 600
    argv = ["engagement-verify"]
    if args.application:
        argv = ["application-verify", "--cases", "vampi-vulnerable=confirmed,vampi-fixed=no_issue_observed", "--exercise-review", "--out", "/tmp/application-campaign"]
    if args.executor:
        argv += ["--executor", args.executor]
    if args.private:
        argv += ["--connectivity", "private"]
    job["spec"]["template"]["spec"]["containers"][0]["args"] = argv
    name = json.loads(kubectl("create", "-f", "-", "-o", "json", data=json.dumps(job)))["metadata"]["name"]
    out = Path("artifacts/tool-execution" if args.executor else "artifacts/application-adapters" if args.application else "artifacts/engagements") / name
    out.mkdir(parents=True, exist_ok=True, mode=0o700)
    status = {}
    for _ in range(610):
        status = json.loads(kubectl("get", "job", name, "-o", "json")).get("status", {})
        if status.get("succeeded") or status.get("failed"):
            break
        time.sleep(1)
    log = kubectl("logs", "job/" + name)
    (out / "verification.log").write_text(log)
    summary = None
    for line in log.splitlines():
        prefix = "APPLICATION_REPORT " if args.application else "ENGAGEMENT_REPORT "
        if line.startswith(prefix):
            summary = json.loads(line[len(prefix):])
            (out / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
        else:
            print(line)
    pattern = r"APPLICATION_RESULT application=\S+ engagement=([a-f0-9]{32})\b" if args.application else r"ENGAGEMENT_RESULT id=([a-f0-9]{32})\b"
    ids = list(dict.fromkeys(re.findall(pattern, log)))
    if ids:
        token = base64.b64decode(kubectl("get", "secret", "tsunami-operator", "-o", "jsonpath={.data.admin-a}")).decode()
        listener = socket.socket()
        listener.bind(("127.0.0.1", 0))
        port = listener.getsockname()[1]
        listener.close()
        forward = subprocess.Popen(base + ["port-forward", "--address=127.0.0.1", "service/tsunami-control", f"{port}:8080"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        try:
            opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
            for identity in ids:
                request = urllib.request.Request(f"http://127.0.0.1:{port}/v1/engagements/{identity}/export", headers={"Authorization": "Bearer " + token})
                bundle = None
                for _ in range(100):
                    if forward.poll() is not None:
                        raise RuntimeError("operator port-forward ended")
                    try:
                        with opener.open(request, timeout=5) as response:
                            raw = response.read(MAX_EXPORT_BYTES + 1)
                            if len(raw) > MAX_EXPORT_BYTES:
                                raise RuntimeError("engagement export exceeds collection limit")
                            bundle = json.loads(raw)
                        break
                    except OSError:
                        time.sleep(0.05)
                if bundle is None:
                    raise RuntimeError("could not collect engagement export")
                target = out / identity
                target.mkdir(mode=0o700)
                (target / "bundle.json").write_text(json.dumps(bundle, indent=2) + "\n")
                (target / "report-draft.md").write_text(bundle["report_markdown"])
                subprocess.run(["go", "run", "./cmd/tsunami", "export-bundle", "--input", str(target / "bundle.json"), "--out", str(target / "export")], check=True)
        finally:
            forward.terminate()
            try:
                forward.wait(timeout=5)
            except subprocess.TimeoutExpired:
                forward.kill()
                forward.wait()
    if not status.get("succeeded") or summary is None:
        raise RuntimeError(f"engagement verification failed; records retained in {out}")
    print(f"PASS scripted {'application' if args.application else 'engagement'} verification; records in {out}; no external model calls.")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, subprocess.SubprocessError, OSError):
        raise SystemExit("Engagement verification failed; inspect retained records (credential output suppressed).")

#!/usr/bin/env python3
"""Disposable local UI test; no installation or production path override.

Keep Chrome's normal sandbox on the host. Only the Go fixture/worker need the
test filesystem view containing the exact reviewed adapter at its fixed path.
This supplies no Kubernetes, worker isolation or network-containment claim.
"""
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time


def main():
    for tool in ("bwrap", "go", "node", "google-chrome", "curl"):
        if not shutil.which(tool):
            raise SystemExit(f"Missing local browser prerequisite: {tool}")
    root = Path.cwd()
    build_cache = subprocess.check_output(["go", "env", "GOCACHE"], text=True).strip()
    with tempfile.TemporaryDirectory(prefix="tsunami-ui-onboarding-", ignore_cleanup_errors=True) as temporary:
        directory = Path(temporary)
        adapter, profile = directory / "adapter", directory / "chrome"
        adapter.mkdir()
        profile.mkdir()
        (profile / "tsunami-test-profile").touch()
        shutil.copyfile(root / "internal/toolrunner/http-read.sh", adapter / "http-read.sh")
        (adapter / "http-read.sh").chmod(0o400)
        chrome = subprocess.Popen([
            "google-chrome", "--headless=new", "--disable-gpu", "--no-proxy-server",
            "--no-first-run", "--no-default-browser-check", "--disable-background-networking",
            "--disable-component-update", "--disable-default-apps", "--disable-sync",
            "--disable-extensions", "--metrics-recording-only",
            "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=0",
            "--user-data-dir=" + str(profile), "about:blank",
        ], env={**os.environ, "XDG_CONFIG_HOME": str(profile / "config")},
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        try:
            command = ["bwrap", "--die-with-parent", "--ro-bind", "/", "/", "--dev-bind", "/dev", "/dev",
                       "--bind", str(root), str(root), "--bind", "/tmp", "/tmp",
                       "--bind", build_cache, build_cache, "--tmpfs", "/opt"]
            for entry in sorted(Path("/opt").iterdir()):
                if entry.name != "tsunami":
                    command.extend(["--ro-bind", str(entry), str(entry)])
            command.extend(["--ro-bind", str(adapter), "/opt/tsunami", "--", "go", "test", "-race",
                            "-count=1", "./internal/webui", "-run", "TestLocalBrowserOnboardingAndReview", "-v"])
            result = subprocess.run(command, env={**os.environ, "TSUNAMI_UI_ONBOARDING_TEST": "1",
                                    "TSUNAMI_TEST_CHROME_PROFILE": str(profile)})
            return result.returncode
        finally:
            chrome.terminate()
            try:
                chrome.wait(timeout=10)
            except subprocess.TimeoutExpired:
                chrome.kill()
                chrome.wait()
            # Chrome descendants may finish profile writes shortly after its
            # main process exits. Remove only this launcher's own temporary tree.
            for attempt in range(10):
                try:
                    shutil.rmtree(directory)
                    break
                except OSError:
                    if attempt == 9:
                        raise
                    time.sleep(0.1 * (attempt + 1))


if __name__ == "__main__":
    raise SystemExit(main())

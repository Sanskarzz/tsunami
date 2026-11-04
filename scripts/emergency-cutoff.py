#!/usr/bin/env python3
"""Latch the synthetic connector closed without contacting the normal Stop API.

This intentionally leaves private execution disabled, including after restart.
Requires trusted kubectl access; never prints the emergency credential.
"""
import base64
import json
import os
import socket
import subprocess
import sys
import time


context = os.environ.get("CONTEXT", "kind-tsunami-lab")
if context != "kind-tsunami-lab":
    sys.exit("This cutoff helper is restricted to the synthetic lab")
base = ["kubectl", "--context", context, "-n", "tsunami-customer-a"]
result = subprocess.run(base+["get", "secret", "tsunami-connector-emergency", "-o", "json"], capture_output=True, text=True)
if result.returncode:
    sys.exit("Cannot read the connector emergency credential with kubectl")
key = base64.b64decode(json.loads(result.stdout)["data"]["emergency-key"]).decode()
with socket.socket() as sock:
    sock.bind(("127.0.0.1", 0)); port = sock.getsockname()[1]
p = subprocess.Popen(base+["port-forward", "--address=127.0.0.1", "service/tsunami-connector", f"{port}:8080"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
try:
    for _ in range(100):
        try:
            with socket.create_connection(("127.0.0.1", port), timeout=0.1):
                break
        except OSError:
            if p.poll() is not None:
                sys.exit("Connector port-forward exited; cutoff was not confirmed")
            time.sleep(0.05)
    else:
        sys.exit("Connector port-forward unavailable; cutoff was not confirmed")
    # Go verifies the connector's real DNS/URI identity over the loopback
    # port-forward. The operator's TLS key stays in the ignored local PKI dir.
    result = subprocess.run(["./bin/tsunami", "connector-cutoff", "--endpoint", f"https://127.0.0.1:{port}"], input=key, text=True)
    if result.returncode:
        sys.exit("Authenticated connector cutoff failed; no stop claim made")
finally:
    p.terminate(); p.wait(timeout=5)

"""Read-only admission checks for this repository's dedicated secured lab.

This is an operator mistake guard, not an authorization or isolation proof.
Never print kubeconfig contents or command responses: they can contain secrets.
"""
import argparse
import json
import os
from pathlib import Path
import stat
import subprocess
from urllib.parse import urlsplit


ROOT = Path(__file__).resolve().parent.parent
CONTEXT = "kind-tsunami-lab"
NODE = "tsunami-lab-control-plane"


def read_command(argv):
    try:
        result = subprocess.run(argv, capture_output=True, text=True, timeout=15)
        if result.returncode or len(result.stdout) > 2 << 20:
            raise ValueError()
        return result.stdout.strip()
    except (OSError, ValueError, subprocess.SubprocessError):
        raise RuntimeError("Secured lab prerequisite unavailable; no deployment admitted (response suppressed)") from None


def require_secured_lab(context=None, runtime=None, *, bootstrap=False):
    context = context if context is not None else os.environ.get("CONTEXT", "")
    runtime = runtime if runtime is not None else os.environ.get("RUNTIME_CLASS", "")
    if context != CONTEXT or runtime != "gvisor":
        raise RuntimeError("Require CONTEXT=kind-tsunami-lab and RUNTIME_CLASS=gvisor; deployment refused")
    expected = ROOT / ".local/sandbox.kubeconfig"
    raw = os.environ.get("KUBECONFIG", "")
    path = Path(raw)
    if (Path.cwd().resolve() != ROOT or not raw or os.pathsep in raw or not path.is_absolute()
            or path != expected or path.is_symlink() or path.parent.is_symlink()):
        raise RuntimeError("Require the absolute repository .local/sandbox.kubeconfig; no ambient/merged kubeconfig")
    if bootstrap:
        if path.exists():
            raise RuntimeError("Existing dedicated kubeconfig found; fresh bootstrap refuses adoption or overwrite")
        read_command(["docker", "info", "--format", "{{.ServerVersion}}"])
        if read_command(["docker", "ps", "-a", "--filter", "label=io.x-k8s.kind.cluster=tsunami-lab", "--format", "{{.Names}}"]):
            raise RuntimeError("Existing tsunami-lab node found; fresh bootstrap requires a separate environment")
        return
    try:
        info = path.stat()
        if not stat.S_ISREG(info.st_mode) or info.st_size > 128 << 10 or info.st_mode & 0o077:
            raise ValueError()
    except (OSError, ValueError):
        raise RuntimeError("Dedicated kubeconfig must be a private regular file (mode 0600)") from None
    kube = ["kubectl", "--kubeconfig", str(expected), "--context", CONTEXT]
    try:
        config = json.loads(read_command(kube + ["config", "view", "--minify", "--raw", "-o", "json"]))
        clusters, contexts, users = config["clusters"], config["contexts"], config["users"]
        if any(len(items) != 1 or items[0]["name"] != CONTEXT for items in (clusters, contexts, users)):
            raise ValueError()
        selected, cluster, user = contexts[0]["context"], clusters[0]["cluster"], users[0]["user"]
        if selected["cluster"] != CONTEXT or selected["user"] != CONTEXT:
            raise ValueError()
        endpoint = urlsplit(cluster["server"])
        if (endpoint.scheme != "https" or endpoint.hostname not in ("127.0.0.1", "::1")
                or not endpoint.port or endpoint.username or endpoint.password or endpoint.path
                or endpoint.query or endpoint.fragment or cluster.get("insecure-skip-tls-verify")
                or cluster.get("proxy-url") or not cluster.get("certificate-authority-data")
                or set(user) != {"client-certificate-data", "client-key-data"}
                or not all(user.values())):
            raise ValueError()
        node = json.loads(read_command(["docker", "inspect", NODE]))[0]
        bindings = node["NetworkSettings"]["Ports"]["6443/tcp"]
        if (node["Config"]["Labels"].get("io.x-k8s.kind.cluster") != "tsunami-lab"
                or not node["State"]["Running"]
                or not any(b["HostIp"] == endpoint.hostname and b["HostPort"] == str(endpoint.port) for b in bindings)):
            raise ValueError()
        nodes = json.loads(read_command(kube + ["get", "nodes", "-o", "json"]))["items"]
        if (len(nodes) != 1 or nodes[0]["metadata"]["name"] != NODE
                or not any(c["type"] == "Ready" and c["status"] == "True" for c in nodes[0]["status"]["conditions"])):
            raise ValueError()
        runtime_class = json.loads(read_command(kube + ["get", "runtimeclass", "gvisor", "-o", "json"]))
        policy = json.loads(read_command(kube + ["-n", "kube-system", "get", "configmap", "cilium-config", "-o", "json"]))
        cilium = json.loads(read_command(kube + ["-n", "kube-system", "get", "daemonset", "cilium", "-o", "json"]))
        if (runtime_class["handler"] != "runsc" or policy["data"]["enable-policy"] != "always"
                or cilium["status"].get("desiredNumberScheduled") != 1
                or cilium["status"].get("numberReady") != 1):
            raise ValueError()
    except (KeyError, IndexError, TypeError, ValueError, AttributeError):
        raise RuntimeError("Dedicated Kind identity, TLS, gVisor or Cilium prerequisite mismatch; deployment refused") from None


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--context")
    parser.add_argument("--runtime-class")
    parser.add_argument("--bootstrap", action="store_true")
    args = parser.parse_args()
    try:
        require_secured_lab(args.context, args.runtime_class, bootstrap=args.bootstrap)
    except RuntimeError as error:
        raise SystemExit(str(error))

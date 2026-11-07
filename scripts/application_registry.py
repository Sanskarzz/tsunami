"""Trusted lab provisioning helpers; never agent tools or customer onboarding."""
import json
import os
from pathlib import Path
import tempfile
import subprocess


def merge_applications(existing, replacements):
    keys = {(c["tenant"], c["plan"]["id"]) for c in replacements}
    merged = [c for c in existing if (c["tenant"], c["plan"]["id"]) not in keys] + replacements
    if len(merged) > 16:
        raise RuntimeError("merged registry exceeds the server's 16 application limit")
    return merged


def write_private_json(path, value):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    fd, temporary = tempfile.mkstemp(prefix=".registry-", dir=path.parent)
    try:
        with os.fdopen(fd, "w") as output:
            json.dump(value, output, indent=2)
            output.write("\n")
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def binding_compiler():
    binary = Path("bin/tsunami").resolve(strict=True)
    check = subprocess.run([str(binary), "prepare-plan", "--help"], capture_output=True, timeout=10)
    if check.returncode:
        raise RuntimeError("build the current CLI with prepare-plan before provisioning")
    return binary


def prepare_plan(facts):
    """No raw documents or credentials in argv; keep inputs private locally."""
    binary = binding_compiler()
    root = Path(".local/application-bindings")
    root.mkdir(parents=True, exist_ok=True, mode=0o700)
    folder = Path(tempfile.mkdtemp(prefix="binding-", dir=root))
    source = folder / "input.json"
    write_private_json(source, facts)
    result = subprocess.run([str(binary), "prepare-plan", "--input", str(source), "--out", str(folder / "prepared"), "--json"], text=True, capture_output=True, timeout=10)
    if result.returncode:
        write_private_json(folder / "diagnostic.json", {"exit_code": result.returncode, "error": result.stderr})
        raise RuntimeError("binding rejected; inspect the private local preparation diagnostic")
    return json.loads((folder / "prepared/plan.json").read_text())

#!/usr/bin/env python3
"""Verify checked-in example bytes/provenance; optionally check local toolkit schemas."""
import argparse
import base64
import hashlib
import json
from pathlib import Path

from toolkit import parse, read_local, validate_export


def check_bundle(path):
    bundle = parse(path.read_bytes())
    engagement = bundle["view"]["engagement"]["id"]
    artifacts = {}
    for item in bundle["artifacts"]:
        data = base64.b64decode(item["data_base64"], validate=True)
        digest = hashlib.sha256(data).hexdigest()
        if item["sha256"] != digest or item["path"] != f"evidence/{digest}.json" or item["id"] != "E-" + digest:
            raise ValueError("invalid artifact identity: " + str(path))
        if item["id"] in artifacts or read_local(path.parent, item["path"]) != data:
            raise ValueError("duplicate or changed materialized artifact: " + str(path))
        capture = parse(data)
        if capture["engagement_id"] != engagement or capture["run_id"] != item["run_id"] or capture["capture"]["id"] != item["capture_id"]:
            raise ValueError("artifact provenance differs: " + str(path))
        artifacts[item["id"]] = item
    for candidate in bundle["finding_candidates"]:
        if any(key not in artifacts for key in candidate["evidence_ids"]):
            raise ValueError("missing candidate evidence: " + str(path))
    for finding in bundle["toolkit_findings"]:
        if parse(read_local(path.parent, "findings/finding_" + finding["id"] + ".json")) != finding:
            raise ValueError("materialized finding differs: " + str(path))
        for evidence in finding["evidence"]:
            if evidence["id"] not in artifacts or parse(read_local(path.parent, "evidence/" + evidence["id"] + ".meta.json")) != evidence:
                raise ValueError("materialized evidence index differs: " + str(path))
    return len(artifacts)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--toolkit", type=Path, help="explicit local reference checkout; never downloaded or executed")
    args = parser.parse_args()
    paths = [Path("examples") / family / app / "bundle.json" for family in ["application-adapters", "tool-execution"] for app in ["vampi-vulnerable", "vampi-fixed"]]
    count = 0
    for path in paths:
        count += check_bundle(path)
        if args.toolkit:
            validate_export(args.toolkit, path)
    archive = parse(Path("examples/inference-diagnostics/archive-sha256.json").read_bytes())
    for name, expected in archive.items():
        if hashlib.sha256(read_local(Path.cwd(), name)).hexdigest() != expected:
            raise ValueError("historical trial bytes changed: " + name)
    print(f"PASS {len(paths)} materialized bundles, {count} artifacts and {len(archive)} historical trial hashes.")
    print("Actual local toolkit schemas: " + ("PASS" if args.toolkit else "not requested; run make verify-toolkit separately."))


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Inspect a local methodology repository as data; never execute its scripts."""
import argparse
import base64
import copy
from collections import Counter
from datetime import datetime, timezone
import hashlib
import importlib.metadata
import json
import os
from pathlib import Path, PurePosixPath
import re
import stat
import tempfile

# Same inline lab-bundle ceiling as platform.MaxInlineExportBytes. This is not
# an assurance that every client/deployment has qualified this memory footprint.
MAX_EXPORT_BYTES = 192 << 20


def read_local(root, relative, limit=2 << 20):
    relative = PurePosixPath(relative)
    if relative.is_absolute() or ".." in relative.parts:
        raise ValueError("local relative path required")
    path = root
    for part in relative.parts:
        path = path / part
        if path.is_symlink():
            raise ValueError("toolkit symlinks are not accepted")
    if not stat.S_ISREG(path.stat().st_mode):
        raise ValueError("regular toolkit file required")
    with path.open("rb") as source:
        data = source.read(limit + 1)
    if len(data) > limit:
        raise ValueError("toolkit file exceeds read limit")
    return data


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("duplicate JSON member")
        result[key] = value
    return result


def parse(data):
    return json.loads(data, object_pairs_hook=unique_object)


def leaves(value, depth=0):
    if depth > 40:
        raise ValueError("checklist nesting exceeds limit")
    found = []
    if isinstance(value, dict):
        children = []
        for nested in value.values():
            if isinstance(nested, (dict, list)):
                children.extend(leaves(nested, depth + 1))
        identifier = value.get("id")
        if not identifier and isinstance(value.get("rank"), int) and isinstance(value.get("cwe_id"), int):
            identifier = "CWE-" + str(value["cwe_id"])
        if isinstance(identifier, str) and not children:
            if len(identifier) > 128 or not re.fullmatch(r"[A-Za-z0-9_:./-]+", identifier):
                raise ValueError("unsupported checklist identifier")
            found.append(identifier)
        found.extend(children)
    elif isinstance(value, list):
        for nested in value:
            found.extend(leaves(nested, depth + 1))
    return found


def inspect(root):
    root = root.resolve(strict=True)
    report = {"version": "tsunami.toolkit-inventory/v1", "checked_at": datetime.now(timezone.utc).isoformat(),
              "scripts_executed": False, "contents_vendored": False,
              "vm_mcp_integration_implemented": False,
              "license_files": [], "scripts": [], "checklists": [], "schemas": [], "prompt_files": [],
              "documented_roles": [], "limitations": [
                  "File hashes and checklist IDs are inventory, not approval to execute a tool.",
                  "All script adapters remain disabled until reviewed and integrated with their enforcement profile.",
                  "Inventory does not verify the toolkit's standards claims, completeness, licensing terms, or script safety."]}
    for p in sorted(root.iterdir()):
        if p.name.lower().startswith(("license", "copying")) and p.is_file():
            data = read_local(root, p.name)
            report["license_files"].append({"path": p.name, "sha256": hashlib.sha256(data).hexdigest()})
    selected = []
    for directory in ["scripts", "checklists", "schemas", "docs/agent_prompts"]:
        base = root / directory
        if base.is_symlink() or not base.is_dir():
            raise ValueError("expected a real toolkit source directory")
        for current, dirs, files in os.walk(base, followlinks=False):
            if any((Path(current) / name).is_symlink() for name in dirs):
                raise ValueError("toolkit directory symlinks are not accepted")
            selected.extend((Path(current) / name).relative_to(root).as_posix() for name in sorted(files))
            if len(selected) > 1024:
                raise ValueError("toolkit inventory exceeds file limit")
    total_bytes = 0
    profiles = {"web": "http-broker-adapter", "network": "isolated-network-executor",
                "mobile": "device-or-artifact-executor", "cloud": "customer-cloud-or-controlled-ssrf",
                "crypto": "offline-artifact-executor", "reporting": "offline-report-executor"}
    for relative in sorted(selected):
        path = PurePosixPath(relative)
        data = read_local(root, relative)
        total_bytes += len(data)
        if total_bytes > 32 << 20:
            raise ValueError("toolkit inventory exceeds total read limit")
        item = {"path": relative, "sha256": hashlib.sha256(data).hexdigest()}
        if path.parts[0] == "scripts" and path.suffix in {".py", ".sh", ".js"}:
            domain = path.parts[1]
            item.update(domain=domain, execution_profile=profiles.get(domain, "unclassified"), state="adapter_not_implemented")
            report["scripts"].append(item)
        elif path.parts[0] == "checklists" and path.suffix == ".json":
            ids = leaves(parse(data))
            item.update(item_ids=sorted(set(ids)), unique_leaf_items=len(set(ids)), duplicate_ids=len(ids)-len(set(ids)),
                        execution_state="reference_only")
            report["checklists"].append(item)
        elif path.parts[0] == "schemas" and path.suffix == ".json":
            parse(data)
            report["schemas"].append(item)
        elif path.parts[:2] == ("docs", "agent_prompts") and path.suffix == ".md":
            report["prompt_files"].append(item)
    role_data = read_local(root, "docs/agent_roles.md")
    report["documented_roles"] = re.findall(r"^\| \*\*([^*]+)\*\* \|", role_data.decode(), re.MULTILINE)
    report["roles_document_sha256"] = hashlib.sha256(role_data).hexdigest()
    report["counts"] = {"scripts": len(report["scripts"]), "checklists": len(report["checklists"]),
                        "schemas": len(report["schemas"]), "prompt_files": len(report["prompt_files"]),
                        "documented_roles": len(report["documented_roles"]),
                        "scripts_by_domain": dict(sorted(Counter(s["domain"] for s in report["scripts"]).items()))}
    return report


def reject_refs(value, allowed=()):
    if isinstance(value, dict):
        if "$ref" in value and value["$ref"] not in allowed:
            raise ValueError("schema references are disabled for this local engagement validator")
        for nested in value.values():
            reject_refs(nested, allowed)
    elif isinstance(value, list):
        for nested in value:
            reject_refs(nested, allowed)


def validate_export(root, bundle_path):
    from jsonschema import Draft7Validator, FormatChecker
    from referencing import Registry, Resource
    schemas = {}
    hashes = {}
    for name in ["engagement.json", "finding.json", "evidence.json"]:
        data = read_local(root.resolve(strict=True), "schemas/" + name)
        schema = parse(data)
        reject_refs(schema, ("evidence.json",) if name == "finding.json" else ())
        if schema.get("$schema") != "http://json-schema.org/draft-07/schema#":
            raise ValueError("unsupported local schema draft")
        try:
            Draft7Validator.check_schema(schema)
        except Exception:
            raise ValueError("invalid local schema contract") from None
        schemas[name] = schema
        hashes[name] = hashlib.sha256(data).hexdigest()

    def no_retrieval(uri):
        raise ValueError("schema resource retrieval is disabled")

    registry = Registry(retrieve=no_retrieval).with_resources((name, Resource.from_contents(value)) for name, value in schemas.items())
    validators = {name: Draft7Validator(value, registry=registry, format_checker=FormatChecker()) for name, value in schemas.items()}

    def validate(name, value):
        try:
            errors = list(validators[name].iter_errors(value))
        except Exception:
            raise ValueError("local schema validation could not be completed") from None
        if errors:
            raise ValueError("projection does not match local " + name)
    with bundle_path.open("rb") as source:
        raw = source.read(MAX_EXPORT_BYTES + 1)
    if len(raw) > MAX_EXPORT_BYTES:
        raise ValueError("engagement export exceeds validation limit")
    bundle = parse(raw)
    if bundle.get("version") not in {"tsunami.engagement-export/v1", "tsunami.engagement-export/v2"} or bundle.get("synthetic_only") is not True:
        raise ValueError("unsupported export contract")
    validate("engagement.json", bundle["toolkit_engagement"])
    view = bundle["view"]
    if view["engagement"]["id"] != bundle["toolkit_engagement"]["id"]:
        raise ValueError("engagement identity mismatch")
    artifacts = {}
    artifact_files_verified = 0
    for item in bundle["artifacts"]:
        data = base64.b64decode(item["data_base64"], validate=True)
        digest = hashlib.sha256(data).hexdigest()
        if item["sha256"] != digest or item["id"] != "E-" + digest or item["path"] != f"evidence/{digest}.json":
            raise ValueError("artifact content/path digest mismatch")
        if item["id"] in artifacts:
            raise ValueError("duplicate artifact identity")
        capture = parse(data)
        if capture["engagement_id"] != view["engagement"]["id"] or capture["run_id"] != item["run_id"] or capture["capture"]["id"] != item["capture_id"]:
            raise ValueError("artifact provenance mismatch")
        if (bundle_path.parent / "evidence").is_dir():
            if read_local(bundle_path.parent, item["path"]) != data:
                raise ValueError("materialized artifact bytes differ from bundle")
            artifact_files_verified += 1
        artifacts[item["id"]] = item
    for candidate in bundle["finding_candidates"]:
        if candidate["status"] not in {"needs_human_review", "reviewed_confirmed", "reviewed_false_positive"} or not candidate["evidence_ids"]:
            raise ValueError("candidate review/evidence state invalid")
        if any(ref not in artifacts or artifacts[ref]["run_id"] not in candidate["run_ids"] for ref in candidate["evidence_ids"]):
            raise ValueError("candidate references missing or unrelated evidence")
    candidates = {candidate["id"]: candidate for candidate in bundle["finding_candidates"]}
    evidence_pairs = {}
    finding_ids = set()
    for finding in bundle.get("toolkit_findings", []):
        validate("finding.json", finding)
        candidate = candidates.get(finding["id"])
        if not candidate or candidate["status"] != "reviewed_confirmed":
            raise ValueError("finding has no reviewed candidate")
        review = candidate["review"]
        if review["decision"] != "confirm" or review["candidate_digest"] != candidate["candidate_digest"]:
            raise ValueError("finding review/evidence changed")
        if review["purpose"] not in {"operator_review", "automated_lab_test"} or review["purpose"] not in finding["tags"]:
            raise ValueError("review provenance missing")
        for field in ["cvss_score", "cvss_vector", "severity"]:
            if finding[field] != review[field]:
                raise ValueError("finding rating differs from review")
        if finding["engagement_id"] != view["engagement"]["id"] or finding["target"] != view["engagement"]["target"]:
            raise ValueError("finding belongs to a different engagement or target")
        if finding["id"] in finding_ids:
            raise ValueError("duplicate finding ID")
        finding_ids.add(finding["id"])
        saved_finding = parse(read_local(bundle_path.parent, "findings/finding_" + finding["id"] + ".json"))
        if saved_finding != finding:
            raise ValueError("materialized finding differs from bundle")
        if {item["id"] for item in finding["evidence"]} != set(candidate["evidence_ids"]):
            raise ValueError("finding omitted or added evidence")
        for item in finding["evidence"]:
            validate("evidence.json", item)
            artifact = artifacts[item["id"]]
            saved_data = read_local(bundle_path.parent, artifact["path"])
            if saved_data != base64.b64decode(artifact["data_base64"], validate=True):
                raise ValueError("materialized evidence differs from bundle")
            saved_metadata = parse(read_local(bundle_path.parent, "evidence/" + item["id"] + ".meta.json"))
            if saved_metadata != item:
                raise ValueError("materialized evidence metadata differs from bundle")
            payload = parse(base64.b64decode(artifact["data_base64"], validate=True))
            if item["finding_id"] != finding["id"] or item["path"] != artifact["path"] or item["captured_at"] != payload["capture"]["started_at"]:
                raise ValueError("evidence projection provenance mismatch")
            pair = (item["id"], finding["id"])
            if pair in evidence_pairs:
                raise ValueError("duplicate finding evidence")
            evidence_pairs[pair] = item
    projected = bundle.get("toolkit_evidence", [])
    if len(projected) != len(evidence_pairs) or any(evidence_pairs.get((item["id"], item["finding_id"])) != item for item in projected):
        raise ValueError("evidence index mismatch")
    if {item["id"] for item in bundle["toolkit_engagement"]["findings"]} != finding_ids:
        raise ValueError("engagement finding index mismatch")
    return {"engagement_schema_valid": True, "artifact_hashes_valid": True,
            "artifact_files_verified": artifact_files_verified,
            "artifacts": len(artifacts), "finding_candidates": len(bundle["finding_candidates"]),
            "reviewed_findings": len(finding_ids), "finding_and_evidence_schemas_valid": True,
            "reviewed_finding_files_verified": len(finding_ids), "reviewed_evidence_files_verified": len(evidence_pairs),
            "final_finding_schema_export": "reviewed_findings_only" if finding_ids else "no_reviewed_findings",
            "schema_sha256": hashes["engagement.json"], "schema_hashes": hashes,
            "jsonschema_version": importlib.metadata.version("jsonschema"),
            "scripts_executed": False, "remote_refs_allowed": False}


def save(path, value):
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    handle, temp = tempfile.mkstemp(prefix=".toolkit-", dir=path.parent)
    try:
        with os.fdopen(handle, "w") as output:
            json.dump(value, output, indent=2)
            output.write("\n")
        os.replace(temp, path)
    finally:
        if os.path.exists(temp):
            os.unlink(temp)


def checklist_coverage(root, bundle_path, mapping_path):
    validate_export(root, bundle_path)
    bundle = parse(bundle_path.read_bytes())
    raw = mapping_path.read_bytes()
    if len(raw) > 64 << 10:
        raise ValueError("checklist mapping exceeds limit")
    mapping = parse(raw)
    engagement = bundle["view"]["engagement"]
    if mapping.get("version") != "tsunami.checklist-map/v1" or mapping.get("engagement_id") != engagement["id"] or mapping.get("scope_digest") != engagement["scope_digest"]:
        raise ValueError("mapping does not match this approved engagement")
    sources = mapping.get("sources", [])
    if not isinstance(sources, list) or not sources or len(sources) > 3:
        raise ValueError("one to three explicit checklist sources required")
    inventory = inspect(root)
    catalogs = {item["path"]: item for item in inventory["checklists"]}
    progress = {item["procedure"]: item for item in bundle["view"]["checklist"]}
    projection = copy.deepcopy(bundle["toolkit_engagement"])
    records = []
    seen = set()
    for source in sources:
        path = source["path"]
        if path in seen or path not in catalogs or source["sha256"] != catalogs[path]["sha256"]:
            raise ValueError("checklist source/version changed or duplicated")
        seen.add(path)
        known_ids = set(catalogs[path]["item_ids"])
        entries = source.get("mappings", [])
        if not entries or len(entries) > 32:
            raise ValueError("bounded explicit partial mappings required")
        samples = []
        mapped = set()
        for entry in entries:
            identifier, procedure = entry["item_id"], entry["procedure"]
            if identifier not in known_ids or identifier in mapped or procedure not in progress or entry.get("coverage") != "partial":
                raise ValueError("unknown/duplicate mapping or unsupported full-coverage claim")
            mapped.add(identifier)
            result = progress[procedure]
            samples.append({"item_id": identifier, "procedure": procedure, "coverage": "partial",
                            "sample_state": "sampled" if result["status"] == "completed" else result["status"],
                            "verdict": result.get("verdict", ""), "run_id": result.get("latest_run", "")})
        # A single read probe samples a broad standard item. It cannot mark
        # that item fully tested or imply application-wide compliance.
        projection["checklist_progress"][PurePosixPath(path).stem] = {"total": len(known_ids), "completed": 0, "skipped": 0, "not_applicable": 0}
        records.append({"path": path, "sha256": source["sha256"], "total_imported_items": len(known_ids),
                        "fully_completed": 0, "unmapped_items": len(known_ids - mapped), "samples": samples})
    projection["metadata"]["tsunami_checklist_samples"] = records
    from jsonschema import Draft7Validator, FormatChecker
    schema = parse(read_local(root.resolve(strict=True), "schemas/engagement.json"))
    reject_refs(schema)
    if list(Draft7Validator(schema, format_checker=FormatChecker()).iter_errors(projection)):
        raise ValueError("coverage projection does not match engagement schema")
    return {"version": "tsunami.checklist-coverage/v1", "engagement": projection,
            "mapping_sha256": hashlib.sha256(raw).hexdigest(), "sources": records,
            "runtime_permissions_changed": False, "scripts_executed": False,
            "limitations": ["Imported IDs are reference data, never executable instructions.",
                            "All mappings are partial samples; fully completed standard items remain zero.",
                            "Normative checklist completeness and procedure mappings still need security review."]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=["inventory", "validate-export", "coverage"])
    parser.add_argument("--toolkit", type=Path, required=True)
    parser.add_argument("--bundle", type=Path)
    parser.add_argument("--mapping", type=Path)
    parser.add_argument("--out", type=Path, required=True)
    args = parser.parse_args()
    if args.command == "inventory":
        report = inspect(args.toolkit)
    elif args.command == "validate-export":
        if not args.bundle:
            parser.error("--bundle is required for validate-export")
        report = validate_export(args.toolkit, args.bundle)
    else:
        if not args.bundle or not args.mapping:
            parser.error("--bundle and --mapping are required for coverage")
        report = checklist_coverage(args.toolkit, args.bundle, args.mapping)
    # Outputs must not overwrite the local methodology source, including aliases.
    root = args.toolkit.resolve(strict=True)
    if args.out.resolve().is_relative_to(root):
        parser.error("write results outside the toolkit repository")
    if any(args.out.resolve() == source.resolve() for source in [args.bundle, args.mapping] if source is not None):
        parser.error("write results separately from input records")
    save(args.out, report)
    result = report.get("counts", report)
    if args.command == "coverage":
        result = {"sources": len(report["sources"]), "fully_completed": 0, "runtime_permissions_changed": False}
    print(json.dumps(result, sort_keys=True))


if __name__ == "__main__":
    try:
        main()
    except (ValueError, OSError, KeyError) as exc:
        raise SystemExit("Toolkit data check failed: " + str(exc))

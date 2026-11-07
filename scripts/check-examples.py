#!/usr/bin/env python3
"""Verify checked-in example bytes/provenance; optionally check local toolkit schemas."""
import argparse
import base64
import hashlib
import json
from pathlib import Path

from toolkit import parse, read_local, validate_export
from showcase import blocked_capture_ids


def check_showcase(root):
    index = parse((root / "showcase.json").read_bytes())
    expected = {"vampi-vulnerable": "confirmed", "vampi-fixed": "no_issue_observed", "stop-lab-run": "stopped",
                "property-vulnerable": "confirmed", "property-filtered": "no_issue_observed",
                "property-ambiguous": "inconclusive", "property-stop": "stopped"}
    if index["version"] != "tsunami.showcase/v1" or index["passed"] is not True or index["phase"] != "complete" or index["mode"] != "scripted_mecatl" or index["approval"] != "automated_owned_lab" or index["external_model_calls"] != 0:
        raise ValueError("showcase execution claim differs")
    if len(index["cases"]) != len(expected) or {c["application"] for c in index["cases"]} != set(expected):
        raise ValueError("showcase cases incomplete or duplicated")
    observed = {}
    for family in ("object", "property"):
        campaign = parse(read_local(root, family + "/summary.json"))
        if campaign["binary_sha256"] != index["binary_sha256"] or campaign["control_image_id"] != index["control_image_id"] or campaign["external_model_calls"] != 0 or not all(c["passed"] for c in campaign["checks"]):
            raise ValueError("showcase campaign/build differs")
        for check in campaign["checks"]:
            if check["application"] in observed:
                raise ValueError("duplicate showcase campaign case")
            observed[check["application"]] = check
    for case in index["cases"]:
        bundle = parse(read_local(root, case["bundle"]))
        read_local(root, case["report"])
        engagement = bundle["view"]["engagement"]
        check = observed[case["application"]]
        if (case["verdict"] != expected[case["application"]] or engagement["id"] != case["engagement_id"]
                or case["run_id"] not in engagement["run_ids"] or engagement["application"]["id"] != case["application"]
                or case["finding_candidates"] != len(bundle["finding_candidates"])
                or case["finding_candidates"] != (1 if case["verdict"] == "confirmed" else 0)
                or case["reviewed_findings"] != 0 or bundle["toolkit_findings"]
                or not case["blocked_capture_ids"] or case["blocked_capture_ids"] != blocked_capture_ids(bundle)
                or case["cleanup"] != "api_resources_absent"):
            raise ValueError("showcase case/evidence index differs")
        if check["run_id"] != case["run_id"] or check["engagement_id"] != case["engagement_id"] or check["after_cleanup"]["cleanup"] != case["cleanup"]:
            raise ValueError("showcase case/campaign identity differs")
        if case["verdict"] == "stopped":
            held, cancelled, final = check["target_held"], check["target_cancelled"], check["target_final"]
            if (check["after_cleanup"]["permission"] != "revoked" or held["active"] != 1 or cancelled["active"] != 0
                    or cancelled["probe_cancelled"] != held["probe_cancelled"] + 1
                    or final["requests"] != held["requests"] or final["probe_completed"] != held["probe_completed"]
                    or check["runtime"] != {"handler": "runsc", "type": "io.containerd.runsc.v1"}
                    or not check["worker_image_id"].startswith("sha256:")):
                raise ValueError("showcase held request Stop evidence differs")
        elif check["result_status"]["assessment"]["verdict"] != case["verdict"]:
            raise ValueError("showcase actual verdict differs")
    network = parse(read_local(root, index["network_record"]))
    if [c["role"] for c in network["checks"]] != ["positive-before", "runner", "positive-after"]:
        raise ValueError("showcase network controls incomplete")
    probes = {(destination, mode, phase) for destination in network["destinations"]
              for mode in ("literal", "resolve", "connect-to") for phase in ("startup", "settled", "repeat")}
    if len(probes) != 54:
        raise ValueError("showcase network destination count differs")
    for check in network["checks"]:
        if check["passed"] is not True or check["runtime_class"] != "gvisor" or check["actual_runtime"] != {"handler": "runsc", "type": "io.containerd.runsc.v1"} or not check["image_id"].startswith("sha256:"):
            raise ValueError("showcase actual sandbox observation differs")
        lines = read_local(root, "network/" + check["role"] + ".log").splitlines()
        if lines[-1] != b"PASS no API token, raw socket denied, root filesystem write denied" or not lines[-2].startswith(b"Kernel: "):
            raise ValueError("showcase sandbox smoke checks missing")
        rows = [parse(line) for line in lines[:-2]]
        attempts = [r for r in rows if r["destination"] != "broker"]
        broker = [r for r in rows if r["destination"] == "broker"]
        if len(attempts) != 54 or {(r["destination"], r["mode"], r["phase"]) for r in attempts} != probes or any(r.get("passed") is not True for r in attempts):
            raise ValueError("showcase network attempts incomplete or failed")
        outcomes = {r["outcome"] for r in attempts}
        if (check["role"] == "runner" and outcomes != {"timeout"}) or (check["role"] != "runner" and outcomes != {"reachable"}):
            raise ValueError("showcase network result/control differs")
        if len(broker) != 3 or {r["phase"] for r in broker} != {"startup", "settled", "repeat"} or any(r["outcome"] != "reachable" for r in broker):
            raise ValueError("showcase broker reachability incomplete")
    print("PASS showcase index, seven unreviewed outcomes and 54 selected network denials with reachable before/after controls.")


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


def check_ui(root):
    qualification = parse((root / "summary.json").read_bytes())
    expected = {"vampi-vulnerable": "confirmed", "vampi-fixed": "no_issue_observed", "stop-lab-run": "stopped"}
    cases = qualification["cases"]
    if qualification["version"] != "tsunami.ui-qualification/v1" or qualification["passed"] is not True or qualification["external_model_calls"] != 0 or qualification["approval"] != "automated_owned_lab" or len(cases) != 3 or {c["application"] for c in cases} != set(expected):
        raise ValueError("UI qualification incomplete or execution claim differs")
    for case in cases:
        bundle = parse(read_local(root, case["bundle"]))
        downloaded = parse(read_local(root, "downloads/tsunami-" + case["engagement_id"] + "-bundle.json"))
        if bundle != downloaded or bundle["view"]["engagement"]["id"] != case["engagement_id"] or case["run_id"] not in bundle["view"]["engagement"]["run_ids"] or case["verdict"] != expected[case["application"]] or case["passed"] is not True or bundle["toolkit_findings"] or len(blocked_capture_ids(bundle)) != 1 or case["candidates"] != len(bundle["finding_candidates"]):
            raise ValueError("UI downloaded evidence/case index differs")
        if case["application"] == "vampi-fixed" and case.get("lost_start_response_recovered") is not True:
            raise ValueError("UI lost Start qualification missing")
        if case["application"] == "stop-lab-run":
            if case.get("stop_during_pending_export") is not True or case["status"]["permission"] != "revoked" or case["status"]["local_operations"] != 0:
                raise ValueError("UI independent Stop qualification missing")
        elif case["status"]["assessment"]["verdict"] != case["verdict"]:
            raise ValueError("UI actual assessment verdict differs")
    requests = parse((root / "operator-requests.json").read_bytes())
    if not requests or not all(r["same_origin"] for r in requests) or len([r for r in requests if r["method"] == "POST" and r["path"].endswith("/runs")]) != 3:
        raise ValueError("UI same-origin or single-attempt request qualification differs")
    print("PASS retained UI downloads, three browser outcomes, lost Start recovery and independent Stop observations.")


def check_onboarding(root):
    summary = parse(read_local(root, "summary.json"))
    if summary["passed"] is not True or summary.get("includes_onboarding") is not True or summary["external_model_calls"] != 0 or summary["approval"] != "automated_owned_lab" or len(summary["cases"]) != 4:
        raise ValueError("onboarding qualification incomplete")
    cases = summary["onboarding"]
    if len(cases) != 3 or {c["case"] for c in cases} != {"missing_backend", "unsupported_login", "owned_onboarding"}:
        raise ValueError("onboarding question controls incomplete")
    for case in cases:
        if case["case"] != "owned_onboarding" and case["target_requests"] != 0:
            raise ValueError("metadata/unsupported login dispatched work")
    owned = next(c for c in cases if c["case"] == "owned_onboarding")
    session = parse(read_local(root, "onboarding-session.json"))
    bundle = parse(read_local(root, owned["bundle"]))
    if (owned["passed"] is not True or owned["validation_requests"] != 4 or owned["actual_runtime"] != {"handler": "runsc", "type": "io.containerd.runsc.v1"}
            or not owned["worker_image_id"].startswith("sha256:") or session["id"] != owned["setup_id"]
            or session["validation_id"] != owned["validation_id"] or session["state"] != "ready_for_approval"
            or session.get("engagement_id") or session["advisor_mode"] != "checked_workflow" or not session.get("validated_at")
            or session["plan"] != bundle["view"]["engagement"]["application"] or session["plan"]["min_interval_ms"] != 1000
            or bundle["toolkit_findings"] or len(bundle["finding_candidates"]) != 1 or len(blocked_capture_ids(bundle)) != 1
            or owned["status"]["assessment"]["verdict"] != "confirmed" or owned["status"]["assessment"]["validation"]["baselines_verified"] != 6):
        raise ValueError("onboarding baseline/runtime/assessment provenance differs")
    for case in summary["cases"]:
        downloaded = parse(read_local(root, "downloads/tsunami-" + case["engagement_id"] + "-bundle.json"))
        actual = parse(read_local(root, case["bundle"] if "bundle" in case else case["application"] + "/bundle.json"))
        if downloaded != actual or actual["view"]["engagement"]["id"] != case["engagement_id"] or case["run_id"] not in actual["view"]["engagement"]["run_ids"] or case["passed"] is not True:
            raise ValueError("onboarding downloaded export identity differs")
    print("PASS onboarding readiness controls, four baseline reads, inferred plan and actual gVisor assessment evidence.")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--toolkit", type=Path, help="explicit local reference checkout; never downloaded or executed")
    args = parser.parse_args()
    paths = [Path("examples") / family / app / "bundle.json" for family in ["application-adapters", "tool-execution"] for app in ["vampi-vulnerable", "vampi-fixed"]]
    paths.extend(Path("examples/operator-journey") / app / "bundle.json" for app in ["vampi-vulnerable", "vampi-fixed", "stop-lab-run"])
    paths.extend(Path("examples/property-access") / app / "bundle.json" for app in ["property-vulnerable", "property-filtered", "property-ambiguous", "property-stop"])
    paths.extend(Path("examples/binding-preparation/runtime/object") / app / "bundle.json" for app in ["vampi-vulnerable", "vampi-fixed", "stop-lab-run"])
    paths.extend(Path("examples/binding-preparation/runtime/property") / app / "bundle.json" for app in ["property-vulnerable", "property-filtered", "property-ambiguous", "property-stop"])
    showcase = Path("examples/showcase")
    paths.extend(showcase / family / app / "bundle.json" for family, apps in [("object", ["vampi-vulnerable", "vampi-fixed", "stop-lab-run"]), ("property", ["property-vulnerable", "property-filtered", "property-ambiguous", "property-stop"])] for app in apps)
    check_showcase(showcase)
    ui = Path("examples/local-ui")
    paths.extend(ui / app / "bundle.json" for app in ["vampi-vulnerable", "vampi-fixed", "stop-lab-run"])
    check_ui(ui)
    onboarding = Path("examples/onboarding")
    paths.extend(onboarding / app / "bundle.json" for app in ["vampi-vulnerable", "vampi-fixed", "stop-lab-run", "onboarded-vampi"])
    check_onboarding(onboarding)
    count = 0
    for path in paths:
        count += check_bundle(path)
        if args.toolkit:
            validate_export(args.toolkit, path)
    archive = parse(Path("examples/inference-diagnostics/archive-sha256.json").read_bytes())
    binding_root = Path("examples/binding-preparation")
    qualification = parse((binding_root / "qualification.json").read_bytes())
    prepared_ids = {p.name for p in (binding_root / "prepared").iterdir() if p.is_dir()}
    entries = qualification["prepared_applications"]
    if len(entries) != len(prepared_ids) or {e["application"] for e in entries} != prepared_ids:
        raise ValueError("prepared qualification index is incomplete or duplicated")
    for entry in qualification["prepared_applications"]:
        directory = binding_root / "prepared" / entry["application"]
        raw = (directory / "plan.json").read_bytes()
        manifest = parse((directory / "manifest.json").read_bytes())
        plan = parse(raw)
        bundle = parse((binding_root / "runtime" / entry["runtime_family"] / entry["application"] / "bundle.json").read_bytes())
        actual = dict(bundle["view"]["engagement"]["application"])
        if not actual.get("credential_set_id"):
            raise ValueError("runtime application lacks credential binding")
        actual["credential_set_id"] = ""
        if plan != actual or plan["credential_set_id"] != "" or manifest["status"] != "pending_operator_registration_and_scope_approval" or manifest["target_requests"] != 0 or hashlib.sha256(raw).hexdigest() != manifest["plan_sha256"] or manifest["plan_sha256"] != entry["plan_sha256"] or manifest["input_sha256"] != entry["input_sha256"] or bundle["view"]["engagement"]["id"] != entry["engagement_id"] or entry["run_id"] not in bundle["view"]["engagement"]["run_ids"]:
            raise ValueError("prepared application/runtime provenance differs")
    for name, expected in archive.items():
        if hashlib.sha256(read_local(Path.cwd(), name)).hexdigest() != expected:
            raise ValueError("historical trial bytes changed: " + name)
    print(f"PASS {len(paths)} materialized bundles, {count} artifacts and {len(archive)} historical trial hashes.")
    print("Actual local toolkit schemas: " + ("PASS" if args.toolkit else "not requested; run make verify-toolkit separately."))


if __name__ == "__main__":
    main()

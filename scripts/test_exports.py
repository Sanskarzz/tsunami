import base64
import copy
import hashlib
import json
from pathlib import Path
import tempfile
import unittest

import toolkit


class ExportIntegrity(unittest.TestCase):
    def fixture(self, root):
        for directory in ["schemas", "scripts", "checklists", "docs/agent_prompts"]:
            (root / directory).mkdir(parents=True)
        # Original minimal test schemas exercise references/provenance checks;
        # actual toolkit contracts are validated separately in the lab campaign.
        for name in ["engagement", "evidence", "finding"]:
            schema = {"$schema": "http://json-schema.org/draft-07/schema#", "$id": name + ".json", "type": "object"}
            if name == "finding":
                schema["properties"] = {"evidence": {"type": "array", "items": {"$ref": "evidence.json"}}}
            (root / "schemas" / (name + ".json")).write_text(json.dumps(schema))
        (root / "docs/agent_roles.md").write_text("| **Orchestrator** | fixture |\n")
        (root / "checklists/api.json").write_text('{"tests":[{"id":"API1:2023"},{"id":"API2:2023"}]}')
        captured = "2026-11-03T00:00:00Z"
        data = json.dumps({"engagement_id": "eng-local", "run_id": "run-local", "capture": {"id": "cap-local", "started_at": captured}}).encode()
        digest = hashlib.sha256(data).hexdigest()
        artifact = {"id": "E-" + digest, "sha256": digest, "path": "evidence/" + digest + ".json", "data_base64": base64.b64encode(data).decode(), "run_id": "run-local", "capture_id": "cap-local"}
        evidence = {"id": artifact["id"], "path": artifact["path"], "finding_id": "F-local", "captured_at": captured}
        review = {"decision": "confirm", "purpose": "automated_lab_test", "candidate_digest": "a" * 64, "cvss_score": 6.5, "cvss_vector": "test-vector", "severity": "Medium"}
        finding = {"id": "F-local", "engagement_id": "eng-local", "target": "http://owned.invalid", "tags": ["automated_lab_test"], "evidence": [evidence], **{key: review[key] for key in ["cvss_score", "cvss_vector", "severity"]}}
        bundle = {"version": "tsunami.engagement-export/v2", "synthetic_only": True,
                  "view": {"engagement": {"id": "eng-local", "target": "http://owned.invalid", "scope_digest": "scope-local"}, "checklist": [{"procedure": "test/object-access/v1", "status": "completed", "verdict": "confirmed", "latest_run": "run-local"}]},
                  "toolkit_engagement": {"id": "eng-local", "findings": [{"id": "F-local"}], "checklist_progress": {}, "metadata": {}},
                  "artifacts": [artifact], "finding_candidates": [{"id": "F-local", "status": "reviewed_confirmed", "candidate_digest": "a" * 64, "review": review, "run_ids": ["run-local"], "evidence_ids": [artifact["id"]]}],
                  "toolkit_findings": [finding], "toolkit_evidence": [evidence]}
        (root / "evidence").mkdir()
        (root / "findings").mkdir()
        (root / artifact["path"]).write_bytes(data)
        (root / "evidence" / (artifact["id"] + ".meta.json")).write_text(json.dumps(evidence))
        (root / "findings/finding_F-local.json").write_text(json.dumps(finding))
        return bundle

    def test_review_and_artifact_substitution_are_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            bundle = self.fixture(root)
            path = root / "bundle.json"
            path.write_text(json.dumps(bundle))
            self.assertEqual(toolkit.validate_export(root, path)["reviewed_findings"], 1)
            for change in ["bytes", "path", "rating", "tenant", "review", "index", "evidence"]:
                bad = copy.deepcopy(bundle)
                if change == "bytes":
                    bad["artifacts"][0]["data_base64"] = base64.b64encode(b"{}").decode()
                elif change == "path":
                    bad["artifacts"][0]["path"] = "../private.key"
                elif change == "rating":
                    bad["toolkit_findings"][0]["cvss_score"] = 10
                elif change == "tenant":
                    bad["toolkit_findings"][0]["engagement_id"] = "other-tenant"
                elif change == "review":
                    bad["finding_candidates"][0]["review"]["candidate_digest"] = "b" * 64
                elif change == "index":
                    bad["toolkit_engagement"]["findings"] = []
                else:
                    bad["toolkit_evidence"] = []
                path.write_text(json.dumps(bad))
                with self.assertRaises(ValueError, msg=change):
                    toolkit.validate_export(root, path)

    def test_partial_mapping_does_not_claim_full_standard_completion(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            bundle = self.fixture(root)
            path = root / "bundle.json"
            path.write_text(json.dumps(bundle))
            source = root / "checklists/api.json"
            mapping = {"version": "tsunami.checklist-map/v1", "engagement_id": "eng-local", "scope_digest": "scope-local", "sources": [{"path": "checklists/api.json", "sha256": hashlib.sha256(source.read_bytes()).hexdigest(), "mappings": [{"item_id": "API1:2023", "procedure": "test/object-access/v1", "coverage": "partial"}]}]}
            mapping_path = root / "mapping.json"
            mapping_path.write_text(json.dumps(mapping))
            result = toolkit.checklist_coverage(root, path, mapping_path)
            self.assertEqual(result["sources"][0]["samples"][0]["sample_state"], "sampled")
            self.assertEqual(result["engagement"]["checklist_progress"]["api"]["completed"], 0)
            mapping["sources"][0]["mappings"][0]["coverage"] = "complete"
            mapping_path.write_text(json.dumps(mapping))
            with self.assertRaises(ValueError):
                toolkit.checklist_coverage(root, path, mapping_path)


if __name__ == "__main__":
    unittest.main()

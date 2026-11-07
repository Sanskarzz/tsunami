import base64
import json
import unittest
from contextlib import contextmanager
import urllib.error

from showcase import Lab, expiry_hint, credential_fingerprint, owned_scope, active_test_pod, blocked_capture_ids


class ShowcaseChecks(unittest.TestCase):
    def test_maintenance_refuses_active_preflight_without_hiding_inspection_errors(self):
        class FakeLab:
            include_property = False
            state = "validating"
            def obj(self, *args): return {"data": {"admin-a": base64.b64encode(b"synthetic-operator").decode()}}
            @contextmanager
            def forward(self): yield "http://127.0.0.1"
            def read(self, origin, path, token):
                if path == "/v1/engagements": return []
                if path == "/v1/onboarding":
                    if isinstance(self.state, int): raise urllib.error.HTTPError(origin + path, self.state, "fixture", {}, None)
                    return [{"state": self.state}]
                return {"items": []}
        lab = FakeLab()
        with self.assertRaisesRegex(RuntimeError, "active setup validation"):
            Lab.permission_idle(lab)
        lab.state = "stopped"
        Lab.permission_idle(lab)
        lab.state = 404
        Lab.permission_idle(lab)
        lab.state = 401
        with self.assertRaisesRegex(RuntimeError, "cannot inspect setup"):
            Lab.permission_idle(lab)

    def test_successful_capture_may_omit_false_blocked_flag(self):
        def artifact(identifier, capture):
            return {"capture_id": identifier, "data_base64": base64.b64encode(json.dumps({"capture": capture}).encode()).decode()}
        bundle = {"artifacts": [artifact("allowed", {"status_code": 200}), artifact("denied", {"blocked": True}), artifact("explicit-false", {"blocked": False})]}
        self.assertEqual(blocked_capture_ids(bundle), ["denied"])
        self.assertEqual(blocked_capture_ids({"artifacts": bundle["artifacts"][:1]}), [])

    def test_permanent_verifier_service_is_not_a_live_assessment(self):
        pod = {"status": {"phase": "Running"}, "metadata": {"labels": {"tsunami-role": "verifier"}, "ownerReferences": [{"kind": "ReplicaSet"}]}}
        self.assertFalse(active_test_pod(pod))
        pod["metadata"]["ownerReferences"] = [{"kind": "Job"}]
        self.assertTrue(active_test_pod(pod))
        pod["status"]["phase"] = "Succeeded"
        self.assertFalse(active_test_pod(pod))
        pod["status"]["phase"] = "Pending"; pod["metadata"]["labels"]["tsunami-role"] = "runner"
        self.assertTrue(active_test_pod(pod))

    def token(self, expiry):
        payload = base64.urlsafe_b64encode(json.dumps({"exp": expiry}).encode()).rstrip(b"=").decode()
        return "header." + payload + ".not-a-verified-signature"

    def test_expiry_is_only_a_hint_and_requires_enough_demo_time(self):
        hint = expiry_hint(self.token(5000), 1000, 900)
        self.assertEqual(hint["remaining_seconds"], 4000)
        self.assertEqual(hint["scope"], "unsigned_expiry_hint_not_account_validation")
        for bad in [self.token(900), self.token(1100), self.token("5000"), self.token(True), self.token(float("nan")), self.token(10**100), "opaque-token", "bad.bad.bad"]:
            with self.assertRaises(RuntimeError):
                expiry_hint(bad, 1000, 900)

    def test_fixed_owned_scope_refuses_arbitrary_targets_and_extra_ips(self):
        plan = {"origin": "http://vampi-fixed.tsunami-app-lab.svc.cluster.local:5000", "approved_addresses": ["10.96.0.25"], "synthetic_data_only": True, "version": "http-object-access/v2"}
        owned_scope(plan, "tsunami-app-lab", "vampi-fixed", 5000, "10.96.0.25")
        for change in [{"origin": "https://customer.example.test"}, {"approved_addresses": ["10.96.0.25", "1.1.1.1"]}, {"synthetic_data_only": False}, {"version": "raw-tcp"}]:
            with self.assertRaises(RuntimeError):
                owned_scope(dict(plan, **change), "tsunami-app-lab", "vampi-fixed", 5000, "10.96.0.25")

    def test_fingerprint_binds_tenant_application_and_actual_credential_set(self):
        keys = {"owner": "synthetic-a", "peer": "synthetic-b"}
        baseline = credential_fingerprint("demo-a", "app", keys)
        self.assertEqual(baseline, credential_fingerprint("demo-a", "app", dict(reversed(list(keys.items())))))
        for tenant, app, values in [("demo-b", "app", keys), ("demo-a", "other", keys), ("demo-a", "app", dict(keys, peer="new"))]:
            self.assertNotEqual(baseline, credential_fingerprint(tenant, app, values))

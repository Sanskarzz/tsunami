import importlib.util
from pathlib import Path
import unittest
import json
import tempfile
from application_registry import write_private_json


spec = importlib.util.spec_from_file_location("setup_vampi", Path(__file__).with_name("setup-vampi.py"))
setup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(setup)


class ApplicationRefreshTest(unittest.TestCase):
    def test_registry_write_is_private_and_replaces_a_local_symlink(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            original = root / "original"
            original.write_text("do not change")
            destination = root / "config.json"
            destination.symlink_to(original)
            write_private_json(destination, {"credentials": "synthetic-test-only"})
            self.assertFalse(destination.is_symlink())
            self.assertEqual(original.read_text(), "do not change")
            self.assertEqual(destination.stat().st_mode & 0o777, 0o600)
            self.assertEqual(json.loads(destination.read_text()), {"credentials": "synthetic-test-only"})
            self.assertFalse(list(root.glob(".registry-*")))

    def test_refresh_preserves_other_plans_and_other_tenant_credentials(self):
        old = {"tenant": "demo-a", "plan": {"id": "vampi-vulnerable"}, "credentials": {"owner": "old"}}
        other = {"tenant": "other-tenant", "plan": {"id": "vampi-vulnerable"}, "credentials": {"owner": "other"}}
        stop = {"tenant": "demo-a", "plan": {"id": "stop-lab-run"}, "credentials": {"owner": "stop"}}
        fresh = {"tenant": "demo-a", "plan": {"id": "vampi-vulnerable"}, "credentials": {"owner": "fresh"}}
        existing = [old, other, stop]
        merged = setup.merge_applications(existing, [fresh])
        self.assertEqual(merged, [other, stop, fresh])
        self.assertEqual(existing, [old, other, stop])

    def test_refresh_refuses_an_oversized_registry(self):
        existing = [{"tenant": "demo-a", "plan": {"id": f"other-{n}"}} for n in range(16)]
        with self.assertRaises(RuntimeError):
            setup.merge_applications(existing, [{"tenant": "demo-a", "plan": {"id": "vampi-fixed"}}])

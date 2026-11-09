"""Reject misleading provenance in the retained D9 qualification record."""
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import shutil
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("example_checks", Path(__file__).with_name("check-examples.py"))
checks = importlib.util.module_from_spec(spec)
spec.loader.exec_module(checks)


class DemoRecordTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / "record"
        shutil.copytree("examples/demo-journey", self.root)

    def verify(self):
        with contextlib.redirect_stdout(io.StringIO()):
            return checks.check_demo(self.root)

    def mutate(self, name, change):
        path = self.root / name
        record = json.loads(path.read_text())
        change(record)
        path.write_text(json.dumps(record))

    def test_retained_campaign_and_every_materialized_artifact(self):
        paths = self.verify()
        self.assertEqual(len(paths), 6)
        self.assertEqual(sum(checks.check_bundle(path) for path in paths), 46)

    def test_late_rehearsal_and_changed_worker_build_are_refused(self):
        for field, bad in [("elapsed_seconds", 304), ("passed", False)]:
            with self.subTest(field=field):
                path = self.root / "browser/summary.json"
                original = path.read_bytes()
                self.mutate("browser/summary.json", lambda d: d["rehearsal"].update({field: bad}))
                with self.assertRaises(ValueError):
                    self.verify()
                path.write_bytes(original)
        self.mutate("build.json", lambda d: d.update(tools_image_id="sha256:" + "0" * 64))
        with self.assertRaises(ValueError):
            self.verify()

    def test_foreign_retest_link_and_lost_history_are_refused(self):
        self.mutate("browser/linked-retest.json", lambda d: d["retest"]["engagement"].update(source_engagement_id="0" * 32))
        with self.assertRaises(ValueError):
            self.verify()
        shutil.copyfile("examples/demo-journey/browser/linked-retest.json", self.root / "browser/linked-retest.json")
        retained = json.loads((self.root / "before.json").read_text())["run_ids"][0]
        self.mutate("after.json", lambda d: d.update(run_ids=[r for r in d["run_ids"] if r != retained]))
        with self.assertRaises(ValueError):
            self.verify()


if __name__ == "__main__":
    unittest.main()

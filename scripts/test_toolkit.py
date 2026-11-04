import json
import importlib.util
import http.server
from pathlib import Path
import tempfile
import threading
import unittest
import urllib.error
import urllib.request

import toolkit


class ToolkitDataBoundaries(unittest.TestCase):
    def test_collector_does_not_forward_operator_header_on_redirect(self):
        spec = importlib.util.spec_from_file_location("engagement_collector", Path(__file__).with_name("verify-engagement.py"))
        collector = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(collector)
        received = []

        class Tripwire(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                received.append(self.headers.get("Authorization"))
                self.send_response(200)
                self.end_headers()

            def log_message(self, *args):
                pass

        with http.server.ThreadingHTTPServer(("127.0.0.1", 0), Tripwire) as target:
            thread = threading.Thread(target=target.serve_forever, daemon=True)
            thread.start()

            class Redirect(Tripwire):
                def do_GET(self):
                    self.send_response(302)
                    self.send_header("Location", f"http://127.0.0.1:{target.server_port}/private")
                    self.end_headers()

            with http.server.ThreadingHTTPServer(("127.0.0.1", 0), Redirect) as redirect:
                server = threading.Thread(target=redirect.serve_forever, daemon=True)
                server.start()
                try:
                    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), collector.NoRedirect())
                    request = urllib.request.Request(f"http://127.0.0.1:{redirect.server_port}/export", headers={"Authorization": "Bearer synthetic-collector-credential"})
                    with self.assertRaises(urllib.error.HTTPError):
                        opener.open(request, timeout=2)
                    self.assertEqual(received, [])
                finally:
                    redirect.shutdown()
                    server.join()
            target.shutdown()
            thread.join()

    def test_leaf_inventory_does_not_count_parent_categories(self):
        data = {"categories": [{"id": "API", "tests": [{"id": "API1:2023"}, {"id": "API2:2023"}]}]}
        self.assertEqual(toolkit.leaves(data), ["API1:2023", "API2:2023"])

    def test_file_and_directory_aliases_are_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "secret").write_text("private")
            (root / "alias").symlink_to(root / "secret")
            (root / "directory").symlink_to(root, target_is_directory=True)
            for path in ["../secret", "alias", "directory/secret"]:
                with self.assertRaises(ValueError):
                    toolkit.read_local(root, path)

    def test_duplicate_members_and_remote_schema_refs_are_rejected(self):
        with self.assertRaises(ValueError):
            toolkit.parse('{"authorized":false,"authorized":true}')
        with self.assertRaises(ValueError):
            toolkit.reject_refs({"properties": {"target": {"$ref": "http://metadata.invalid/schema"}}})

    def test_inventory_never_executes_scripts_or_copies_their_contents(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for name in ["scripts/web", "schemas", "checklists", "docs/agent_prompts"]:
                (root / name).mkdir(parents=True)
            marker = root / "executed"
            script = f"#!/bin/sh\ntouch {marker}\n# sensitive-toolkit-content-canary\n"
            (root / "scripts/web/test.sh").write_text(script)
            (root / "schemas/engagement.json").write_text('{"type":"object"}')
            (root / "checklists/tests.json").write_text('{"tests":[{"id":"T-1"}]}')
            (root / "docs/agent_roles.md").write_text('| **Orchestrator** | role |\n')
            report = toolkit.inspect(root)
            self.assertEqual(report["counts"]["scripts"], 1)
            self.assertFalse(marker.exists())
            self.assertNotIn("sensitive-toolkit-content-canary", json.dumps(report))
            self.assertFalse(report["scripts_executed"])
            self.assertEqual(report["scripts"][0]["state"], "adapter_not_implemented")


if __name__ == "__main__":
    unittest.main()

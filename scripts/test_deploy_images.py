import json
import os
from pathlib import Path
import subprocess
import unittest

from deploy_images import image_from_control, with_image


class SelectedImages(unittest.TestCase):
    def controller(self):
        return {"kind": "Deployment", "metadata": {"name": "tsunami-control"}, "spec": {"template": {"spec": {
            "containers": [{"name": "control", "image": "tsunami:dev", "env": [{"name": "TSUNAMI_RUNTIME_CLASS", "value": "old-runtime"}]}]}}}}

    def test_renderer_binds_native_workloads_and_worker_image(self):
        control = self.controller()
        fixture = self.controller()
        fixture["metadata"]["name"] = "tsunami-fixed"
        third_party = {"kind": "Pod", "spec": {"containers": [{"name": "app", "image": "customer-app:pinned"}]}}
        raw = "\n".join(json.dumps(item) for item in [control, fixture, third_party])
        script = Path(__file__).with_name("render-lab.py")
        env = dict(os.environ, IMAGE="tsunami:review", RUNTIME_CLASS="gvisor")
        result = subprocess.run(["python3", "-B", str(script)], input=raw, text=True, capture_output=True, env=env, check=True)
        items = json.loads(result.stdout)["items"]
        for item in items[:2]:
            self.assertEqual(item["spec"]["template"]["spec"]["containers"][0]["image"], "tsunami:review")
        selected = items[0]["spec"]["template"]["spec"]["containers"][0]
        values = {e["name"]: e["value"] for e in selected["env"]}
        self.assertEqual(values["TSUNAMI_IMAGE"], "tsunami:review")
        self.assertEqual(values["TSUNAMI_RUNTIME_CLASS"], "gvisor")
        self.assertEqual(items[2], third_party)
        self.assertEqual(image_from_control(items[0]), "tsunami:review")

    def test_verifier_binding_does_not_change_source_or_other_images(self):
        manifest = {"kind": "Job", "spec": {"template": {"spec": {"containers": [
            {"image": "tsunami:dev"}, {"image": "customer-app:pinned"}]}}}}
        bound = with_image(manifest, "registry.example.test/tsunami:review")
        images = [c["image"] for c in bound["spec"]["template"]["spec"]["containers"]]
        self.assertEqual(images, ["registry.example.test/tsunami:review", "customer-app:pinned"])
        self.assertEqual(manifest["spec"]["template"]["spec"]["containers"][0]["image"], "tsunami:dev")
        control = self.controller()
        control["spec"]["template"]["spec"]["containers"][0]["env"].append({"name": "TSUNAMI_IMAGE", "value": "stale:image"})
        with self.assertRaises(RuntimeError):
            image_from_control(control)

    def test_make_propagates_custom_native_and_tool_tags(self):
        root = Path(__file__).resolve().parent.parent
        result = subprocess.check_output(["make", "-n", "deploy", "IMAGE=tsunami:review", "RUNTIME_CLASS=gvisor", "CONTEXT=kind-tsunami-lab"], text=True, cwd=root)
        self.assertIn('IMAGE="tsunami:review"', result)
        self.assertIn("kind load docker-image tsunami:review", result)
        result = subprocess.check_output(["make", "-n", "tools-image", "TOOL_IMAGE=tsunami-tools:review"], text=True, cwd=root)
        self.assertIn("-t tsunami-tools:review", result)


if __name__ == "__main__":
    unittest.main()

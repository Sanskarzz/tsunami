import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import deployment_guard as guard


READ_COMMAND = guard.read_command
REPOSITORY = Path(__file__).resolve().parent.parent


class SecuredLabIdentity(unittest.TestCase):
    def setUp(self):
        self.folder = tempfile.TemporaryDirectory()
        self.addCleanup(self.folder.cleanup)
        self.root = Path(self.folder.name)
        self.config = self.root / ".local/sandbox.kubeconfig"
        self.config.parent.mkdir()
        self.config.write_text("synthetic config; command output is stubbed")
        self.config.chmod(0o600)
        self.enterContext(patch.object(guard, "ROOT", self.root))
        self.enterContext(patch.dict(os.environ, {"CONTEXT": guard.CONTEXT, "RUNTIME_CLASS": "gvisor", "KUBECONFIG": str(self.config)}))
        previous = Path.cwd()
        os.chdir(self.root)
        self.addCleanup(os.chdir, previous)
        self.responses = {
            "config": {"clusters": [{"name": guard.CONTEXT, "cluster": {"server": "https://127.0.0.1:12345", "certificate-authority-data": "synthetic-ca"}}],
                       "contexts": [{"name": guard.CONTEXT, "context": {"cluster": guard.CONTEXT, "user": guard.CONTEXT}}],
                       "users": [{"name": guard.CONTEXT, "user": {"client-certificate-data": "synthetic-cert", "client-key-data": "synthetic-key"}}]},
            "docker": [{"Config": {"Labels": {"io.x-k8s.kind.cluster": "tsunami-lab"}}, "State": {"Running": True},
                        "NetworkSettings": {"Ports": {"6443/tcp": [{"HostIp": "127.0.0.1", "HostPort": "12345"}]}}}],
            "nodes": {"items": [{"metadata": {"name": guard.NODE}, "status": {"conditions": [{"type": "Ready", "status": "True"}]}}]},
            "runtime": {"handler": "runsc"}, "policy": {"data": {"enable-policy": "always"}},
            "cilium": {"status": {"desiredNumberScheduled": 1, "numberReady": 1}},
        }
        self.reader = self.enterContext(patch.object(guard, "read_command", side_effect=self.read))

    def read(self, argv):
        if argv[:2] == ["docker", "inspect"]:
            return json.dumps(self.responses["docker"])
        self.assertEqual(argv[:5], ["kubectl", "--kubeconfig", str(self.config), "--context", guard.CONTEXT])
        suffix = argv[5:]
        if suffix == ["config", "view", "--minify", "--raw", "-o", "json"]:
            key = "config"
        elif suffix == ["get", "nodes", "-o", "json"]:
            key = "nodes"
        elif suffix == ["get", "runtimeclass", "gvisor", "-o", "json"]:
            key = "runtime"
        elif suffix == ["-n", "kube-system", "get", "configmap", "cilium-config", "-o", "json"]:
            key = "policy"
        elif suffix == ["-n", "kube-system", "get", "daemonset", "cilium", "-o", "json"]:
            key = "cilium"
        else:
            self.fail("guard attempted an unexpected command")
        return json.dumps(self.responses[key])

    def test_matching_identity_only_performs_reads(self):
        guard.require_secured_lab()
        self.assertEqual(self.reader.call_count, 6)
        self.assertEqual(sorted(p.name for p in self.root.iterdir()), [".local"])

    def test_missing_or_wrong_context_and_runtime_never_invoke_a_tool(self):
        for key, value in [("CONTEXT", ""), ("CONTEXT", "kind-kind"), ("RUNTIME_CLASS", ""), ("RUNTIME_CLASS", "runc")]:
            with self.subTest(key=key, value=value), patch.dict(os.environ, {key: value}):
                with self.assertRaises(RuntimeError):
                    guard.require_secured_lab()
        self.reader.assert_not_called()

    def test_missing_ambient_merged_relative_or_foreign_config_never_invokes_a_tool(self):
        for value in ("", "~/.kube/config", ".local/sandbox.kubeconfig", str(self.root / "other"), str(self.config) + ":/other"):
            with self.subTest(value=value), patch.dict(os.environ, KUBECONFIG=value):
                with self.assertRaises(RuntimeError):
                    guard.require_secured_lab()
        self.reader.assert_not_called()

    def test_absent_public_symlink_and_nonregular_config_fail_before_tools(self):
        self.config.chmod(0o644)
        with self.assertRaises(RuntimeError):
            guard.require_secured_lab()
        self.config.unlink()
        with self.assertRaises(RuntimeError):
            guard.require_secured_lab()
        outside = self.root / "other"
        outside.write_text("unchanged")
        self.config.symlink_to(outside)
        with self.assertRaises(RuntimeError):
            guard.require_secured_lab()
        self.config.unlink()
        self.config.mkdir()
        with self.assertRaises(RuntimeError):
            guard.require_secured_lab()
        self.reader.assert_not_called()
        self.assertEqual(outside.read_text(), "unchanged")

    def test_untrusted_config_identity_tls_plugins_and_endpoint_are_refused(self):
        original = copy.deepcopy(self.responses["config"])
        mutations = [
            lambda x: x["contexts"][0]["context"].update(cluster="kind-kind"),
            lambda x: x["clusters"][0].update(name="kind-kind"),
            lambda x: x["users"][0]["user"].update(exec={"command": "must-not-run"}),
            lambda x: x["users"][0]["user"].update(**{"auth-provider": {"name": "must-not-run"}}),
            lambda x: x["clusters"][0]["cluster"].update(**{"insecure-skip-tls-verify": True}),
            lambda x: x["clusters"][0]["cluster"].update(**{"proxy-url": "http://unapproved.invalid"}),
            lambda x: x["clusters"][0]["cluster"].update(server="https://unapproved.invalid:12345"),
            lambda x: x["clusters"][0]["cluster"].update(server="http://127.0.0.1:12345"),
            lambda x: x["clusters"][0]["cluster"].update(server="https://127.0.0.1:9999"),
        ]
        for mutate in mutations:
            self.responses["config"] = copy.deepcopy(original)
            mutate(self.responses["config"])
            with self.subTest(mutation=mutate), self.assertRaises(RuntimeError):
                guard.require_secured_lab()

    def test_wrong_node_runtime_or_cilium_cannot_admit_deployment(self):
        original = copy.deepcopy(self.responses)
        mutations = [
            lambda x: x["docker"][0]["Config"]["Labels"].update({"io.x-k8s.kind.cluster": "other"}),
            lambda x: x["docker"][0]["State"].update(Running=False),
            lambda x: x["nodes"]["items"][0]["metadata"].update(name="other-node"),
            lambda x: x["nodes"]["items"][0]["status"]["conditions"][0].update(status="False"),
            lambda x: x["runtime"].update(handler="runc"),
            lambda x: x["policy"]["data"].update({"enable-policy": "default"}),
            lambda x: x["cilium"]["status"].update(numberReady=0),
        ]
        for mutate in mutations:
            self.responses = copy.deepcopy(original)
            mutate(self.responses)
            with self.subTest(mutation=mutate), self.assertRaises(RuntimeError):
                guard.require_secured_lab()

    def test_command_failure_and_malformed_output_do_not_echo_secret_canaries(self):
        for output in ("private-response-canary", "null", "{}"):
            with patch.object(guard, "read_command", return_value=output), self.assertRaises(RuntimeError) as caught:
                guard.require_secured_lab()
            self.assertNotIn("private-response-canary", str(caught.exception))
        with patch.object(guard.subprocess, "run", return_value=subprocess.CompletedProcess([], 1, "private-response-canary", "private-response-canary")):
            with self.assertRaises(RuntimeError) as caught:
                READ_COMMAND(["stub"])
            self.assertNotIn("private-response-canary", str(caught.exception))

    def test_fresh_bootstrap_checks_before_creation_and_refuses_adoption(self):
        with self.assertRaises(RuntimeError):
            guard.require_secured_lab(bootstrap=True)
        self.reader.assert_not_called()
        self.config.unlink()
        with patch.object(guard, "read_command", side_effect=["synthetic-docker-version", ""]) as reader:
            guard.require_secured_lab(bootstrap=True)
            self.assertEqual(reader.call_count, 2)
        self.assertFalse(self.config.exists())
        with patch.object(guard, "read_command", side_effect=["synthetic-docker-version", guard.NODE]):
            with self.assertRaises(RuntimeError):
                guard.require_secured_lab(bootstrap=True)


class DeploymentEntryPoints(unittest.TestCase):
    def setUp(self):
        self.folder = tempfile.TemporaryDirectory()
        self.addCleanup(self.folder.cleanup)
        self.root = Path(self.folder.name)
        shutil.copytree(REPOSITORY / "scripts", self.root / "scripts", ignore=shutil.ignore_patterns("__pycache__"))
        shutil.copyfile(REPOSITORY / "Makefile", self.root / "Makefile")
        self.bin = self.root / "stubs"
        self.bin.mkdir()
        self.log = self.root / "unexpected-command"
        for name in ("go", "docker", "kind", "kubectl", "helm", "curl", "zstd", "sha512sum"):
            script = self.bin / name
            script.write_text('#!/bin/sh\nprintf "%s\\n" "$0 $*" >> "$GUARD_COMMAND_LOG"\nexit 99\n')
            script.chmod(0o755)
        self.env = dict(os.environ, PATH=str(self.bin) + os.pathsep + os.environ["PATH"], GUARD_COMMAND_LOG=str(self.log),
                        KUBECONFIG="", CONTEXT="", RUNTIME_CLASS="")
        self.env.pop("PYTHONDONTWRITEBYTECODE", None)

    def refused(self, argv, env=None):
        before = {folder: (self.root / folder).exists() for folder in ("bin", ".local", "artifacts")}
        result = subprocess.run(argv, cwd=self.root, env=env or self.env, capture_output=True, text=True, timeout=10)
        self.assertNotEqual(result.returncode, 0, result.stdout)
        self.assertTrue(result.stderr)
        self.assertFalse(self.log.exists(), "a build/download/cluster command ran before refusal")
        self.assertFalse((self.root / "scripts/__pycache__").exists(), "bytecode written before refusal")
        for folder in ("bin", ".local", "artifacts"):
            self.assertEqual((self.root / folder).exists(), before[folder], "generated state before refusal")

    def test_make_deployments_refuse_before_build_even_in_parallel_mode(self):
        for target in ("deploy", "tools-deploy", "private-deploy", "application-setup", "property-lab-setup", "stop-lab-setup", "ui-deploy", "showcase-prepare", "configure-model-fixture", "disable-model", "sandbox-cluster"):
            with self.subTest(target=target):
                defaults = dict(self.env)
                defaults.pop("CONTEXT")
                defaults.pop("RUNTIME_CLASS")
                self.refused(["make", "-j4", target], defaults)
                self.refused(["make", "-j4", target])
                self.refused(["make", "-j4", target, "CONTEXT=kind-kind", "RUNTIME_CLASS=gvisor"])
                self.refused(["make", "-j4", target, "CONTEXT=kind-tsunami-lab", "RUNTIME_CLASS=runc"])
                self.refused(["make", "-j4", target, "CONTEXT=kind-tsunami-lab", "RUNTIME_CLASS=gvisor"])

    def test_direct_helpers_cannot_bypass_the_guard(self):
        entries = [("configure-model.py", "--provider", "fixture"), ("configure-private.py",), ("configure-tools.py",),
                   ("configure-ui.py",), ("setup-vampi.py",), ("setup-stop-lab.py",), ("setup-property-lab.py",),
                   ("showcase.py", "prepare"), ("showcase.py", "run")]
        for entry in entries:
            with self.subTest(entry=entry):
                self.refused([sys.executable, "scripts/" + entry[0], *entry[1:]])
                self.refused([sys.executable, "scripts/" + entry[0], *entry[1:]],
                             dict(self.env, CONTEXT=guard.CONTEXT, RUNTIME_CLASS="gvisor"))
        self.refused(["sh", "scripts/setup-sandbox.sh"])
        config = self.root / ".local/sandbox.kubeconfig"
        config.parent.mkdir()
        config.write_text("{}")
        config.chmod(0o600)
        valid = dict(self.env, CONTEXT=guard.CONTEXT, RUNTIME_CLASS="gvisor", KUBECONFIG=str(config))
        for entry in entries:
            with self.subTest(guard_read_first=entry):
                result = subprocess.run([sys.executable, "scripts/" + entry[0], *entry[1:]], cwd=self.root,
                                        env=valid, capture_output=True, text=True, timeout=10)
                self.assertNotEqual(result.returncode, 0)
                calls = self.log.read_text().splitlines()
                self.assertEqual(len(calls), 1)
                self.assertIn("config view --minify --raw -o json", calls[0])
                self.assertIn("--kubeconfig " + str(config), calls[0])
                self.log.unlink()
                self.assertEqual(config.read_text(), "{}")
                self.assertFalse((self.root / "artifacts").exists())
                self.assertFalse((self.root / "bin").exists())
        for target in ("deploy", "tools-deploy", "private-deploy", "application-setup", "property-lab-setup"):
            with self.subTest(make_guard_read_first=target):
                result = subprocess.run(["make", "-j4", target], cwd=self.root, env=valid, capture_output=True, text=True, timeout=10)
                self.assertNotEqual(result.returncode, 0)
                calls = self.log.read_text().splitlines()
                self.assertEqual(len(calls), 1)
                self.assertIn("config view --minify --raw -o json", calls[0])
                self.log.unlink()
                self.assertFalse((self.root / "bin").exists())
                self.assertFalse((self.root / "artifacts").exists())

    def test_source_build_and_explicit_image_build_are_not_deployment_guarded(self):
        for target in ("test", "build", "image", "tools-image"):
            result = subprocess.run(["make", "-n", target], cwd=self.root, env=self.env, capture_output=True, text=True, check=True)
            self.assertNotIn("deployment_guard", result.stdout)
            self.assertFalse(self.log.exists())
        default = subprocess.run(["make", "-n"], cwd=self.root, env=self.env, capture_output=True, text=True, check=True)
        self.assertIn("go test -race", default.stdout)
        self.assertNotIn("deployment_guard", default.stdout)


if __name__ == "__main__":
    unittest.main()

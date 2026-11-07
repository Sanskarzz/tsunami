#!/usr/bin/env python3
"""Bind a trusted manifest's Tsunami image to the installed control build."""
import argparse
import json
import subprocess

from deploy_images import deployed_image, with_image, manifest_json

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--context", required=True)
parser.add_argument("--manifest", required=True)
args = parser.parse_args()
base = ["kubectl", "--context", args.context]
def kube(*argv):
    return subprocess.check_output(base + list(argv), text=True)
manifest = manifest_json(kube("create", "--dry-run=client", "-f", args.manifest, "-o", "json"))
print(json.dumps(with_image(manifest, deployed_image(kube))))

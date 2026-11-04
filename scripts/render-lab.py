#!/usr/bin/env python3
"""Bind the selected native image and worker runtime before deployment."""
import json
import os
import sys

from deploy_images import with_image

# kubectl emits consecutive JSON documents for a multi-document YAML input.
remaining = sys.stdin.read().strip()
decoder = json.JSONDecoder()
items = []
while remaining:
    item, offset = decoder.raw_decode(remaining)
    items.append(item)
    remaining = remaining[offset:].lstrip()
controllers = [item for item in items if item.get("kind") == "Deployment" and item["metadata"]["name"] == "tsunami-control"]
if len(controllers) != 1:
    sys.exit("Expected exactly one control-plane Deployment")
image = os.environ.get("IMAGE", "tsunami:dev")
items = with_image(items, image)
controller = next(item for item in items if item.get("kind") == "Deployment" and item["metadata"]["name"] == "tsunami-control")
container = controller["spec"]["template"]["spec"]["containers"][0]
env = [entry for entry in container.get("env", []) if entry["name"] not in ("TSUNAMI_RUNTIME_CLASS", "TSUNAMI_IMAGE")]
env.append({"name": "TSUNAMI_IMAGE", "value": image})
runtime = os.environ.get("RUNTIME_CLASS", "")
if runtime:
    env.append({"name": "TSUNAMI_RUNTIME_CLASS", "value": runtime})
container["env"] = env
json.dump({"apiVersion": "v1", "kind": "List", "items": items}, sys.stdout)

"""Shared image binding for trusted lab deployment/verification helpers."""
import copy
import json


def image_from_control(control):
    containers = control["spec"]["template"]["spec"]["containers"]
    container = next(c for c in containers if c["name"] == "control")
    image = container["image"]
    configured = next((e.get("value") for e in container.get("env", []) if e["name"] == "TSUNAMI_IMAGE"), image)
    if not image or configured != image:
        raise RuntimeError("control and worker image configuration differ; redeploy the selected build")
    return image


def deployed_image(kube):
    return image_from_control(json.loads(kube("-n", "tsunami", "get", "deployment", "tsunami-control", "-o", "json")))


def deployed_tool_image(kube):
    control = json.loads(kube("-n", "tsunami", "get", "deployment", "tsunami-control", "-o", "json"))
    container = next(c for c in control["spec"]["template"]["spec"]["containers"] if c["name"] == "control")
    image = next((e.get("value") for e in container.get("env", []) if e["name"] == "TSUNAMI_TOOL_IMAGE"), None)
    if not image:
        raise RuntimeError("reviewed tool image is not configured")
    return image


def with_image(manifest, image):
    """Replace only the known native placeholder; leave third-party images alone."""
    if not image or any(c.isspace() for c in image):
        raise ValueError("an explicit image without whitespace is required")
    result = copy.deepcopy(manifest)
    def visit(value):
        if isinstance(value, dict):
            if value.get("image") == "tsunami:dev":
                value["image"] = image
            for child in value.values():
                visit(child)
        elif isinstance(value, list):
            for child in value:
                visit(child)
    visit(result)
    return result

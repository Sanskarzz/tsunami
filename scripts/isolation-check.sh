#!/bin/sh
set -eu
context=${CONTEXT:-kind-kind}
if ! kubectl --context "$context" get runtimeclass gvisor >/dev/null 2>&1; then
  printf 'NOT QUALIFIED: gVisor RuntimeClass is absent. Testing network behavior separately.\n'
  CONTEXT="$context" python3 scripts/network-check.py
  printf 'FAIL: sandbox runtime is absent even if the tested network paths pass.\n' >&2
  exit 1
fi
# A RuntimeClass object alone is insufficient. Verify a real pod through CRI.
CONTEXT="$context" python3 scripts/runtime-check.py
CONTEXT="$context" RUNTIME_CLASS=gvisor python3 scripts/network-check.py
printf 'PASS runtime smoke test and listed network paths. Shell remains disabled; this is not full containment qualification.\n'

#!/bin/sh
set -eu
context=${CONTEXT:-kind-kind}
job=$(python3 scripts/render-workload.py --context "$context" --manifest deploy/verify.yaml | kubectl --context "$context" -n tsunami create -f - -o name)
attempt=0
while :; do
  completed=$(kubectl --context "$context" -n tsunami get "$job" -o 'jsonpath={.status.succeeded}')
  failed=$(kubectl --context "$context" -n tsunami get "$job" -o 'jsonpath={.status.failed}')
  if [ "$completed" = "1" ] || [ "${failed:-0}" -gt 0 ]; then break; fi
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 120 ]; then
    printf 'Verifier did not complete: %s\n' "$job" >&2
    exit 1
  fi
  sleep 1
done
# Read after completion. Following logs can lose output when this laptop runs
# out of fsnotify watchers; Job success is the result, not the log stream.
kubectl --context "$context" -n tsunami logs "$job"
if [ "$completed" != "1" ]; then
  printf 'Verification failed: %s\n' "$job" >&2
  exit 1
fi
printf 'Verification job retained for inspection: %s\n' "$job"

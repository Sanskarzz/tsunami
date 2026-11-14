#!/bin/sh
set -eu
# All installation is inside this dedicated Kind node. No sudo, host runtime
# installation, host daemon restart, or changes to the user's kubeconfig.
cd "$(dirname "$0")/.."
python3 -B scripts/deployment_guard.py --bootstrap --context "${CONTEXT:-}" --runtime-class "${RUNTIME_CLASS:-}"
name=tsunami-lab
node=tsunami-lab-control-plane
config="$PWD/.local/sandbox.kubeconfig"
runtime_dir="$PWD/.local/gvisor-20261026.0"
[ "$(uname -m)" = "x86_64" ] || { printf 'This lab setup is tested only on Linux x86_64\n' >&2; exit 1; }
for tool in kind docker kubectl helm curl sha512sum zstd; do
  command -v "$tool" >/dev/null || { printf 'Missing %s\n' "$tool" >&2; exit 1; }
done
mkdir -p "$runtime_dir"
chmod 700 .local
if [ ! -f "$runtime_dir/gvisor.tar.zstd" ]; then
  curl -fL --retry 2 --max-time 180 -o "$runtime_dir/gvisor.tar.zstd.tmp" https://storage.googleapis.com/gvisor/releases/release/20261026.0/x86_64/gvisor.tar.zstd
  mv "$runtime_dir/gvisor.tar.zstd.tmp" "$runtime_dir/gvisor.tar.zstd"
fi
# Pinned checksum from the official release, not a mutable latest URL.
(cd "$runtime_dir" && printf '%s  gvisor.tar.zstd\n' e6aa030fb8379fe92b53e9b8b696419e0dce8a3bbcc7d4c1cf76b2ea3d6569eabd7360e694deda6ccbdcd77095b3e82ca0f90a3d8fd13a3e5d6b11cd29fe4649 | sha512sum -c -)
tar --zstd -xf "$runtime_dir/gvisor.tar.zstd" -C "$runtime_dir"
if ! docker inspect "$node" >/dev/null 2>&1; then
  kind create cluster --name "$name" --kubeconfig "$config" --config deploy/sandbox/kind.yaml --image kindest/node:v1.34.0@sha256:7416a61b42b1662ca6ca89f02028ac133a309a2a30ba309614e8ec94d976dc5a
fi
[ -f "$config" ] || { printf 'Dedicated kubeconfig missing; refusing to use another context\n' >&2; exit 1; }
chmod 600 "$config"
installed_version=$(docker exec "$node" /usr/local/bin/runsc --version 2>/dev/null || true)
case "$installed_version" in
  'runsc version release-20261026.0'*) printf 'Pinned runtime already installed; leaving containerd running.\n' ;;
  *)
    printf 'Installing runtime and restarting containerd inside the dedicated lab node.\n'
    docker cp "$runtime_dir/runsc" "$node:/usr/local/bin/runsc"
    docker cp "$runtime_dir/containerd-shim-runsc-v1" "$node:/usr/local/bin/containerd-shim-runsc-v1"
    docker cp "$runtime_dir/gvisor-bin" "$node:/usr/local/bin/"
    docker cp deploy/sandbox/runsc.toml "$node:/etc/containerd/runsc.toml"
    docker exec "$node" systemctl restart containerd
    ;;
esac
docker exec "$node" /usr/local/bin/runsc --version
kubectl --kubeconfig "$config" --context kind-tsunami-lab create namespace tsunami --dry-run=client -o yaml | kubectl --kubeconfig "$config" --context kind-tsunami-lab apply -f -
kubectl --kubeconfig "$config" --context kind-tsunami-lab apply -f deploy/sandbox/trusted-network.yaml
helm upgrade --install cilium cilium --repo https://helm.cilium.io --version 1.20.4 --kubeconfig "$config" --kube-context kind-tsunami-lab --namespace kube-system -f deploy/sandbox/cilium-values.yaml
kubectl --kubeconfig "$config" --context kind-tsunami-lab apply -f deploy/sandbox/runtimeclass.yaml
kubectl --kubeconfig "$config" --context kind-tsunami-lab rollout status daemonset/cilium -n kube-system --timeout=180s
kubectl --kubeconfig "$config" --context kind-tsunami-lab wait node --all --for=condition=Ready --timeout=90s
printf '\nCreated separate sandbox cluster. Next: KUBECONFIG=.local/sandbox.kubeconfig make deploy CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor\n'

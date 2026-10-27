#!/usr/bin/env bash
set -euo pipefail
version=0.70.0
checksum=8b4376d5d6befe5c24d503f10ff136d9e0c49f9127a4279fd110b727929a5aa9
if command -v trivy >/dev/null && [[ "$(trivy --config /dev/null --version | head -n 1)" == "Version: $version" ]]; then
  command -v trivy
  exit 0
fi
[[ "$(uname -s)" == Linux && "$(uname -m)" == x86_64 ]] || { echo 'Pinned installer requires Linux x64.' >&2; exit 2; }
destination="${RUNNER_TOOL_CACHE:-${XDG_CACHE_HOME:-$HOME/.cache}}/security-fleet/tools/$version"
install -d -m 700 "$destination"
exec 9>"$destination/install.lock"
flock 9
if [[ ! -x "$destination/trivy" ]]; then
  work=$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/security-fleet-install.XXXXXX")
  trap 'rm -rf "$work"' EXIT
  curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
    "https://github.com/aquasecurity/trivy/releases/download/v${version}/trivy_${version}_Linux-64bit.tar.gz" -o "$work/trivy.tar.gz"
  printf '%s  %s\n' "$checksum" "$work/trivy.tar.gz" | sha256sum --check --strict >&2
  tar -xzf "$work/trivy.tar.gz" -C "$work" trivy
  install -m 700 "$work/trivy" "$destination/trivy"
fi
[[ "$("$destination/trivy" --config /dev/null --version | head -n 1)" == "Version: $version" ]]
printf '%s\n' "$destination/trivy"

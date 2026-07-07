#!/usr/bin/env bash
set -euo pipefail

version="${1:?version is required}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
dist="$root/dist"

rm -rf "$dist"
mkdir -p "$dist"

commit="$(git -C "$root" rev-parse --short=12 HEAD 2>/dev/null || echo unknown)"
date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
ldflags="-s -w -X main.version=$version -X main.commit=$commit -X main.date=$date"

platforms=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

for platform in "${platforms[@]}"; do
  os="${platform%/*}"
  arch="${platform#*/}"
  name="interplan_${version}_${os}_${arch}"
  bin="interplan"
  archive="$name.tar.gz"

  if [[ "$os" == "windows" ]]; then
    bin="interplan.exe"
    archive="$name.zip"
  fi

  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  echo "Building $platform"
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags "$ldflags" -o "$tmp/$bin" ./cmd/interplan

  cp "$root/LICENSE" "$tmp/LICENSE"
  cp "$root/README.md" "$tmp/README.md"

  if [[ "$os" == "windows" ]]; then
    (cd "$tmp" && zip -q "$dist/$archive" "$bin" LICENSE README.md)
  else
    (cd "$tmp" && tar -czf "$dist/$archive" "$bin" LICENSE README.md)
  fi

  rm -rf "$tmp"
  trap - EXIT
done

(
  cd "$dist"
  shasum -a 256 * > checksums.txt
)

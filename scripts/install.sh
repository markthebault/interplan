#!/usr/bin/env sh
set -eu

repo="markthebault/interplan"
binary="interplan"

fail() {
  echo "interplan installer: $*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

need uname
need mktemp

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1"; }
  download() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- "$1"; }
  download() { wget -qO "$2" "$1"; }
else
  fail "required command not found: curl or wget"
fi

os_raw="$(uname -s)"
arch_raw="$(uname -m)"

case "$os_raw" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) fail "unsupported operating system: $os_raw. Download a release manually from https://github.com/$repo/releases" ;;
esac

case "$arch_raw" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) fail "unsupported architecture: $arch_raw" ;;
esac

install_dir="${INTERPLAN_INSTALL_DIR:-}"
if [ -z "$install_dir" ]; then
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    install_dir="/usr/local/bin"
  else
    install_dir="$HOME/.local/bin"
  fi
fi

latest_json="$(fetch "https://api.github.com/repos/$repo/releases/latest")"
tag="$(printf '%s\n' "$latest_json" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
[ -n "$tag" ] || fail "could not determine latest release tag"

archive="interplan_${tag}_${os}_${arch}.tar.gz"
base_url="https://github.com/$repo/releases/download/$tag"
archive_url="$base_url/$archive"
checksums_url="$base_url/checksums.txt"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "Installing interplan $tag for $os/$arch"
download "$archive_url" "$tmp/$archive" || fail "could not download $archive_url"
download "$checksums_url" "$tmp/checksums.txt" || fail "could not download $checksums_url"

expected="$(grep " $archive\$" "$tmp/checksums.txt" | awk '{print $1}')"
[ -n "$expected" ] || fail "checksum for $archive not found"

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$archive" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')"
else
  fail "required command not found: sha256sum or shasum"
fi

[ "$actual" = "$expected" ] || fail "checksum mismatch for $archive"

tar -xzf "$tmp/$archive" -C "$tmp" || fail "could not extract $archive"
[ -x "$tmp/$binary" ] || fail "archive did not contain executable $binary"

mkdir -p "$install_dir"
cp "$tmp/$binary" "$install_dir/$binary"
chmod 755 "$install_dir/$binary"

echo "Installed $binary to $install_dir/$binary"

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *)
    echo "Note: $install_dir is not in your PATH. Add it to use '$binary' from any shell."
    ;;
esac

"$install_dir/$binary" --help >/dev/null 2>&1 || true

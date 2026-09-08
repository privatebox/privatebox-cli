#!/usr/bin/env bash
# Install or upgrade the latest stable Private Box CLI on Linux.
set -euo pipefail

fail() { printf 'Error: %s\n' "$*" >&2; exit 1; }
[[ $# -eq 0 ]] || fail 'Usage: bash install.sh'
[[ $(uname -s) == Linux ]] || fail 'This installer supports Linux/WSL. Use Homebrew on macOS.'
case "$(uname -m)" in
  x86_64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) fail 'Supported architectures: AMD64 and ARM64.' ;;
esac

for tool in curl sha256sum awk mktemp; do
  command -v "$tool" >/dev/null || fail "Install the required tool: $tool"
done

if command -v apt-get >/dev/null && command -v dpkg >/dev/null; then
  format=deb
elif command -v dnf >/dev/null; then
  format=rpm
else
  command -v tar >/dev/null || fail 'Install the required tool: tar'
  command -v install >/dev/null || fail 'Install the required tool: install'
  format=tar.gz
fi

elevate=()
if [[ $EUID -ne 0 ]]; then
  command -v sudo >/dev/null || fail 'Run as root or install sudo.'
  elevate=(sudo)
fi

repository=https://github.com/privatebox/privatebox-cli
release_url=$(curl --proto '=https' --proto-redir '=https' -fsSL -o /dev/null \
  -w '%{url_effective}' "$repository/releases/latest")
tag=${release_url##*/}
[[ $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail 'Could not resolve the latest stable release.'
version=${tag#v}
asset="privatebox_${version}_${arch}.${format}"
if [[ $format == tar.gz ]]; then
  asset="privatebox_${version}_linux_${arch}.tar.gz"
fi

# A private temporary directory prevents stale downloads from being installed.
download_dir=$(mktemp -d)
trap 'rm -rf -- "$download_dir"' EXIT
printf 'Downloading Private Box %s for Linux %s...\n' "$tag" "$arch"
curl --proto '=https' --proto-redir '=https' -fsSL \
  "$repository/releases/download/$tag/$asset" -o "$download_dir/$asset"
curl --proto '=https' --proto-redir '=https' -fsSL \
  "$repository/releases/download/$tag/checksums.txt" -o "$download_dir/checksums.txt"

checksum=$(awk -v asset="$asset" '$2 == asset { print $1 }' "$download_dir/checksums.txt")
[[ $checksum =~ ^[[:xdigit:]]{64}$ ]] || fail 'Missing, duplicate or invalid asset checksum.'
printf '%s  %s\n' "$checksum" "$asset" | (cd "$download_dir" && sha256sum --check -)

printf 'Installing %s (administrator access may be requested)...\n' "$tag"
case "$format" in
  deb)
    "${elevate[@]}" apt-get -o APT::Sandbox::User=root install -y "$download_dir/$asset"
    ;;
  rpm)
    "${elevate[@]}" dnf install -y "$download_dir/$asset"
    ;;
  tar.gz)
    tar -xzf "$download_dir/$asset" -C "$download_dir" privatebox
    "${elevate[@]}" install -d /usr/local/bin
    "${elevate[@]}" install -m 0755 "$download_dir/privatebox" /usr/local/bin/privatebox
    ;;
esac
printf 'Installed Private Box %s. Run: privatebox --version\n' "$tag"

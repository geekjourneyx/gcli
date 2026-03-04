#!/usr/bin/env bash
set -euo pipefail

VERSION="0.1.0"
REPO="your-org/gcli"
BASE="https://github.com/${REPO}/releases/download/v${VERSION}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "unsupported arch: $arch"; exit 1 ;;
esac
case "$os" in
  linux|darwin) ;;
  *) echo "unsupported os: $os"; exit 1 ;;
esac

bin_name="gcli"
asset="${bin_name}-${os}-${arch}"
url="${BASE}/${asset}"
install_dir="${HOME}/.local/bin"
mkdir -p "$install_dir"

tmp="/tmp/${bin_name}-$$"
if ! curl -fsSL -o "$tmp" "$url"; then
  echo "download failed"
  echo "url: $url"
  echo "platform: ${os}/${arch}"
  exit 1
fi

size="$(wc -c < "$tmp" || true)"
if [ -z "$size" ] || [ "$size" -eq 0 ]; then
  echo "downloaded file is empty"
  echo "url: $url"
  exit 1
fi

chmod +x "$tmp"
mv "$tmp" "${install_dir}/${bin_name}"

if [[ ":$PATH:" != *":${install_dir}:"* ]]; then
  echo "add PATH: export PATH=\"\$PATH:${install_dir}\""
fi

echo "installed: ${install_dir}/${bin_name}"

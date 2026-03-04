#!/usr/bin/env bash
set -euo pipefail

if [ ! -f Makefile ] || [ ! -f scripts/install.sh ] || [ ! -f CHANGELOG.md ]; then
  echo "required files missing (Makefile/scripts/install.sh/CHANGELOG.md)"
  exit 1
fi

make_ver="$(sed -n 's/^VERSION[[:space:]]*?=[[:space:]]*//p' Makefile | head -n1)"
install_ver="$(awk -F'"' '/^VERSION=/{print $2; exit}' scripts/install.sh)"
changelog_ver="$(sed -n 's/^## \[\([^]]*\)\].*/\1/p' CHANGELOG.md | head -n1)"

if [ -z "$make_ver" ] || [ -z "$install_ver" ] || [ -z "$changelog_ver" ]; then
  echo "failed to parse versions"
  echo "make: '$make_ver' install: '$install_ver' changelog: '$changelog_ver'"
  exit 1
fi

if [ "$make_ver" != "$install_ver" ] || [ "$make_ver" != "$changelog_ver" ]; then
  echo "version mismatch"
  echo "Makefile:       $make_ver"
  echo "scripts/install.sh: $install_ver"
  echo "CHANGELOG.md:   $changelog_ver"
  exit 1
fi

echo "release-check passed (version: $make_ver)"

#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<USAGE
Usage: $0 --env-file <path>

Required env vars in file:
  GCLI_GMAIL_CLIENT_ID
  GCLI_GMAIL_CLIENT_SECRET
  GCLI_GMAIL_REFRESH_TOKEN
USAGE
}

env_file=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-file)
      env_file="$2"
      shift 2
      ;;
    *)
      echo "unknown arg: $1"
      usage
      exit 1
      ;;
  esac
done

if [[ -z "$env_file" ]]; then
  usage
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$env_file"
set +a

echo "running smoke: gcli mail list --limit 1"
./bin/gcli mail list --limit 1 >/tmp/gcli-smoke.json

if ! grep -q '"version":"v1"' /tmp/gcli-smoke.json; then
  echo "smoke failed: invalid output"
  cat /tmp/gcli-smoke.json
  exit 1
fi

echo "smoke passed"

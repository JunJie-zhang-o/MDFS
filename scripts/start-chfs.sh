#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHFS_BIN="${ROOT_DIR}/.local/chfs/chfs-mac-amd64-3.1"
PORT="8091"

if [[ ! -x "${CHFS_BIN}" ]]; then
  printf 'chfs executable not found: %s\n' "${CHFS_BIN}" >&2
  printf 'Download it first, then run this script again.\n' >&2
  exit 1
fi

printf 'Starting chfs...\n'
printf 'URL: http://[::1]:%s/\n' "${PORT}"
printf 'Shared path: %s\n' "${ROOT_DIR}"

exec "${CHFS_BIN}" -path "${ROOT_DIR}" -port "${PORT}"

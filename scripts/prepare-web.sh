#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEBUI_DIR="${ROOT_DIR}/backend/internal/webui/dist"

cd "${ROOT_DIR}/frontend"
npm ci
npm run build

rm -rf "${WEBUI_DIR}"
mkdir -p "${WEBUI_DIR}"
cp -R "${ROOT_DIR}/frontend/dist/." "${WEBUI_DIR}/"
printf '%s\n' 'The frontend production build is copied into this directory by scripts/prepare-web.sh.' \
  > "${WEBUI_DIR}/README.txt"


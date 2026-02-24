#!/usr/bin/env bash
set -euo pipefail

APP_NAME="${APP_NAME:-nitrocoredx}"
GOOS_TARGET="${GOOS_TARGET:-linux}"
GOARCH_TARGET="${GOARCH_TARGET:-amd64}"
VERSION="${1:-}"

if [[ -z "${VERSION}" ]]; then
  if VERSION="$(git describe --tags --always --dirty 2>/dev/null)"; then
    :
  else
    VERSION="dev-$(date +%Y%m%d)"
  fi
fi

if [[ "${GOOS_TARGET}" != "linux" ]]; then
  echo "This local packaging script currently supports GOOS_TARGET=linux only." >&2
  echo "Use the GitHub Actions workflow for Windows release packages." >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
STAGE_DIR="${DIST_DIR}/${APP_NAME}-${VERSION}-${GOOS_TARGET}-${GOARCH_TARGET}"
ARCHIVE_PATH="${DIST_DIR}/${APP_NAME}-${VERSION}-${GOOS_TARGET}-${GOARCH_TARGET}.tar.gz"

mkdir -p "${DIST_DIR}"
rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}"

echo "Building ${APP_NAME} (${GOOS_TARGET}/${GOARCH_TARGET})..."
(
  cd "${ROOT_DIR}"
  GOOS="${GOOS_TARGET}" GOARCH="${GOARCH_TARGET}" \
    go build -tags no_sdl_ttf -o "${STAGE_DIR}/${APP_NAME}" ./cmd/corelx_devkit
)

cp "${ROOT_DIR}/LICENSE" "${STAGE_DIR}/LICENSE"
cp "${ROOT_DIR}/README.md" "${STAGE_DIR}/README.md"

cat > "${STAGE_DIR}/README_RELEASE.txt" <<'EOF'
Nitro-Core-DX (Integrated App) Release Package
=============================================

This package contains the Nitro-Core-DX integrated app (editor + embedded emulator).

Quick start:
  ./nitrocoredx

Notes (Linux):
- This binary is built with the "no_sdl_ttf" build tag.
- SDL2_ttf is not required.
- SDL2 runtime libraries are still required on the host system.
- The app includes an "Emulator Only" view if you just want to run/test ROMs.

If input seems unresponsive:
- click the emulator pane
- enable "Capture Game Input"
EOF

rm -f "${ARCHIVE_PATH}"
tar -C "${DIST_DIR}" -czf "${ARCHIVE_PATH}" "$(basename "${STAGE_DIR}")"

echo "Created: ${ARCHIVE_PATH}"

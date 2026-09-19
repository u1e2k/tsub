#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# tsub installer / updater script
# ==============================================================================

REPO_OWNER="${REPO_OWNER:-u1e2k}"
REPO_NAME="${REPO_NAME:-tsub}"
BINARY_NAME="${BINARY_NAME:-tsub}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo "==> Detecting system architecture..."
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64)
    ASSET_NAME="${BINARY_NAME}-linux-amd64"
    ;;
  aarch64|arm64)
    ASSET_NAME="${BINARY_NAME}-linux-arm64"
    ;;
  armv7l|armv7|armhf)
    ASSET_NAME="${BINARY_NAME}-linux-armv7"
    ;;
  *)
    echo "Error: Unsupported architecture: ${ARCH}" >&2
    echo "tsub supports x86_64, aarch64 (arm64), and armv7l (armv7)." >&2
    exit 1
    ;;
esac

echo "    Target binary: ${ASSET_NAME}"

# Setup temporary directory
TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

DOWNLOAD_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest/download/${ASSET_NAME}"
TMP_FILE="${TMP_DIR}/${BINARY_NAME}"

echo "==> Downloading latest release from:"
echo "    ${DOWNLOAD_URL}"

if command -v curl >/dev/null 2>&1; then
  curl -fL --progress-bar "${DOWNLOAD_URL}" -o "${TMP_FILE}"
elif command -v wget >/dev/null 2>&1; then
  wget -q --show-progress -O "${TMP_FILE}" "${DOWNLOAD_URL}"
else
  echo "Error: Neither curl nor wget was found. Please install one of them." >&2
  exit 1
fi

chmod +x "${TMP_FILE}"

echo "==> Installing to ${INSTALL_DIR}/${BINARY_NAME}..."

# Resolve sudo prefix if destination is not writable
SUDO=""
if [ ! -w "${INSTALL_DIR}" ]; then
  if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
      SUDO="sudo"
    else
      echo "Error: Directory ${INSTALL_DIR} is not writable and sudo is not available." >&2
      exit 1
    fi
  fi
fi

${SUDO} mkdir -p "${INSTALL_DIR}"

# Use atomic copy and mv to avoid "Text file busy" (ETXTBSY) error
# when self-updating (tsub -u) while the binary is actively running.
TMP_INSTALL="${INSTALL_DIR}/${BINARY_NAME}.tmp.$$"
${SUDO} cp "${TMP_FILE}" "${TMP_INSTALL}"
${SUDO} chmod +x "${TMP_INSTALL}"
${SUDO} mv -f "${TMP_INSTALL}" "${INSTALL_DIR}/${BINARY_NAME}"

echo ""
echo "========================================================"
echo " Successfully installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
echo "========================================================"
echo ""
echo "Run '${BINARY_NAME}' to get started!"

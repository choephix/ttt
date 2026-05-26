#!/bin/sh
set -e

REPO="eugenioenko/ttt"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux)  BINARY="ttt-linux-${ARCH}" ;;
  darwin) BINARY="ttt-darwin-${ARCH}" ;;
  *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

if [ -n "$1" ]; then
  VERSION="$1"
else
  VERSION=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
fi

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version"
  exit 1
fi

URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}"

echo "Downloading ttt ${VERSION} for ${OS}/${ARCH}..."
curl -sSfL "$URL" -o ttt

chmod +x ttt

if [ -w "$INSTALL_DIR" ]; then
  mv ttt "${INSTALL_DIR}/ttt"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv ttt "${INSTALL_DIR}/ttt"
fi

echo "ttt ${VERSION} installed to ${INSTALL_DIR}/ttt"

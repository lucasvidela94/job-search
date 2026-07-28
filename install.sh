#!/bin/sh
set -eu

REPO="lucasvidela94/job-search"
PROJECT="jobsearch"
INSTALL_DIR="/usr/local/bin"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --dir)
      if [ -z "${2-}" ]; then
        echo "Error: --dir requires an argument" >&2
        exit 1
      fi
      INSTALL_DIR="$2"
      shift 2
      ;;
    --dir=*)
      INSTALL_DIR="${1#--dir=}"
      shift
      ;;
    -h|--help)
      echo "Usage: sh install.sh [--dir <path>]"
      echo ""
      echo "Environment variables:"
      echo "  JOBSEARCH_VERSION  Pin a specific version (default: latest)"
      echo "  JOBSEARCH_DIR      Override install directory (overrides --dir)"
      exit 0
      ;;
    *)
      echo "Error: Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

if [ -n "${JOBSEARCH_DIR-}" ]; then
  INSTALL_DIR="$JOBSEARCH_DIR"
fi

PINNED_VERSION="${JOBSEARCH_VERSION-}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux|darwin) ;;
  *)
    echo "Error: Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

DOWNLOAD_CMD=
if command -v curl >/dev/null 2>&1; then
  DOWNLOAD_CMD="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
  DOWNLOAD_CMD="wget -qO-"
else
  echo "Error: curl or wget is required" >&2
  exit 1
fi

TMPDIR=
cleanup() {
  if [ -n "$TMPDIR" ]; then
    rm -rf "$TMPDIR"
  fi
}
trap cleanup EXIT
TMPDIR="$(mktemp -d "/tmp/${PROJECT}-install.XXXXXX")"

if [ -z "$PINNED_VERSION" ]; then
  BASE_URL="https://github.com/${REPO}/releases/latest/download"

  echo "Downloading checksums..." >&2
  $DOWNLOAD_CMD "${BASE_URL}/checksums.txt" > "$TMPDIR/checksums.txt" || {
    echo "Error: Failed to download checksums.txt" >&2
    exit 1
  }

  ARCHIVE_LINE="$(grep -F "_${OS}_${ARCH}.tar.gz" "$TMPDIR/checksums.txt" | head -n 1)"
  if [ -z "$ARCHIVE_LINE" ]; then
    echo "Error: No release found for ${OS}/${ARCH}" >&2
    exit 1
  fi

  CHECKSUM_EXPECTED="$(echo "$ARCHIVE_LINE" | awk '{print $1}')"
  ARCHIVE_FILENAME="$(echo "$ARCHIVE_LINE" | awk '{print $2}')"
  VERSION="$(echo "$ARCHIVE_FILENAME" | sed "s/^${PROJECT}_//; s/_${OS}_${ARCH}\\.tar\\.gz$//")"
else
  VERSION="$PINNED_VERSION"
  ARCHIVE_FILENAME="${PROJECT}_${VERSION}_${OS}_${ARCH}.tar.gz"
  BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

  echo "Downloading checksums..." >&2
  $DOWNLOAD_CMD "${BASE_URL}/checksums.txt" > "$TMPDIR/checksums.txt" || {
    echo "Error: Failed to download checksums.txt" >&2
    exit 1
  }

  CHECKSUM_EXPECTED="$(grep -F "$ARCHIVE_FILENAME" "$TMPDIR/checksums.txt" | awk '{print $1}')"
  if [ -z "$CHECKSUM_EXPECTED" ]; then
    echo "Error: No checksum found for ${ARCHIVE_FILENAME}" >&2
    echo "  Verify that version ${VERSION} exists and supports ${OS}/${ARCH}" >&2
    exit 1
  fi
fi

echo "Downloading ${ARCHIVE_FILENAME}..." >&2
$DOWNLOAD_CMD "${BASE_URL}/${ARCHIVE_FILENAME}" > "$TMPDIR/${ARCHIVE_FILENAME}" || {
  echo "Error: Failed to download ${ARCHIVE_FILENAME}" >&2
  exit 1
}

CHECKSUM_TOOL=
if command -v sha256sum >/dev/null 2>&1; then
  CHECKSUM_TOOL="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  CHECKSUM_TOOL="shasum -a 256"
else
  echo "Error: sha256sum or shasum is required" >&2
  exit 1
fi

CHECKSUM_ACTUAL="$($CHECKSUM_TOOL "$TMPDIR/${ARCHIVE_FILENAME}" | awk '{print $1}')"
if [ "$CHECKSUM_EXPECTED" != "$CHECKSUM_ACTUAL" ]; then
  echo "Error: Checksum mismatch" >&2
  echo "  Expected: ${CHECKSUM_EXPECTED}" >&2
  echo "  Actual:   ${CHECKSUM_ACTUAL}" >&2
  exit 1
fi

echo "Checksum verified" >&2

echo "Extracting..." >&2
(cd "$TMPDIR" && tar -xzf "${ARCHIVE_FILENAME}") || {
  echo "Error: Failed to extract archive" >&2
  exit 1
}

install_to() {
  dir="$1"
  if [ ! -d "$dir" ]; then
    mkdir -p "$dir" 2>/dev/null || return 1
  fi
  cp "$TMPDIR/${PROJECT}" "$dir/${PROJECT}" 2>/dev/null && chmod +x "$dir/${PROJECT}" 2>/dev/null
}

if install_to "$INSTALL_DIR"; then
  echo "${PROJECT} ${VERSION} installed to ${INSTALL_DIR}/${PROJECT}"
else
  # Fall back to user-local bin directory
  FALLBACK="${HOME}/.local/bin"
  echo "Warning: Cannot write to ${INSTALL_DIR}. Trying ${FALLBACK}..." >&2

  if install_to "$FALLBACK"; then
    echo "${PROJECT} ${VERSION} installed to ${FALLBACK}/${PROJECT}"
    case ":${PATH}:" in
      *:"${FALLBACK}":*) ;;
      *) echo "  Add ${FALLBACK} to your PATH:  export PATH=\"${FALLBACK}:\$PATH\"" >&2
         echo "  Or run:  source ~/.profile" >&2 ;;
    esac
  else
    echo "Error: Cannot write to ${INSTALL_DIR} or ${FALLBACK}." >&2
    echo "Try:  ${PROJECT}_${VERSION}_\$(uname -s | tr A-Z a-z)_\$(uname -m).tar.gz" >&2
    echo "Extract and place the '${PROJECT}' binary somewhere in your PATH." >&2
    exit 1
  fi
fi

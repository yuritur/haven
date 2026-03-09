#!/bin/sh
set -eu

REPO="yuritur/haven"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"
GITHUB_RELEASES="https://github.com/${REPO}/releases/download"

fail() {
    echo "Error: $1" >&2
    exit 1
}

detect_os() {
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux)  echo "linux" ;;
        darwin) echo "darwin" ;;
        *)      fail "unsupported OS: $os" ;;
    esac
}

detect_arch() {
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)              fail "unsupported architecture: $arch" ;;
    esac
}

http_get() {
    url="$1"
    if command -v curl >/dev/null 2>&1; then
        curl -sSL -f "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$url"
    else
        fail "neither curl nor wget found; install one and retry"
    fi
}

http_download() {
    url="$1"
    output="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -sSL -f -o "$output" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$output" "$url"
    else
        fail "neither curl nor wget found; install one and retry"
    fi
}

get_latest_version() {
    response="$(http_get "$GITHUB_API")" || fail "could not fetch latest release from GitHub API"
    version="$(echo "$response" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')"
    if [ -z "$version" ]; then
        fail "could not determine latest version from GitHub API response"
    fi
    echo "$version"
}

verify_checksum() {
    archive="$1"
    checksums_file="$2"
    expected="$(grep "$(basename "$archive")" "$checksums_file" | awk '{print $1}')"
    if [ -z "$expected" ]; then
        fail "archive not found in checksums file"
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual="$(sha256sum "$archive" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
        actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
    else
        fail "neither sha256sum nor shasum found; cannot verify checksum"
    fi

    if [ "$expected" != "$actual" ]; then
        fail "checksum mismatch: expected $expected, got $actual"
    fi
}

main() {
    os="$(detect_os)"
    arch="$(detect_arch)"

    echo "Detected platform: ${os}/${arch}"

    tag="$(get_latest_version)"
    version="${tag#v}"

    echo "Latest version: ${tag}"

    archive="haven_${version}_${os}_${arch}.tar.gz"
    checksums="haven_${version}_checksums.txt"

    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    echo "Downloading ${archive}..."
    http_download "${GITHUB_RELEASES}/${tag}/${archive}" "${tmpdir}/${archive}"

    echo "Downloading checksums..."
    http_download "${GITHUB_RELEASES}/${tag}/${checksums}" "${tmpdir}/${checksums}"

    echo "Verifying checksum..."
    verify_checksum "${tmpdir}/${archive}" "${tmpdir}/${checksums}"

    echo "Extracting binary..."
    tar -xzf "${tmpdir}/${archive}" -C "${tmpdir}"

    if [ ! -f "${tmpdir}/haven" ]; then
        fail "binary not found in archive"
    fi

    echo "Installing to ${INSTALL_DIR}/haven..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "${tmpdir}/haven" "${INSTALL_DIR}/haven"
    else
        sudo mv "${tmpdir}/haven" "${INSTALL_DIR}/haven"
    fi
    chmod +x "${INSTALL_DIR}/haven"

    if [ "$os" = "darwin" ]; then
        xattr -d com.apple.quarantine "${INSTALL_DIR}/haven" 2>/dev/null || true
    fi

    echo ""
    echo "haven ${tag} installed successfully to ${INSTALL_DIR}/haven"
    echo "Run 'haven --help' to get started."
}

main

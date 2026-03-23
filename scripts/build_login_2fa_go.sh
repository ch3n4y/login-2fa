#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p dist
mkdir -p .cache/go-build .cache/go-mod

export GOCACHE="$ROOT_DIR/.cache/go-build"
export GOMODCACHE="$ROOT_DIR/.cache/go-mod"

TARGET_OS="${TARGET_OS:-linux}"
TARGET_ARCH="${TARGET_ARCH:-$(go env GOARCH)}"
TARGET_ARM="${TARGET_ARM:-}"
BUILD_PAM="${BUILD_PAM:-1}"

KEY_FILE="$ROOT_DIR/dist/login-2fa.key"
umask 077
head -c 32 /dev/urandom | od -An -tx1 -v | tr -d ' \n' >"$KEY_FILE"
printf '\n' >>"$KEY_FILE"

CLI_ENV=(GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" CGO_ENABLED=0)
if [ -n "$TARGET_ARM" ]; then
  CLI_ENV+=(GOARM="$TARGET_ARM")
fi
env "${CLI_ENV[@]}" go build -trimpath -ldflags="-s -w" -o dist/login-2fa ./cmd/login-2fa

if [ "$BUILD_PAM" = "1" ]; then
  PAM_ENV=(GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" CGO_ENABLED=1)
  if [ -n "$TARGET_ARM" ]; then
    PAM_ENV+=(GOARM="$TARGET_ARM")
  fi
  if [ -z "${CC:-}" ]; then
    case "$TARGET_ARCH" in
      amd64)
        ;;
      arm64)
        PAM_ENV+=(CC=aarch64-linux-gnu-gcc)
        ;;
      arm)
        PAM_ENV+=(CC=arm-linux-gnueabihf-gcc)
        ;;
    esac
  fi
  env "${PAM_ENV[@]}" go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o dist/pam_login_2fa.so ./cmd/pam_login_2fa
fi

echo "Binary ready: $ROOT_DIR/dist/login-2fa"
if [ "$BUILD_PAM" = "1" ]; then
  echo "PAM module ready: $ROOT_DIR/dist/pam_login_2fa.so"
else
  echo "PAM module skipped: BUILD_PAM=0"
fi
echo "Master key file: $KEY_FILE"

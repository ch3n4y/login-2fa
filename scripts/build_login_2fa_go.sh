#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p dist
mkdir -p .cache/go-build .cache/go-mod

export GOCACHE="$ROOT_DIR/.cache/go-build"
export GOMODCACHE="$ROOT_DIR/.cache/go-mod"

KEY_FILE="$ROOT_DIR/dist/login-2fa.key"
umask 077
head -c 32 /dev/urandom | od -An -tx1 -v | tr -d ' \n' >"$KEY_FILE"
printf '\n' >>"$KEY_FILE"

go build -trimpath -ldflags="-s -w" -o dist/login-2fa ./cmd/login-2fa
go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o dist/pam_login_2fa.so ./cmd/pam_login_2fa

echo "Binary ready: $ROOT_DIR/dist/login-2fa"
echo "PAM module ready: $ROOT_DIR/dist/pam_login_2fa.so"
echo "Master key file: $KEY_FILE"

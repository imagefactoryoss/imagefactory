#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/release/dist}"
DIST_DIR="$(cd "$(dirname "$DIST_DIR")" && pwd)/$(basename "$DIST_DIR")"
VERSION="${VERSION:-$(git -C "$ROOT_DIR" describe --tags --always 2>/dev/null || git -C "$ROOT_DIR" rev-parse --short HEAD)}"
TARGETS="${TARGETS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64}"

BUILD_COMMIT="$(git -C "$ROOT_DIR" rev-parse HEAD 2>/dev/null || echo unknown)"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
BUILD_DIRTY="$([ -n "$(git -C "$ROOT_DIR" status --porcelain 2>/dev/null)" ] && echo true || echo false)"
LDFLAGS="-X main.buildVersion=${VERSION} -X main.buildCommit=${BUILD_COMMIT} -X main.buildTime=${BUILD_TIME} -X main.buildDirty=${BUILD_DIRTY}"

ARTIFACTS=(
  "image-factory-server:./cmd/server"
  "image-factory-dispatcher:./cmd/dispatcher"
  "image-factory-notification-worker:./cmd/notification-worker"
  "image-factory-email-worker:./cmd/email-worker"
  "image-factory-internal-registry-gc-worker:./cmd/internal-registry-gc-worker"
  "image-factory-docs-server:./cmd/docs-server"
  "image-factory-migrate:./cmd/migrate"
  "image-factory-essential-config-seeder:./cmd/essential-config-seeder"
  "image-factory-external-tenant-service:./cmd/external-tenant-service"
)

mkdir -p "$DIST_DIR"
rm -f "$DIST_DIR"/*.tar.gz "$DIST_DIR"/checksums.txt

for target in $TARGETS; do
  GOOS="${target%/*}"
  GOARCH="${target#*/}"
  STAGE_DIR="$DIST_DIR/.stage/${GOOS}_${GOARCH}"
  rm -rf "$STAGE_DIR"
  mkdir -p "$STAGE_DIR"

  for artifact in "${ARTIFACTS[@]}"; do
    NAME="${artifact%%:*}"
    CMD_PATH="${artifact#*:}"
    BIN_NAME="$NAME"
    if [ "$GOOS" = "windows" ]; then
      BIN_NAME="${BIN_NAME}.exe"
    fi

    echo "Building ${NAME} for ${GOOS}/${GOARCH}"
    (
      cd "$BACKEND_DIR"
      CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
        go build -ldflags "$LDFLAGS" -o "$STAGE_DIR/$BIN_NAME" "$CMD_PATH"
    )

    TAR_NAME="${NAME}_${VERSION}_${GOOS}_${GOARCH}.tar.gz"
    tar -C "$STAGE_DIR" -czf "$DIST_DIR/$TAR_NAME" "$BIN_NAME"
    rm -f "$STAGE_DIR/$BIN_NAME"
  done
done

if command -v shasum >/dev/null 2>&1; then
  (cd "$DIST_DIR" && shasum -a 256 ./*.tar.gz > checksums.txt)
elif command -v sha256sum >/dev/null 2>&1; then
  (cd "$DIST_DIR" && sha256sum ./*.tar.gz > checksums.txt)
else
  echo "No checksum tool found (expected shasum or sha256sum)" >&2
  exit 1
fi

rm -rf "$DIST_DIR/.stage"
echo "Release artifacts written to $DIST_DIR"

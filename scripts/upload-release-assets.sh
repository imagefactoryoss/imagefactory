#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/release/dist}"
TAG="${TAG:-}"
REPO="${REPO:-imagefactoryoss/imagefactory}"

if [ -z "$TAG" ]; then
  echo "TAG is required, for example: TAG=v0.1.0" >&2
  exit 1
fi

if [ ! -d "$DIST_DIR" ]; then
  echo "Distribution directory not found: $DIST_DIR" >&2
  exit 1
fi

gh release upload "$TAG" "$DIST_DIR"/*.tar.gz "$DIST_DIR"/checksums.txt --repo "$REPO" --clobber

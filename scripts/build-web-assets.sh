#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
WEB_DIR="$ROOT_DIR/web-svelte"
DESKTOP_DIST_DIR="$ROOT_DIR/desktop/frontend/dist"
STATIC_DIR="$ROOT_DIR/internal/web/static"

cd "$WEB_DIR"
bun run build

rm -rf "$DESKTOP_DIST_DIR"
mkdir -p "$DESKTOP_DIST_DIR"
cp -r dist/* "$DESKTOP_DIST_DIR/"

rm -rf "$STATIC_DIR"/*
mkdir -p "$STATIC_DIR"
cp -r dist/* "$STATIC_DIR/"
touch "$STATIC_DIR/.gitkeep"

echo "✅ Web assets synced to desktop/frontend/dist and internal/web/static"

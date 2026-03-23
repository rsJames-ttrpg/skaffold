#!/usr/bin/env bash
set -euo pipefail

# Usage: build_image_tar.sh <binary_path> <output_tar>
# Assembles a minimal Docker-compatible image tar from a static binary.

BINARY="$1"
OUT="$2"

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# Create the root filesystem layer
mkdir -p "$WORK/layer"
cp "$BINARY" "$WORK/layer/app"
chmod +x "$WORK/layer/app"
tar -cf "$WORK/layer.tar" -C "$WORK/layer" .
LAYER_SHA=$(sha256sum "$WORK/layer.tar" | cut -d' ' -f1)

# Create the image config
printf '{"architecture":"amd64","os":"linux","config":{"Entrypoint":["/app"]},"rootfs":{"type":"layers","diff_ids":["sha256:%s"]}}' "$LAYER_SHA" > "$WORK/config.json"
CONFIG_SHA=$(sha256sum "$WORK/config.json" | cut -d' ' -f1)

# Arrange files by sha256
mkdir -p "$WORK/$LAYER_SHA"
mv "$WORK/layer.tar" "$WORK/$LAYER_SHA/layer.tar"
mv "$WORK/config.json" "$WORK/$CONFIG_SHA.json"

# Create manifest.json (what `docker load` reads)
printf '[{"Config":"%s.json","RepoTags":["skaffold-buck2:latest"],"Layers":["%s/layer.tar"]}]' "$CONFIG_SHA" "$LAYER_SHA" > "$WORK/manifest.json"

# Package into final tar
tar -cf "$OUT" -C "$WORK" manifest.json "$CONFIG_SHA.json" "$LAYER_SHA/layer.tar"

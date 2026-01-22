#!/usr/bin/env bash
set -euo pipefail

IMAGE="ghcr.io/zenapo-hub/flight-simulator2"
PLATFORMS="linux/amd64,linux/arm64"
BUILDER="flight-sim-builder"

# Version: allow argument OR derive from git tag
VERSION="${1:-}"

if [[ -z "${VERSION}" ]]; then
  # Try git tag (exact match) like v0.1.0
  VERSION="$(git describe --tags --exact-match 2>/dev/null || true)"
fi

if [[ -z "${VERSION}" ]]; then
  echo "❌ Version not provided and no git tag found."
  echo "Usage:"
  echo "  ./build-and-push.sh v0.1.0"
  exit 1
fi

# Ensure go.sum exists
if [[ ! -f go.sum ]]; then
  echo "❌ go.sum is missing. Please run:"
  echo "  go mod tidy"
  exit 1
fi

echo "✅ Building and pushing:"
echo "  Image:     ${IMAGE}"
echo "  Version:   ${VERSION}"
echo "  Platforms: ${PLATFORMS}"

# Create builder only if missing
if ! docker buildx inspect "${BUILDER}" >/dev/null 2>&1; then
  docker buildx create --use --name "${BUILDER}"
else
  docker buildx use "${BUILDER}"
fi

# Build and push multi-arch images
docker buildx build \
  --platform "${PLATFORMS}" \
  -t "${IMAGE}:latest" \
  -t "${IMAGE}:${VERSION}" \
  --push \
  .

echo "✅ Done. Published:"
echo "  ${IMAGE}:latest"
echo "  ${IMAGE}:${VERSION}"

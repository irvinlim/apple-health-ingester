#!/bin/bash

set -euxo pipefail

IMAGE_NAME="irvinlim/apple-health-ingester"
IMAGE_TAG_ARGS=(--tag "${IMAGE_NAME}:latest")

# Use git commit hash for untagged git builds
COMMIT_TAG=$(git rev-parse HEAD | cut -c1-12)
IMAGE_TAG_ARGS+=(--tag "${IMAGE_NAME}:${COMMIT_TAG}")

# Include additional tag if git tag is set
GIT_TAG=$(git tag --points-at HEAD)
if [[ -n "${GIT_TAG}" ]]; then
  IMAGE_TAG_ARGS+=(--tag "${IMAGE_NAME}:${GIT_TAG}")
fi

# create builder with multi-arch build support
docker buildx create --use
docker buildx build --platform 'linux/amd64,linux/arm64' "${IMAGE_TAG_ARGS[@]}" --push .

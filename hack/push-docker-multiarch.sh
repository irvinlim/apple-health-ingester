#!/bin/bash

set -euxo pipefail

IMAGE_NAME="irvinlim/apple-health-ingester"

docker build -t "${IMAGE_NAME}:latest" .

# Use git commit hash for untagged git builds
COMMIT_TAG=$(git rev-parse HEAD | cut -c1-12)
docker tag "${IMAGE_NAME}:latest" "${IMAGE_NAME}:${COMMIT_TAG}"
docker push "${IMAGE_NAME}:latest"
docker push "${IMAGE_NAME}:${COMMIT_TAG}"

# Also push with additional tag if git tag is set
GIT_TAG=$(git tag --points-at HEAD)
if [[ -n "${GIT_TAG}" ]]; then
  docker tag "${IMAGE_NAME}:latest" "${IMAGE_NAME}:${GIT_TAG}"
  docker push "${IMAGE_NAME}:${GIT_TAG}"
fi

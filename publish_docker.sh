#!/bin/sh

# Set the image name
IMAGE_NAME="corlinp/victor"

# Set the platforms you want to support
PLATFORMS="linux/amd64,linux/arm64/v8"

# Create a new builder instance with multi-platform support (if not already created)
docker buildx create --name multiplatform-builder --use

# Build the Docker image for multiple platforms
docker buildx build --platform $PLATFORMS -t $IMAGE_NAME:latest --push .

echo "Docker image $IMAGE_NAME pushed"

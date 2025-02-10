#!/bin/bash

# Extract version from config.go
VERSION=$(grep 'Version: ' config/config.go | cut -d '"' -f 2)

if [ -z "$VERSION" ]
  then
    echo "Error: Unable to extract version from config/config.go"
    exit 1
fi

# Check if docker is installed
if ! [ -x "$(command -v docker)" ]; then
  echo 'Error: docker is not installed.' >&2
  exit 1
fi

echo "Building container for Kerbecs v$VERSION"

# Build the docker container
docker build -t bk1031/kerbecs:"$VERSION" -t bk1031/kerbecs:latest --platform linux/amd64,linux/arm64,linux/arm/v7,linux/arm/v6 --push --progress=plain .

echo "Container deployed successfully"

# Check if GitHub CLI is installed
if ! command -v gh &> /dev/null
then
    echo "GitHub CLI (gh) is not installed. Please install it to proceed."
    exit 1
fi

# Create a release tag
git tag -s v$VERSION -m "Release version $VERSION"
git push origin v$VERSION

# Create a release
gh release create v$VERSION --generate-notes

echo "Package released successfully for version $VERSION"
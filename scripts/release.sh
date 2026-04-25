#!/usr/bin/env bash
set -euo pipefail

if ! command -v gh &> /dev/null; then
    echo "Error: gh CLI is required (https://cli.github.com)"
    exit 1
fi

BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$BRANCH" != "main" ]]; then
    echo "Error: must be on main branch (currently on $BRANCH)"
    exit 1
fi

git fetch origin main --tags --quiet
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main)
if [[ "$LOCAL" != "$REMOTE" ]]; then
    echo "Error: local main is not up to date with origin/main"
    echo "  local:  $LOCAL"
    echo "  remote: $REMOTE"
    echo "Run 'git pull' first."
    exit 1
fi

if ! git diff-index --quiet HEAD --; then
    echo "Error: working tree has uncommitted changes"
    exit 1
fi

PREV_TAG=$(gh release list --limit 1 --json tagName --jq '.[0].tagName' 2>/dev/null || true)

echo ""
if [[ -n "$PREV_TAG" ]]; then
    echo "Current release: $PREV_TAG"
else
    echo "Current release: (none)"
fi
echo ""
read -rp "Enter new version: " INPUT

if [[ -z "$INPUT" ]]; then
    echo "Error: version cannot be empty"
    exit 1
fi

INPUT="${INPUT#v}"
if [[ ! "$INPUT" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: version must be a valid semver (e.g. 3.1.0)"
    exit 1
fi
VERSION="v${INPUT}"
SEMVER="${INPUT}"

if git tag -l "$VERSION" | grep -q "^${VERSION}$"; then
    echo "Error: tag $VERSION already exists locally"
    exit 1
fi
if git ls-remote --tags origin "refs/tags/$VERSION" | grep -q .; then
    echo "Error: tag $VERSION already exists on origin"
    exit 1
fi

CONFIG_FILE="config/config.go"

echo ""
echo "=== Release Summary ==="
echo "  Version: $VERSION"
echo "  Commit:  $(git rev-parse --short HEAD)"
echo "  Branch:  main"
echo ""
echo "  File to update:"
echo "    $CONFIG_FILE"
echo ""
echo "  Docker images that will be tagged:"
echo "    ghcr.io/bk1031/kerbecs:${SEMVER}"
echo "    docker.io/bk1031/kerbecs:${SEMVER}"
echo ""
read -rp "Proceed? (y/N) " CONFIRM
if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
    echo "Aborted."
    exit 0
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
sed -i '' -E "s/(Version[[:space:]]*=[[:space:]]*)\"[^\"]*\"/\1\"${SEMVER}\"/" "${REPO_ROOT}/${CONFIG_FILE}"

git add "${CONFIG_FILE}"
git commit -m "release: ${VERSION}"
git push origin main

gh release create "$VERSION" \
    --target main \
    --title "$VERSION" \
    --generate-notes

echo ""
echo "Release $VERSION created successfully."
echo "CI workflows will tag Docker images once builds complete."

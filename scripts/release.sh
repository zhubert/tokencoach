#!/bin/bash
#
# Release script for tokencoach
# Usage: ./scripts/release.sh <patch|minor|major> [--dry-run]
#
# Bumps the version, tags, and pushes to trigger the GitHub Actions release workflow.
#
# Examples:
#   ./scripts/release.sh patch      # v0.0.3 -> v0.0.4
#   ./scripts/release.sh minor      # v0.0.3 -> v0.1.0
#   ./scripts/release.sh major      # v0.0.3 -> v1.0.0

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

BUMP_TYPE=""
DRY_RUN=false

for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN=true
            ;;
        patch|minor|major)
            BUMP_TYPE="$arg"
            ;;
        *)
            echo -e "${RED}Unknown argument: $arg${NC}"
            echo "Usage: ./scripts/release.sh <patch|minor|major> [--dry-run]"
            exit 1
            ;;
    esac
done

if [ -z "$BUMP_TYPE" ]; then
    echo -e "${RED}Error: Bump type argument required (patch, minor, or major)${NC}"
    echo "Usage: ./scripts/release.sh <patch|minor|major> [--dry-run]"
    exit 1
fi

# Get the latest version tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

if ! [[ "$LATEST_TAG" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    echo -e "${RED}Error: Latest tag '$LATEST_TAG' is not in format vX.Y.Z${NC}"
    exit 1
fi

MAJOR="${BASH_REMATCH[1]}"
MINOR="${BASH_REMATCH[2]}"
PATCH="${BASH_REMATCH[3]}"

case $BUMP_TYPE in
    patch) PATCH=$((PATCH + 1)) ;;
    minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
    major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
esac

VERSION="v${MAJOR}.${MINOR}.${PATCH}"

echo -e "Current version: ${YELLOW}${LATEST_TAG}${NC}"
echo -e "New version:     ${GREEN}${VERSION}${NC} (${BUMP_TYPE} bump)"
echo ""

# Prerequisites
echo "Checking prerequisites..."

if ! command -v gh &> /dev/null; then
    echo -e "${RED}Error: gh CLI is not installed${NC}"
    exit 1
fi
echo "  gh CLI: found"

if ! gh auth status &> /dev/null; then
    echo -e "${RED}Error: Not authenticated with gh CLI${NC}"
    exit 1
fi
echo "  gh auth: authenticated"

if [ -n "$(git status --porcelain)" ]; then
    echo -e "${RED}Error: Working directory is not clean${NC}"
    git status --short
    exit 1
fi
echo "  Working directory: clean"

CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo -e "${RED}Error: Not on main branch (currently on: $CURRENT_BRANCH)${NC}"
    exit 1
fi
echo "  Branch: main"

if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo -e "${RED}Error: Tag $VERSION already exists${NC}"
    exit 1
fi
echo "  Tag $VERSION: available"

echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}Dry run — would create and push tag ${VERSION}${NC}"
    echo "The GitHub Actions release workflow would then build and publish the release."
else
    echo "Creating tag ${VERSION}..."
    git tag "$VERSION"

    echo "Pushing tag to origin..."
    git push origin "$VERSION"

    echo ""
    echo -e "${GREEN}Tag ${VERSION} pushed. GitHub Actions will handle the release.${NC}"
    echo ""
    echo "Watch the workflow: https://github.com/zhubert/tokencoach/actions"
    echo "Release will appear at: https://github.com/zhubert/tokencoach/releases/tag/${VERSION}"
fi

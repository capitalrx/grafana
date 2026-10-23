#!/usr/bin/env bash
# Builds the quickwit datasource plugin from source and stages its dist into
# custom-plugins/ so `docker build` bakes it into the image (see the COPY in
# Dockerfile and custom-plugins/README.md).
#
# Usage:
#   ./scripts/build-custom-plugins.sh              # build + stage
#   SKIP_BUILD=1 ./scripts/build-custom-plugins.sh # stage an existing dist only
#
# The quickwit-datasource frontend build needs Node 20/22 (see its .nvmrc).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
QUICKWIT_DS_DIR="${QUICKWIT_DS_DIR:-$REPO_ROOT/../quickwit-datasource}"
STAGE_DIR="$REPO_ROOT/custom-plugins"

mage_cmd() {
  if command -v mage >/dev/null 2>&1; then
    mage "$@"
  else
    go run github.com/magefile/mage@v1.17.2 "$@"
  fi
}

if [[ "${SKIP_BUILD:-}" != "1" ]]; then
  echo ">> Building quickwit-datasource (${QUICKWIT_DS_DIR})"
  # Frontend first: webpack cleans dist/, which would delete the backend binaries.
  (cd "$QUICKWIT_DS_DIR" && npm ci && npm run build)
  (cd "$QUICKWIT_DS_DIR" && mage_cmd -v build:linux build:linuxARM64)
fi

echo ">> Staging dists into ${STAGE_DIR}"
find "$STAGE_DIR" -mindepth 1 -maxdepth 1 ! -name README.md ! -name .gitkeep -exec rm -rf {} +
cp -a "$QUICKWIT_DS_DIR/dist" "$STAGE_DIR/quickwit-quickwit-datasource"

# A locally built dist must be plain unsigned: Grafana always rejects a
# modified signed plugin, but allow_loading_unsigned_plugins waives unsigned ones.
rm -f "$STAGE_DIR"/*/MANIFEST.txt

echo ">> Staged plugins:"
ls -1 "$STAGE_DIR"

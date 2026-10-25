#!/usr/bin/env bash
# Builds the plugins baked into the Grafana image from source and stages their dists into
# custom-plugins/ so `docker build` copies them in (see the COPY in Dockerfile and custom-plugins/README.md):
#   - quickwit-quickwit-datasource (../quickwit-datasource)
#   - grafana-lokiexplore-app, the Quickwit port of Logs Drilldown (../logs-drilldown)
#
# Usage:
#   ./scripts/build-custom-plugins.sh              # build + stage
#   SKIP_BUILD=1 ./scripts/build-custom-plugins.sh # stage existing dists only
#
# Needs Node 24.21 (logs-drilldown's .nvmrc; the quickwit-datasource build works on it too) and pnpm.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
QUICKWIT_DS_DIR="${QUICKWIT_DS_DIR:-$REPO_ROOT/../quickwit-datasource}"
LOGS_DRILLDOWN_DIR="${LOGS_DRILLDOWN_DIR:-$REPO_ROOT/../logs-drilldown}"
STAGE_DIR="$REPO_ROOT/custom-plugins"

mage_cmd() {
  if command -v mage >/dev/null 2>&1; then
    mage "$@"
  else
    go run github.com/magefile/mage@v1.17.2 "$@"
  fi
}

if [[ "${SKIP_BUILD:-}" != "1" ]]; then
  # .nvmrc may pin a major ("24") or a full version ("24.21.0"); match it as a prefix.
  required_node="$(tr -d '[:space:]' < "$LOGS_DRILLDOWN_DIR/.nvmrc")"
  required_node="${required_node#v}"
  if [[ "$(node -v)" != "v${required_node}" && "$(node -v)" != "v${required_node}."* ]]; then
    echo "!! Node $(node -v) found, logs-drilldown needs v${required_node} (e.g. nvm use ${required_node})" >&2
    exit 1
  fi

  echo ">> Building quickwit-datasource (${QUICKWIT_DS_DIR})"
  # Frontend first: webpack cleans dist/, which would delete the backend binaries.
  (cd "$QUICKWIT_DS_DIR" && npm ci && npm run build)
  (cd "$QUICKWIT_DS_DIR" && mage_cmd -v build:linux build:linuxARM64)

  echo ">> Building logs-drilldown (${LOGS_DRILLDOWN_DIR})"
  (cd "$LOGS_DRILLDOWN_DIR" && pnpm install --frozen-lockfile && pnpm run build)
fi

echo ">> Staging dists into ${STAGE_DIR}"
find "$STAGE_DIR" -mindepth 1 -maxdepth 1 ! -name README.md ! -name .gitkeep -exec rm -rf {} +
cp -a "$QUICKWIT_DS_DIR/dist" "$STAGE_DIR/quickwit-quickwit-datasource"
cp -a "$LOGS_DRILLDOWN_DIR/dist" "$STAGE_DIR/grafana-lokiexplore-app"

# A locally built dist must be plain unsigned: Grafana always rejects a
# modified signed plugin, but allow_loading_unsigned_plugins waives unsigned ones.
rm -f "$STAGE_DIR"/*/MANIFEST.txt

echo ">> Staged plugins:"
ls -1 "$STAGE_DIR"

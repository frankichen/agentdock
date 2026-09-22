#!/usr/bin/env bash
set -euo pipefail

SOURCE_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
RUNTIME_DIR="${FEISHU_MCP_RUNTIME_DIR:-${HOME}/AgentDock/mcp/feishu-openapi-mcp}"
START_SCRIPT="${RUNTIME_DIR}/start-stdio.sh"

if [[ ! -f "${START_SCRIPT}" ]]; then
  echo "Feishu MCP start script not found: ${START_SCRIPT}" >&2
  exit 66
fi

install -m 0755 "${SOURCE_DIR}/ensure-comment-openapi-extensions.js" "${RUNTIME_DIR}/ensure-comment-openapi-extensions.js"
node "${SOURCE_DIR}/patch-start-stdio.js" "${START_SCRIPT}"
echo "Installed Feishu comment OpenAPI extensions into ${RUNTIME_DIR}"

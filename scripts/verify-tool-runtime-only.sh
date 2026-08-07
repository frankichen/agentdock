#!/usr/bin/env sh
set -eu

compose_cmd="${COMPOSE_CMD:-docker compose}"

config="$($compose_cmd -f docker-compose.yml -f docker-compose.tool-runtime-only.yml config)"

printf '%s\n' "$config" | grep -q 'AGENTDOCK_ACP_ENABLED: "false"' || {
  echo "tool-runtime-only verification failed: AGENTDOCK_ACP_ENABLED is not forced to false" >&2
  exit 1
}

echo "tool-runtime-only verification passed"

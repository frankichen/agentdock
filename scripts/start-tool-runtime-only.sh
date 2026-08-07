#!/usr/bin/env sh
set -eu

# Force ACP off even when the parent environment enables it.
export AGENTDOCK_ACP_ENABLED=false

exec agentdock "$@"

#!/usr/bin/env sh
set -eu

# Enable the program-level guard and keep ACP off as defense in depth.
export AGENTDOCK_TOOL_RUNTIME_ONLY=true
export AGENTDOCK_ACP_ENABLED=false

exec agentdock "$@"

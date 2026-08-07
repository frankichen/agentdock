# Compliance and Intended Use

This fork is intended to use AgentDock as a controlled MCP tool runtime for operating machines, files, repositories, containers, browser sessions, and deployment environments.

## Intended use

AgentDock may be connected to supported AI clients through their documented MCP interfaces and used to expose explicitly permitted tools and resources.

This fork is not intended, marketed, or configured as a mechanism to bypass model-provider quotas, rate limits, usage limits, account restrictions, safety controls, or other service protections.

## Provider limits remain applicable

Using AgentDock does not remove or override any terms, quotas, rate limits, usage limits, or safety requirements imposed by an AI/model provider. Users remain responsible for complying with the terms and policies of every service they connect.

Do not configure this fork for account rotation, credential pooling, automated account switching, or other mechanisms whose purpose is to evade provider limits or restrictions.

## Tool-Runtime-Only profile

For deployments that only need MCP tool execution, enable the program-level hardening guard:

```text
AGENTDOCK_TOOL_RUNTIME_ONLY=true
AGENTDOCK_ACP_ENABLED=false
```

When `AGENTDOCK_TOOL_RUNTIME_ONLY=true`, the runtime forces ACP off even if `AGENTDOCK_ACP_ENABLED=true` is supplied elsewhere. Residual ACP adapter configuration is ignored rather than parsed. The provided `docker-compose.tool-runtime-only.yml` and `scripts/start-tool-runtime-only.sh` set the guard for hardened deployments; the Compose profile also pins `AGENTDOCK_ACP_ENABLED=false` as defense in depth.

The hardened Docker profile builds the current fork checkout instead of silently running the upstream `uvwt/agentdock` image. This ensures the program-level guard in this fork is actually present in the running container.

This keeps AgentDock focused on its file, command, Git, MCP, browser, task, and deployment capabilities without launching a provider-backed ACP coding agent. It does not modify, bypass, or replace any provider-side quota or usage-limit enforcement.

## Least privilege

Run AgentDock with the minimum operating-system and filesystem privileges required for the task. In particular:

- avoid running as root unless strictly necessary;
- do not expose the Docker socket unless the deployment requires Docker control;
- mount only directories that the agent needs to access;
- protect authentication tokens and OAuth secrets;
- require authentication for non-loopback network exposure;
- review browser automation and authenticated-session access carefully;
- keep write-capable tools scoped to the smallest practical environment.

## Upstream synchronization

This fork is expected to track upstream AgentDock while retaining its compliance hardening. Upstream changes should be merged through a review branch and checked for changes that weaken these restrictions or reintroduce quota-circumvention positioning.

This document is operational guidance, not a certification by OpenAI or any other provider.

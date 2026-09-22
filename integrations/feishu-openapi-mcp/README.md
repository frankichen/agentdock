# Feishu comment OpenAPI extensions

This integration persists two Feishu Drive comment operations that are not present in the pinned `@larksuiteoapi/lark-mcp@0.5.1` generated tool set used by the LensHub AgentDock runtime:

- `drive.v1.fileComment.createV2`
- `drive.v1.fileCommentReply.create`

The important capability for LensHub is `drive.v1.fileComment.createV2`. It calls:

```text
POST /open-apis/drive/v1/files/:file_token/new_comments
```

and supports a Bitable record anchor through:

```text
file_type = bitable
anchor.block_id = <table_id>
anchor.base_record_id = <record_id>
anchor.base_view_id = <view_id>
```

This is preferable to browser automation because it uses Feishu OpenAPI authentication and does not depend on browser cookies or CSRF state.

## Why the handler fallback is patched

The pinned Lark MCP package does not have an SDK method for the injected endpoint. Its generic fallback previously forwarded a literal path containing `:file_token`. The integration patches the generic fallback to:

1. resolve named path parameters from `input.path`;
2. URL-encode them;
3. forward `data` and `params` explicitly;
4. fail when a required path parameter is absent.

This patch is generic and remains compatible with existing generated tools.

## Runtime installation

```bash
bash integrations/feishu-openapi-mcp/install-runtime.sh
```

The installer derives the runtime from `$HOME/AgentDock` by default (or `FEISHU_MCP_RUNTIME_DIR` when set) and copies the extension injector into the existing Feishu MCP runtime and idempotently adds the two tool names to `start-stdio.sh`.

No App Secret, access token, cookie, CSRF value, or user credential is stored in this repository.

## Bitable record-comment example

```json
{
  "path": {
    "file_token": "<base app token>"
  },
  "data": {
    "file_type": "bitable",
    "anchor": {
      "block_id": "<table id>",
      "base_record_id": "<record id>",
      "base_view_id": "<view id>"
    },
    "reply_elements": [
      {
        "type": "text",
        "text": "comment text"
      }
    ]
  },
  "useUAT": true
}
```

Use user access-token mode (`useUAT=true`) when the comment must be authored as the current Feishu user.

## Verification

```bash
node --test integrations/feishu-openapi-mcp/tests/patch.test.js
```

Runtime acceptance also requires refreshing the dynamic `feishu` MCP and successfully creating a comment on a known Bitable record.

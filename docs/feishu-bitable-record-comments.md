# Feishu Base record comments on lark-mcp 0.5.1

This compatibility patch is for deployments that pin `@larksuiteoapi/lark-mcp@0.5.1`.

That release does not expose Drive comment V2, while Feishu's current OpenAPI uses:

- `POST /open-apis/drive/v1/files/{file_token}/new_comments`
- `file_type=bitable`
- `anchor.block_id=<table_id>`
- `anchor.base_record_id=<record_id>`
- `anchor.base_view_id=<view_id>`

for Base record-local comments.

It also fixes the generic MCP fallback so missing SDK methods expand `:path_parameters` before `client.request`. Without that expansion, the V2 comment tool reaches Feishu with an invalid file token and returns `1069302`.

## Apply

```bash
node scripts/patch-feishu-lark-mcp-comments.js \
  --root /path/to/node_modules/@larksuiteoapi/lark-mcp
```

The patch is idempotent and fails closed if the upstream generated-file layout changes.

After applying it, add these tools to the lark-mcp allowlist used by the deployment:

```text
drive.v1.fileComment.createV2
drive.v1.fileCommentReply.create
```

For Base record comments, call `drive.v1.fileComment.createV2` with a body equivalent to:

```json
{
  "file_type": "bitable",
  "anchor": {
    "block_id": "tbl...",
    "base_record_id": "rec...",
    "base_view_id": "vew..."
  },
  "reply_elements": [
    {
      "type": "text",
      "text": "record-local comment"
    }
  ]
}
```

Do not store Feishu app credentials or access tokens in this repository.

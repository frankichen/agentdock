"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const test = require("node:test");

const root = path.resolve(__dirname, "..");
const ensureScript = path.join(root, "ensure-comment-openapi-extensions.js");
const patchStartScript = path.join(root, "patch-start-stdio.js");

function runNode(script, args = [], env = {}) {
  const result = spawnSync(process.execPath, [script, ...args], {
    env: { ...process.env, ...env },
    encoding: "utf8",
  });
  assert.equal(result.status, 0, result.stderr || result.stdout);
}

function syntaxCheck(file) {
  const result = spawnSync(process.execPath, ["--check", file], { encoding: "utf8" });
  assert.equal(result.status, 0, result.stderr || result.stdout);
}

test("injects V2 Base comment and reply-create tools idempotently", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "feishu-comment-oapi-"));
  const drive = path.join(dir, "drive_v1.js");
  const handler = path.join(dir, "handler.js");

  fs.writeFileSync(drive, [
    '"use strict";',
    'const zod_1 = require("zod");',
    'exports.driveV1FileCommentCreate = {};',
    'exports.driveV1FileCommentGet = {};',
    'exports.driveV1FileCommentReplyDelete = {};',
    'exports.driveV1FileCommentReplyList = {};',
    'exports.driveV1Tools = [',
    '    exports.driveV1FileCommentCreate,',
    '    exports.driveV1FileCommentGet,',
    '    exports.driveV1FileCommentReplyDelete,',
    '    exports.driveV1FileCommentReplyList,',
    '];',
    '',
  ].join("\n"));

  fs.writeFileSync(handler, [
    '"use strict";',
    'async function h(client, path, httpMethod, func, params) {',
    '    for (const part of [1]) {',
    '        if (!func) {',
    '            func = async (params, ...args) => await client.request({ method: httpMethod, url: path, ...params }, ...args);',
    '            break;',
    '        }',
    '    }',
    '    if (!(func instanceof Function)) {',
    '        func = async (params, ...args) => await client.request({ method: httpMethod, url: path, ...params }, ...args);',
    '    }',
    '}',
    '',
  ].join("\n"));

  const env = { LARK_MCP_DRIVE_V1_JS: drive, LARK_MCP_HANDLER_JS: handler };
  runNode(ensureScript, [], env);

  const onceDrive = fs.readFileSync(drive, "utf8");
  const onceHandler = fs.readFileSync(handler, "utf8");
  assert.match(onceDrive, /drive\.v1\.fileComment\.createV2/);
  assert.match(onceDrive, /\/open-apis\/drive\/v1\/files\/:file_token\/new_comments/);
  assert.match(onceDrive, /base_record_id/);
  assert.match(onceDrive, /base_view_id/);
  assert.match(onceDrive, /drive\.v1\.fileCommentReply\.create/);
  assert.match(onceHandler, /const resolvedPath = path\.replace/);
  assert.match(onceHandler, /encodeURIComponent/);
  assert.match(onceHandler, /request\.data = input\.data/);
  assert.match(onceHandler, /request\.params = input\.params/);
  syntaxCheck(drive);
  syntaxCheck(handler);

  runNode(ensureScript, [], env);
  assert.equal(fs.readFileSync(drive, "utf8"), onceDrive);
  assert.equal(fs.readFileSync(handler, "utf8"), onceHandler);
});

test("patches Feishu start script idempotently", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "feishu-start-stdio-"));
  const target = path.join(dir, "start-stdio.sh");
  fs.writeFileSync(target, [
    '#!/bin/sh',
    'set -eu',
    'export APP_ID APP_SECRET',
    "TOOLS='bitable.v1.app.create,drive.v1.fileComment.create'",
    'exec lark-mcp mcp --tools "$TOOLS"',
    '',
  ].join("\n"));

  runNode(patchStartScript, [target]);
  const once = fs.readFileSync(target, "utf8");
  assert.match(once, /ensure-comment-openapi-extensions\.js/);
  assert.match(once, /drive\.v1\.fileComment\.createV2/);
  assert.match(once, /drive\.v1\.fileCommentReply\.create/);

  runNode(patchStartScript, [target]);
  assert.equal(fs.readFileSync(target, "utf8"), once);
  assert.equal((once.match(/drive\.v1\.fileComment\.createV2/g) || []).length, 1);
});

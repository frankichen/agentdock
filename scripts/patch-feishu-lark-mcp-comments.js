#!/usr/bin/env node
"use strict";

const fs = require("fs");
const path = require("path");

function fail(message) {
  throw new Error(message);
}

function parseArgs(argv) {
  let root = process.env.LARK_MCP_ROOT || "";
  for (let i = 2; i < argv.length; i += 1) {
    if (argv[i] === "--root") {
      root = argv[++i] || "";
      continue;
    }
    fail("unknown argument: " + argv[i]);
  }
  if (!root) {
    fail("missing lark-mcp root; pass --root <path> or set LARK_MCP_ROOT");
  }
  return path.resolve(root);
}

function insertBeforeOnce(source, anchor, block, marker) {
  if (source.includes(marker)) return source;
  if (!source.includes(anchor)) fail("layout changed: missing anchor for " + marker);
  return source.replace(anchor, block + anchor);
}

function replaceOnce(source, oldText, newText, marker) {
  if (source.includes(newText)) return source;
  if (!source.includes(oldText)) fail("layout changed: missing anchor for " + marker);
  return source.replace(oldText, newText);
}

function patchDriveTools(root) {
  const target = path.join(root, "dist/mcp-tool/tools/zh/gen-tools/zod/drive_v1.js");
  let source = fs.readFileSync(target, "utf8");

  const createV2Block = `exports.driveV1FileCommentCreateV2 = {
    project: 'drive',
    name: 'drive.v1.fileComment.createV2',
    sdkName: 'drive.v1.fileComment.createV2',
    path: '/open-apis/drive/v1/files/:file_token/new_comments',
    httpMethod: 'POST',
    description: '[Feishu/Lark]-云文档-评论-添加评论(V2)-支持多维表格记录局部评论；Base 使用 anchor.block_id + anchor.base_record_id + anchor.base_view_id',
    accessTokens: ['tenant', 'user'],
    schema: {
        data: zod_1.z.object({
            file_type: zod_1.z.enum(['doc', 'docx', 'sheet', 'file', 'slides', 'bitable', 'apps']).describe('云文档类型；多维表格/Base 必须传 bitable'),
            anchor: zod_1.z.object({
                block_id: zod_1.z.string().describe('评论锚点；Base 传 table_id'),
                parent_file_token: zod_1.z.string().optional(),
                parent_file_type: zod_1.z.string().optional(),
                sheet_row: zod_1.z.number().int().optional(),
                sheet_col: zod_1.z.number().int().optional(),
                slide_block_type: zod_1.z.string().optional(),
                base_record_id: zod_1.z.string().describe('Base 记录 ID').optional(),
                base_view_id: zod_1.z.string().describe('Base 视图 ID').optional(),
                file_page_num: zod_1.z.string().optional(),
                file_extra: zod_1.z.string().optional(),
                content_anchor_id: zod_1.z.string().optional(),
            }).optional(),
            reply_elements: zod_1.z.array(zod_1.z.object({
                type: zod_1.z.enum(['text', 'mention_user', 'link']).describe('评论元素类型'),
                text: zod_1.z.string().optional(),
                mention_user: zod_1.z.string().optional(),
                link: zod_1.z.string().optional(),
            })).min(1).describe('评论内容元素集合'),
            extra: zod_1.z.string().optional(),
        }),
        path: zod_1.z.object({
            file_token: zod_1.z.string().describe('文档 token；Base 使用 app/base token'),
        }),
        useUAT: zod_1.z.boolean().describe('使用用户身份请求, 否则使用应用身份').optional(),
    },
};
`;

  const replyCreateBlock = `exports.driveV1FileCommentReplyCreate = {
    project: 'drive',
    name: 'drive.v1.fileCommentReply.create',
    sdkName: 'drive.v1.fileCommentReply.create',
    path: '/open-apis/drive/v1/files/:file_token/comments/:comment_id/replies',
    httpMethod: 'POST',
    description: '[Feishu/Lark]-云文档-评论-添加回复-对云文档中的某条评论进行回复',
    accessTokens: ['tenant', 'user'],
    schema: {
        data: zod_1.z.object({
            content: zod_1.z.object({
                elements: zod_1.z.array(zod_1.z.object({
                    type: zod_1.z.enum(['text_run', 'docs_link', 'person']),
                    text_run: zod_1.z.object({ text: zod_1.z.string() }).optional(),
                    docs_link: zod_1.z.object({ url: zod_1.z.string() }).optional(),
                    person: zod_1.z.object({ user_id: zod_1.z.string() }).optional(),
                })),
            }),
            extra: zod_1.z.string().optional(),
        }),
        params: zod_1.z.object({
            file_type: zod_1.z.enum(['doc', 'sheet', 'file', 'docx', 'slides', 'bitable', 'apps']),
            user_id_type: zod_1.z.enum(['user_id', 'union_id', 'open_id']).optional(),
        }),
        path: zod_1.z.object({
            file_token: zod_1.z.string(),
            comment_id: zod_1.z.string(),
        }),
        useUAT: zod_1.z.boolean().optional(),
    },
};
`;

  source = insertBeforeOnce(
    source,
    "exports.driveV1FileCommentGet = {",
    createV2Block,
    "exports.driveV1FileCommentCreateV2 = {",
  );
  source = insertBeforeOnce(
    source,
    "exports.driveV1FileCommentReplyDelete = {",
    replyCreateBlock,
    "exports.driveV1FileCommentReplyCreate = {",
  );
  source = replaceOnce(
    source,
    "    exports.driveV1FileCommentCreate,\n    exports.driveV1FileCommentGet,",
    "    exports.driveV1FileCommentCreate,\n    exports.driveV1FileCommentCreateV2,\n    exports.driveV1FileCommentGet,",
    "createV2 tools array entry",
  );
  source = replaceOnce(
    source,
    "    exports.driveV1FileCommentReplyDelete,\n    exports.driveV1FileCommentReplyList,",
    "    exports.driveV1FileCommentReplyDelete,\n    exports.driveV1FileCommentReplyCreate,\n    exports.driveV1FileCommentReplyList,",
    "reply create tools array entry",
  );

  fs.writeFileSync(target, source, "utf8");
  for (const needle of [
    "drive.v1.fileComment.createV2",
    "/open-apis/drive/v1/files/:file_token/new_comments",
    "base_record_id",
    "base_view_id",
    "drive.v1.fileCommentReply.create",
  ]) {
    if (!source.includes(needle)) fail("drive tool verification failed: " + needle);
  }
  return target;
}

function patchFallback(root) {
  const target = path.join(root, "dist/mcp-tool/utils/handler.js");
  let source = fs.readFileSync(target, "utf8");

  const oldFallback = `        if (!func) {
            func = async (params, ...args) => await client.request({ method: httpMethod, url: path, ...params }, ...args);
            break;
        }`;
  const newFallback = `        if (!func) {
            func = async (params, ...args) => {
                const input = params || {};
                const pathParams = input.path || {};
                const resolvedPath = path.replace(/:([A-Za-z0-9_]+)/g, (match, key) => {
                    const value = pathParams[key];
                    if (value === undefined || value === null || String(value) === '') {
                        throw new Error('Missing path parameter: ' + key);
                    }
                    return encodeURIComponent(String(value));
                });
                const request = { method: httpMethod, url: resolvedPath };
                if (input.data !== undefined) request.data = input.data;
                if (input.params !== undefined) request.params = input.params;
                return await client.request(request, ...args);
            };
            break;
        }`;

  const oldNonFunctionFallback = `    if (!(func instanceof Function)) {
        func = async (params, ...args) => await client.request({ method: httpMethod, url: path, ...params }, ...args);
    }`;
  const newNonFunctionFallback = `    if (!(func instanceof Function)) {
        func = async (params, ...args) => {
            const input = params || {};
            const pathParams = input.path || {};
            const resolvedPath = path.replace(/:([A-Za-z0-9_]+)/g, (match, key) => {
                const value = pathParams[key];
                if (value === undefined || value === null || String(value) === '') {
                    throw new Error('Missing path parameter: ' + key);
                }
                return encodeURIComponent(String(value));
            });
            const request = { method: httpMethod, url: resolvedPath };
            if (input.data !== undefined) request.data = input.data;
            if (input.params !== undefined) request.params = input.params;
            return await client.request(request, ...args);
        };
    }`;

  if (!source.includes("const resolvedPath = path.replace(/:([A-Za-z0-9_]+)/g")) {
    if (!source.includes(oldFallback) || !source.includes(oldNonFunctionFallback)) {
      fail("handler layout changed: generic fallback anchors missing");
    }
    source = source.replace(oldFallback, newFallback);
    source = source.replace(oldNonFunctionFallback, newNonFunctionFallback);
    fs.writeFileSync(target, source, "utf8");
  }

  for (const needle of [
    "const resolvedPath = path.replace(/:([A-Za-z0-9_]+)/g",
    "Missing path parameter:",
    "request.data = input.data",
    "request.params = input.params",
  ]) {
    if (!source.includes(needle)) fail("handler verification failed: " + needle);
  }
  return target;
}

const root = parseArgs(process.argv);
const changed = {
  drive_tools: patchDriveTools(root),
  handler: patchFallback(root),
};
process.stdout.write(JSON.stringify({ ok: true, root, changed }) + "\n");

#!/usr/bin/env node
"use strict";

const fs = require("fs");
const target = process.argv[2];
if (!target) throw new Error("usage: patch-start-stdio.js <start-stdio.sh>");
let src = fs.readFileSync(target, "utf8");

if (!src.includes("ensure-comment-openapi-extensions.js")) {
  const anchor = "export APP_ID APP_SECRET\n";
  if (!src.includes(anchor)) throw new Error("start-stdio.sh layout changed: APP export anchor missing");
  src = src.replace(anchor, anchor + [
    'RUNTIME_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"',
    'node "$RUNTIME_DIR/ensure-comment-openapi-extensions.js"',
    '',
  ].join("\n"));
}

for (const tool of ["drive.v1.fileComment.createV2", "drive.v1.fileCommentReply.create"]) {
  if (src.includes(tool)) continue;
  const toolsAnchor = "TOOLS='";
  if (!src.includes(toolsAnchor)) throw new Error("start-stdio.sh layout changed: TOOLS anchor missing");
  src = src.replace(toolsAnchor, toolsAnchor + tool + ",");
}

fs.writeFileSync(target, src, "utf8");

package scripts

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFeishuLarkMCPCommentPatchIsIdempotent(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required")
	}

	root := t.TempDir()
	drivePath := filepath.Join(root, "dist", "mcp-tool", "tools", "zh", "gen-tools", "zod", "drive_v1.js")
	handlerPath := filepath.Join(root, "dist", "mcp-tool", "utils", "handler.js")
	if err := os.MkdirAll(filepath.Dir(drivePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(handlerPath), 0o755); err != nil {
		t.Fatal(err)
	}

	driveFixture := `"use strict";
exports.driveV1FileCommentCreate = {
    project: 'drive',
};
exports.driveV1FileCommentGet = {
    project: 'drive',
};
exports.driveV1FileCommentReplyDelete = {
    project: 'drive',
};
exports.driveV1FileCommentReplyList = {
    project: 'drive',
};
exports.driveV1Tools = [
    exports.driveV1FileCommentCreate,
    exports.driveV1FileCommentGet,
    exports.driveV1FileCommentReplyDelete,
    exports.driveV1FileCommentReplyList,
];
`
	handlerFixture := `"use strict";
const sdkFuncCall = async (client, params, options) => {
    const { tool } = options || {};
    const { sdkName, path, httpMethod } = tool || {};
    const chain = sdkName.split('.');
    let func = client;
    for (const element of chain) {
        func = func[element];
        if (!func) {
            func = async (params, ...args) => await client.request({ method: httpMethod, url: path, ...params }, ...args);
            break;
        }
    }
    if (!(func instanceof Function)) {
        func = async (params, ...args) => await client.request({ method: httpMethod, url: path, ...params }, ...args);
    }
};
`
	if err := os.WriteFile(drivePath, []byte(driveFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(handlerPath, []byte(handlerFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	script, err := filepath.Abs("patch-feishu-lark-mcp-comments.js")
	if err != nil {
		t.Fatal(err)
	}
	run := func() {
		t.Helper()
		cmd := exec.Command(node, script, "--root", root)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("patch failed: %v\n%s", err, out)
		}
	}
	run()

	firstDrive, err := os.ReadFile(drivePath)
	if err != nil {
		t.Fatal(err)
	}
	firstHandler, err := os.ReadFile(handlerPath)
	if err != nil {
		t.Fatal(err)
	}

	for _, needle := range []string{
		"drive.v1.fileComment.createV2",
		"/open-apis/drive/v1/files/:file_token/new_comments",
		"base_record_id",
		"base_view_id",
		"drive.v1.fileCommentReply.create",
	} {
		if !bytes.Contains(firstDrive, []byte(needle)) {
			t.Fatalf("patched drive tool missing %q", needle)
		}
	}
	for _, needle := range []string{
		"const resolvedPath = path.replace(/:([A-Za-z0-9_]+)/g",
		"encodeURIComponent(String(value))",
		"request.data = input.data",
		"request.params = input.params",
	} {
		if !bytes.Contains(firstHandler, []byte(needle)) {
			t.Fatalf("patched handler missing %q", needle)
		}
	}

	run()
	secondDrive, _ := os.ReadFile(drivePath)
	secondHandler, _ := os.ReadFile(handlerPath)
	if !bytes.Equal(firstDrive, secondDrive) {
		t.Fatal("drive tool patch is not idempotent")
	}
	if !bytes.Equal(firstHandler, secondHandler) {
		t.Fatal("handler patch is not idempotent")
	}

	if strings.Count(string(secondDrive), "exports.driveV1FileCommentCreateV2 = {") != 1 {
		t.Fatal("createV2 tool was inserted more than once")
	}
	if strings.Count(string(secondDrive), "exports.driveV1FileCommentReplyCreate = {") != 1 {
		t.Fatal("reply-create tool was inserted more than once")
	}
}

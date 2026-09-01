"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const { expandValue, loadConfig } = require("../src/config.js");

test("configuration expands variables and ignores disabled server placeholders", () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "kaeser-chat-config-"));
    const configPath = path.join(directory, "config.json");
    fs.writeFileSync(configPath, JSON.stringify({
        llm: { model: "test-model" },
        mcpServers: {
            local: { enabled: true, transport: "stdio", command: "node", args: ["${PROJECT_ROOT}/server.js"] },
            remote: { enabled: false, transport: "http", url: "${MISSING_REMOTE_URL}" }
        }
    }), "utf8");

    try {
        const loaded = loadConfig(configPath);
        assert.deepEqual(Object.keys(loaded.config.mcpServers), ["local"]);
        assert.match(loaded.config.mcpServers.local.args[0], /server\.js$/);
        assert.throws(() => expandValue("${NOT_DEFINED}", {}), /Missing environment variable/);
    } finally {
        fs.rmSync(directory, { recursive: true, force: true });
    }
});

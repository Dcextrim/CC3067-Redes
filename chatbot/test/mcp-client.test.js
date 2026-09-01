"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { once } = require("node:events");

const { createRemoteServer } = require("../../mcp-kaeser-server/remote-server.js");
const { InteractionLogger } = require("../src/logger.js");
const { McpClient } = require("../src/mcp-client.js");
const { HttpTransport } = require("../src/transports/http.js");
const { StdioTransport } = require("../src/transports/stdio.js");

const localServerPath = path.resolve(__dirname, "..", "..", "mcp-kaeser-server", "server.js");

test("manual stdio client discovers and calls the local MCP server", async () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "kaeser-chat-stdio-"));
    const logger = new InteractionLogger(directory);
    const transport = new StdioTransport("kaeser_local", {
        command: process.execPath,
        args: [localServerPath],
        cwd: path.dirname(localServerPath),
        env: { KAESER_LECTURAS_PATH: path.join(directory, "lecturas.json") }
    }, logger);
    const client = new McpClient("kaeser_local", transport);

    try {
        const tools = await client.connect();
        assert.equal(tools.length, 5);
        const result = await client.callTool("consultar_specs", { modelo: "SM 13" });
        assert.equal(result.isError, false);
        assert.equal(result.structuredContent.potencia_kw, 7.5);
        const logEntries = logger.readTail(50);
        assert.ok(logEntries.some((entry) => entry.direction === "sent" && entry.message.method === "tools/call"));
        assert.ok(logEntries.some((entry) => entry.direction === "received" && entry.message.result?.structuredContent));
    } finally {
        await client.close();
        fs.rmSync(directory, { recursive: true, force: true });
    }
});

test("manual HTTP client uses the same tools through the remote transport", async () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "kaeser-chat-http-"));
    const server = createRemoteServer({ authToken: "test-token" });
    server.listen(0, "127.0.0.1");
    await once(server, "listening");
    const address = server.address();
    const logger = new InteractionLogger(directory);
    const transport = new HttpTransport("kaeser_remote", {
        url: `http://127.0.0.1:${address.port}/mcp`,
        headers: { Authorization: "Bearer test-token" }
    }, logger);
    const client = new McpClient("kaeser_remote", transport);

    try {
        const tools = await client.connect();
        assert.equal(tools.length, 5);
        const result = await client.callTool("diagnosticar_falla", { codigo_mensaje: "0044 A", modelo: "SM 13" });
        assert.equal(result.structuredContent.requiere_servicio_autorizado, true);
        assert.ok(logger.readTail(50).every((entry) => entry.transport === "http"));
    } finally {
        await client.close();
        server.close();
        await once(server, "close");
        fs.rmSync(directory, { recursive: true, force: true });
    }
});

"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { once } = require("node:events");

const { createRemoteServer } = require("../remote-server.js");

function mcpHeaders(extra = {}) {
    return {
        Accept: "application/json, text/event-stream",
        Authorization: "Bearer test-token",
        "Content-Type": "application/json",
        ...extra
    };
}

test("remote server implements authenticated Streamable HTTP JSON responses", async () => {
    const server = createRemoteServer({
        authToken: "test-token",
        allowedOrigins: ["https://allowed.example"]
    });
    server.listen(0, "127.0.0.1");
    await once(server, "listening");

    const address = server.address();
    const baseUrl = `http://127.0.0.1:${address.port}`;

    try {
        const healthResponse = await fetch(`${baseUrl}/health`);
        assert.equal(healthResponse.status, 200);
        assert.equal((await healthResponse.json()).transport, "streamable-http");

        const unauthorizedResponse = await fetch(`${baseUrl}/mcp`, {
            method: "POST",
            headers: mcpHeaders({ Authorization: "" }),
            body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "ping" })
        });
        assert.equal(unauthorizedResponse.status, 401);

        const forbiddenOriginResponse = await fetch(`${baseUrl}/mcp`, {
            method: "POST",
            headers: mcpHeaders({ Origin: "https://blocked.example" }),
            body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "ping" })
        });
        assert.equal(forbiddenOriginResponse.status, 403);

        const getResponse = await fetch(`${baseUrl}/mcp`, {
            headers: {
                Accept: "text/event-stream",
                Authorization: "Bearer test-token"
            }
        });
        assert.equal(getResponse.status, 405);

        const initializeResponse = await fetch(`${baseUrl}/mcp`, {
            method: "POST",
            headers: mcpHeaders(),
            body: JSON.stringify({
                jsonrpc: "2.0",
                id: 1,
                method: "initialize",
                params: {
                    protocolVersion: "2025-11-25",
                    capabilities: {},
                    clientInfo: { name: "remote-test", version: "1.0.0" }
                }
            })
        });
        assert.equal(initializeResponse.status, 200);
        const initialize = await initializeResponse.json();
        assert.equal(initialize.result.protocolVersion, "2025-11-25");

        const notificationResponse = await fetch(`${baseUrl}/mcp`, {
            method: "POST",
            headers: mcpHeaders({ "MCP-Protocol-Version": "2025-11-25" }),
            body: JSON.stringify({ jsonrpc: "2.0", method: "notifications/initialized" })
        });
        assert.equal(notificationResponse.status, 202);

        const toolsResponse = await fetch(`${baseUrl}/mcp`, {
            method: "POST",
            headers: mcpHeaders({ "MCP-Protocol-Version": "2025-11-25" }),
            body: JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} })
        });
        const tools = await toolsResponse.json();
        assert.equal(tools.result.tools.length, 5);

        const toolCallResponse = await fetch(`${baseUrl}/mcp`, {
            method: "POST",
            headers: mcpHeaders({ "MCP-Protocol-Version": "2025-11-25" }),
            body: JSON.stringify({
                jsonrpc: "2.0",
                id: 3,
                method: "tools/call",
                params: {
                    name: "diagnosticar_falla",
                    arguments: { codigo_mensaje: "0044 A", modelo: "SM 13" }
                }
            })
        });
        const toolCall = await toolCallResponse.json();
        assert.equal(toolCall.result.isError, false);
        assert.equal(toolCall.result.structuredContent.codigo_mensaje, "0044 A");

        const invalidVersionResponse = await fetch(`${baseUrl}/mcp`, {
            method: "POST",
            headers: mcpHeaders({ "MCP-Protocol-Version": "invalid" }),
            body: JSON.stringify({ jsonrpc: "2.0", id: 4, method: "ping" })
        });
        assert.equal(invalidVersionResponse.status, 400);
    } finally {
        server.close();
        await once(server, "close");
    }
});

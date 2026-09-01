"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const { once } = require("node:events");

const { AnthropicClient } = require("../src/anthropic-client.js");

test("Anthropic client sends conversation history and MCP-compatible tools", async () => {
    let receivedHeaders;
    let receivedBody;
    const server = http.createServer((request, response) => {
        const chunks = [];
        request.on("data", (chunk) => chunks.push(chunk));
        request.on("end", () => {
            receivedHeaders = request.headers;
            receivedBody = JSON.parse(Buffer.concat(chunks).toString("utf8"));
            response.writeHead(200, { "Content-Type": "application/json" });
            response.end(JSON.stringify({
                id: "msg_test",
                type: "message",
                role: "assistant",
                content: [{ type: "text", text: "Hello" }],
                stop_reason: "end_turn"
            }));
        });
    });
    server.listen(0, "127.0.0.1");
    await once(server, "listening");

    try {
        const address = server.address();
        const client = new AnthropicClient({
            apiKey: "test-key",
            model: "test-model",
            maxTokens: 500,
            baseUrl: `http://127.0.0.1:${address.port}`
        });
        const result = await client.createMessage(
            [{ role: "user", content: "Hello" }],
            [{ name: "server__tool", description: "Tool", input_schema: { type: "object" } }],
            "System prompt"
        );

        assert.equal(result.content[0].text, "Hello");
        assert.equal(receivedHeaders["x-api-key"], "test-key");
        assert.equal(receivedHeaders["anthropic-version"], "2023-06-01");
        assert.equal(receivedBody.model, "test-model");
        assert.equal(receivedBody.messages.length, 1);
        assert.equal(receivedBody.tools[0].name, "server__tool");
    } finally {
        server.close();
        await once(server, "close");
    }
});

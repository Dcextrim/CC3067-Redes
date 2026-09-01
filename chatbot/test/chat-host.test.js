"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const { ChatHost } = require("../src/chat-host.js");

test("chat host executes MCP tools and keeps context between user turns", async () => {
    const toolCalls = [];
    const apiCalls = [];
    const fakeClient = {
        async connect() {
            return [{
                name: "consultar_specs",
                description: "Consulta especificaciones",
                inputSchema: {
                    type: "object",
                    properties: { modelo: { type: "string" } },
                    required: ["modelo"]
                }
            }];
        },
        async callTool(name, args) {
            toolCalls.push({ name, args });
            return { content: [{ type: "text", text: "7.5 kW" }], isError: false };
        },
        async close() {}
    };
    const responses = [
        {
            content: [{
                type: "tool_use",
                id: "toolu_1",
                name: "kaeser__consultar_specs",
                input: { modelo: "SM 13" }
            }]
        },
        { content: [{ type: "text", text: "El SM 13 tiene 7.5 kW." }] },
        { content: [{ type: "text", text: "Sí, seguimos hablando del SM 13." }] }
    ];
    const llmClient = {
        async createMessage(messages, tools) {
            apiCalls.push({ messages: structuredClone(messages), tools: structuredClone(tools) });
            return responses.shift();
        }
    };
    const assistantText = [];
    const host = new ChatHost({
        config: {
            mcpServers: {
                kaeser: { transport: "stdio", required: true }
            }
        },
        logger: { log() {} },
        llmClient,
        clientFactory: () => fakeClient,
        onAssistantText: (text) => assistantText.push(text)
    });

    await host.connect();
    await host.ask("¿Cuál es la potencia del SM 13?");
    await host.ask("¿De qué modelo hablábamos?");

    assert.equal(host.apiTools.length, 1);
    assert.deepEqual(toolCalls, [{ name: "consultar_specs", args: { modelo: "SM 13" } }]);
    assert.ok(apiCalls[1].messages.some((message) => message.role === "user" && Array.isArray(message.content)));
    assert.ok(apiCalls[2].messages.some((message) => message.content === "¿Cuál es la potencia del SM 13?"));
    assert.deepEqual(assistantText, ["El SM 13 tiene 7.5 kW.", "Sí, seguimos hablando del SM 13."]);
    await host.close();
});

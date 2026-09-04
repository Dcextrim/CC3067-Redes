"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const { once } = require("node:events");

const { GroqClient, selectRelevantTools } = require("../src/groq-client.js");

test("Groq tool routing avoids sending every discovered MCP schema", () => {
    const names = [
        "kaeser_remote__consultar_specs",
        "kaeser_remote__diagnosticar_falla",
        "kaeser_remote__calcular_mantenimiento_pendiente",
        "kaeser_remote__consultar_disponibilidad_repuesto",
        "kaeser_remote__registrar_lectura_sensor",
        ...Array.from({ length: 14 }, (_, index) => `filesystem__unrelated_${index}`),
        ...Array.from({ length: 12 }, (_, index) => `git__unrelated_${index}`)
    ];
    const tools = names.map((name) => ({ name, input_schema: { type: "object" } }));
    const selected = selectRelevantTools(
        [{ role: "user", content: "Usa consultar_specs para mostrar las especificaciones del SM 13." }],
        tools
    );

    assert.deepEqual(selected.map((tool) => tool.name), ["kaeser_remote__consultar_specs"]);
});

test("Groq client maps MCP tools and completes a tool-call round trip", async () => {
    const requestBodies = [];
    let receivedHeaders;
    let receivedUrl;
    const server = http.createServer((request, response) => {
        const chunks = [];
        request.on("data", (chunk) => chunks.push(chunk));
        request.on("end", () => {
            receivedHeaders = request.headers;
            receivedUrl = request.url;
            requestBodies.push(JSON.parse(Buffer.concat(chunks).toString("utf8")));
            response.writeHead(200, { "Content-Type": "application/json" });
            if (requestBodies.length === 1) {
                response.end(JSON.stringify({
                    choices: [{
                        message: {
                            role: "assistant",
                            content: null,
                            tool_calls: [{
                                id: "call_1",
                                type: "function",
                                function: {
                                    name: "kaeser__consultar_specs",
                                    arguments: "{\"modelo\":\"SM 13\"}"
                                }
                            }]
                        },
                        finish_reason: "tool_calls"
                    }]
                }));
                return;
            }
            response.end(JSON.stringify({
                choices: [{
                    message: { role: "assistant", content: "El SM 13 tiene 7.5 kW." },
                    finish_reason: "stop"
                }]
            }));
        });
    });
    server.listen(0, "127.0.0.1");
    await once(server, "listening");

    try {
        const address = server.address();
        const client = new GroqClient({
            apiKey: "test-key",
            model: "test-model",
            maxTokens: 500,
            baseUrl: `http://127.0.0.1:${address.port}`
        });
        const tools = [{
            name: "kaeser__consultar_specs",
            description: "Consulta especificaciones",
            input_schema: {
                type: "object",
                properties: { modelo: { type: "string" } },
                required: ["modelo"]
            }
        }];

        const first = await client.createMessage(
            [{ role: "user", content: "Consulta el SM 13" }],
            tools,
            "System prompt"
        );
        assert.deepEqual(first.content[0], {
            type: "tool_use",
            id: "call_1",
            name: "kaeser__consultar_specs",
            input: { modelo: "SM 13" }
        });

        const history = [
            { role: "user", content: "Consulta el SM 13" },
            { role: "assistant", content: first.content },
            { role: "user", content: [{ type: "tool_result", tool_use_id: "call_1", content: "7.5 kW" }] }
        ];
        const second = await client.createMessage(history, tools, "System prompt");

        assert.equal(receivedUrl, "/openai/v1/chat/completions");
        assert.equal(receivedHeaders.authorization, "Bearer test-key");
        assert.equal(requestBodies[0].model, "test-model");
        assert.equal(requestBodies[0].max_completion_tokens, 500);
        assert.equal(requestBodies[0].messages[0].role, "system");
        assert.equal(requestBodies[0].tools[0].function.name, "kaeser__consultar_specs");
        assert.equal(requestBodies[1].messages[2].tool_calls[0].id, "call_1");
        assert.equal(requestBodies[1].messages[3].role, "tool");
        assert.equal(requestBodies[1].messages[3].tool_call_id, "call_1");
        assert.equal(second.content[0].text, "El SM 13 tiene 7.5 kW.");
    } finally {
        server.close();
        await once(server, "close");
    }
});

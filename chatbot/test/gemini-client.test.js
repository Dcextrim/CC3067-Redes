"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const { once } = require("node:events");

const { GeminiClient, isRetryableError } = require("../src/gemini-client.js");

function readJson(request) {
    return new Promise((resolve) => {
        const chunks = [];
        request.on("data", (chunk) => chunks.push(chunk));
        request.on("end", () => resolve(JSON.parse(Buffer.concat(chunks).toString("utf8"))));
    });
}

test("Gemini client sends conversation history and MCP-compatible tools", async () => {
    let receivedHeaders;
    let receivedBody;
    let receivedUrl;
    const server = http.createServer(async (request, response) => {
        receivedHeaders = request.headers;
        receivedUrl = request.url;
        receivedBody = await readJson(request);
        response.writeHead(200, { "Content-Type": "application/json" });
        response.end(JSON.stringify({
            candidates: [{
                content: { role: "model", parts: [{ text: "Hola" }] },
                finishReason: "STOP"
            }]
        }));
    });
    server.listen(0, "127.0.0.1");
    await once(server, "listening");

    try {
        const address = server.address();
        const client = new GeminiClient({
            apiKey: "test-key",
            model: "test-model",
            maxTokens: 500,
            baseUrl: `http://127.0.0.1:${address.port}`
        });
        const result = await client.createMessage(
            [{ role: "user", content: "Hola" }],
            [{
                name: "server__tool",
                description: "Tool",
                input_schema: { type: "object", properties: { value: { type: "string" } } }
            }],
            "System prompt"
        );

        assert.equal(result.content[0].text, "Hola");
        assert.equal(receivedHeaders["x-goog-api-key"], "test-key");
        assert.equal(receivedUrl, "/v1beta/models/test-model:generateContent");
        assert.equal(receivedBody.contents[0].role, "user");
        assert.equal(receivedBody.contents[0].parts[0].text, "Hola");
        assert.equal(receivedBody.systemInstruction.parts[0].text, "System prompt");
        assert.equal(receivedBody.generationConfig.maxOutputTokens, 500);
        assert.equal(receivedBody.tools[0].functionDeclarations[0].name, "server__tool");
        assert.equal(receivedBody.tools[0].functionDeclarations[0].parametersJsonSchema.type, "object");
    } finally {
        server.close();
        await once(server, "close");
    }
});

test("Gemini client maps function calls, results, ids, and thought signatures", async () => {
    const requestBodies = [];
    let requestCount = 0;
    const server = http.createServer(async (request, response) => {
        requestBodies.push(await readJson(request));
        requestCount += 1;
        response.writeHead(200, { "Content-Type": "application/json" });

        if (requestCount === 1) {
            response.end(JSON.stringify({
                candidates: [{
                    content: {
                        role: "model",
                        parts: [{
                            functionCall: {
                                id: "call_1",
                                name: "kaeser__consultar_specs",
                                args: { modelo: "SM 13" }
                            },
                            thoughtSignature: "signed-thought"
                        }]
                    },
                    finishReason: "STOP"
                }]
            }));
            return;
        }

        response.end(JSON.stringify({
            candidates: [{
                content: { role: "model", parts: [{ text: "El SM 13 tiene 7.5 kW." }] },
                finishReason: "STOP"
            }]
        }));
    });
    server.listen(0, "127.0.0.1");
    await once(server, "listening");

    try {
        const address = server.address();
        const client = new GeminiClient({
            apiKey: "test-key",
            model: "test-model",
            baseUrl: `http://127.0.0.1:${address.port}`
        });
        const tools = [{
            name: "kaeser__consultar_specs",
            description: "Consulta especificaciones",
            input_schema: { type: "object", properties: { modelo: { type: "string" } } }
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

        const second = await client.createMessage([
            { role: "user", content: "Consulta el SM 13" },
            { role: "assistant", content: first.content },
            {
                role: "user",
                content: [{
                    type: "tool_result",
                    tool_use_id: "call_1",
                    content: "7.5 kW",
                    is_error: false
                }]
            }
        ], tools, "System prompt");

        assert.equal(second.content[0].text, "El SM 13 tiene 7.5 kW.");
        const modelPart = requestBodies[1].contents[1].parts[0];
        const responsePart = requestBodies[1].contents[2].parts[0];
        assert.equal(modelPart.functionCall.id, "call_1");
        assert.equal(modelPart.thoughtSignature, "signed-thought");
        assert.equal(responsePart.functionResponse.id, "call_1");
        assert.equal(responsePart.functionResponse.name, "kaeser__consultar_specs");
        assert.deepEqual(responsePart.functionResponse.response, { result: "7.5 kW" });
    } finally {
        server.close();
        await once(server, "close");
    }
});

test("Gemini client retries temporary API failures", async () => {
    let requestCount = 0;
    const delays = [];
    const server = http.createServer(async (request, response) => {
        await readJson(request);
        requestCount += 1;
        response.writeHead(requestCount < 3 ? 503 : 200, { "Content-Type": "application/json" });
        if (requestCount < 3) {
            response.end(JSON.stringify({ error: { message: "Temporary high demand" } }));
            return;
        }
        response.end(JSON.stringify({
            candidates: [{
                content: { role: "model", parts: [{ text: "Disponible" }] },
                finishReason: "STOP"
            }]
        }));
    });
    server.listen(0, "127.0.0.1");
    await once(server, "listening");

    try {
        const address = server.address();
        const client = new GeminiClient({
            apiKey: "test-key",
            model: "test-model",
            baseUrl: `http://127.0.0.1:${address.port}`,
            retryDelayMs: 1,
            sleep: async (delay) => delays.push(delay)
        });
        const result = await client.createMessage([{ role: "user", content: "Hola" }], [], "");

        assert.equal(result.content[0].text, "Disponible");
        assert.equal(requestCount, 3);
        assert.equal(delays.length, 2);
    } finally {
        server.close();
        await once(server, "close");
    }
});

test("Gemini client recognizes network timeouts as retryable", () => {
    assert.equal(isRetryableError({ code: "ETIMEDOUT" }), true);
    assert.equal(isRetryableError({ code: "ECONNRESET" }), true);
    assert.equal(isRetryableError({ statusCode: 503 }), true);
    assert.equal(isRetryableError({ statusCode: 400 }), false);
});

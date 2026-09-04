"use strict";

const http = require("node:http");
const https = require("node:https");

const MAX_RESPONSE_LENGTH = 10 * 1024 * 1024;
const RETRYABLE_STATUS_CODES = new Set([408, 429, 500, 502, 503, 504]);
const RETRYABLE_ERROR_CODES = new Set(["ECONNRESET", "ETIMEDOUT", "EAI_AGAIN"]);

function isRetryableError(error) {
    return RETRYABLE_STATUS_CODES.has(error?.statusCode) || RETRYABLE_ERROR_CODES.has(error?.code);
}

function parseRetryAfter(value) {
    if (!value) return 0;
    const seconds = Number(value);
    if (Number.isFinite(seconds)) return Math.max(0, seconds * 1000);
    const date = Date.parse(value);
    return Number.isFinite(date) ? Math.max(0, date - Date.now()) : 0;
}

function findToolNames(messages) {
    const names = new Map();
    for (const message of messages) {
        if (!Array.isArray(message.content)) continue;
        for (const block of message.content) {
            if (block.type === "tool_use") names.set(block.id, block.name);
        }
    }
    return names;
}

function toGeminiContents(messages, toolMetadata = new Map()) {
    const toolNames = findToolNames(messages);

    return messages.map((message) => {
        const blocks = typeof message.content === "string"
            ? [{ type: "text", text: message.content }]
            : message.content || [];
        const parts = [];

        for (const block of blocks) {
            if (block.type === "text") {
                parts.push({ text: block.text });
                continue;
            }

            if (block.type === "tool_use") {
                const functionCall = {
                    name: block.name,
                    args: block.input || {}
                };
                if (block.id) functionCall.id = block.id;

                const part = { functionCall };
                const metadata = toolMetadata.get(block.id);
                if (metadata?.thoughtSignature) part.thoughtSignature = metadata.thoughtSignature;
                parts.push(part);
                continue;
            }

            if (block.type === "tool_result") {
                const name = toolNames.get(block.tool_use_id);
                if (!name) throw new Error(`Unable to match Gemini function response ${block.tool_use_id} to a tool name.`);

                const functionResponse = {
                    name,
                    response: block.is_error
                        ? { error: String(block.content) }
                        : { result: block.content }
                };
                if (block.tool_use_id) functionResponse.id = block.tool_use_id;
                parts.push({ functionResponse });
            }
        }

        return {
            role: message.role === "assistant" ? "model" : "user",
            parts
        };
    }).filter((message) => message.parts.length > 0);
}

function toGeminiTools(tools) {
    if (tools.length === 0) return [];
    return [{
        functionDeclarations: tools.map((tool) => ({
            name: tool.name,
            description: tool.description || tool.name,
            parametersJsonSchema: tool.input_schema || { type: "object", properties: {} }
        }))
    }];
}

function toHostResponse(response, toolMetadata = new Map()) {
    const candidate = response.candidates?.[0];
    const parts = candidate?.content?.parts || [];
    const content = [];

    for (let index = 0; index < parts.length; index += 1) {
        const part = parts[index];
        if (typeof part.text === "string" && !part.thought) {
            content.push({ type: "text", text: part.text });
        }
        if (part.functionCall) {
            const id = part.functionCall.id || `gemini_tool_${Date.now()}_${index}`;
            toolMetadata.set(id, { thoughtSignature: part.thoughtSignature });
            content.push({
                type: "tool_use",
                id,
                name: part.functionCall.name,
                input: part.functionCall.args || {}
            });
        }
    }

    if (content.length === 0) {
        const reason = candidate?.finishReason || response.promptFeedback?.blockReason || "empty response";
        throw new Error(`Gemini API returned no usable content (${reason}).`);
    }

    return { content, stop_reason: candidate?.finishReason };
}

class GeminiClient {
    constructor(options) {
        this.apiKey = options.apiKey;
        this.model = options.model;
        this.maxTokens = options.maxTokens || 4096;
        this.baseUrl = new URL(options.baseUrl || "https://generativelanguage.googleapis.com");
        this.client = this.baseUrl.protocol === "https:" ? https : http;
        this.toolMetadata = new Map();
        this.maxRetries = options.maxRetries ?? 4;
        this.retryDelayMs = options.retryDelayMs ?? 1000;
        this.sleep = options.sleep || ((milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds)));
        this.onRetry = options.onRetry || (() => {});
    }

    async createMessage(messages, tools, system) {
        const body = {
            contents: toGeminiContents(messages, this.toolMetadata),
            generationConfig: { maxOutputTokens: this.maxTokens },
            tools: toGeminiTools(tools)
        };
        if (body.tools.length === 0) delete body.tools;
        if (system) body.systemInstruction = { parts: [{ text: system }] };

        const payload = JSON.stringify(body);
        const model = encodeURIComponent(this.model);
        const url = new URL(`/v1beta/models/${model}:generateContent`, this.baseUrl);

        for (let attempt = 0; ; attempt += 1) {
            try {
                return await this.requestOnce(url, payload);
            } catch (error) {
                if (!isRetryableError(error) || attempt >= this.maxRetries) throw error;
                const exponentialDelay = Math.min(60000, this.retryDelayMs * (2 ** attempt));
                const delay = Math.max(error.retryAfterMs || 0, exponentialDelay) + Math.floor(Math.random() * 250);
                this.onRetry({
                    attempt: attempt + 1,
                    maxRetries: this.maxRetries,
                    delayMs: delay,
                    statusCode: error.statusCode,
                    errorCode: error.code
                });
                await this.sleep(delay);
            }
        }
    }

    requestOnce(url, payload) {
        return new Promise((resolve, reject) => {
            const request = this.client.request(url, {
                method: "POST",
                headers: {
                    "Content-Length": Buffer.byteLength(payload),
                    "Content-Type": "application/json",
                    "x-goog-api-key": this.apiKey
                }
            }, (response) => {
                const chunks = [];
                let length = 0;
                response.on("data", (chunk) => {
                    length += chunk.length;
                    if (length > MAX_RESPONSE_LENGTH) {
                        request.destroy(new Error("Gemini API response exceeds the supported size."));
                        return;
                    }
                    chunks.push(chunk);
                });
                response.on("end", () => {
                    const responseBody = Buffer.concat(chunks).toString("utf8");
                    let parsed;
                    try {
                        parsed = JSON.parse(responseBody);
                    } catch {
                        reject(new Error(`Gemini API returned invalid JSON (HTTP ${response.statusCode}).`));
                        return;
                    }

                    if (response.statusCode < 200 || response.statusCode >= 300) {
                        const error = new Error(`Gemini API error ${response.statusCode}: ${parsed.error?.message || responseBody}`);
                        error.statusCode = response.statusCode;
                        error.retryAfterMs = parseRetryAfter(response.headers["retry-after"]);
                        reject(error);
                        return;
                    }

                    try {
                        resolve(toHostResponse(parsed, this.toolMetadata));
                    } catch (error) {
                        reject(error);
                    }
                });
            });

            request.setTimeout(60000, () => {
                const error = new Error("Gemini API request timed out.");
                error.code = "ETIMEDOUT";
                request.destroy(error);
            });
            request.on("error", reject);
            request.end(payload);
        });
    }
}

module.exports = { GeminiClient, findToolNames, isRetryableError, parseRetryAfter, toGeminiContents, toGeminiTools, toHostResponse };

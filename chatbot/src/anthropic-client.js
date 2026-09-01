"use strict";

const http = require("node:http");
const https = require("node:https");

const MAX_RESPONSE_LENGTH = 10 * 1024 * 1024;

class AnthropicClient {
    constructor(options) {
        this.apiKey = options.apiKey;
        this.model = options.model;
        this.maxTokens = options.maxTokens || 1500;
        this.baseUrl = new URL(options.baseUrl || "https://api.anthropic.com");
        this.client = this.baseUrl.protocol === "https:" ? https : http;
    }

    createMessage(messages, tools, system) {
        const body = {
            model: this.model,
            max_tokens: this.maxTokens,
            messages
        };
        if (tools.length > 0) body.tools = tools;
        if (system) body.system = system;

        const payload = JSON.stringify(body);
        const url = new URL("/v1/messages", this.baseUrl);

        return new Promise((resolve, reject) => {
            const request = this.client.request(url, {
                method: "POST",
                headers: {
                    "anthropic-version": "2023-06-01",
                    "Content-Length": Buffer.byteLength(payload),
                    "Content-Type": "application/json",
                    "x-api-key": this.apiKey
                }
            }, (response) => {
                const chunks = [];
                let length = 0;
                response.on("data", (chunk) => {
                    length += chunk.length;
                    if (length > MAX_RESPONSE_LENGTH) {
                        request.destroy(new Error("Claude API response exceeds the supported size."));
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
                        reject(new Error(`Claude API returned invalid JSON (HTTP ${response.statusCode}).`));
                        return;
                    }

                    if (response.statusCode < 200 || response.statusCode >= 300) {
                        reject(new Error(`Claude API error ${response.statusCode}: ${parsed.error?.message || responseBody}`));
                        return;
                    }
                    resolve(parsed);
                });
            });

            request.setTimeout(60000, () => request.destroy(new Error("Claude API request timed out.")));
            request.on("error", reject);
            request.end(payload);
        });
    }
}

module.exports = { AnthropicClient };

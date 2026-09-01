"use strict";

const fs = require("node:fs");
const http = require("node:http");
const https = require("node:https");
const path = require("node:path");

const MAX_RESPONSE_LENGTH = 5 * 1024 * 1024;

function parseSse(body, expectedId) {
    const messages = [];
    for (const event of body.split(/\r?\n\r?\n/)) {
        const data = event.split(/\r?\n/)
            .filter((line) => line.startsWith("data:"))
            .map((line) => line.slice(5).trimStart())
            .join("\n");
        if (!data) continue;
        messages.push(JSON.parse(data));
    }

    return messages.find((message) => message.id === expectedId) || messages.at(-1);
}

class HttpTransport {
    constructor(serverName, config, logger) {
        this.serverName = serverName;
        this.url = new URL(config.url);
        this.headers = config.headers || {};
        this.logger = logger;
        this.protocolVersion = null;
        this.sessionId = null;
        this.client = this.url.protocol === "https:" ? https : http;
        this.agent = new this.client.Agent({ keepAlive: true });
        this.keylogSockets = new WeakSet();
    }

    async start() {
        if (!["http:", "https:"].includes(this.url.protocol)) {
            throw new Error(`Unsupported HTTP protocol for ${this.serverName}: ${this.url.protocol}`);
        }
    }

    setProtocolVersion(protocolVersion) {
        this.protocolVersion = protocolVersion;
    }

    attachTlsKeyLogger(socket) {
        const keylogFile = process.env.TLS_KEYLOG_FILE;
        if (!keylogFile || this.url.protocol !== "https:" || this.keylogSockets.has(socket)) return;
        this.keylogSockets.add(socket);
        fs.mkdirSync(path.dirname(path.resolve(keylogFile)), { recursive: true });
        socket.on("keylog", (line) => fs.appendFileSync(keylogFile, line));
    }

    send(message, timeoutMs = 15000) {
        this.logger.log({ server: this.serverName, transport: "http", direction: "sent", message });
        const payload = JSON.stringify(message);
        const headers = {
            Accept: "application/json, text/event-stream",
            "Content-Length": Buffer.byteLength(payload),
            "Content-Type": "application/json",
            ...this.headers
        };
        if (this.protocolVersion) headers["MCP-Protocol-Version"] = this.protocolVersion;
        if (this.sessionId) headers["MCP-Session-Id"] = this.sessionId;

        return new Promise((resolve, reject) => {
            const request = this.client.request(this.url, {
                method: "POST",
                headers,
                agent: this.agent
            }, (response) => {
                if (response.headers["mcp-session-id"]) this.sessionId = response.headers["mcp-session-id"];

                const chunks = [];
                let length = 0;
                response.on("data", (chunk) => {
                    length += chunk.length;
                    if (length > MAX_RESPONSE_LENGTH) {
                        request.destroy(new Error(`Response from ${this.serverName} is too large.`));
                        return;
                    }
                    chunks.push(chunk);
                });
                response.on("end", () => {
                    const body = Buffer.concat(chunks).toString("utf8");
                    if (response.statusCode === 202) {
                        this.logger.log({
                            server: this.serverName,
                            transport: "http",
                            direction: "received",
                            category: "http-status",
                            message: { status: 202, body: null }
                        });
                        resolve(undefined);
                        return;
                    }
                    if (response.statusCode < 200 || response.statusCode >= 300) {
                        this.logger.log({
                            server: this.serverName,
                            transport: "http",
                            direction: "received",
                            category: "http-error",
                            message: { status: response.statusCode, body }
                        });
                        reject(new Error(`${this.serverName} returned HTTP ${response.statusCode}: ${body}`));
                        return;
                    }

                    try {
                        const contentType = response.headers["content-type"] || "";
                        const result = contentType.includes("text/event-stream")
                            ? parseSse(body, message.id)
                            : JSON.parse(body);
                        this.logger.log({ server: this.serverName, transport: "http", direction: "received", message: result });
                        resolve(result);
                    } catch (error) {
                        reject(new Error(`Unable to parse ${this.serverName} response: ${error.message}`));
                    }
                });
            });

            request.setTimeout(timeoutMs, () => request.destroy(new Error(`Request to ${this.serverName} timed out.`)));
            request.on("socket", (socket) => this.attachTlsKeyLogger(socket));
            request.on("error", reject);
            request.end(payload);
        });
    }

    async close() {
        this.agent.destroy();
    }
}

module.exports = { HttpTransport, parseSse };

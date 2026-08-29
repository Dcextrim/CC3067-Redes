"use strict";

const crypto = require("node:crypto");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");

const { createResponder } = require("./mcp/json-rpc.js");
const { createMessageHandler } = require("./mcp/message-handler.js");
const { SUPPORTED_PROTOCOL_VERSIONS } = require("./mcp/request-handler.js");

const MAX_BODY_LENGTH = 1024 * 1024;

function parseAllowedOrigins(value = "") {
    return value.split(",").map((origin) => origin.trim()).filter(Boolean);
}

function secureTokenEquals(received, expected) {
    const receivedBuffer = Buffer.from(received);
    const expectedBuffer = Buffer.from(expected);
    return receivedBuffer.length === expectedBuffer.length
        && crypto.timingSafeEqual(receivedBuffer, expectedBuffer);
}

function isAuthorized(request, authToken) {
    if (!authToken) return true;
    const authorization = request.headers.authorization || "";
    const prefix = "Bearer ";
    return authorization.startsWith(prefix)
        && secureTokenEquals(authorization.slice(prefix.length), authToken);
}

function getCorsHeaders(request, allowedOrigins) {
    const origin = request.headers.origin;
    if (!origin) return {};
    if (!allowedOrigins.includes("*") && !allowedOrigins.includes(origin)) return null;
    return {
        "Access-Control-Allow-Origin": allowedOrigins.includes("*") ? "*" : origin,
        "Access-Control-Allow-Headers": "Authorization, Content-Type, MCP-Protocol-Version",
        "Access-Control-Allow-Methods": "POST, OPTIONS",
        Vary: "Origin"
    };
}

function sendJson(response, statusCode, body, headers = {}) {
    const payload = JSON.stringify(body);
    response.writeHead(statusCode, {
        "Cache-Control": "no-store",
        "Content-Length": Buffer.byteLength(payload),
        "Content-Type": "application/json; charset=utf-8",
        "X-Content-Type-Options": "nosniff",
        ...headers
    });
    response.end(payload);
}

function sendHttpError(response, statusCode, message, headers = {}) {
    sendJson(response, statusCode, {
        jsonrpc: "2.0",
        id: null,
        error: { code: -32000, message }
    }, headers);
}

function readRequestBody(request) {
    return new Promise((resolve, reject) => {
        const chunks = [];
        let length = 0;

        request.on("data", (chunk) => {
            length += chunk.length;
            if (length > MAX_BODY_LENGTH) {
                reject(new Error("Request body exceeds the maximum supported length."));
                return;
            }
            chunks.push(chunk);
        });
        request.on("end", () => resolve(Buffer.concat(chunks).toString("utf8")));
        request.on("error", reject);
    });
}

function createRemoteServer(options = {}) {
    const authToken = options.authToken ?? process.env.MCP_AUTH_TOKEN ?? "";
    const allowedOrigins = options.allowedOrigins ?? parseAllowedOrigins(process.env.ALLOWED_ORIGINS);

    return http.createServer(async (request, response) => {
        const requestUrl = new URL(request.url, "http://localhost");

        if (request.method === "GET" && requestUrl.pathname === "/health") {
            sendJson(response, 200, { status: "ok", server: "kaeser-assistant", transport: "streamable-http" });
            return;
        }

        if (requestUrl.pathname !== "/mcp") {
            sendHttpError(response, 404, "Not found");
            return;
        }

        const corsHeaders = getCorsHeaders(request, allowedOrigins);
        if (corsHeaders === null) {
            sendHttpError(response, 403, "Forbidden origin");
            return;
        }

        if (request.method === "OPTIONS") {
            response.writeHead(204, corsHeaders);
            response.end();
            return;
        }

        if (!isAuthorized(request, authToken)) {
            sendHttpError(response, 401, "Unauthorized", { "WWW-Authenticate": "Bearer", ...corsHeaders });
            return;
        }

        if (request.method === "GET" || request.method === "DELETE") {
            sendHttpError(response, 405, "This stateless server does not expose an SSE stream or explicit session termination.", {
                Allow: "POST, OPTIONS",
                ...corsHeaders
            });
            return;
        }

        if (request.method !== "POST") {
            sendHttpError(response, 405, "Method not allowed", { Allow: "POST, OPTIONS", ...corsHeaders });
            return;
        }

        const contentType = request.headers["content-type"] || "";
        if (!contentType.toLowerCase().startsWith("application/json")) {
            sendHttpError(response, 415, "Content-Type must be application/json", corsHeaders);
            return;
        }

        const accept = request.headers.accept || "";
        if (!accept.includes("application/json") || !accept.includes("text/event-stream")) {
            sendHttpError(response, 406, "Accept must include application/json and text/event-stream", corsHeaders);
            return;
        }

        const protocolVersion = request.headers["mcp-protocol-version"];
        if (protocolVersion && !SUPPORTED_PROTOCOL_VERSIONS.includes(protocolVersion)) {
            sendHttpError(response, 400, `Unsupported MCP protocol version: ${protocolVersion}`, corsHeaders);
            return;
        }

        let body;
        try {
            body = await readRequestBody(request);
        } catch (error) {
            if (!response.headersSent) sendHttpError(response, 413, error.message, corsHeaders);
            return;
        }

        let message;
        try {
            message = JSON.parse(body);
        } catch {
            sendJson(response, 400, {
                jsonrpc: "2.0",
                id: null,
                error: { code: -32700, message: "Parse error" }
            }, corsHeaders);
            return;
        }

        const outgoing = [];
        const responder = createResponder((result) => outgoing.push(result));
        createMessageHandler(responder)(message);

        if (outgoing.length === 0) {
            response.writeHead(202, { "Cache-Control": "no-store", ...corsHeaders });
            response.end();
            return;
        }

        sendJson(response, 200, outgoing[0], corsHeaders);
    });
}

function startRemoteServer() {
    if (!process.env.KAESER_LECTURAS_PATH) {
        process.env.KAESER_LECTURAS_PATH = path.join(os.tmpdir(), "kaeser-lecturas.json");
    }

    const port = Number.parseInt(process.env.PORT || "8080", 10);
    const server = createRemoteServer();
    server.listen(port, "0.0.0.0", () => {
        console.log(`KAESER remote MCP server listening on 0.0.0.0:${port}/mcp`);
    });
}

if (require.main === module) startRemoteServer();

module.exports = { createRemoteServer, parseAllowedOrigins, startRemoteServer };

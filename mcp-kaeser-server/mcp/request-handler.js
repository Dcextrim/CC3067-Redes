"use strict";

const toolDefinitions = require("./tool-definitions.js");
const { createToolHandler } = require("./tool-handler.js");
const { isPlainObject, validateInitializeParams } = require("./validation.js");

const SUPPORTED_PROTOCOL_VERSIONS = [
    "2025-11-25",
    "2025-06-18",
    "2025-03-26",
    "2024-11-05"
];

function createRequestHandler(responder) {
    const { sendRequestError, sendResult } = responder;
    const handleToolCall = createToolHandler(responder);

    function handleInitialize(request) {
        const validationError = validateInitializeParams(request.params);
        if (validationError) {
            sendRequestError(request, -32602, "Invalid params", validationError);
            return;
        }

        const requestedVersion = request.params.protocolVersion;
        const protocolVersion = SUPPORTED_PROTOCOL_VERSIONS.includes(requestedVersion)
            ? requestedVersion
            : SUPPORTED_PROTOCOL_VERSIONS[0];

        sendResult(request, {
            protocolVersion,
            serverInfo: {
                name: "kaeser-assistant",
                version: "1.1.0",
                description: "Local industrial compressor maintenance assistant"
            },
            capabilities: { tools: { listChanged: false } },
            instructions: "Technical data and maintenance records are demonstrative. Always follow the equipment manual and plant safety procedures."
        });
    }

    return function handleRequest(request) {
        switch (request.method) {
            case "initialize":
                handleInitialize(request);
                return;
            case "notifications/initialized":
            case "notifications/cancelled":
                return;
            case "ping":
                sendResult(request, {});
                return;
            case "tools/list":
                if (request.params !== undefined && !isPlainObject(request.params)) {
                    sendRequestError(request, -32602, "Invalid params", "tools/list params must be an object when provided.");
                    return;
                }
                sendResult(request, { tools: toolDefinitions });
                return;
            case "tools/call":
                handleToolCall(request);
                return;
            default:
                sendRequestError(request, -32601, "Method not found");
        }
    };
}

module.exports = { SUPPORTED_PROTOCOL_VERSIONS, createRequestHandler };

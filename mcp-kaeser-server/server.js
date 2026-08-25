"use strict";

const { stdioResponder } = require("./mcp/json-rpc.js");
const { createMessageHandler } = require("./mcp/message-handler.js");

const MAX_MESSAGE_LENGTH = 1024 * 1024;
const handleMessage = createMessageHandler(stdioResponder);

function processLine(line) {
    if (line.trim().length === 0) return;

    let request;
    try {
        request = JSON.parse(line);
    } catch {
        stdioResponder.sendProtocolError(null, -32700, "Parse error");
        return;
    }

    handleMessage(request);
}

let inputBuffer = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", (chunk) => {
    inputBuffer += chunk;

    if (inputBuffer.length > MAX_MESSAGE_LENGTH && !inputBuffer.includes("\n")) {
        stdioResponder.sendProtocolError(null, -32700, "Parse error", "Message exceeds the maximum supported length.");
        inputBuffer = "";
        return;
    }

    let newlineIndex;
    while ((newlineIndex = inputBuffer.indexOf("\n")) !== -1) {
        const line = inputBuffer.slice(0, newlineIndex);
        inputBuffer = inputBuffer.slice(newlineIndex + 1);
        processLine(line);
    }
});

process.stdin.on("end", () => {
    if (inputBuffer.trim().length > 0) processLine(inputBuffer);
});

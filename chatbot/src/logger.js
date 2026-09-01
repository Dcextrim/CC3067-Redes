"use strict";

const fs = require("node:fs");
const path = require("node:path");

function safeTimestamp() {
    return new Date().toISOString().replace(/[:.]/g, "-");
}

class InteractionLogger {
    constructor(logDirectory) {
        this.logDirectory = path.resolve(logDirectory);
        fs.mkdirSync(this.logDirectory, { recursive: true });
        this.filePath = path.join(this.logDirectory, `mcp-${safeTimestamp()}.jsonl`);
    }

    log({ server, transport, direction, message, category = "json-rpc" }) {
        const entry = {
            timestamp: new Date().toISOString(),
            category,
            server,
            transport,
            direction,
            message
        };
        fs.appendFileSync(this.filePath, `${JSON.stringify(entry)}\n`, "utf8");
    }

    readTail(limit = 20) {
        if (!fs.existsSync(this.filePath)) return [];
        const lines = fs.readFileSync(this.filePath, "utf8").trim().split(/\r?\n/).filter(Boolean);
        return lines.slice(-Math.max(1, limit)).map((line) => JSON.parse(line));
    }
}

module.exports = { InteractionLogger };

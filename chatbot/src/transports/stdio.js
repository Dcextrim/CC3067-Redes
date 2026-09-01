"use strict";

const { spawn } = require("node:child_process");

class StdioTransport {
    constructor(serverName, config, logger) {
        this.serverName = serverName;
        this.config = config;
        this.logger = logger;
        this.child = null;
        this.pending = new Map();
        this.stdoutBuffer = "";
        this.closed = false;
    }

    async start() {
        this.child = spawn(this.config.command, this.config.args || [], {
            cwd: this.config.cwd,
            env: { ...process.env, ...(this.config.env || {}) },
            shell: false,
            stdio: ["pipe", "pipe", "pipe"],
            windowsHide: true
        });

        this.child.stdout.setEncoding("utf8");
        this.child.stdout.on("data", (chunk) => this.handleStdout(chunk));
        this.child.stderr.setEncoding("utf8");
        this.child.stderr.on("data", (chunk) => {
            this.logger.log({
                server: this.serverName,
                transport: "stdio",
                direction: "received",
                category: "stderr",
                message: chunk.trimEnd()
            });
        });
        this.child.on("close", (code) => {
            this.closed = true;
            this.rejectPending(new Error(`MCP server ${this.serverName} exited with code ${code}.`));
        });

        await new Promise((resolve, reject) => {
            this.child.once("spawn", resolve);
            this.child.once("error", reject);
        });
    }

    handleStdout(chunk) {
        this.stdoutBuffer += chunk;
        let newlineIndex;
        while ((newlineIndex = this.stdoutBuffer.indexOf("\n")) !== -1) {
            const line = this.stdoutBuffer.slice(0, newlineIndex);
            this.stdoutBuffer = this.stdoutBuffer.slice(newlineIndex + 1);
            if (!line.trim()) continue;

            let message;
            try {
                message = JSON.parse(line);
            } catch {
                this.rejectPending(new Error(`${this.serverName} wrote non-JSON data to stdout.`));
                continue;
            }

            this.logger.log({ server: this.serverName, transport: "stdio", direction: "received", message });
            if (Object.prototype.hasOwnProperty.call(message, "id") && !message.method) {
                const pending = this.pending.get(message.id);
                if (pending) {
                    clearTimeout(pending.timer);
                    this.pending.delete(message.id);
                    pending.resolve(message);
                }
                continue;
            }

            if (message.method && Object.prototype.hasOwnProperty.call(message, "id")) {
                this.writeMessage({
                    jsonrpc: "2.0",
                    id: message.id,
                    error: { code: -32601, message: "Client method not found" }
                });
            }
        }
    }

    writeMessage(message) {
        if (!this.child || this.closed || !this.child.stdin.writable) {
            throw new Error(`MCP server ${this.serverName} is not running.`);
        }
        this.logger.log({ server: this.serverName, transport: "stdio", direction: "sent", message });
        this.child.stdin.write(`${JSON.stringify(message)}\n`);
    }

    send(message, timeoutMs = 30000) {
        if (!Object.prototype.hasOwnProperty.call(message, "id")) {
            this.writeMessage(message);
            return Promise.resolve(undefined);
        }

        return new Promise((resolve, reject) => {
            const timer = setTimeout(() => {
                this.pending.delete(message.id);
                reject(new Error(`Timed out waiting for ${this.serverName} to answer request ${message.id}.`));
            }, timeoutMs);
            this.pending.set(message.id, { resolve, reject, timer });

            try {
                this.writeMessage(message);
            } catch (error) {
                clearTimeout(timer);
                this.pending.delete(message.id);
                reject(error);
            }
        });
    }

    rejectPending(error) {
        for (const pending of this.pending.values()) {
            clearTimeout(pending.timer);
            pending.reject(error);
        }
        this.pending.clear();
    }

    async close() {
        if (!this.child || this.closed) return;
        await new Promise((resolve) => {
            const timer = setTimeout(() => {
                if (!this.closed) this.child.kill();
            }, 1500);
            timer.unref();
            this.child.once("close", () => {
                clearTimeout(timer);
                resolve();
            });
            this.child.stdin.end();
        });
    }
}

module.exports = { StdioTransport };

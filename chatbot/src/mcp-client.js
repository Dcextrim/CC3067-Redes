"use strict";

class McpClient {
    constructor(serverName, transport) {
        this.serverName = serverName;
        this.transport = transport;
        this.nextRequestId = 1;
        this.serverInfo = null;
        this.tools = [];
    }

    async request(method, params) {
        const request = { jsonrpc: "2.0", id: this.nextRequestId, method };
        this.nextRequestId += 1;
        if (params !== undefined) request.params = params;

        const response = await this.transport.send(request);
        if (!response) throw new Error(`${this.serverName} returned no JSON-RPC response.`);
        if (response.error) {
            const error = new Error(`${response.error.message}${response.error.data ? `: ${response.error.data}` : ""}`);
            error.code = response.error.code;
            throw error;
        }
        return response.result;
    }

    async notify(method, params) {
        const notification = { jsonrpc: "2.0", method };
        if (params !== undefined) notification.params = params;
        await this.transport.send(notification);
    }

    async connect() {
        await this.transport.start();
        const initialization = await this.request("initialize", {
            protocolVersion: "2025-11-25",
            capabilities: {},
            clientInfo: { name: "kaeser-terminal-chatbot", version: "1.0.0" }
        });
        this.serverInfo = initialization.serverInfo;
        if (typeof this.transport.setProtocolVersion === "function") {
            this.transport.setProtocolVersion(initialization.protocolVersion);
        }
        await this.notify("notifications/initialized");
        const toolList = await this.request("tools/list", {});
        this.tools = toolList.tools || [];
        return this.tools;
    }

    callTool(name, args) {
        return this.request("tools/call", { name, arguments: args });
    }

    close() {
        return this.transport.close();
    }
}

module.exports = { McpClient };

"use strict";

const { McpClient } = require("./mcp-client.js");
const { HttpTransport } = require("./transports/http.js");
const { StdioTransport } = require("./transports/stdio.js");

const SYSTEM_PROMPT = [
    "You are a technical assistant running inside a terminal MCP host.",
    "Use the available MCP tools whenever the user asks for information or actions they provide.",
    "Never invent tool results. Clearly distinguish official technical data from simulated data.",
    "For industrial maintenance, repeat safety warnings and never claim to control physical machinery."
].join(" ");

function sanitizeToolName(serverName, toolName) {
    return `${serverName}__${toolName}`.replace(/[^a-zA-Z0-9_-]/g, "_").slice(0, 64);
}

function createMcpClient(serverName, config, logger) {
    const transport = config.transport === "http"
        ? new HttpTransport(serverName, config, logger)
        : new StdioTransport(serverName, config, logger);
    return new McpClient(serverName, transport);
}

function toolResultText(result) {
    if (result?.structuredContent !== undefined) return JSON.stringify(result.structuredContent, null, 2);
    if (Array.isArray(result?.content)) {
        const text = result.content.filter((block) => block.type === "text").map((block) => block.text).join("\n");
        if (text) return text;
    }
    return JSON.stringify(result ?? null, null, 2);
}

class ChatHost {
    constructor(options) {
        this.config = options.config;
        this.logger = options.logger;
        this.llmClient = options.llmClient;
        this.clientFactory = options.clientFactory || createMcpClient;
        this.onStatus = options.onStatus || (() => {});
        this.onAssistantText = options.onAssistantText || (() => {});
        this.onToolUse = options.onToolUse || (() => {});
        this.clients = new Map();
        this.toolRegistry = new Map();
        this.apiTools = [];
        this.messages = [];
    }

    async connect() {
        for (const [serverName, serverConfig] of Object.entries(this.config.mcpServers)) {
            const client = this.clientFactory(serverName, serverConfig, this.logger);
            try {
                this.onStatus({ serverName, state: "connecting", transport: serverConfig.transport });
                const tools = await client.connect();
                this.clients.set(serverName, client);
                this.registerTools(serverName, serverConfig, client, tools);
                this.onStatus({ serverName, state: "connected", transport: serverConfig.transport, toolCount: tools.length });
            } catch (error) {
                await client.close().catch(() => {});
                this.onStatus({ serverName, state: "failed", transport: serverConfig.transport, error });
                if (serverConfig.required !== false) {
                    await this.close();
                    throw new Error(`Required MCP server ${serverName} failed: ${error.message}`);
                }
            }
        }
        return this.apiTools;
    }

    registerTools(serverName, serverConfig, client, tools) {
        for (const tool of tools) {
            const apiName = sanitizeToolName(serverName, tool.name);
            if (this.toolRegistry.has(apiName)) throw new Error(`Duplicate API tool name: ${apiName}`);
            this.toolRegistry.set(apiName, { serverName, originalName: tool.name, client });
            this.apiTools.push({
                name: apiName,
                description: `[MCP server: ${serverName}] ${tool.description || tool.name}${serverConfig.toolHint ? ` ${serverConfig.toolHint}` : ""}`,
                input_schema: tool.inputSchema || { type: "object", properties: {} }
            });
        }
    }

    async executeToolUse(toolUse) {
        const registration = this.toolRegistry.get(toolUse.name);
        if (!registration) {
            return {
                type: "tool_result",
                tool_use_id: toolUse.id,
                content: `Unknown tool selected by the model: ${toolUse.name}`,
                is_error: true
            };
        }

        this.onToolUse({
            serverName: registration.serverName,
            toolName: registration.originalName,
            input: toolUse.input
        });

        try {
            const result = await registration.client.callTool(registration.originalName, toolUse.input || {});
            return {
                type: "tool_result",
                tool_use_id: toolUse.id,
                content: toolResultText(result),
                is_error: Boolean(result?.isError)
            };
        } catch (error) {
            return {
                type: "tool_result",
                tool_use_id: toolUse.id,
                content: error.message,
                is_error: true
            };
        }
    }

    async ask(userText) {
        this.messages.push({ role: "user", content: userText });

        for (let round = 0; round < 8; round += 1) {
            const response = await this.llmClient.createMessage(this.messages, this.apiTools, SYSTEM_PROMPT);
            const content = Array.isArray(response.content) ? response.content : [];
            this.messages.push({ role: "assistant", content });

            for (const block of content.filter((item) => item.type === "text")) {
                this.onAssistantText(block.text);
            }

            const toolUses = content.filter((item) => item.type === "tool_use");
            if (toolUses.length === 0) return response;

            const toolResults = [];
            for (const toolUse of toolUses) toolResults.push(await this.executeToolUse(toolUse));
            this.messages.push({ role: "user", content: toolResults });
        }

        throw new Error("The model exceeded the maximum of 8 consecutive tool-use rounds.");
    }

    clearContext() {
        this.messages = [];
    }

    async close() {
        await Promise.allSettled([...this.clients.values()].map((client) => client.close()));
        this.clients.clear();
    }
}

module.exports = { ChatHost, SYSTEM_PROMPT, sanitizeToolName, toolResultText };

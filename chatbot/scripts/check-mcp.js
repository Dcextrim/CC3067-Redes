"use strict";

const path = require("node:path");

const { loadConfig } = require("../src/config.js");
const { InteractionLogger } = require("../src/logger.js");
const { McpClient } = require("../src/mcp-client.js");
const { HttpTransport } = require("../src/transports/http.js");
const { StdioTransport } = require("../src/transports/stdio.js");

function getConfigPath() {
    const configIndex = process.argv.indexOf("--config");
    if (configIndex !== -1 && process.argv[configIndex + 1]) return process.argv[configIndex + 1];
    return path.join(__dirname, "..", "config.example.json");
}

async function main() {
    const { config } = loadConfig(getConfigPath());
    const logger = new InteractionLogger(path.join(__dirname, "..", "logs"));
    let requiredFailure = false;

    for (const [serverName, serverConfig] of Object.entries(config.mcpServers)) {
        const transport = serverConfig.transport === "http"
            ? new HttpTransport(serverName, serverConfig, logger)
            : new StdioTransport(serverName, serverConfig, logger);
        const client = new McpClient(serverName, transport);

        try {
            const tools = await client.connect();
            console.log(`[OK] ${serverName} (${serverConfig.transport}): ${tools.length} tools`);
            for (const tool of tools) console.log(`  - ${tool.name}`);
        } catch (error) {
            console.error(`[FAIL] ${serverName}: ${error.message}`);
            if (serverConfig.required !== false) requiredFailure = true;
        } finally {
            await client.close().catch(() => {});
        }
    }

    console.log(`MCP interaction log: ${logger.filePath}`);
    if (requiredFailure) process.exitCode = 1;
}

main().catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
});

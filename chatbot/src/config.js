"use strict";

const fs = require("node:fs");
const path = require("node:path");

const VARIABLE_PATTERN = /\$\{([A-Z][A-Z0-9_]*)\}/g;

function expandString(value, variables) {
    return value.replace(VARIABLE_PATTERN, (_, name) => {
        if (!Object.prototype.hasOwnProperty.call(variables, name) || variables[name] === "") {
            throw new Error(`Missing environment variable or built-in value: ${name}`);
        }
        return variables[name];
    });
}

function expandValue(value, variables) {
    if (typeof value === "string") return expandString(value, variables);
    if (Array.isArray(value)) return value.map((item) => expandValue(item, variables));
    if (value && typeof value === "object") {
        return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, expandValue(item, variables)]));
    }
    return value;
}

function validateConfig(config) {
    if (!config || typeof config !== "object" || Array.isArray(config)) throw new Error("The chatbot configuration must be a JSON object.");
    if (!config.llm || typeof config.llm !== "object") throw new Error("The chatbot configuration requires an llm object.");
    if (!config.llm.model || typeof config.llm.model !== "string") throw new Error("llm.model must be a non-empty string.");
    if (!config.mcpServers || typeof config.mcpServers !== "object" || Array.isArray(config.mcpServers)) {
        throw new Error("The chatbot configuration requires an mcpServers object.");
    }

    for (const [name, server] of Object.entries(config.mcpServers)) {
        if (!server || typeof server !== "object") throw new Error(`Invalid MCP server configuration: ${name}`);
        if (server.enabled === false) continue;
        if (!["stdio", "http"].includes(server.transport)) throw new Error(`Unsupported transport for ${name}: ${server.transport}`);
        if (server.transport === "stdio" && typeof server.command !== "string") throw new Error(`${name}.command is required for stdio.`);
        if (server.transport === "http" && typeof server.url !== "string") throw new Error(`${name}.url is required for HTTP.`);
    }
}

function loadConfig(configPath) {
    const absolutePath = path.resolve(configPath);
    if (!fs.existsSync(absolutePath)) {
        throw new Error(`Configuration file not found: ${absolutePath}. Copy config.example.json to config.json first.`);
    }

    let parsed;
    try {
        parsed = JSON.parse(fs.readFileSync(absolutePath, "utf8"));
    } catch (error) {
        throw new Error(`Unable to read chatbot configuration: ${error.message}`);
    }

    const projectRoot = path.resolve(path.dirname(absolutePath), "..");
    const variables = { ...process.env, PROJECT_ROOT: projectRoot };
    const enabledServers = Object.fromEntries(
        Object.entries(parsed.mcpServers || {}).filter(([, server]) => server.enabled !== false)
    );
    const config = {
        ...parsed,
        llm: expandValue(parsed.llm || {}, variables),
        mcpServers: expandValue(enabledServers, variables)
    };

    if (process.env.ANTHROPIC_MODEL) config.llm.model = process.env.ANTHROPIC_MODEL;
    validateConfig(config);
    return { config, configPath: absolutePath, projectRoot };
}

module.exports = { expandValue, loadConfig, validateConfig };

"use strict";

const path = require("node:path");
const readline = require("node:readline/promises");

const { AnthropicClient } = require("./src/anthropic-client.js");
const { ChatHost } = require("./src/chat-host.js");
const { loadConfig } = require("./src/config.js");
const { InteractionLogger } = require("./src/logger.js");

const useColor = process.stdout.isTTY && !process.env.NO_COLOR;
const color = {
    cyan: (text) => useColor ? `\u001b[36m${text}\u001b[0m` : text,
    green: (text) => useColor ? `\u001b[32m${text}\u001b[0m` : text,
    yellow: (text) => useColor ? `\u001b[33m${text}\u001b[0m` : text,
    red: (text) => useColor ? `\u001b[31m${text}\u001b[0m` : text,
    bold: (text) => useColor ? `\u001b[1m${text}\u001b[0m` : text
};

function getConfigPath() {
    const configIndex = process.argv.indexOf("--config");
    if (configIndex !== -1 && process.argv[configIndex + 1]) return process.argv[configIndex + 1];
    return process.env.CHATBOT_CONFIG || path.join(__dirname, "config.json");
}

function printHelp() {
    console.log([
        "Available commands:",
        "  /help       Show this help",
        "  /tools      List discovered MCP tools",
        "  /log [n]    Show the last n MCP log entries (default: 20)",
        "  /clear      Clear the LLM conversation context",
        "  /exit       Close all MCP connections and exit"
    ].join("\n"));
}

function printLog(logger, count) {
    const entries = logger.readTail(count);
    if (entries.length === 0) {
        console.log("The MCP log is empty.");
        return;
    }
    for (const entry of entries) {
        console.log(`${entry.timestamp} ${entry.server} ${entry.transport} ${entry.direction}`);
        console.log(JSON.stringify(entry.message, null, 2));
    }
}

async function main() {
    if (!process.env.ANTHROPIC_API_KEY) {
        throw new Error("ANTHROPIC_API_KEY is required. Set it in the terminal before starting the chatbot.");
    }

    const { config, projectRoot } = loadConfig(getConfigPath());
    const logger = new InteractionLogger(path.join(__dirname, "logs"));
    const llmClient = new AnthropicClient({
        apiKey: process.env.ANTHROPIC_API_KEY,
        model: config.llm.model,
        maxTokens: config.llm.maxTokens,
        baseUrl: process.env.ANTHROPIC_BASE_URL
    });
    const host = new ChatHost({
        config,
        logger,
        llmClient,
        onStatus(status) {
            if (status.state === "connecting") console.log(color.cyan(`[MCP] Connecting to ${status.serverName} using ${status.transport}...`));
            if (status.state === "connected") console.log(color.green(`[MCP] ${status.serverName} connected (${status.toolCount} tools).`));
            if (status.state === "failed") console.warn(color.yellow(`[MCP] Optional server ${status.serverName} unavailable: ${status.error.message}`));
        },
        onToolUse(event) {
            console.log(color.cyan(`[MCP] Calling ${event.serverName}/${event.toolName}`));
            console.log(JSON.stringify(event.input, null, 2));
        },
        onAssistantText(text) {
            console.log(`\n${color.bold("Claude:")} ${text}`);
        }
    });

    console.log(color.bold("KAESER MCP Terminal Chatbot"));
    console.log(`Project: ${projectRoot}`);
    console.log(`Model: ${config.llm.model}`);
    console.log(`MCP log: ${logger.filePath}\n`);
    await host.connect();
    console.log(color.green(`\nReady with ${host.apiTools.length} MCP tools. Type /help for commands.`));

    const terminal = readline.createInterface({ input: process.stdin, output: process.stdout });
    try {
        while (true) {
            const input = (await terminal.question(`\n${color.bold("You:")} `)).trim();
            if (!input) continue;
            if (input === "/exit") break;
            if (input === "/help") {
                printHelp();
                continue;
            }
            if (input === "/tools") {
                for (const tool of host.apiTools) console.log(`- ${tool.name}: ${tool.description}`);
                continue;
            }
            if (input.startsWith("/log")) {
                const requested = Number.parseInt(input.split(/\s+/)[1] || "20", 10);
                printLog(logger, Number.isFinite(requested) ? requested : 20);
                continue;
            }
            if (input === "/clear") {
                host.clearContext();
                console.log(color.green("Conversation context cleared."));
                continue;
            }

            try {
                await host.ask(input);
            } catch (error) {
                console.error(color.red(`Request failed: ${error.message}`));
            }
        }
    } finally {
        terminal.close();
        await host.close();
    }
}

main().catch((error) => {
    console.error(color.red(error.message));
    process.exitCode = 1;
});

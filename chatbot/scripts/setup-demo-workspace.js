"use strict";

const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const projectRoot = path.resolve(__dirname, "..", "..");
const demoWorkspace = path.join(projectRoot, "demo-workspace");

fs.mkdirSync(demoWorkspace, { recursive: true });

function runGit(args) {
    const result = spawnSync("git", args, { cwd: demoWorkspace, encoding: "utf8" });
    if (result.status !== 0) throw new Error(result.stderr || result.stdout || `git ${args.join(" ")} failed`);
}

if (!fs.existsSync(path.join(demoWorkspace, ".git"))) runGit(["init"]);
runGit(["config", "user.name", "MCP Demo"]);
runGit(["config", "user.email", "mcp-demo@example.invalid"]);

console.log(`Demonstration repository ready at ${demoWorkspace}`);

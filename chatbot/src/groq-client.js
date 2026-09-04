"use strict";

const http = require("node:http");
const https = require("node:https");

const MAX_RESPONSE_LENGTH = 10 * 1024 * 1024;
const MAX_TOOLS_PER_REQUEST = 8;

const TOOL_KEYWORDS = {
    consultar_specs: ["especificacion", "spec", "potencia", "caudal", "dimension", "modelo", "compresor"],
    diagnosticar_falla: ["diagnost", "falla", "codigo", "alarma", "mensaje", "averia"],
    calcular_mantenimiento_pendiente: ["mantenimiento", "servicio", "horas", "serie", "proximo mantenimiento"],
    consultar_disponibilidad_repuesto: ["repuesto", "inventario", "disponibilidad", "numero de parte"],
    registrar_lectura_sensor: ["sensor", "lectura", "presion", "temperatura", "registrar"],
    read_file: ["leer archivo", "lee el archivo", "read file"],
    read_text_file: ["leer archivo", "lee el archivo", "contenido del archivo", "read file"],
    read_media_file: ["imagen", "audio", "media"],
    read_multiple_files: ["leer archivos", "varios archivos", "multiple files"],
    write_file: ["crear archivo", "escribir archivo", "guarda el archivo", "write file"],
    edit_file: ["editar archivo", "modificar archivo", "edit file"],
    create_directory: ["crear directorio", "crear carpeta", "create directory"],
    list_directory: ["listar directorio", "listar carpeta", "lista los archivos", "list directory"],
    list_directory_with_sizes: ["tamanos", "sizes"],
    directory_tree: ["arbol de directorios", "directory tree"],
    move_file: ["mover archivo", "renombrar archivo", "move file"],
    search_files: ["buscar archivos", "encontrar archivo", "search files"],
    get_file_info: ["informacion del archivo", "metadatos del archivo", "file info"],
    list_allowed_directories: ["directorios permitidos", "allowed directories"],
    git_status: ["git status", "estado del repositorio"],
    git_diff_unstaged: ["git diff unstaged", "cambios sin preparar"],
    git_diff_staged: ["git diff staged", "cambios preparados"],
    git_diff: ["git diff", "diferencias"],
    git_commit: ["git commit", "crear commit", "haz commit"],
    git_add: ["git add", "agregar al staging", "preparar archivo"],
    git_reset: ["git reset", "quitar del staging"],
    git_log: ["git log", "historial de commits"],
    git_create_branch: ["crear rama", "create branch"],
    git_checkout: ["cambiar rama", "checkout"],
    git_show: ["git show", "mostrar commit"],
    git_branch: ["listar ramas", "list branches"]
};

function normalizeText(value) {
    return String(value || "").normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLowerCase();
}

function latestUserText(messages) {
    for (let index = messages.length - 1; index >= 0; index -= 1) {
        if (messages[index].role === "user" && typeof messages[index].content === "string") {
            return messages[index].content;
        }
    }
    return "";
}

function recentToolNames(messages) {
    const names = [];
    for (let index = messages.length - 1; index >= 0 && names.length < 2; index -= 1) {
        const content = messages[index].content;
        if (!Array.isArray(content)) continue;
        for (const block of content) {
            if (block.type === "tool_use" && !names.includes(block.name)) names.push(block.name);
        }
    }
    return names;
}

function selectRelevantTools(messages, tools) {
    if (tools.length <= MAX_TOOLS_PER_REQUEST) return tools;

    const prompt = normalizeText(latestUserText(messages));
    const previous = new Set(recentToolNames(messages));
    const scored = tools.map((tool, index) => {
        const shortName = tool.name.split("__").pop();
        const normalizedName = normalizeText(shortName);
        let score = prompt.includes(normalizedName) ? 100 : 0;
        for (const keyword of TOOL_KEYWORDS[shortName] || []) {
            if (prompt.includes(normalizeText(keyword))) score += 10;
        }
        if (previous.has(tool.name)) score += 5;
        return { tool, score, index };
    });

    return scored
        .filter((item) => item.score > 0)
        .sort((left, right) => right.score - left.score || left.index - right.index)
        .slice(0, MAX_TOOLS_PER_REQUEST)
        .map((item) => item.tool);
}

function toGroqMessages(messages, system) {
    const result = [];
    if (system) result.push({ role: "system", content: system });

    for (const message of messages) {
        if (typeof message.content === "string") {
            result.push({ role: message.role, content: message.content });
            continue;
        }

        const blocks = Array.isArray(message.content) ? message.content : [];
        if (message.role === "assistant") {
            const text = blocks
                .filter((block) => block.type === "text")
                .map((block) => block.text)
                .join("\n");
            const toolCalls = blocks
                .filter((block) => block.type === "tool_use")
                .map((block) => ({
                    id: block.id,
                    type: "function",
                    function: {
                        name: block.name,
                        arguments: JSON.stringify(block.input || {})
                    }
                }));
            const converted = { role: "assistant", content: text || null };
            if (toolCalls.length > 0) converted.tool_calls = toolCalls;
            result.push(converted);
            continue;
        }

        const text = blocks
            .filter((block) => block.type === "text")
            .map((block) => block.text)
            .join("\n");
        if (text) result.push({ role: "user", content: text });
        for (const block of blocks.filter((item) => item.type === "tool_result")) {
            result.push({
                role: "tool",
                tool_call_id: block.tool_use_id,
                content: String(block.content)
            });
        }
    }

    return result;
}

function toGroqTools(tools) {
    return tools.map((tool) => ({
        type: "function",
        function: {
            name: tool.name,
            description: tool.description || tool.name,
            parameters: tool.input_schema || { type: "object", properties: {} }
        }
    }));
}

function toHostResponse(response) {
    const choice = response.choices?.[0];
    const message = choice?.message;
    const content = [];

    if (typeof message?.content === "string" && message.content.trim()) {
        content.push({ type: "text", text: message.content });
    }

    for (const [index, toolCall] of (message?.tool_calls || []).entries()) {
        let input;
        try {
            input = JSON.parse(toolCall.function?.arguments || "{}");
        } catch {
            throw new Error(`Groq returned invalid JSON arguments for tool ${toolCall.function?.name || index}.`);
        }
        content.push({
            type: "tool_use",
            id: toolCall.id || `groq_tool_${Date.now()}_${index}`,
            name: toolCall.function?.name,
            input
        });
    }

    if (content.length === 0) {
        throw new Error(`Groq API returned no usable content (${choice?.finish_reason || "empty response"}).`);
    }

    return { content, stop_reason: choice?.finish_reason };
}

class GroqClient {
    constructor(options) {
        this.apiKey = options.apiKey;
        this.model = options.model;
        this.maxTokens = options.maxTokens || 4096;
        this.baseUrl = new URL(options.baseUrl || "https://api.groq.com");
        this.client = this.baseUrl.protocol === "https:" ? https : http;
    }

    createMessage(messages, tools, system) {
        const relevantTools = selectRelevantTools(messages, tools);
        const body = {
            model: this.model,
            messages: toGroqMessages(messages, system),
            max_completion_tokens: this.maxTokens
        };
        if (relevantTools.length > 0) {
            body.tools = toGroqTools(relevantTools);
            body.tool_choice = "auto";
        }

        const payload = JSON.stringify(body);
        const url = new URL("/openai/v1/chat/completions", this.baseUrl);

        return new Promise((resolve, reject) => {
            const request = this.client.request(url, {
                method: "POST",
                headers: {
                    Authorization: `Bearer ${this.apiKey}`,
                    "Content-Length": Buffer.byteLength(payload),
                    "Content-Type": "application/json"
                }
            }, (response) => {
                const chunks = [];
                let length = 0;
                response.on("data", (chunk) => {
                    length += chunk.length;
                    if (length > MAX_RESPONSE_LENGTH) {
                        request.destroy(new Error("Groq API response exceeds the supported size."));
                        return;
                    }
                    chunks.push(chunk);
                });
                response.on("end", () => {
                    const responseBody = Buffer.concat(chunks).toString("utf8");
                    let parsed;
                    try {
                        parsed = JSON.parse(responseBody);
                    } catch {
                        reject(new Error(`Groq API returned invalid JSON (HTTP ${response.statusCode}).`));
                        return;
                    }

                    if (response.statusCode < 200 || response.statusCode >= 300) {
                        reject(new Error(`Groq API error ${response.statusCode}: ${parsed.error?.message || responseBody}`));
                        return;
                    }

                    try {
                        resolve(toHostResponse(parsed));
                    } catch (error) {
                        reject(error);
                    }
                });
            });

            request.setTimeout(60000, () => request.destroy(new Error("Groq API request timed out.")));
            request.on("error", reject);
            request.end(payload);
        });
    }
}

module.exports = { GroqClient, selectRelevantTools, toGroqMessages, toGroqTools, toHostResponse };

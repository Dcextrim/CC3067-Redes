"use strict";

const herramientas = require("./herramientas.js");

const MAX_MESSAGE_LENGTH = 1024 * 1024;
const SUPPORTED_PROTOCOL_VERSIONS = [
    "2025-11-25",
    "2025-06-18",
    "2025-03-26",
    "2024-11-05"
];

const sourceFields = {
    fuente: { type: "string" },
    advertencia: { type: "string" }
};

const toolDefinitions = [
    {
        name: "consultar_specs",
        description: "Consulta potencia, presion, caudal y dimensiones de un modelo de compresor.",
        inputSchema: {
            type: "object",
            properties: {
                modelo: { type: "string", minLength: 1, description: "Ej. SXC 3, SM 13 o ASK 34 SFC" }
            },
            required: ["modelo"],
            additionalProperties: false
        },
        outputSchema: {
            type: "object",
            properties: {
                modelo: { type: "string" },
                familia: { type: "string" },
                configuracion: { type: "string" },
                presion_trabajo_bar: { type: "number" },
                caudal_m3_min: {
                    oneOf: [
                        { type: "number" },
                        {
                            type: "object",
                            properties: { min: { type: "number" }, max: { type: "number" } },
                            required: ["min", "max"],
                            additionalProperties: false
                        }
                    ]
                },
                presion_max_bar: { type: "number" },
                potencia_kw: { type: "number" },
                dimensiones_mm: { type: "string" },
                ...sourceFields
            },
            required: ["modelo", "familia", "configuracion", "presion_trabajo_bar", "caudal_m3_min", "presion_max_bar", "potencia_kw", "dimensiones_mm", "fuente", "advertencia"],
            additionalProperties: true
        }
    },
    {
        name: "diagnosticar_falla",
        description: "Interpreta un mensaje documentado del SIGMA CONTROL 2 y propone verificaciones.",
        inputSchema: {
            type: "object",
            properties: {
                codigo_mensaje: { type: "string", pattern: "^\\d{4} [AaWw]$", description: "Ej. 0044 A" },
                modelo: { type: "string", minLength: 1 }
            },
            required: ["codigo_mensaje", "modelo"],
            additionalProperties: false
        },
        outputSchema: {
            type: "object",
            properties: {
                codigo_mensaje: { type: "string" },
                modelo_evaluado: { type: "string" },
                tipo: { type: "string" },
                descripcion: { type: "string" },
                causas_probables: { type: "array", items: { type: "string" } },
                acciones: { type: "array", items: { type: "string" } },
                requiere_servicio_autorizado: { type: "boolean" },
                ...sourceFields
            },
            required: ["codigo_mensaje", "modelo_evaluado", "tipo", "descripcion", "causas_probables", "acciones", "requiere_servicio_autorizado", "fuente", "advertencia"],
            additionalProperties: false
        }
    },
    {
        name: "calcular_mantenimiento_pendiente",
        description: "Compara las horas acumuladas con el plan local simulado de mantenimiento.",
        inputSchema: {
            type: "object",
            properties: {
                numero_serie: { type: "string", minLength: 1 },
                horas_operacion: { type: "number", minimum: 0 }
            },
            required: ["numero_serie", "horas_operacion"],
            additionalProperties: false
        },
        outputSchema: {
            type: "object",
            properties: {
                numero_serie: { type: "string" },
                estado: { type: "string", enum: ["al_dia", "proximo", "vencido"] },
                ultimo_servicio_horas: { type: "number" },
                siguiente_servicio_horas: { type: "number" },
                horas_restantes: { type: "number" },
                horas_vencidas: { type: "number" },
                recomendacion: { type: "string" },
                ...sourceFields
            },
            required: ["numero_serie", "estado", "ultimo_servicio_horas", "siguiente_servicio_horas", "horas_restantes", "horas_vencidas", "recomendacion", "fuente", "advertencia"],
            additionalProperties: false
        }
    },
    {
        name: "consultar_disponibilidad_repuesto",
        description: "Consulta el inventario local simulado de repuestos.",
        inputSchema: {
            type: "object",
            properties: {
                numero_parte: { type: "string", minLength: 1 }
            },
            required: ["numero_parte"],
            additionalProperties: false
        },
        outputSchema: {
            type: "object",
            properties: {
                numero_parte: { type: "string" },
                cantidad: { type: "integer", minimum: 0 },
                bodega: { type: "string" },
                fecha_reabastecimiento: { type: "string" },
                ultima_actualizacion: { type: "string" },
                ...sourceFields
            },
            required: ["numero_parte", "cantidad", "bodega", "ultima_actualizacion", "fuente", "advertencia"],
            additionalProperties: false
        }
    },
    {
        name: "registrar_lectura_sensor",
        description: "Guarda una lectura simulada y la evalua contra un rango configurado.",
        inputSchema: {
            type: "object",
            properties: {
                numero_serie: { type: "string", minLength: 1 },
                tipo: { type: "string", enum: ["presion", "temperatura"] },
                valor: { type: "number" },
                unidad: { type: "string", enum: ["bar", "degC"] }
            },
            required: ["numero_serie", "tipo", "valor", "unidad"],
            additionalProperties: false
        },
        outputSchema: {
            type: "object",
            properties: {
                numero_serie: { type: "string" },
                tipo: { type: "string", enum: ["presion", "temperatura"] },
                valor: { type: "number" },
                unidad: { type: "string", enum: ["bar", "degC"] },
                registrado: { type: "boolean" },
                timestamp: { type: "string" },
                dentro_de_rango: { type: "boolean" },
                nivel_alerta: { type: "string", enum: ["NORMAL", "BAJA", "ALTA"] },
                rango_configurado: {
                    type: "object",
                    properties: { min: { type: "number" }, max: { type: "number" }, unidad: { type: "string" } },
                    required: ["min", "max", "unidad"],
                    additionalProperties: false
                },
                mensaje: { type: "string" },
                ...sourceFields
            },
            required: ["numero_serie", "tipo", "valor", "unidad", "registrado", "timestamp", "dentro_de_rango", "nivel_alerta", "rango_configurado", "mensaje", "fuente", "advertencia"],
            additionalProperties: false
        }
    }
];

const toolsByName = new Map(toolDefinitions.map((tool) => [tool.name, tool]));

function sendResponse(response) {
    process.stdout.write(`${JSON.stringify(response)}\n`);
}

function hasRequestId(request) {
    return Object.prototype.hasOwnProperty.call(request, "id");
}

function sendResult(request, result) {
    if (!hasRequestId(request)) return;
    sendResponse({ jsonrpc: "2.0", id: request.id, result });
}

function sendRequestError(request, code, message, data) {
    if (!hasRequestId(request)) return;
    const error = { code, message };
    if (data !== undefined) error.data = data;
    sendResponse({ jsonrpc: "2.0", id: request.id, error });
}

function sendProtocolError(id, code, message, data) {
    const error = { code, message };
    if (data !== undefined) error.data = data;
    sendResponse({ jsonrpc: "2.0", id, error });
}

function isPlainObject(value) {
    return value !== null && typeof value === "object" && !Array.isArray(value);
}

function validateRequestEnvelope(request) {
    if (!isPlainObject(request)) return "The JSON-RPC message must be an object.";
    if (request.jsonrpc !== "2.0") return 'The "jsonrpc" member must be exactly "2.0".';
    if (typeof request.method !== "string" || request.method.length === 0) return 'The "method" member must be a non-empty string.';
    if (hasRequestId(request) && request.id !== null && typeof request.id !== "string" && typeof request.id !== "number") {
        return 'The "id" member must be a string, number, or null.';
    }
    if (request.params !== undefined && !isPlainObject(request.params) && !Array.isArray(request.params)) {
        return 'The "params" member must be an object or array.';
    }
    return null;
}

function validateInitializeParams(params) {
    if (!isPlainObject(params)) return "initialize requires an object in params.";
    if (typeof params.protocolVersion !== "string") return "protocolVersion must be a string.";
    if (!isPlainObject(params.capabilities)) return "capabilities must be an object.";
    if (!isPlainObject(params.clientInfo) || typeof params.clientInfo.name !== "string" || typeof params.clientInfo.version !== "string") {
        return "clientInfo must contain string name and version fields.";
    }
    return null;
}

function validateToolArguments(toolName, args) {
    if (!isPlainObject(args)) return ["arguments must be an object."];

    const rules = {
        consultar_specs: {
            required: ["modelo"], allowed: ["modelo"], strings: ["modelo"]
        },
        diagnosticar_falla: {
            required: ["codigo_mensaje", "modelo"], allowed: ["codigo_mensaje", "modelo"], strings: ["codigo_mensaje", "modelo"]
        },
        calcular_mantenimiento_pendiente: {
            required: ["numero_serie", "horas_operacion"], allowed: ["numero_serie", "horas_operacion"], strings: ["numero_serie"], numbers: ["horas_operacion"]
        },
        consultar_disponibilidad_repuesto: {
            required: ["numero_parte"], allowed: ["numero_parte"], strings: ["numero_parte"]
        },
        registrar_lectura_sensor: {
            required: ["numero_serie", "tipo", "valor", "unidad"], allowed: ["numero_serie", "tipo", "valor", "unidad"], strings: ["numero_serie", "tipo", "unidad"], numbers: ["valor"]
        }
    };

    const rule = rules[toolName];
    const errors = [];
    for (const name of rule.required) {
        if (!Object.prototype.hasOwnProperty.call(args, name)) errors.push(`Missing required argument: ${name}.`);
    }
    for (const name of Object.keys(args)) {
        if (!rule.allowed.includes(name)) errors.push(`Unexpected argument: ${name}.`);
    }
    for (const name of rule.strings || []) {
        if (Object.prototype.hasOwnProperty.call(args, name) && (typeof args[name] !== "string" || args[name].trim().length === 0)) {
            errors.push(`${name} must be a non-empty string.`);
        }
    }
    for (const name of rule.numbers || []) {
        if (Object.prototype.hasOwnProperty.call(args, name) && (typeof args[name] !== "number" || !Number.isFinite(args[name]))) {
            errors.push(`${name} must be a finite number.`);
        }
    }

    if (toolName === "diagnosticar_falla" && typeof args.codigo_mensaje === "string" && !/^\d{4} [AW]$/i.test(args.codigo_mensaje.trim())) {
        errors.push("codigo_mensaje must use the format '0000 A' or '0000 W'.");
    }
    if (toolName === "calcular_mantenimiento_pendiente" && typeof args.horas_operacion === "number" && args.horas_operacion < 0) {
        errors.push("horas_operacion must be greater than or equal to zero.");
    }
    if (toolName === "registrar_lectura_sensor") {
        if (typeof args.tipo === "string" && !["presion", "temperatura"].includes(args.tipo)) {
            errors.push("tipo must be either 'presion' or 'temperatura'.");
        }
        const expectedUnit = args.tipo === "presion" ? "bar" : args.tipo === "temperatura" ? "degC" : null;
        if (expectedUnit && args.unidad !== expectedUnit) errors.push(`unidad must be '${expectedUnit}' when tipo is '${args.tipo}'.`);
    }

    return errors;
}

function createToolResult(data) {
    return {
        content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
        structuredContent: data,
        isError: false
    };
}

function createToolError(message) {
    return {
        content: [{ type: "text", text: message }],
        isError: true
    };
}

function callTool(name, args) {
    switch (name) {
        case "consultar_specs":
            return herramientas.consultarSpecs(args.modelo);
        case "diagnosticar_falla":
            return herramientas.diagnosticarFalla(args.codigo_mensaje, args.modelo);
        case "calcular_mantenimiento_pendiente":
            return herramientas.calcularMantenimiento(args.numero_serie, args.horas_operacion);
        case "consultar_disponibilidad_repuesto":
            return herramientas.consultarRepuesto(args.numero_parte);
        case "registrar_lectura_sensor":
            return herramientas.registrarLectura(args.numero_serie, args.tipo, args.valor, args.unidad);
        default:
            throw new Error(`Unhandled tool: ${name}`);
    }
}

function handleToolCall(request) {
    if (!isPlainObject(request.params) || typeof request.params.name !== "string" || !isPlainObject(request.params.arguments)) {
        sendRequestError(request, -32602, "Invalid params", "tools/call requires string name and object arguments fields.");
        return;
    }

    const { name, arguments: args } = request.params;
    if (!toolsByName.has(name)) {
        sendRequestError(request, -32602, "Invalid params", `Unknown tool: ${name}`);
        return;
    }

    const validationErrors = validateToolArguments(name, args);
    if (validationErrors.length > 0) {
        sendResult(request, createToolError(validationErrors.join(" ")));
        return;
    }

    try {
        sendResult(request, createToolResult(callTool(name, args)));
    } catch (error) {
        if (error instanceof herramientas.ToolExecutionError) {
            sendResult(request, createToolError(error.message));
            return;
        }
        console.error("Unexpected tool error:", error);
        sendRequestError(request, -32603, "Internal error");
    }
}

function handleRequest(request) {
    switch (request.method) {
        case "initialize": {
            const validationError = validateInitializeParams(request.params);
            if (validationError) {
                sendRequestError(request, -32602, "Invalid params", validationError);
                return;
            }
            const requestedVersion = request.params.protocolVersion;
            const protocolVersion = SUPPORTED_PROTOCOL_VERSIONS.includes(requestedVersion)
                ? requestedVersion
                : SUPPORTED_PROTOCOL_VERSIONS[0];
            sendResult(request, {
                protocolVersion,
                serverInfo: {
                    name: "kaeser-assistant",
                    version: "1.1.0",
                    description: "Local industrial compressor maintenance assistant"
                },
                capabilities: { tools: { listChanged: false } },
                instructions: "Technical data and maintenance records are demonstrative. Always follow the equipment manual and plant safety procedures."
            });
            return;
        }
        case "notifications/initialized":
        case "notifications/cancelled":
            return;
        case "ping":
            sendResult(request, {});
            return;
        case "tools/list":
            if (request.params !== undefined && !isPlainObject(request.params)) {
                sendRequestError(request, -32602, "Invalid params", "tools/list params must be an object when provided.");
                return;
            }
            sendResult(request, { tools: toolDefinitions });
            return;
        case "tools/call":
            handleToolCall(request);
            return;
        default:
            sendRequestError(request, -32601, "Method not found");
    }
}

function processLine(line) {
    if (line.trim().length === 0) return;

    let request;
    try {
        request = JSON.parse(line);
    } catch {
        sendProtocolError(null, -32700, "Parse error");
        return;
    }

    const validationError = validateRequestEnvelope(request);
    if (validationError) {
        const id = isPlainObject(request) && hasRequestId(request) && ["string", "number"].includes(typeof request.id)
            ? request.id
            : null;
        sendProtocolError(id, -32600, "Invalid Request", validationError);
        return;
    }

    handleRequest(request);
}

let inputBuffer = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", (chunk) => {
    inputBuffer += chunk;

    if (inputBuffer.length > MAX_MESSAGE_LENGTH && !inputBuffer.includes("\n")) {
        sendProtocolError(null, -32700, "Parse error", "Message exceeds the maximum supported length.");
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

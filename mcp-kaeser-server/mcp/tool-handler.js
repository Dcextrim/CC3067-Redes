"use strict";

const herramientas = require("../herramientas.js");
const toolDefinitions = require("./tool-definitions.js");
const { isPlainObject, validateToolArguments } = require("./validation.js");

const toolsByName = new Map(toolDefinitions.map((tool) => [tool.name, tool]));

const toolExecutors = {
    consultar_specs: ({ modelo }) => herramientas.consultarSpecs(modelo),
    diagnosticar_falla: ({ codigo_mensaje, modelo }) => herramientas.diagnosticarFalla(codigo_mensaje, modelo),
    calcular_mantenimiento_pendiente: ({ numero_serie, horas_operacion }) => herramientas.calcularMantenimiento(numero_serie, horas_operacion),
    consultar_disponibilidad_repuesto: ({ numero_parte }) => herramientas.consultarRepuesto(numero_parte),
    registrar_lectura_sensor: ({ numero_serie, tipo, valor, unidad }) => herramientas.registrarLectura(numero_serie, tipo, valor, unidad)
};

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

function createToolHandler(responder) {
    const { sendRequestError, sendResult } = responder;

    return function handleToolCall(request) {
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
            sendResult(request, createToolResult(toolExecutors[name](args)));
        } catch (error) {
            if (error instanceof herramientas.ToolExecutionError) {
                sendResult(request, createToolError(error.message));
                return;
            }
            console.error("Unexpected tool error:", error);
            sendRequestError(request, -32603, "Internal error");
        }
    };
}

module.exports = { createToolHandler };

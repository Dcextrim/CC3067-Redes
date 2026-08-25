"use strict";

const toolRules = {
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

function isPlainObject(value) {
    return value !== null && typeof value === "object" && !Array.isArray(value);
}

function validateRequestEnvelope(request) {
    if (!isPlainObject(request)) return "The JSON-RPC message must be an object.";
    if (request.jsonrpc !== "2.0") return 'The "jsonrpc" member must be exactly "2.0".';
    if (typeof request.method !== "string" || request.method.length === 0) return 'The "method" member must be a non-empty string.';
    if (Object.prototype.hasOwnProperty.call(request, "id") && request.id !== null && typeof request.id !== "string" && typeof request.id !== "number") {
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

    const rule = toolRules[toolName];
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

module.exports = {
    isPlainObject,
    validateInitializeParams,
    validateRequestEnvelope,
    validateToolArguments
};

"use strict";

const fs = require("node:fs");
const path = require("node:path");

const diagnosticos = require("./data/diagnosticos.json");
const especificaciones = require("./data/especificaciones.json");
const inventario = require("./data/inventario.json");
const mantenimiento = require("./data/mantenimiento.json");
const rangosSensores = require("./data/rangos-sensores.json");

const SAFETY_WARNING = "Prototipo de demostracion: no sustituye el manual del equipo ni los procedimientos de seguridad de planta.";

const specificationSources = {
    SXC: "KAESER SXC Series brochure, Technical specifications",
    SX: "KAESER SX Series brochure, Technical specifications",
    SM: "KAESER SM Series brochure, Technical specifications",
    SK: "KAESER SK Series brochure, Technical data",
    ASK: "KAESER ASK Series brochure, Technical data"
};

class ToolExecutionError extends Error {
    constructor(message) {
        super(message);
        this.name = "ToolExecutionError";
    }
}

function normalizeKey(value) {
    return value.trim().toUpperCase().replace(/\s+/g, " ");
}

function getReadingsPath() {
    return process.env.KAESER_LECTURAS_PATH || path.join(__dirname, "data", "lecturas.json");
}

function readStoredReadings(filePath) {
    if (!fs.existsSync(filePath)) return [];

    let parsed;
    try {
        parsed = JSON.parse(fs.readFileSync(filePath, "utf8"));
    } catch (error) {
        throw new Error(`No fue posible leer el archivo de lecturas: ${error.message}`);
    }

    if (!Array.isArray(parsed)) throw new Error("El archivo de lecturas debe contener un arreglo JSON.");
    return parsed;
}

function persistReading(reading) {
    const readingsPath = getReadingsPath();
    const directory = path.dirname(readingsPath);
    const temporaryPath = `${readingsPath}.${process.pid}.tmp`;
    const readings = readStoredReadings(readingsPath);
    readings.push(reading);

    fs.mkdirSync(directory, { recursive: true });
    fs.writeFileSync(temporaryPath, `${JSON.stringify(readings, null, 2)}\n`, "utf8");
    fs.renameSync(temporaryPath, readingsPath);
}

function consultarSpecs(modelo) {
    const normalizedModel = normalizeKey(modelo);
    const specification = especificaciones[normalizedModel];
    if (!specification) throw new ToolExecutionError(`Modelo no encontrado en el catalogo: ${normalizedModel}.`);

    return {
        modelo: normalizedModel,
        ...specification,
        fuente: specificationSources[specification.familia] || specification.fuente,
        advertencia: SAFETY_WARNING
    };
}

function diagnosticarFalla(codigo, modelo) {
    const normalizedCode = normalizeKey(codigo);
    const diagnosis = diagnosticos[normalizedCode];
    if (!diagnosis) throw new ToolExecutionError(`Codigo no documentado en el prototipo: ${normalizedCode}.`);

    return {
        codigo_mensaje: normalizedCode,
        modelo_evaluado: normalizeKey(modelo),
        ...diagnosis,
        advertencia: SAFETY_WARNING
    };
}

function calcularMantenimiento(numeroSerie, horasOperacion) {
    const normalizedSerial = normalizeKey(numeroSerie);
    const equipment = mantenimiento.equipos[normalizedSerial];
    if (!equipment) throw new ToolExecutionError(`Numero de serie no encontrado en el plan simulado: ${normalizedSerial}.`);
    if (horasOperacion < equipment.ultimo_servicio_horas) {
        throw new ToolExecutionError("Las horas de operacion no pueden ser menores que las registradas en el ultimo servicio.");
    }

    const nextService = equipment.ultimo_servicio_horas + mantenimiento.intervalo_horas;
    const difference = nextService - horasOperacion;
    let status = "al_dia";
    let recommendation = "Continuar operacion y conservar el monitoreo programado.";

    if (difference <= 0) {
        status = "vencido";
        recommendation = difference === 0
            ? "El servicio corresponde en las horas de operacion actuales. Programarlo antes de continuar el ciclo."
            : `El servicio esta vencido por ${Math.abs(difference)} horas. Programarlo de inmediato.`;
    } else if (difference <= mantenimiento.umbral_proximo_horas) {
        status = "proximo";
        recommendation = "Agendar el mantenimiento preventivo proximo.";
    }

    return {
        numero_serie: normalizedSerial,
        estado: status,
        ultimo_servicio_horas: equipment.ultimo_servicio_horas,
        siguiente_servicio_horas: nextService,
        horas_restantes: Math.max(difference, 0),
        horas_vencidas: Math.max(-difference, 0),
        recomendacion: recommendation,
        fuente: "Plan local de mantenimiento simulado (data/mantenimiento.json)",
        advertencia: SAFETY_WARNING
    };
}

function consultarRepuesto(numeroParte) {
    const normalizedPart = normalizeKey(numeroParte);
    const part = inventario[normalizedPart];
    if (!part) throw new ToolExecutionError(`Numero de parte no encontrado en el inventario simulado: ${normalizedPart}.`);

    return {
        numero_parte: normalizedPart,
        ...part,
        fuente: "Inventario local simulado (data/inventario.json)",
        advertencia: SAFETY_WARNING
    };
}

function registrarLectura(numeroSerie, tipo, valor, unidad) {
    const range = rangosSensores[tipo];
    if (!range) throw new ToolExecutionError(`Tipo de sensor no configurado: ${tipo}.`);
    if (unidad !== range.unidad) throw new ToolExecutionError(`La unidad configurada para ${tipo} es ${range.unidad}.`);

    const normalizedSerial = normalizeKey(numeroSerie);
    const alertLevel = valor < range.min ? "BAJA" : valor > range.max ? "ALTA" : "NORMAL";
    const withinRange = alertLevel === "NORMAL";
    const reading = {
        numero_serie: normalizedSerial,
        tipo,
        valor,
        unidad,
        timestamp: new Date().toISOString()
    };

    try {
        persistReading(reading);
    } catch (error) {
        throw new ToolExecutionError(`No fue posible guardar la lectura: ${error.message}`);
    }

    return {
        ...reading,
        registrado: true,
        dentro_de_rango: withinRange,
        nivel_alerta: alertLevel,
        rango_configurado: { ...range },
        mensaje: withinRange
            ? "Lectura registrada dentro del rango configurado."
            : `Lectura ${alertLevel.toLowerCase()} fuera del rango configurado; seguir el procedimiento de seguridad de planta.`,
        fuente: "Rangos y lectura local simulados (data/rangos-sensores.json y data/lecturas.json)",
        advertencia: SAFETY_WARNING
    };
}

module.exports = {
    ToolExecutionError,
    consultarSpecs,
    diagnosticarFalla,
    calcularMantenimiento,
    consultarRepuesto,
    registrarLectura
};

"use strict";

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

module.exports = toolDefinitions;

// Importamos nuestras herramientas desde el archivo modular
const herramientas = require('./herramientas.js');

// Función segura para enviar respuestas por stdout
function sendResponse(response) {
    process.stdout.write(JSON.stringify(response) + '\n');
}

// Función para enviar errores JSON-RPC
function sendError(id, code, message) {
    sendResponse({ jsonrpc: "2.0", id, error: { code, message } });
}

// Se escuchan los mensajes del cliente MCP (Anfitrión)
process.stdin.on('data', (data) => {
    const mensajes = data.toString().split('\n');
    for (const mensaje of mensajes) {
        if (!mensaje.trim()) continue;
        
        try {
            const req = JSON.parse(mensaje);
            
            // ==========================================
            // 1. Fase de Inicialización
            // ==========================================
            if (req.method === 'initialize') {
                sendResponse({
                    jsonrpc: "2.0",
                    id: req.id,
                    result: {
                        protocolVersion: "2024-11-05",
                        serverInfo: { name: "kaeser-assistant", version: "1.0.0" },
                        capabilities: { tools: {} }
                    }
                });
            }
            else if (req.method === 'notifications/initialized') {
            }
            
            // ==========================================
            // 2. Fase de Listado de Herramientas
            // ==========================================
            else if (req.method === 'tools/list') {
                sendResponse({
                    jsonrpc: "2.0",
                    id: req.id,
                    result: {
                        tools: [
                            {
                                name: "consultar_specs",
                                description: "Consulta potencia, presión, caudal y dimensiones de un modelo de compresor.",
                                inputSchema: {
                                    type: "object",
                                    properties: { modelo: { type: "string", description: "Ej. SXC 3, SM 13" } },
                                    required: ["modelo"]
                                }
                            },
                            {
                                name: "diagnosticar_falla",
                                description: "Interpreta un mensaje documentado del SIGMA CONTROL 2 y propone verificaciones.",
                                inputSchema: {
                                    type: "object",
                                    properties: { 
                                        codigo_mensaje: { type: "string", description: "Ej. 0044 A" },
                                        modelo: { type: "string" }
                                    },
                                    required: ["codigo_mensaje", "modelo"]
                                }
                            },
                            {
                                name: "calcular_mantenimiento_pendiente",
                                description: "Compara horas acumuladas con el plan simulado de mantenimiento.",
                                inputSchema: {
                                    type: "object",
                                    properties: { 
                                        numero_serie: { type: "string" },
                                        horas_operacion: { type: "number" }
                                    },
                                    required: ["numero_serie", "horas_operacion"]
                                }
                            },
                            {
                                name: "consultar_disponibilidad_repuesto",
                                description: "Consulta el inventario local simulado de repuestos KAESER.",
                                inputSchema: {
                                    type: "object",
                                    properties: { numero_parte: { type: "string" } },
                                    required: ["numero_parte"]
                                }
                            },
                            {
                                name: "registrar_lectura_sensor",
                                description: "Guarda una lectura simulada y la evalúa contra un rango configurado.",
                                inputSchema: {
                                    type: "object",
                                    properties: { 
                                        numero_serie: { type: "string" },
                                        tipo: { type: "string", enum: ["presion", "temperatura"] },
                                        valor: { type: "number" },
                                        unidad: { type: "string" }
                                    },
                                    required: ["numero_serie", "tipo", "valor", "unidad"]
                                }
                            }
                        ]
                    }
                });
            }
            
            // ==========================================
            // 3. Fase de Ejecución de Herramientas
            // ==========================================
            else if (req.method === 'tools/call') {
                const toolName = req.params.name;
                const args = req.params.arguments;
                let resultado = {};

                switch (toolName) {
                    case 'consultar_specs':
                        resultado = herramientas.consultarSpecs(args.modelo);
                        break;
                    case 'diagnosticar_falla':
                        resultado = herramientas.diagnosticarFalla(args.codigo_mensaje, args.modelo);
                        break;
                    case 'calcular_mantenimiento_pendiente':
                        resultado = herramientas.calcularMantenimiento(args.numero_serie, args.horas_operacion);
                        break;
                    case 'consultar_disponibilidad_repuesto':
                        resultado = herramientas.consultarRepuesto(args.numero_parte);
                        break;
                    case 'registrar_lectura_sensor':
                        resultado = herramientas.registrarLectura(args.numero_serie, args.tipo, args.valor, args.unidad);
                        break;
                    default:
                        sendError(req.id, -32601, "Herramienta no encontrada");
                        return; // Si no existe la herramienta se termin la ejecucion
                }

                // Enviar la respuesta como lo exige MCP
                sendResponse({
                    jsonrpc: "2.0",
                    id: req.id,
                    result: {
                        content: [{ type: "text", text: JSON.stringify(resultado, null, 2) }]
                    }
                });

            } else {
                // Si el método JSON-RPC no es soportado, manda mensaje de error
                if (req.id) sendError(req.id, -32601, "Method not found");
            }
        } catch (e) {
            // Si hay errores se imprimen logs en la terminal sin afectar el protocolo stdout
            console.error("Error procesando mensaje:", e);
        }
    }
});
const diagnosticos = require('./data/diagnosticos.json');
const especificaciones = require('./data/especificaciones.json');
const inventario = require('./data/inventario.json');

function consultarSpecs(modelo) {
    return especificaciones[modelo] || { error: "Modelo no encontrado en catálogo de demostración." };
}

function diagnosticarFalla(codigo, modelo) {
    const resultado = diagnosticos[codigo] || { error: "Código no documentado en el prototipo." };
    resultado.modelo_evaluado = modelo;
    return resultado;
}

function calcularMantenimiento(numero_serie, horas) {
    const limite = 3000;
    const horasRestantes = limite - (horas % limite);
    return {
        estado: horasRestantes < 200 ? "proximo" : "al_dia",
        siguiente_servicio_horas: horas + horasRestantes,
        horas_restantes: horasRestantes,
        recomendacion: horasRestantes < 200 ? "Agendar mantenimiento pronto" : "Operación normal",
        aviso: "Datos basados en plan simulado."
    };
}

function consultarRepuesto(numero_parte) {
    return inventario[numero_parte] || { error: "Número de parte no existe en inventario local." };
}

function registrarLectura(numero_serie, tipo, valor, unidad) {
    const alerta = (tipo === 'temperatura' && valor > 90) ? "ALTA" : "NORMAL";
    return {
        registrado: true,
        timestamp: new Date().toISOString(),
        dentro_de_rango: alerta === "NORMAL",
        nivel_alerta: alerta,
        mensaje: alerta === "ALTA" ? "Lectura fuera de rango, revise ventilación." : "Lectura guardada localmente (Demo)."
    };
}

// Exportamos las funciones para poder usarlas en server.js
module.exports = {
    consultarSpecs,
    diagnosticarFalla,
    calcularMantenimiento,
    consultarRepuesto,
    registrarLectura
};
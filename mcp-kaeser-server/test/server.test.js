"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawn } = require("node:child_process");

const tools = require("../herramientas.js");
const serverPath = path.join(__dirname, "..", "server.js");

function createTemporaryReadingsPath() {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "kaeser-mcp-test-"));
    return { directory, file: path.join(directory, "lecturas.json") };
}

function removeTemporaryReadings({ directory, file }) {
    if (fs.existsSync(file)) fs.unlinkSync(file);
    if (fs.existsSync(directory)) fs.rmdirSync(directory);
}

function startServer(readingsPath) {
    const child = spawn(process.execPath, [serverPath], {
        cwd: path.dirname(serverPath),
        env: { ...process.env, KAESER_LECTURAS_PATH: readingsPath },
        stdio: ["pipe", "pipe", "pipe"]
    });

    const messages = [];
    let stdoutBuffer = "";
    let stderr = "";
    const waiters = [];

    function resolveWaiters() {
        for (let index = waiters.length - 1; index >= 0; index -= 1) {
            if (messages.length >= waiters[index].count) {
                clearTimeout(waiters[index].timer);
                waiters[index].resolve(messages);
                waiters.splice(index, 1);
            }
        }
    }

    child.stdout.setEncoding("utf8");
    child.stdout.on("data", (chunk) => {
        stdoutBuffer += chunk;
        let newlineIndex;
        while ((newlineIndex = stdoutBuffer.indexOf("\n")) !== -1) {
            const line = stdoutBuffer.slice(0, newlineIndex);
            stdoutBuffer = stdoutBuffer.slice(newlineIndex + 1);
            if (line.trim()) messages.push(JSON.parse(line));
        }
        resolveWaiters();
    });
    child.stderr.setEncoding("utf8");
    child.stderr.on("data", (chunk) => { stderr += chunk; });

    return {
        child,
        messages,
        getStderr: () => stderr,
        send(message) {
            child.stdin.write(`${JSON.stringify(message)}\n`);
        },
        write(raw) {
            child.stdin.write(raw);
        },
        waitForMessages(count, timeoutMs = 3000) {
            if (messages.length >= count) return Promise.resolve(messages);
            return new Promise((resolve, reject) => {
                const timer = setTimeout(() => reject(new Error(`Timed out waiting for ${count} server messages; received ${messages.length}. stderr=${stderr}`)), timeoutMs);
                waiters.push({ count, resolve, reject, timer });
            });
        },
        stop() {
            child.stdin.end();
            return new Promise((resolve, reject) => {
                child.once("error", reject);
                child.once("close", (code) => resolve(code));
            });
        }
    };
}

test("technical specifications include source-backed SFC variants", () => {
    const result = tools.consultarSpecs("ask 34 sfc");
    assert.equal(result.modelo, "ASK 34 SFC");
    assert.deepEqual(result.caudal_m3_min, { min: 0.94, max: 3.6 });
    assert.equal(result.potencia_kw, 18.5);
    assert.match(result.fuente, /ASK Series/);
    assert.match(result.advertencia, /no sustituye/);
});

test("proposal diagnostic codes 0015 A and 0044 A are available", () => {
    const temperature = tools.diagnosticarFalla("0015 a", "SM 13");
    const pressure = tools.diagnosticarFalla("0044 A", "SM 13");

    assert.match(temperature.descripcion, /temperature/i);
    assert.equal(temperature.requiere_servicio_autorizado, false);
    assert.match(pressure.descripcion, /pressure/i);
    assert.equal(pressure.requiere_servicio_autorizado, true);
    assert.ok(pressure.acciones.some((action) => /acoplamiento/.test(action)));
});

test("maintenance calculation reaches all three states", () => {
    assert.equal(tools.calcularMantenimiento("KC-2291", 4200).estado, "al_dia");
    assert.equal(tools.calcularMantenimiento("KC-2291", 5900).estado, "proximo");

    const due = tools.calcularMantenimiento("KC-2291", 6000);
    assert.equal(due.estado, "vencido");
    assert.equal(due.horas_restantes, 0);
    assert.equal(due.horas_vencidas, 0);

    const overdue = tools.calcularMantenimiento("KC-2291", 6150);
    assert.equal(overdue.estado, "vencido");
    assert.equal(overdue.horas_vencidas, 150);
});

test("sensor readings validate pressure and persist locally", () => {
    const temporary = createTemporaryReadingsPath();
    const previousPath = process.env.KAESER_LECTURAS_PATH;
    process.env.KAESER_LECTURAS_PATH = temporary.file;

    try {
        const result = tools.registrarLectura("KC-2291", "presion", 9.2, "bar");
        assert.equal(result.registrado, true);
        assert.equal(result.nivel_alerta, "ALTA");
        assert.equal(result.dentro_de_rango, false);
        assert.deepEqual(result.rango_configurado, { min: 5.5, max: 8, unidad: "bar" });

        const secondResult = tools.registrarLectura("KC-2291", "temperatura", 70, "degC");
        assert.equal(secondResult.nivel_alerta, "NORMAL");

        const stored = JSON.parse(fs.readFileSync(temporary.file, "utf8"));
        assert.equal(stored.length, 2);
        assert.equal(stored[0].numero_serie, "KC-2291");
        assert.equal(stored[0].valor, 9.2);
    } finally {
        if (previousPath === undefined) delete process.env.KAESER_LECTURAS_PATH;
        else process.env.KAESER_LECTURAS_PATH = previousPath;
        removeTemporaryReadings(temporary);
    }
});

test("stdio server handles framing, validation, structured output, and JSON-RPC errors", async () => {
    const temporary = createTemporaryReadingsPath();
    const server = startServer(temporary.file);

    try {
        server.send({
            jsonrpc: "2.0",
            id: 1,
            method: "initialize",
            params: {
                protocolVersion: "2025-11-25",
                capabilities: {},
                clientInfo: { name: "test-client", version: "1.0.0" }
            }
        });
        await server.waitForMessages(1);
        assert.equal(server.messages[0].result.protocolVersion, "2025-11-25");

        const listRequest = `${JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} })}\n`;
        server.write(listRequest.slice(0, 20));
        await new Promise((resolve) => setTimeout(resolve, 100));
        assert.equal(server.messages.length, 1, "a partial frame must not be parsed early");
        server.write(listRequest.slice(20));
        await server.waitForMessages(2);
        assert.equal(server.messages[1].result.tools.length, 5);
        assert.ok(server.messages[1].result.tools.every((tool) => tool.outputSchema));

        server.send({
            jsonrpc: "2.0",
            id: 3,
            method: "tools/call",
            params: { name: "diagnosticar_falla", arguments: { codigo_mensaje: "0044 A", modelo: "SM 13" } }
        });
        await server.waitForMessages(3);
        assert.equal(server.messages[2].result.isError, false);
        assert.equal(server.messages[2].result.structuredContent.codigo_mensaje, "0044 A");

        server.send({
            jsonrpc: "2.0",
            id: 4,
            method: "tools/call",
            params: { name: "calcular_mantenimiento_pendiente", arguments: { numero_serie: "KC-2291", horas_operacion: -1 } }
        });
        await server.waitForMessages(4);
        assert.equal(server.messages[3].result.isError, true);

        server.send({
            jsonrpc: "2.0",
            id: 5,
            method: "tools/call",
            params: { name: "registrar_lectura_sensor", arguments: { numero_serie: "KC-2291", tipo: "presion", valor: 7, unidad: "psi" } }
        });
        await server.waitForMessages(5);
        assert.equal(server.messages[4].result.isError, true);

        server.send({ jsonrpc: "2.0", id: 0, method: "does/not/exist" });
        await server.waitForMessages(6);
        assert.equal(server.messages[5].id, 0);
        assert.equal(server.messages[5].error.code, -32601);

        server.write("not-json\n");
        await server.waitForMessages(7);
        assert.equal(server.messages[6].id, null);
        assert.equal(server.messages[6].error.code, -32700);
    } finally {
        const exitCode = await server.stop();
        assert.equal(exitCode, 0);
        assert.equal(server.getStderr(), "");
        removeTemporaryReadings(temporary);
    }
});
